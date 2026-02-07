package core

import "time"

type User struct {
	Email string `yaml:"email" json:"email"`
	Role  string `yaml:"role" json:"role"`
	Dept  string `yaml:"dept" json:"dept"`
}

type Client struct {
	ID           string   `yaml:"id" json:"id"`
	Secret       string   `yaml:"secret" json:"secret,omitempty"`
	RedirectURIs []string `yaml:"redirect_uris" json:"redirect_uris"`
}

type AuthCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	UserID              string
	ExpiresAt           time.Time
	Used                bool
	CodeChallenge       string
	CodeChallengeMethod string
}

type RefreshToken struct {
	Token     string
	ClientID  string
	UserID    string
	Scope     string
	ExpiresAt time.Time
	IssuedAt  time.Time
	Revoked   bool
}

type RevokedToken struct {
	Token     string
	RevokedAt time.Time
}

type LogEntry struct {
	Time      time.Time
	Method    string
	Path      string
	Status    int
	Duration  time.Duration
	RemoteIP  string
	UserAgent string
	Bytes     int
	Error     string
}
