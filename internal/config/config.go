package config

import (
	"fmt"
	"jwtea/internal/core"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server            ServerConfig        `yaml:"server"`
	OAuth             OAuthConfig         `yaml:"oauth"`
	Tokens            TokenConfig         `yaml:"tokens"`
	Introspection     IntrospectionConfig `yaml:"introspection"`
	Revocation        RevocationConfig    `yaml:"revocation"`
	Users             []UserConfig        `yaml:"users"`
	Clients           []core.Client       `yaml:"clients"`
	CallbackServer    CallbackServer      `yaml:"callback_server"`
	ExternalCallbacks []string            `yaml:"external_callbacks"`
	Dashboard         DashboardConfig     `yaml:"dashboard"`
	Logging           LoggingConfig       `yaml:"logging"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type OAuthConfig struct {
	Issuer                string   `yaml:"issuer"`
	AuthCodeExpiry        Duration `yaml:"auth_code_expiry"`
	DefaultScopes         []string `yaml:"default_scopes"`
	SupportedScopes       []string `yaml:"supported_scopes"`
	AllowedGrantTypes     []string `yaml:"allowed_grant_types"`
	PKCERequired          bool     `yaml:"pkce_required"`
	PKCERequiredForPublic bool     `yaml:"pkce_required_for_public"`
}

type TokenConfig struct {
	AccessTokenExpiry    Duration          `yaml:"access_token_expiry"`
	IDTokenExpiry        Duration          `yaml:"id_token_expiry"`
	RefreshTokenExpiry   Duration          `yaml:"refresh_token_expiry"`
	Algorithm            string            `yaml:"algorithm"`
	CustomClaims         map[string]string `yaml:"custom_claims"`
	IssueRefreshToken    bool              `yaml:"issue_refresh_token"`
	RefreshTokenRotation bool              `yaml:"refresh_token_rotation"`
}

type UserConfig struct {
	Email string `yaml:"email"`
	Role  string `yaml:"role"`
	Dept  string `yaml:"dept"`
}

type CallbackServer struct {
	Enabled  bool   `yaml:"enabled"`
	Path     string `yaml:"path"`
	ClientID string `yaml:"client_id"`
}

type DashboardConfig struct {
	TickInterval  Duration `yaml:"tick_interval"`
	LogBufferSize int      `yaml:"log_buffer_size"`
	DefaultTab    string   `yaml:"default_tab"`
	ShowHelp      bool     `yaml:"show_help"`
	ColorScheme   string   `yaml:"color_scheme"`
}

type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	BufferSize int    `yaml:"buffer_size"`
}

type IntrospectionConfig struct {
	Enabled           bool     `yaml:"enabled"`
	RequireClientAuth bool     `yaml:"require_client_auth"`
	AllowedClients    []string `yaml:"allowed_clients"`
}

type RevocationConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequireClientAuth bool `yaml:"require_client_auth"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			_, err := fmt.Fprintln(os.Stderr, "error closing config file:", err)
			if err != nil {
				return
			}
		}
	}(f)

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	cfg.applyEnvOverrides()

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}

	if c.OAuth.Issuer == "" {
		c.OAuth.Issuer = "http://localhost:8080"
	}
	if c.OAuth.AuthCodeExpiry.Duration == 0 {
		c.OAuth.AuthCodeExpiry.Duration = 10 * time.Minute
	}
	if len(c.OAuth.DefaultScopes) == 0 {
		c.OAuth.DefaultScopes = []string{"openid"}
	}
	if len(c.OAuth.SupportedScopes) == 0 {
		c.OAuth.SupportedScopes = []string{"openid", "profile", "email"}
	}
	if len(c.OAuth.AllowedGrantTypes) == 0 {
		c.OAuth.AllowedGrantTypes = []string{"authorization_code", "client_credentials", "refresh_token"}
	}
	if len(c.OAuth.SupportedScopes) > 0 && !containsScope(c.OAuth.SupportedScopes, "offline_access") {
		c.OAuth.SupportedScopes = append(c.OAuth.SupportedScopes, "offline_access")
	}

	if c.Tokens.AccessTokenExpiry.Duration == 0 {
		c.Tokens.AccessTokenExpiry.Duration = 5 * time.Minute
	}
	if c.Tokens.IDTokenExpiry.Duration == 0 {
		c.Tokens.IDTokenExpiry.Duration = 5 * time.Minute
	}
	if c.Tokens.RefreshTokenExpiry.Duration == 0 {
		c.Tokens.RefreshTokenExpiry.Duration = 24 * time.Hour
	}
	if c.Tokens.Algorithm == "" {
		c.Tokens.Algorithm = "RS256"
	}

	if c.Dashboard.TickInterval.Duration == 0 {
		c.Dashboard.TickInterval.Duration = 1000 * time.Millisecond
	}
	if c.Dashboard.LogBufferSize == 0 {
		c.Dashboard.LogBufferSize = 500
	}
	if c.Dashboard.DefaultTab == "" {
		c.Dashboard.DefaultTab = "generate"
	}
	if c.Dashboard.ColorScheme == "" {
		c.Dashboard.ColorScheme = "default"
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.BufferSize == 0 {
		c.Logging.BufferSize = 500
	}

	if !c.Introspection.Enabled {
		c.Introspection.Enabled = true
	}
	if !c.Introspection.RequireClientAuth {
		c.Introspection.RequireClientAuth = true
	}

	if !c.Revocation.Enabled {
		c.Revocation.Enabled = true
	}
	if !c.Revocation.RequireClientAuth {
		c.Revocation.RequireClientAuth = true
	}

	if len(c.Users) == 0 {
		c.Users = []UserConfig{
			{Email: "alice@test.com", Role: "user", Dept: "engineering"},
			{Email: "bob@test.com", Role: "user", Dept: "sales"},
			{Email: "admin@test.com", Role: "admin", Dept: ""},
		}
	}

	if c.CallbackServer.Path == "" {
		c.CallbackServer = CallbackServer{
			Enabled:  true,
			Path:     "/callback",
			ClientID: "demo-client",
		}
	}

	if len(c.ExternalCallbacks) == 0 {
		c.ExternalCallbacks = []string{
			"https://oauth.pstmn.io/v1/callback",
		}
	}

	if len(c.Clients) == 0 {
		redirectURIs := []string{}

		if c.CallbackServer.Enabled {
			callbackURL := fmt.Sprintf("http://%s:%d%s", c.Server.Host, c.Server.Port, c.CallbackServer.Path)
			redirectURIs = append(redirectURIs, callbackURL)
		}

		redirectURIs = append(redirectURIs, c.ExternalCallbacks...)

		c.Clients = []core.Client{
			{
				ID:           "demo-client",
				Secret:       "demo-secret",
				RedirectURIs: redirectURIs,
			},
		}
	}
}

