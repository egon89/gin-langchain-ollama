package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/egon89/gin-langchain-ollama/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
)

var chatMemories = make(map[string]*memory.ConversationBuffer)
var mu sync.Mutex

func ChatStreamHandler(
	llm *ollama.LLM,
	retrieverService *service.RetrieverService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
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

func getChatMemory(
	chatId string,
) *memory.ConversationBuffer {
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
