package api

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hassanaziz0012/drive-syncer-video/auth"
	"github.com/hassanaziz0012/drive-syncer-video/internal/config"
	"github.com/hassanaziz0012/drive-syncer-video/internal/models"
	"google.golang.org/api/drive/v3"
)

const FolderMimeType = "application/vnd.google-apps.folder"

var CONFIG = config.ReadConfig("config.json")

var DRV = auth.NewDrive(CONFIG)

func createBaseFolder(drv *models.Drive) (base *drive.File) {
	query := fmt.Sprintf("mimeType='%s' and name='%s' and trashed=false", FolderMimeType, drv.Config.BaseFolder)
	r, err := drv.Api.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		log.Fatalf("failed to query base folder. cannot proceed with sync: %v", err)
	}

	if len(r.Files) > 1 {
		log.Fatalf("found multiple base folder matches")
	}

	for _, f := range r.Files {
		if f.Name == drv.Config.BaseFolder {
			return f
		}
	}

	newbase := drive.File{
		Name:     drv.Config.BaseFolder,
		MimeType: FolderMimeType,
	}
	f, err := drv.Api.Files.Create(&newbase).Do()
	if err != nil {
		log.Fatalf("failed to create base folder %s: %v\n", drv.Config.BaseFolder, err)
	}
	return f
}

func CountTotalFiles(to_ignore []string, fp string) int {
	var totalFiles int
	filepath.WalkDir(fp, func(path string, d fs.DirEntry, err error) error {
		for _, ignore := range to_ignore {
			ignored_dir := filepath.Join(fp, ignore)
			shouldSkip := path == ignored_dir || strings.HasPrefix(path, ignored_dir+string(os.PathSeparator))
			if shouldSkip {
				log.Printf("skipping %s\n", path)
				if d.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}
		}
		if !d.IsDir() {
			totalFiles++
		}
		return nil
	})

	return totalFiles
}

func createProjectFolder(drv *models.Drive, basefolder_id string, name string) (*drive.File, error) {
	return createSubFolder(drv, basefolder_id, name)
}

func UploadFolder(folder *models.Folder, progressChan chan *models.UploadProgress) {
	defer close(progressChan)

	log.Printf("starting upload process for folder %s\n", folder.Name)

	base := createBaseFolder(DRV)

	// if err := delPrevFolder(DRV, base.Id, folder); err != nil {
	// 	log.Printf("failed to delete previous folder. you may see duplicates in your upload. error: %v\n", err)
	// }

	// metadata := drive.File{
	// 	Name:     folder.Name,
	// 	Parents:  []string{base.Id},
	// 	MimeType: FolderMimeType,
	// }
	// projFolder, err := DRV.api.Files.Create(&metadata).Do()
	// if err != nil {
	// 	log.Fatalf("failed to create project folder %s: %v\n", folder.Name, err)
	// }

	subfolders := map[string]string{}

	tracker := NewTracker(DRV, folder, base.Id)
	projectExists, err := tracker.projectExists()
	if err != nil {
		log.Fatalf("%s: checking if project exists already: %v\n", folder.Name, err)
	}
	if projectExists {
		log.Printf("%s: project already exists. walking project tree.\n", folder.Name)
		tracker.walkProjectTree()
		tracker.mapSubfolders(subfolders)
	}

	to_ignore := make([]string, 0, len(DRV.Config.Ignore))
	to_ignore = append(to_ignore, DRV.Config.Ignore...)

	if DRV.Config.UseGitignore {
		gitignore, err := readGitignore(folder)
		if err != nil {
			log.Printf("cannot read gitignore. will upload all files. error: %v\n", err)
		}
		to_ignore = append(to_ignore, gitignore...)
	}

	totalFiles := CountTotalFiles(to_ignore, folder.Path)
	progress := &models.UploadProgress{
		Folder:        folder,
		TotalFiles:    totalFiles,
		UploadedFiles: 0,
	}
	progressChan <- progress

	filepath.WalkDir(folder.Path, func(path string, d fs.DirEntry, err error) error {
		if path == folder.Path {
			if projectExists {
				log.Println("project folder already exists. assigning ID.")
				subfolders[folder.Path] = tracker.folderId
				return nil
			} else {
				r, err := createProjectFolder(DRV, base.Id, folder.Name)
				if err != nil {
					log.Fatalf("failed to create project folder: %v\n", err)
				}
				subfolders[folder.Path] = r.Id
				return nil
			}
		}

		for _, ignore := range to_ignore {
			ignored_dir := filepath.Join(folder.Path, ignore)
			shouldSkip := path == ignored_dir || strings.HasPrefix(path, ignored_dir+string(os.PathSeparator))
			if shouldSkip {
				log.Printf("skipping %s\n", path)
				if d.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}
		}

		shouldUpload, err := tracker.shouldUpload(path)
		if err != nil {
			log.Printf("%s: cannot upload %s: %v\n", folder.Name, path, err)
			return err
		}
		if !shouldUpload {
			log.Printf("skipping %s (unchanged)\n", path)
			return nil
		}

		if d.IsDir() {
			subname := filepath.Base(path)
			parent_path := filepath.Dir(path)

			if _, ok := subfolders[path]; !ok {
				parent_id, p_ok := subfolders[parent_path]
				var parent string
				if p_ok {
					parent = parent_id
				} else {
					parent = base.Id
				}
				sub, err := createSubFolder(DRV, parent, d.Name())
				if err != nil {
					log.Printf("failed to create subfolder %s: %v\n", subname, err)
					return err
				}
				subfolders[path] = sub.Id
				log.Printf("created subfolder %s in parent %s", subname, parent_path)
			}
			return nil
		} else {
			parent_path := filepath.Dir(path)
			if parent_id, ok := subfolders[parent_path]; ok {
				progress.CurFile = d.Name()
				progressChan <- progress
				err := uploadFile(DRV, path, d, parent_id)
				if err != nil {
					return err
				}
				progress.UploadedFiles++
				progressChan <- progress
				return nil
			} else {
				err := fmt.Sprintf("failed to find parent folder %s for file %s. upload failed", parent_path, d.Name())
				log.Printf(err)
				return errors.New(err)
			}
		}
	})
}

