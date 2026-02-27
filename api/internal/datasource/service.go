package datasource

import (
	"context"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
)

type SourceInfo struct {
	ID          int64  `json:"id"`
	Provider    string `json:"provider"`
	Connected   bool   `json:"connected"`
	ConnectedAt string `json:"connectedAt"`
}

type ListResponse struct {
	Sources []SourceInfo `json:"sources"`
}

type Service struct {
	q *dbgen.Queries
}

func NewService(q *dbgen.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) List(ctx context.Context, userID int64) (*ListResponse, error) {
	rows, err := s.q.ListDataSourcesByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Internalf("list data sources: %w", err)
	}

	sources := make([]SourceInfo, 0, len(rows))
	for _, r := range rows {
		connectedAt := ""
		if r.CreatedAt.Valid {
			connectedAt = r.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		sources = append(sources, SourceInfo{
			ID:          r.ID,
			Provider:    r.Provider,
			Connected:   true,
			ConnectedAt: connectedAt,
		})
	}

	return &ListResponse{Sources: sources}, nil
}
