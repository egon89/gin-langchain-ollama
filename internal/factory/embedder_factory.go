package factory

import (
	"log"

	"github.com/tmc/langchaingo/embeddings"
)

func CreateEmbedderOrFail(client embeddings.EmbedderClient) embeddings.Embedder {
	embedder, err := embeddings.NewEmbedder(client)
	if err != nil {
		log.Fatalf("Error creating embedder: %v", err)
	}

	return embedder
}
