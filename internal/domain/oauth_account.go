package domain

import "time"

const (
	OAuthProviderGoogle = "google"
	OAuthProviderYandex = "yandex"
	OAuthProviderVKID   = "vkid"
)

var ValidOAuthProviders = map[string]bool{
	OAuthProviderGoogle: true,
	OAuthProviderYandex: true,
	OAuthProviderVKID:   true,
}

type OAuthAccount struct {
	ID         int64
	UserID     int64
	Provider   string
	ProviderID string
	CreatedAt  time.Time
}

type ExternalUserInfo struct {
	ProviderID    string
	Email         string
	EmailVerified bool
	Username      string
	AvatarURL     string
}

type OAuthState struct {
	State        string
	Provider     string
	CodeVerifier string
	CreatedAt    time.Time
}
