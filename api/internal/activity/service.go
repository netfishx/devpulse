package activity

import (
	"context"
	"encoding/json"
	"time"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
)

type ActivityResponse struct {
	ID         int64           `json:"id"`
	Source     string          `json:"source"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurredAt"`
}

type ListResponse struct {
	Activities []ActivityResponse `json:"activities"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PerPage    int                `json:"perPage"`
}

type Service struct {
	q *dbgen.Queries
}

func NewService(q *dbgen.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) List(ctx context.Context, userID int64, page, perPage int, source string) (*ListResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	rows, err := s.q.ListActivitiesByUser(ctx, dbgen.ListActivitiesByUserParams{
		UserID:  userID,
		Limit:   int32(perPage),
		Offset:  int32(offset),
		Column4: source, // "" = no filter
	})
	if err != nil {
		return nil, apperror.Internalf("list activities: %w", err)
	}

	total, err := s.q.CountActivitiesByUser(ctx, dbgen.CountActivitiesByUserParams{
		UserID:  userID,
		Column2: source, // "" = no filter
	})
	if err != nil {
		return nil, apperror.Internalf("count activities: %w", err)
	}

	activities := make([]ActivityResponse, 0, len(rows))
	for _, r := range rows {
		activities = append(activities, ActivityResponse{
			ID:         r.ID,
			Source:     r.Source,
			Type:       r.Type,
			Payload:    r.Payload,
			OccurredAt: r.OccurredAt.Time,
		})
	}

	return &ListResponse{
		Activities: activities,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
	}, nil
}

type RepoStats struct {
	Name       string `json:"name"`
	Count      int    `json:"count"`
	LastActive string `json:"lastActive"`
}

type TopReposResponse struct {
	Repos []RepoStats `json:"repos"`
}

func (s *Service) TopRepos(ctx context.Context, userID int64, days int, source string) (*TopReposResponse, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	rows, err := s.q.ListTopRepos(ctx, dbgen.ListTopReposParams{
		UserID:  userID,
		Column2: int32(days),
	})
	if err != nil {
		return nil, apperror.Internalf("list top repos: %w", err)
	}

	repos := make([]RepoStats, 0, len(rows))
	for _, r := range rows {
		name, _ := r.Name.(string)
		lastActive := ""
		if t, ok := r.LastActive.(time.Time); ok {
			lastActive = t.Format(time.DateOnly)
		}
		repos = append(repos, RepoStats{
			Name:       name,
			Count:      int(r.Count),
			LastActive: lastActive,
		})
	}

	return &TopReposResponse{Repos: repos}, nil
}
