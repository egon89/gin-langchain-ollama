package main

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// instance of ollama LLM
var llm = ollamaFactory()

func main() {
	router := gin.Default()

	router.Static("/public", "./public")

	router.Use(ErrorHandler())

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

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

		log.Printf("Received message for chat %s: %s", chatId, message)

		var lastChunk string
		var punctuationRegex = regexp.MustCompile(`^[.,!?)]`)

		_, err := llms.GenerateFromSinglePrompt(
			ctx,
			llm,
			message,
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
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

		c.SSEvent("message", "<END_OF_RESPONSE>")
		c.Writer.Flush()
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

func ollamaFactory() *ollama.LLM {
	// Default configuration (localhost:11434)
	llm, err := ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Println("Ollama config error")
		panic(err)
	}

	log.Println("Ollama LLM initialized with model")

	return llm
}
