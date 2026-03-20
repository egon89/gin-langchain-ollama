package processor

import (
	"bytes"
	"log"
	"net/http"
	"os"
)

func StartTikaProcessor(fileQueue <-chan string, tikaURL string, concurrency uint) {
	sem := make(chan struct{}, concurrency)

	for filePath := range fileQueue {
		sem <- struct{}{}

		go func(fp string) {
			defer func() { <-sem }()

			log.Printf("Processing file: %s\n", fp)

			processFile(fp, tikaURL)
		}(filePath)
	}
}

func processFile(filePath, tikaURL string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", tikaURL+"/tika", file)
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error calling Tika:", err)
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	log.Println("Extracted content:", buf.String())
}
