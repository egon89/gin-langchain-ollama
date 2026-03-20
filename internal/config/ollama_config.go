package config

import (
	"log"

	"github.com/tmc/langchaingo/llms/ollama"
)

func OllamaFactory() *ollama.LLM {
	// Default configuration (localhost:11434)
	llm, err := ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Println("Ollama config error")
		panic(err)
	}

	log.Println("Ollama LLM initialized with model")

	return llm
}
