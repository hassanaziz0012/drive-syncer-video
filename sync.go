package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"google.golang.org/api/drive/v3"
)

const FolderMimeType = "application/vnd.google-apps.folder"

func createBaseFolder(drv *Drive) (base *drive.File) {
	query := fmt.Sprintf("mimeType='%s' and name='%s' and trashed=false", FolderMimeType, drv.config.BaseFolder)
	r, err := drv.api.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		log.Fatalf("failed to query base folder. cannot proceed with sync: %v", err)
	}

	if len(r.Files) > 1 {
		log.Fatalf("found multiple base folder matches")
	}

	for _, f := range r.Files {
		if f.Name == drv.config.BaseFolder {
			return f
		}
	}

	newbase := drive.File{
		Name:     drv.config.BaseFolder,
		MimeType: FolderMimeType,
	}
	f, err := drv.api.Files.Create(&newbase).Do()
	if err != nil {
		log.Fatalf("failed to create base folder %s: %v\n", drv.config.BaseFolder, err)
	}
	return f
}

func UploadFolder(drv *Drive, folder *Folder) {
	log.Printf("starting upload process for folder %s\n", folder.Name)

	base := createBaseFolder(drv)

	if err := delPrevFolder(drv, base.Id, folder); err != nil {
		log.Printf("failed to delete previous folder. you may see duplicates in your upload. error: %v\n", err)
	}

	// metadata := drive.File{
	// 	Name:     folder.Name,
	// 	Parents:  []string{base.Id},
	// 	MimeType: FolderMimeType,
	// }
	// projFolder, err := drv.api.Files.Create(&metadata).Do()
	// if err != nil {
	// 	log.Fatalf("failed to create project folder %s: %v\n", folder.Name, err)
	// }

	subfolders := map[string]string{
		base.Name: base.Id,
	}

	filepath.WalkDir(folder.Path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			subname := filepath.Base(path)
			parent_name := filepath.Base(filepath.Dir(path))

			if _, ok := subfolders[subname]; !ok {
				parent_id, p_ok := subfolders[parent_name]
				var parent string
				if p_ok {
					parent = parent_id
				} else {
					parent = base.Id
				}
				sub, err := createSubFolder(drv, parent, d)
				if err != nil {
					log.Printf("failed to create subfolder %s: %v\n", subname, err)
					return err
				}
				subfolders[subname] = sub.Id
				log.Printf("created subfolder %s in parent %s", subname, parent_name)
			}
			return nil
		} else {
			parent_name := filepath.Base(filepath.Dir(path))
			if parent_id, ok := subfolders[parent_name]; ok {
				return uploadFile(drv, path, d, parent_id)
			} else {
				err := fmt.Sprintf("failed to find parent folder %s for file %s. upload failed", parent_name, d.Name())
				log.Printf(err)
				return errors.New(err)
			}
		}
	})
}

func createSubFolder(drv *Drive, parent_id string, d fs.DirEntry) (*drive.File, error) {
	sub := drive.File{
		Name:     d.Name(),
		MimeType: FolderMimeType,
		Parents:  []string{parent_id},
	}
	r, err := drv.api.Files.Create(&sub).Do()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func uploadFile(drv *Drive, path string, d fs.DirEntry, parent_id string) error {
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

	r, err := drv.api.Files.Create(&metadata).Media(f).Do()
	if err != nil {
		log.Printf("failed to upload file %s: %v\n", d.Name(), err)
		return err
	}

	log.Printf("uploaded %s to drive (%s)\n", d.Name(), r.Id)
	return nil
}

func delPrevFolder(drv *Drive, basefolder_id string, folder *Folder) error {
	query := fmt.Sprintf("mimeType='%s' and name='%s' and '%s' in parents and trashed=false", FolderMimeType, folder.Name, basefolder_id)
	r, err := drv.api.Files.List().Q(query).Fields("files(id, name)").PageSize(1).Do()
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

	err = drv.api.Files.Delete(r.Files[0].Id).Do()
	if err != nil {
		return err
	}

	return nil
}
