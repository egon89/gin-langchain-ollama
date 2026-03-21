package runner

import (
	"log"
	"os"
	"path/filepath"
)

type StartupFolderScanRunner struct{}

func NewStartupFolderScanRunner() *StartupFolderScanRunner {
	return &StartupFolderScanRunner{}
}

func (r *StartupFolderScanRunner) Run(inputDir string, fileQueue chan<- string) {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		log.Fatalf("Failed to read input directory: %v", err)
	}

	log.Println("Starting initial folder scan...")

	for _, entry := range entries {
		if entry.Type().IsRegular() {
			filePath := filepath.Join(inputDir, entry.Name())

			log.Printf("Discovered file: %s", filePath)

			fileQueue <- filePath
		}
	}

	log.Println("Folder scan completed.")
}
