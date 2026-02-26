package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
	"github.com/ethanwang/devpulse/api/internal/jwtutil"
)

type Service struct {
	q         *dbgen.Queries
	jwtSecret string
}

func NewService(q *dbgen.Queries, jwtSecret string) *Service {
	return &Service{q: q, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internalf("hash password: %w", err)
	}

	row, err := s.q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashed),
	})
	if err != nil {
		return nil, apperror.Conflict("email already exists")
	}

	return toUserResponse(row), nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.q.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.Unauthorized("invalid credentials")
		}
		return nil, apperror.Internalf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	token, err := jwtutil.Generate(user.ID, s.jwtSecret)
	if err != nil {
		return nil, apperror.Internalf("generate token: %w", err)
	}

	return &LoginResponse{
		Token: token,
		User:  *toUserResponseFromFull(user),
	}, nil
}

func (s *Service) GetMe(ctx context.Context, userID int64) (*UserResponse, error) {
	row, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.Internalf("get user: %w", err)
	}
	return toUserResponseFromGetByID(row), nil
}

// Converters: sqlc row types → response DTOs

func toUserResponse(row dbgen.CreateUserRow) *UserResponse {
	resp := &UserResponse{
		ID:    row.ID,
		Email: row.Email,
		Name:  row.Name,
	}
	if row.AvatarUrl.Valid {
		resp.AvatarURL = &row.AvatarUrl.String
	}
	if row.CreatedAt.Valid {
		resp.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		resp.UpdatedAt = row.UpdatedAt.Time
	}
	return resp
}

func toUserResponseFromFull(u dbgen.User) *UserResponse {
	resp := &UserResponse{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
	}
	if u.AvatarUrl.Valid {
		resp.AvatarURL = &u.AvatarUrl.String
	}
	if u.CreatedAt.Valid {
		resp.CreatedAt = u.CreatedAt.Time
	}
	if u.UpdatedAt.Valid {
		resp.UpdatedAt = u.UpdatedAt.Time
	}
	return resp
}

func toUserResponseFromGetByID(row dbgen.GetUserByIDRow) *UserResponse {
	resp := &UserResponse{
		ID:    row.ID,
		Email: row.Email,
		Name:  row.Name,
	}
	if row.AvatarUrl.Valid {
		resp.AvatarURL = &row.AvatarUrl.String
	}
	if row.CreatedAt.Valid {
		resp.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		resp.UpdatedAt = row.UpdatedAt.Time
	}
	return resp
}

// pgText converts *string to pgtype.Text for sqlc params.
func pgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
