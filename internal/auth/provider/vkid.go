package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

const (
	vkidDefaultAuthURL     = "https://id.vk.com/authorize"
	vkidDefaultTokenURL    = "https://id.vk.com/oauth2/auth" //nolint:gosec // G101 false positive: this is a public endpoint URL, not a secret
	vkidDefaultUserInfoURL = "https://id.vk.com/oauth2/user_info"
	vkidScope              = "email"
)

type vkidTokenResp struct {
	AccessToken string `json:"access_token"`
	UserID      int64  `json:"user_id"`
}

type vkidUserResp struct {
	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
}

type vkidUserInfoResp struct {
	User vkidUserResp `json:"user"`
}

type VKIDProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	authURL      string
	tokenURL     string
	userInfoURL  string
	httpClient   *http.Client
}

func NewVKIDProvider(clientID, clientSecret, redirectURL string, httpClient *http.Client) *VKIDProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &VKIDProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		authURL:      vkidDefaultAuthURL,
		tokenURL:     vkidDefaultTokenURL,
		userInfoURL:  vkidDefaultUserInfoURL,
		httpClient:   httpClient,
	}
}

func (p *VKIDProvider) Name() string { return domain.OAuthProviderVKID }

func (p *VKIDProvider) AuthorizationURL(state, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", vkidScope)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	return p.authURL + "?" + q.Encode()
}

func (p *VKIDProvider) Exchange(ctx context.Context, code, codeVerifier, deviceID string) (*domain.ExternalUserInfo, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", p.redirectURL)
	if deviceID != "" {
		form.Set("device_id", deviceID)
	}

	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("vkid build token request: %w", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")

	tokenResp, err := p.httpClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("vkid token exchange: %w", err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(tokenResp.Body, 1024))
		return nil, fmt.Errorf("vkid token exchange: status %d: %s", tokenResp.StatusCode, string(body))
	}

	tokenBody, err := io.ReadAll(io.LimitReader(tokenResp.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("vkid read token body: %w", err)
	}
	var token vkidTokenResp
	if err := json.Unmarshal(tokenBody, &token); err != nil {
		return nil, fmt.Errorf("vkid decode token: %w (body=%s)", err, string(tokenBody))
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("vkid token exchange: empty access_token (body=%s)", string(tokenBody))
	}

	infoForm := url.Values{}
	infoForm.Set("access_token", token.AccessToken)
	infoForm.Set("client_id", p.clientID)
	infoReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.userInfoURL, strings.NewReader(infoForm.Encode()))
	if err != nil {
		return nil, fmt.Errorf("vkid build userinfo request: %w", err)
	}
	infoReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	infoReq.Header.Set("Accept", "application/json")

	infoResp, err := p.httpClient.Do(infoReq)
	if err != nil {
		return nil, fmt.Errorf("vkid userinfo: %w", err)
	}
	defer infoResp.Body.Close()
	if infoResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(infoResp.Body, 1024))
		return nil, fmt.Errorf("vkid userinfo: status %d: %s", infoResp.StatusCode, string(body))
	}

	infoBody, err := io.ReadAll(io.LimitReader(infoResp.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("vkid read userinfo body: %w", err)
	}
	var info vkidUserInfoResp
	if err := json.Unmarshal(infoBody, &info); err != nil {
		return nil, fmt.Errorf("vkid decode userinfo: %w (body=%s)", err, string(infoBody))
	}

	providerID := info.User.UserID
	if providerID == "" && token.UserID != 0 {
		providerID = strconv.FormatInt(token.UserID, 10)
	}
	if providerID == "" {
		return nil, fmt.Errorf("vkid userinfo: empty user_id")
	}

	username := info.User.FirstName
	if username == "" {
		username = info.User.LastName
	}

	return &domain.ExternalUserInfo{
		ProviderID:    providerID,
		Email:         info.User.Email,
		EmailVerified: info.User.Email != "",
		Username:      username,
		AvatarURL:     info.User.Avatar,
	}, nil
}
