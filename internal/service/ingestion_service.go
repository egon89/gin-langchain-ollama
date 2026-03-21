package service

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
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

	/*
		documentEmbeddings, err := s.llm.CreateEmbedding(ctx, chunks)
		if err != nil {
			log.Println("Error creating embeddings:", err)
			return err
		}

		// print the embeddings for demonstration purposes
		for i, embedding := range documentEmbeddings {
			log.Printf("Chunk %d: %v\n", i, embedding)
		}
	*/

	embedder, err := embeddings.NewEmbedder(s.llm)
	if err != nil {
		log.Println("Error creating embedder:", err)
		return err
	}

	store, err := pgvector.New(
		ctx,
		pgvector.WithConn(s.dbConnection),
		pgvector.WithCollectionTableName("collection_table"),
		pgvector.WithEmbedder(embedder),
		pgvector.WithEmbeddingTableName("emb_table"),
	)
	if err != nil {
		log.Println("Error creating vector store:", err)
		return err
	}

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
		textsplitter.WithChunkSize(500),
		textsplitter.WithChunkOverlap(50),
	)

	chunks, err := splitter.SplitText(content)
	if err != nil {
		log.Println("Error splitting content:", err)
		return nil
	}

	return chunks
}
