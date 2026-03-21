package processor

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/egon89/gin-langchain-ollama/internal/db"
	"github.com/egon89/gin-langchain-ollama/internal/service"
	"github.com/egon89/gin-langchain-ollama/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type TikaProcessor struct {
	tikaURL              string
	processedFileService *service.ProcessedFileService
}

func NewTikaProcessor(tikaURL string, processedFileService *service.ProcessedFileService) *TikaProcessor {
	return &TikaProcessor{
		tikaURL:              tikaURL,
		processedFileService: processedFileService,
	}
}

func (tp *TikaProcessor) Start(fileQueue <-chan string, concurrency uint) {
	ctx := context.Background()
	sem := make(chan struct{}, concurrency)

	for filePath := range fileQueue {
		sem <- struct{}{}

		go func(fp string) {
			defer func() { <-sem }()

			log.Printf("Processing file: %s\n", fp)

			tp.processFile(ctx, fp, tp.tikaURL)
		}(filePath)
	}
}

func (tp *TikaProcessor) processFile(ctx context.Context, filePath, tikaURL string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// read file content and compute checksum
	fileBuffer := new(bytes.Buffer)

	fileBuffer.ReadFrom(file)

	checksum := util.ComputeSHA256Checksum(fileBuffer.Bytes())

	alreadyProcessed, _ := tp.processedFileService.AlreadyProcessed(ctx, checksum)
	if alreadyProcessed {
		log.Printf("File already processed, skipping: %s\n", filePath)
		return
	}

	// the file pointer is at the end after reading, so we need to reset it before sending to Tika
	file.Seek(0, 0)

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

	content := buf.String()
	log.Printf("Extracted content for file %s: %s\n", filePath, content)

	id, err := util.NewUUID(uuid.New().String())
	if err != nil {
		log.Println("Error generating UUID:", err)
		return
	}

	err = tp.processedFileService.Save(ctx, db.CreateProcessedFileParams{
		ID:          id,
		FileName:    filePath,
		Checksum:    checksum,
		ProcessedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		log.Println("Error saving processed file or file already processed:", err)
		return
	}

	log.Printf("Finished processing file: %s\n", filePath)
}
