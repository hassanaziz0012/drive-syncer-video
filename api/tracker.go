package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/hassanaziz0012/drive-syncer-video/internal/models"
)

type Tracker struct {
	basefolder_id string
	drv           *models.Drive
	folder        *models.Folder
	folderId      string
	nodes         map[string]*Node
}

func NewTracker(drv *models.Drive, folder *models.Folder, basefolder_id string) *Tracker {
	t := Tracker{
		drv:           drv,
		folder:        folder,
		basefolder_id: basefolder_id,
	}
	return &t
}

func (t *Tracker) projectExists() (bool, error) {
	query := fmt.Sprintf("mimeType='%s' and '%s' in parents and trashed=false", FolderMimeType, t.basefolder_id)
	if r, err := t.drv.Api.Files.List().Q(query).Fields("files(id, name)").PageSize(1).Do(); err == nil {
		switch len(r.Files) {
		case 0:
			return false, nil
		case 1:
			t.folderId = r.Files[0].Id
			return true, nil
		default: // impossible to reach because of PageSize(1)
			return false, errors.New("multiple project folders found")
		}
	} else {
		return false, err
	}
}

func (t *Tracker) mapSubfolders(current map[string]string) {
	log.Println("mapping subfolders from GDrive to local")
	for k, v := range t.nodes {
		if v.isFolder {
			current[k] = v.id
		}
	}
}

func (t *Tracker) walkProjectTree() {
	fmt.Print("==== ==== ==== PROJECT TREE START ==== ==== ====\n")

	nodes := map[string]*Node{}
	queue := []string{t.folderId}

	folderPaths := map[string]string{}
	folderPaths[t.folderId] = t.folder.Path

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		basepath := folderPaths[current]

		pageToken := ""

		fmt.Printf("Walking %s\n", basepath)

		for {
			query := fmt.Sprintf("'%s' in parents and trashed=false", current)
			req := t.drv.Api.Files.List().Fields("files(id, name, mimeType, size, modifiedTime, sha256Checksum, parents)").Q(query)

			if pageToken != "" {
				req.PageToken(pageToken)
			}
			r, err := req.Do()
			if err != nil {
				fmt.Println(err)
				break
			}

			for _, f := range r.Files {
				if f.MimeType == FolderMimeType {
					folderPaths[f.Id] = filepath.Join(basepath, f.Name)
					queue = append(queue, f.Id)
				}

				var parent string
				if len(f.Parents) == 1 {
					parent = f.Parents[0]
				}

				localAbsPath := filepath.Join(basepath, f.Name)
				nodes[localAbsPath] = &Node{
					id:           f.Id,
					name:         f.Name,
					mimeType:     f.MimeType,
					isFolder:     f.MimeType == FolderMimeType,
					modifiedTime: f.ModifiedTime,
					checksum:     f.Sha256Checksum,
					parent:       parent,
				}
			}

			if r.NextPageToken == "" {
				break
			}
			pageToken = r.NextPageToken
		}
	}

	t.nodes = nodes

	for k, v := range t.nodes {
		fmt.Println(k, " = ", v.id)
	}

	fmt.Print("==== ==== ==== PROJECT TREE END ==== ==== ====\n")
}

func (t *Tracker) shouldUpload(path string) (bool, error) {
	if node, ok := t.nodes[path]; ok {
		if node.isFolder {
			return false, nil
		} else {
			f, err := os.Open(path)
			if err != nil {
				return false, err
			}
			defer f.Close()

			local_checksum, err := t.getChecksum(f)
			if err != nil {
				return false, err
			}
			return local_checksum != node.checksum, nil
		}
	}

	return true, nil
}

func (t *Tracker) getChecksum(f *os.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	sum := hash.Sum(nil)
	return hex.EncodeToString(sum), nil
}

type Node struct {
	id           string
	name         string
	mimeType     string
	isFolder     bool
	size         int
	modifiedTime string
	checksum     string
	parent       string
}
