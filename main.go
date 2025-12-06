package main

import (
	"context"
	"fmt"

	"github.com/hassanaziz0012/drive-syncer-video/api"
	"github.com/hassanaziz0012/drive-syncer-video/auth"
	"github.com/hassanaziz0012/drive-syncer-video/cmd"
)

func main() {
	fmt.Println()

	ctx := context.Background()

	// f := configureLogger()
	// defer f.Close()

	auth.ConnectToDrive(api.DRV, false, ctx)

	cmd.Execute()

	// r, err := drv.api.Files.List().PageSize(10).Do()
	// if err != nil {
	// 	log.Fatalf("Unable to fetch files from Drive: %v\n", err)
	// }
	// for _, f := range r.Files {
	// 	fmt.Println(f.Name)
	// }
}
