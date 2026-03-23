package service

import (
	"context"
	"log"

	"github.com/egon89/gin-langchain-ollama/internal/factory"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

type IngestionService struct {
	llm          *ollama.LLM
	dbConnection *pgxpool.Pool
}

func NewIngestionService(llm *ollama.LLM, dbConnection *pgxpool.Pool) *IngestionService {
	return &IngestionService{llm: llm, dbConnection: dbConnection}
}

func (s *IngestionService) IngestContent(ctx context.Context, id uuid.UUID, filePath, content string) error {
	chunks := s.split(content)

	embedder := factory.CreateEmbedderOrFail(s.llm)

	store := factory.CreateStoreOrFail(ctx, s.dbConnection, embedder)

	docs := make([]schema.Document, len(chunks))

	for i, chunk := range chunks {
		docs[i] = schema.Document{
			PageContent: chunk,
			Metadata: map[string]any{
				"id":        id.String(),
				"file_path": filePath,
			},
		}
	}

	store.AddDocuments(ctx, docs)

	return nil
}

func (s *IngestionService) split(content string) []string {
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(50),
	)

	chunks, err := splitter.SplitText(content)
	if err != nil {
		log.Println("Error splitting content:", err)
		return nil
	}

	return chunks
}
