package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/egon89/gin-langchain-ollama/internal/config"
	"github.com/egon89/gin-langchain-ollama/internal/db"
	"github.com/egon89/gin-langchain-ollama/internal/processor"
	"github.com/egon89/gin-langchain-ollama/internal/runner"
	"github.com/egon89/gin-langchain-ollama/internal/service"
	"github.com/egon89/gin-langchain-ollama/internal/watcher"
	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
)

// instance of ollama LLM
var llm = config.OllamaFactory()

var chatMemories = make(map[string]*memory.ConversationBuffer)
var mu sync.Mutex

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

	router.GET("/chat", func(c *gin.Context) {
		message := c.DefaultQuery("message", "Give me a short poem")

		completion, err := llms.GenerateFromSinglePrompt(c.Request.Context(), llm, message)
		if err != nil {
			c.Error(err)
			return
		}

		c.String(http.StatusOK, completion)
	})

	router.GET("/chat/:id/stream", func(c *gin.Context) {
		ctx := c.Request.Context()
		chatId := c.Param("id")
		message := c.Query("message")
		if message == "" {
			message = "Give me a short poem"
		}

		// Get conversation history
		chatMemory := getChatMemory(chatId)
		chatMessages, _ := chatMemory.ChatHistory.Messages(ctx)
		systemContext := retrieverService.RetrieveAnswer(ctx, message)

		messageContentList := createMessageContentList(systemContext, chatMessages, message)

		log.Printf("Received message for chat %s: %s", chatId, message)

		var lastChunk string
		var punctuationRegex = regexp.MustCompile(`^[.,!?)]`)

		resp, err := llm.GenerateContent(ctx, messageContentList, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if isDisconnected(ctx) {
				log.Println("client disconnected, stopping stream")
				return ctx.Err()
			}

			current := string(chunk)

			// First chunk → just send
			if lastChunk == "" {
				lastChunk = current
				c.SSEvent("message", current)
				c.Writer.Flush()
				return nil
			}

			// Check if we need a space
			needsSpace := !strings.HasSuffix(lastChunk, " ") &&
				!punctuationRegex.MatchString(current)

			output := current
			if needsSpace {
				output = " " + current
			}

			lastChunk = current

			c.SSEvent("message", output)
			c.Writer.Flush()

			return nil
		}),
		)
		if err != nil {
			log.Fatal(err)
		}

		finalResponse := getResponse(resp, c)

		c.SSEvent("message", "<END_OF_RESPONSE>")
		c.Writer.Flush()

		fmt.Printf("Final response for chat %s: %s\n", chatId, finalResponse)

		// Save the conversation to memory
		chatMemory.ChatHistory.AddUserMessage(ctx, message)
		chatMemory.ChatHistory.AddAIMessage(ctx, finalResponse)
	})

	router.GET("/api/stream", func(c *gin.Context) {
		ctx := c.Request.Context()

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				log.Println("client disconnected, stopping work")
				return
			case <-time.After(1 * time.Second):
				c.SSEvent("message", gin.H{"count": i})
				c.Writer.Flush()
			}
		}
	})

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

func isDisconnected(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func getChatMemory(chatId string) *memory.ConversationBuffer {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Retrieving memory for chat ID: %s", chatId)

	mem, exists := chatMemories[chatId]
	if !exists {
		mem = memory.NewConversationBuffer()
		chatMemories[chatId] = mem
	}

	return mem
}

func getResponse(response *llms.ContentResponse, c *gin.Context) string {
	choices := response.Choices

	if len(choices) < 1 {
		c.String(http.StatusOK, "no response from model")
	}

	return choices[0].Content
}

func createMessageContentList(
	systemContext string,
	chatMessages []llms.ChatMessage,
	lastMessage string,
) []llms.MessageContent {
	var messageContentList []llms.MessageContent

	messageContentList = append(messageContentList, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemContext}},
	})

	for _, msg := range chatMessages {
		switch msg.GetType() {
		case "human":
			messageContentList = append(messageContentList, llms.MessageContent{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextContent{Text: msg.GetContent()}},
			})
		case "ai":
			messageContentList = append(messageContentList, llms.MessageContent{
				Role:  llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{llms.TextContent{Text: msg.GetContent()}},
			})
		}
	}

	messageContentList = append(messageContentList, llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: lastMessage}},
	})

	return messageContentList
}
