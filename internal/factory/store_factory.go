package factory

import (
	"context"
	"log"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
)

type StoreInput struct {
	CollectionTableName string
	EmbeddingTableName  string
}

func CreateStoreOrFail(ctx context.Context, dbConnection pgvector.PGXConn, embedder embeddings.Embedder) pgvector.Store {
	return CreateStoreWithCustomTablesOrFail(ctx, dbConnection, embedder, &StoreInput{
		CollectionTableName: "collection",
		EmbeddingTableName:  "vector_store",
	})
}

func CreateStoreWithCustomTablesOrFail(ctx context.Context, dbConnection pgvector.PGXConn, embedder embeddings.Embedder, input *StoreInput) pgvector.Store {
	store, err := pgvector.New(
		ctx,
		pgvector.WithConn(dbConnection),
		pgvector.WithCollectionTableName(input.CollectionTableName),
		pgvector.WithEmbedder(embedder),
		pgvector.WithEmbeddingTableName(input.EmbeddingTableName),
	)
	if err != nil {
		log.Fatalf("Error creating store: %v", err)
	}

	return store
}