func (c *Config) applyEnvOverrides() {
	if port := os.Getenv("JWTEA_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}
	if host := os.Getenv("JWTEA_SERVER_HOST"); host != "" {
		c.Server.Host = host
	}

	if issuer := os.Getenv("JWTEA_OAUTH_ISSUER"); issuer != "" {
		c.OAuth.Issuer = issuer
	}
	if expiry := os.Getenv("JWTEA_OAUTH_AUTH_CODE_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			c.OAuth.AuthCodeExpiry.Duration = d
		}
	}
	if scopes := os.Getenv("JWTEA_OAUTH_DEFAULT_SCOPES"); scopes != "" {
		c.OAuth.DefaultScopes = strings.Split(scopes, ",")
	}
	if scopes := os.Getenv("JWTEA_OAUTH_SUPPORTED_SCOPES"); scopes != "" {
		c.OAuth.SupportedScopes = strings.Split(scopes, ",")
	}

	if expiry := os.Getenv("JWTEA_TOKENS_ACCESS_TOKEN_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			c.Tokens.AccessTokenExpiry.Duration = d
		}
	}
	if expiry := os.Getenv("JWTEA_TOKENS_ID_TOKEN_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			c.Tokens.IDTokenExpiry.Duration = d
		}
	}
	if expiry := os.Getenv("JWTEA_TOKENS_REFRESH_TOKEN_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			c.Tokens.RefreshTokenExpiry.Duration = d
		}
	}
	if algo := os.Getenv("JWTEA_TOKENS_ALGORITHM"); algo != "" {
		c.Tokens.Algorithm = algo
	}

	if enabled := os.Getenv("JWTEA_CALLBACK_SERVER_ENABLED"); enabled != "" {
		c.CallbackServer.Enabled = enabled == "true" || enabled == "1"
	}
	if path := os.Getenv("JWTEA_CALLBACK_SERVER_PATH"); path != "" {
		c.CallbackServer.Path = path
	}
	if clientID := os.Getenv("JWTEA_CALLBACK_SERVER_CLIENT_ID"); clientID != "" {
		c.CallbackServer.ClientID = clientID
	}

	if interval := os.Getenv("JWTEA_DASHBOARD_TICK_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			c.Dashboard.TickInterval.Duration = d
		}
	}
	if size := os.Getenv("JWTEA_DASHBOARD_LOG_BUFFER_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil {
			c.Dashboard.LogBufferSize = s
		}
	}
	if tab := os.Getenv("JWTEA_DASHBOARD_DEFAULT_TAB"); tab != "" {
		c.Dashboard.DefaultTab = tab
	}
	if help := os.Getenv("JWTEA_DASHBOARD_SHOW_HELP"); help != "" {
		c.Dashboard.ShowHelp = help == "true" || help == "1"
	}
	if scheme := os.Getenv("JWTEA_DASHBOARD_COLOR_SCHEME"); scheme != "" {
		c.Dashboard.ColorScheme = scheme
	}

	if level := os.Getenv("JWTEA_LOGGING_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if format := os.Getenv("JWTEA_LOGGING_FORMAT"); format != "" {
		c.Logging.Format = format
	}
	if size := os.Getenv("JWTEA_LOGGING_BUFFER_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil {
			c.Logging.BufferSize = s
		}
	}
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	return cfg
}

func SaveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		err := os.Remove(tmpPath)
		if err != nil {
			return err
		}
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

func (c *Config) SyncFromStore(s *core.Store) {
	if s == nil {
		return
	}

	users := s.ListUsers()
	c.Users = make([]UserConfig, len(users))
	for i, u := range users {
		c.Users[i] = UserConfig{
			Email: u.Email,
			Role:  u.Role,
			Dept:  u.Dept,
		}
	}
	sort.Slice(c.Users, func(i, j int) bool {
		return c.Users[i].Email < c.Users[j].Email
	})

	clients := s.ListClients()
	c.Clients = make([]core.Client, len(clients))
	copy(c.Clients, clients)
	sort.Slice(c.Clients, func(i, j int) bool {
		return c.Clients[i].ID < c.Clients[j].ID
	})
}

func containsScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}
