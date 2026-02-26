package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
	"github.com/jackc/pgx/v5/pgtype"
)

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

type Service struct {
	q      *dbgen.Queries
	github GitHubConfig
}

func NewService(q *dbgen.Queries, github GitHubConfig) *Service {
	return &Service{q: q, github: github}
}

// GitHubAuthURL returns the URL to redirect users to for GitHub authorization.
func (s *Service) GitHubAuthURL() string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user,repo",
		url.QueryEscape(s.github.ClientID),
		url.QueryEscape(s.github.CallbackURL),
	)
}

// ExchangeGitHubCode exchanges an authorization code for an access token
// and stores it in the database.
func (s *Service) ExchangeGitHubCode(ctx context.Context, userID int64, code string) error {
	tokenResp, err := exchangeCodeForToken(s.github, code)
	if err != nil {
		return apperror.Internalf("github token exchange: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return apperror.BadRequest("github authorization failed")
	}

	// TODO: AES encrypt access_token before storing
	_, err = s.q.UpsertDataSource(ctx, dbgen.UpsertDataSourceParams{
		UserID:       userID,
		Provider:     "github",
		AccessToken:  []byte(tokenResp.AccessToken),
		RefreshToken: nil,
		ExpiresAt:    pgtype.Timestamptz{},
	})
	if err != nil {
		return apperror.Internalf("save github token: %w", err)
	}

	return nil
}

type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func exchangeCodeForToken(cfg GitHubConfig, code string) (*githubTokenResponse, error) {
	body := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
	}
	req, err := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.URL.RawQuery = body.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post to github: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp githubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &tokenResp, nil
}
