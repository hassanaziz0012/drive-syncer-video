package main

import (
	"log"
	"os"
)

func configureLogger() *os.File {
	f, err := os.OpenFile("drive-syncer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // (read-write, read, read)
	if err != nil {
		log.Fatalf("error opening log file: %v\n", err)
	}

	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return f
}
