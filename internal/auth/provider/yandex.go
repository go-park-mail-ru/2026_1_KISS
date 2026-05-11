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
	yandexDefaultAuthURL     = "https://oauth.yandex.ru/authorize"
	yandexDefaultTokenURL    = "https://oauth.yandex.ru/token"
	yandexDefaultUserInfoURL = "https://login.yandex.ru/info"
	yandexScope              = "login:email login:info"
)

type YandexProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	authURL      string
	tokenURL     string
	userInfoURL  string
	httpClient   *http.Client
}

func NewYandexProvider(clientID, clientSecret, redirectURL string, httpClient *http.Client) *YandexProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &YandexProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		authURL:      yandexDefaultAuthURL,
		tokenURL:     yandexDefaultTokenURL,
		userInfoURL:  yandexDefaultUserInfoURL,
		httpClient:   httpClient,
	}
}

func (p *YandexProvider) Name() string { return domain.OAuthProviderYandex }

func (p *YandexProvider) AuthorizationURL(state, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", yandexScope)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("force_confirm", "yes")
	return p.authURL + "?" + q.Encode()
}

func (p *YandexProvider) Exchange(ctx context.Context, code, codeVerifier string) (*domain.ExternalUserInfo, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("grant_type", "authorization_code")

	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("yandex build token request: %w", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")

	tokenResp, err := p.httpClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("yandex token exchange: %w", err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(tokenResp.Body, 1024))
		return nil, fmt.Errorf("yandex token exchange: status %d: %s", tokenResp.StatusCode, string(body))
	}

	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("yandex decode token: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("yandex token exchange: empty access_token")
	}

	infoURL := p.userInfoURL + "?format=json"
	infoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("yandex build userinfo request: %w", err)
	}
	infoReq.Header.Set("Authorization", "OAuth "+token.AccessToken)
	infoReq.Header.Set("Accept", "application/json")

	infoResp, err := p.httpClient.Do(infoReq)
	if err != nil {
		return nil, fmt.Errorf("yandex userinfo: %w", err)
	}
	defer infoResp.Body.Close()
	if infoResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(infoResp.Body, 1024))
		return nil, fmt.Errorf("yandex userinfo: status %d: %s", infoResp.StatusCode, string(body))
	}

	var info struct {
		ID              string `json:"id"`
		Login           string `json:"login"`
		DefaultEmail    string `json:"default_email"`
		DisplayName     string `json:"display_name"`
		RealName        string `json:"real_name"`
		FirstName       string `json:"first_name"`
		DefaultAvatarID string `json:"default_avatar_id"`
		IsAvatarEmpty   bool   `json:"is_avatar_empty"`
	}
	if err := json.NewDecoder(infoResp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("yandex decode userinfo: %w", err)
	}
	if info.ID == "" {
		return nil, fmt.Errorf("yandex userinfo: empty id")
	}

	username := info.DisplayName
	if username == "" {
		username = info.FirstName
	}
	if username == "" {
		username = info.Login
	}

	avatar := ""
	if !info.IsAvatarEmpty && info.DefaultAvatarID != "" {
		avatar = "https://avatars.yandex.net/get-yapic/" + info.DefaultAvatarID + "/islands-200"
	}

	return &domain.ExternalUserInfo{
		ProviderID:    info.ID,
		Email:         info.DefaultEmail,
		EmailVerified: info.DefaultEmail != "",
		Username:      username,
		AvatarURL:     avatar,
	}, nil
}
