package summary

import (
	"context"
	"time"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
)

type SummaryResponse struct {
	Date          string `json:"date"`
	TotalCommits  int32  `json:"totalCommits"`
	TotalPrs      int32  `json:"totalPrs"`
	CodingMinutes int32  `json:"codingMinutes"`
}

type ListSummariesResponse struct {
	Summaries []SummaryResponse `json:"summaries"`
}

type Service struct {
	q *dbgen.Queries
}

func NewService(q *dbgen.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) List(ctx context.Context, userID int64, days int) (*ListSummariesResponse, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	rows, err := s.q.ListSummariesByUser(ctx, dbgen.ListSummariesByUserParams{
		UserID:  userID,
		Column2: int32(days),
	})
	if err != nil {
		return nil, apperror.Internalf("list summaries: %w", err)
	}

	summaries := make([]SummaryResponse, 0, len(rows))
	for _, r := range rows {
		summaries = append(summaries, SummaryResponse{
			Date:          r.Date.Time.Format(time.DateOnly),
			TotalCommits:  r.TotalCommits.Int32,
			TotalPrs:      r.TotalPrs.Int32,
			CodingMinutes: r.CodingMinutes.Int32,
		})
	}

	return &ListSummariesResponse{Summaries: summaries}, nil
}
