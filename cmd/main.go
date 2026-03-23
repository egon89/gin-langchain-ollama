package main

import (
	"log"
	"net/http"

	"github.com/egon89/gin-langchain-ollama/internal/config"
	"github.com/egon89/gin-langchain-ollama/internal/db"
	"github.com/egon89/gin-langchain-ollama/internal/handler"
	"github.com/egon89/gin-langchain-ollama/internal/processor"
	"github.com/egon89/gin-langchain-ollama/internal/runner"
	"github.com/egon89/gin-langchain-ollama/internal/service"
	"github.com/egon89/gin-langchain-ollama/internal/watcher"
	"github.com/gin-gonic/gin"
)

// instance of ollama LLM
var llm = config.OllamaFactory()

func main() {
	config.LoadEnv()

	router := gin.Default()

	dbConnection, err := config.NewDB(config.DbUrl)
	if err != nil {
		log.Fatal(err)
	}

	queries := db.New(dbConnection)

	processedFileService := service.NewProcessedFileService(queries)

	ingestionService := service.NewIngestionService(llm, dbConnection)

	retrieverService := service.NewRetrieverService(llm, dbConnection)

	tikaProcessor := processor.NewTikaProcessor(config.TikaHost, processedFileService, ingestionService)

	startupScanRunner := runner.NewStartupFolderScanRunner()

	router.Static("/public", "./public")

	router.Use(ErrorHandler())

	router.GET("/chat", handler.ChatHandler(llm))

	router.GET("/chat/:id/stream", handler.ChatStreamHandler(llm, retrieverService))

	router.GET("/api/stream", handler.SSEExampleHandler())

	fileQueue := make(chan string, 100)

	go startupScanRunner.Run(config.RagPath, fileQueue)

	// Start watcher
	err = watcher.StartFolderWatcher(config.RagPath, fileQueue)
	if err != nil {
		log.Fatalf("Failed to start folder watcher: %v", err)
	}

	go tikaProcessor.Start(fileQueue, 3)

	router.Run() // listens on 0.0.0.0:8080 by default
}

// ErrorHandler captures errors and returns a consistent JSON error response
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first

		// Check if any errors were added to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		}
	}
}
