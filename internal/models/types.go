package models

import "google.golang.org/api/drive/v3"

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

type Drive struct {
	Config *Config
	Api    *drive.Service
}

type UploadProgress struct {
	Folder        *Folder
	CurFile       string
	TotalFiles    int
	UploadedFiles int
}
