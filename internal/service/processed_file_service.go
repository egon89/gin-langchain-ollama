package service

import (
	"context"
	"errors"

	"github.com/egon89/gin-langchain-ollama/internal/db"
	"github.com/jackc/pgx"
)

type ProcessedFileService struct {
	queries *db.Queries
}

func NewProcessedFileService(q *db.Queries) *ProcessedFileService {
	return &ProcessedFileService{queries: q}
}

func (s *ProcessedFileService) AlreadyProcessed(ctx context.Context, checksum string) (bool, error) {
	_, err := s.queries.FindByChecksum(ctx, checksum)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *ProcessedFileService) Save(ctx context.Context, file db.CreateProcessedFileParams) error {
	return s.queries.CreateProcessedFile(ctx, file)
}
