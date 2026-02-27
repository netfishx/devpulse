package summary

import (
	"context"
	"fmt"
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

// --- Period (weekly/monthly) summaries ---

type PeriodSummary struct {
	Period        string `json:"period"`
	TotalCommits  int32  `json:"totalCommits"`
	TotalPrs      int32  `json:"totalPrs"`
	CodingMinutes int32  `json:"codingMinutes"`
}

type PeriodSummariesResponse struct {
	Summaries []PeriodSummary `json:"summaries"`
}

func (s *Service) ListWeekly(ctx context.Context, userID int64, weeks int) (*PeriodSummariesResponse, error) {
	if weeks < 1 || weeks > 52 {
		weeks = 12
	}

	rows, err := s.q.ListWeeklySummaries(ctx, dbgen.ListWeeklySummariesParams{
		UserID:  userID,
		Column2: int32(weeks),
	})
	if err != nil {
		return nil, apperror.Internalf("list weekly summaries: %w", err)
	}

	summaries := make([]PeriodSummary, 0, len(rows))
	for _, r := range rows {
		year, week := r.Period.Time.ISOWeek()
		summaries = append(summaries, PeriodSummary{
			Period:        fmt.Sprintf("%d-W%02d", year, week),
			TotalCommits:  r.TotalCommits,
			TotalPrs:      r.TotalPrs,
			CodingMinutes: r.CodingMinutes,
		})
	}

	return &PeriodSummariesResponse{Summaries: summaries}, nil
}

func (s *Service) ListMonthly(ctx context.Context, userID int64, months int) (*PeriodSummariesResponse, error) {
	if months < 1 || months > 24 {
		months = 12
	}

	rows, err := s.q.ListMonthlySummaries(ctx, dbgen.ListMonthlySummariesParams{
		UserID:  userID,
		Column2: int32(months),
	})
	if err != nil {
		return nil, apperror.Internalf("list monthly summaries: %w", err)
	}

	summaries := make([]PeriodSummary, 0, len(rows))
	for _, r := range rows {
		summaries = append(summaries, PeriodSummary{
			Period:        r.Period.Time.Format("2006-01"),
			TotalCommits:  r.TotalCommits,
			TotalPrs:      r.TotalPrs,
			CodingMinutes: r.CodingMinutes,
		})
	}

	return &PeriodSummariesResponse{Summaries: summaries}, nil
}

// --- Heatmap ---

type HeatmapDay struct {
	Date  string `json:"date"`
	Level int    `json:"level"`
	Count int    `json:"count"`
}

type HeatmapResponse struct {
	Days []HeatmapDay `json:"days"`
}

func commitCountToLevel(count int) int {
	switch {
	case count == 0:
		return 0
	case count <= 3:
		return 1
	case count <= 9:
		return 2
	case count <= 19:
		return 3
	default:
		return 4
	}
}

func (s *Service) Heatmap(ctx context.Context, userID int64, days int) (*HeatmapResponse, error) {
	if days < 1 || days > 365 {
		days = 365
	}

	rows, err := s.q.ListDailySummariesForHeatmap(ctx, dbgen.ListDailySummariesForHeatmapParams{
		UserID:  userID,
		Column2: int32(days),
	})
	if err != nil {
		return nil, apperror.Internalf("list heatmap: %w", err)
	}

	heatmap := make([]HeatmapDay, 0, len(rows))
	for _, r := range rows {
		count := int(r.TotalCommits)
		heatmap = append(heatmap, HeatmapDay{
			Date:  r.Date.Time.Format(time.DateOnly),
			Level: commitCountToLevel(count),
			Count: count,
		})
	}

	return &HeatmapResponse{Days: heatmap}, nil
}
