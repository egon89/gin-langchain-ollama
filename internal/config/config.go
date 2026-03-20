package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	DbUrl      string
	OllamaHost string
	TikaHost   string
	RagPath    string
)

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	DbUrl = os.Getenv("DB_URL")
	OllamaHost = Getenv("OLLAMA_HOST", "http://localhost:11434")
	TikaHost = Getenv("TIKA_HOST", "http://localhost:9998")
	RagPath = os.Getenv("RAG_PATH")

	log.Println("Environment variables loaded.")
}

func Getenv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
