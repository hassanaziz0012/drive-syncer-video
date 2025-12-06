package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/hassanaziz0012/drive-syncer-video/internal/models"
)

func ReadConfig(fp string) *models.Config {
	f, err := os.Open(fp)
	if err != nil {
		log.Fatalf("Cannot read config file at %s: %v\n", fp, err)
	}
	defer f.Close()

	var c *models.Config
	json.NewDecoder(f).Decode(&c)

	addFolderNames(c)

	return c
}

func addFolderNames(c *models.Config) {
	for _, f := range c.Folders {
		if f.Name == "" {
			f.Name = filepath.Base(f.Path)
		}
	}
}
