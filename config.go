package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	BaseFolder   string
	Folders      []*Folder
	Ignore       []string
	UseGitignore bool `json:"use_gitignore"`
}

type Folder struct {
	Name string
	Path string
}

func readConfig(fp string) *Config {
	f, err := os.Open(fp)
	if err != nil {
		log.Fatalf("Cannot read config file at %s: %v\n", fp, err)
	}
	defer f.Close()

	var c *Config
	json.NewDecoder(f).Decode(&c)

	addFolderNames(c)

	for _, f := range c.Folders {
		fmt.Println(f.Name, " = ", f.Path)
	}

	return c
}

func addFolderNames(c *Config) {
	for _, f := range c.Folders {
		if f.Name == "" {
			f.Name = filepath.Base(f.Path)
		}
	}
}
