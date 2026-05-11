package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

const (
	googleDefaultAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleDefaultTokenURL    = "https://oauth2.googleapis.com/token"
	googleDefaultUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	googleScope              = "openid email profile"
)

type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	authURL      string
	tokenURL     string
	userInfoURL  string
	httpClient   *http.Client
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string, httpClient *http.Client) *GoogleProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		authURL:      googleDefaultAuthURL,
		tokenURL:     googleDefaultTokenURL,
		userInfoURL:  googleDefaultUserInfoURL,
		httpClient:   httpClient,
	}
}

func (p *GoogleProvider) Name() string { return domain.OAuthProviderGoogle }

func (p *GoogleProvider) AuthorizationURL(state, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", googleScope)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("access_type", "online")
	q.Set("prompt", "select_account")
	return p.authURL + "?" + q.Encode()
}

func (p *GoogleProvider) Exchange(ctx context.Context, code, codeVerifier string) (*domain.ExternalUserInfo, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", p.redirectURL)

	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("google build token request: %w", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")

	tokenResp, err := p.httpClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("google token exchange: %w", err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(tokenResp.Body, 1024))
		return nil, fmt.Errorf("google token exchange: status %d: %s", tokenResp.StatusCode, string(body))
	}

	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("google decode token: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("google token exchange: empty access_token")
	}

	infoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("google build userinfo request: %w", err)
	}
	infoReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	infoReq.Header.Set("Accept", "application/json")

	infoResp, err := p.httpClient.Do(infoReq)
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer infoResp.Body.Close()
	if infoResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(infoResp.Body, 1024))
		return nil, fmt.Errorf("google userinfo: status %d: %s", infoResp.StatusCode, string(body))
	}

	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(infoResp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("google decode userinfo: %w", err)
	}
	if info.Sub == "" {
		return nil, fmt.Errorf("google userinfo: empty sub")
	}

	username := info.GivenName
	if username == "" {
		username = info.Name
	}

	return &domain.ExternalUserInfo{
		ProviderID:    info.Sub,
		Email:         info.Email,
		EmailVerified: info.EmailVerified,
		Username:      username,
		AvatarURL:     info.Picture,
	}, nil
}
