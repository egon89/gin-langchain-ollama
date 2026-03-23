package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func ChatHandler(
	llm *ollama.LLM,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		message := c.DefaultQuery("message", "Give me a short poem")

		completion, err := llms.GenerateFromSinglePrompt(c.Request.Context(), llm, message)
		if err != nil {
			c.Error(err)
			return
		}

		c.String(http.StatusOK, completion)
	}
}
