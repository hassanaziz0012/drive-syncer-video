package utils

import (
	"fmt"
	"strings"

	"github.com/hassanaziz0012/drive-syncer-video/internal/models"
)

func RenderProgressBar(progress *models.UploadProgress) string {
	uploaded := progress.UploadedFiles
	total := progress.TotalFiles

	percent := float64(uploaded) / float64(total) * 100
	barwidth := 50

	filled := int(percent / 100 * float64(barwidth))
	empty := barwidth - filled

	bar := fmt.Sprintf("%s [%s%s] %.2f%% (uploading \"%s\") (%d/%d)",
		progress.Folder.Name,
		strings.Repeat("#", filled),
		strings.Repeat("-", empty),
		percent,
		progress.CurFile,
		progress.UploadedFiles,
		progress.TotalFiles,
	)

	return bar
}
