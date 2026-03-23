package service

import (
	"context"
	"log"
	"strings"

	"github.com/egon89/gin-langchain-ollama/internal/factory"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

type RetrieverService struct {
	llm          *ollama.LLM
	dbConnection *pgxpool.Pool
}

func NewRetrieverService(llm *ollama.LLM, dbConnection *pgxpool.Pool) *RetrieverService {
	return &RetrieverService{llm: llm, dbConnection: dbConnection}
}

func (s *RetrieverService) RetrieveDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	store := factory.CreateStoreOrFail(ctx, s.dbConnection, factory.CreateEmbedderOrFail(s.llm))

	retriever := vectorstores.ToRetriever(store, 5)

	result, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		return nil, err
	}

	for _, r := range result {
		log.Printf("Retrieved document: %s", r.Metadata)
	}

	return result, nil
}

func (s *RetrieverService) RetrieveAnswer(ctx context.Context, query string) string {
	documents, err := s.RetrieveDocuments(ctx, query)
	if err != nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("Use ONLY the provided context to answer.\n")
	sb.WriteString("If the answer is not in the context, say you don't know.\n\n")
	sb.WriteString("Context:\n")

	for _, doc := range documents {
		sb.WriteString(doc.PageContent)
		sb.WriteString("\n---\n")
	}

	return sb.String()
}
