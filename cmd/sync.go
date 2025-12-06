package cmd

import (
	"fmt"
	"sync"

	"github.com/hassanaziz0012/drive-syncer-video/api"
	"github.com/hassanaziz0012/drive-syncer-video/internal/models"
	"github.com/hassanaziz0012/drive-syncer-video/utils"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Start syncing projects to Google Drive",
	Run: func(cmd *cobra.Command, args []string) {
		bars := make([]string, len(api.CONFIG.Folders))

		var wg sync.WaitGroup
		wg.Add(len(api.CONFIG.Folders) * 2)

		for i := 0; i < len(api.CONFIG.Folders); i++ {
			f := api.CONFIG.Folders[i]
			ch := make(chan *models.UploadProgress)

			go func() {
				api.UploadFolder(f, ch)
				wg.Done()
			}()

			go func() {
				for progress := range ch {
					bar := utils.RenderProgressBar(progress)
					bars[i] = bar

					// renderBars(bars)
				}
				wg.Done()
			}()
		}

		wg.Wait()
	},
}

func renderBars(bars []string) {
	for i := 0; i < len(bars); i++ {
		fmt.Print("\033[F") // move cursor up by one line
		fmt.Print("\033[K") // clear the entire line
	}

	for _, bar := range bars {
		fmt.Println(bar)
	}
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