func createSubFolder(drv *models.Drive, parent_id string, name string) (*drive.File, error) {
	sub := drive.File{
		Name:     name,
		MimeType: FolderMimeType,
		Parents:  []string{parent_id},
	}
	r, err := drv.Api.Files.Create(&sub).Do()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func uploadFile(drv *models.Drive, path string, d fs.DirEntry, parent_id string) error {
	metadata := drive.File{
		Name:    d.Name(),
		Parents: []string{parent_id},
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("failed to open file %s: %v\n", d.Name(), err)
		return err
	}
	defer f.Close()

	r, err := drv.Api.Files.Create(&metadata).Media(f).Do()
	if err != nil {
		log.Printf("failed to upload file %s: %v\n", d.Name(), err)
		return err
	}

	log.Printf("uploaded %s to drive (%s)\n", d.Name(), r.Id)
	return nil
}

func delPrevFolder(drv *models.Drive, basefolder_id string, folder *models.Folder) error {
	query := fmt.Sprintf("mimeType='%s' and name='%s' and '%s' in parents and trashed=false", FolderMimeType, folder.Name, basefolder_id)
	r, err := drv.Api.Files.List().Q(query).Fields("files(id, name)").PageSize(1).Do()
	if err != nil {
		return err
	}

	log.Printf("attempting to delete previous folder for project %s\n", folder.Name)
	if len(r.Files) == 0 {
		log.Println("previous folder does not exist. uploading new")
		return nil
	} else if len(r.Files) == 1 {
		log.Printf("deleting previous folder %s\n", folder.Name)
	} else if len(r.Files) > 1 {
		return errors.New("multiple folder matches returned from api. cannot delete")
	}

	err = drv.Api.Files.Delete(r.Files[0].Id).Do()
	if err != nil {
		return err
	}

	return nil
}

func readGitignore(folder *models.Folder) (to_ignore []string, err error) {
	f, err := os.Open(filepath.Join(folder.Path, ".gitignore"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		to_ignore = append(to_ignore, strings.TrimSpace(s.Text()))
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	return to_ignore, nil
}
