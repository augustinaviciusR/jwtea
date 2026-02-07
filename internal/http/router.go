package http

import (
	"crypto/rsa"
	"jwtea/internal/core"
	"net/http"

	"jwtea/internal/callback"
	"jwtea/internal/config"
	"jwtea/internal/keys"
)

type RouterConfig struct {
	Store   *core.Store
	Config  *config.Config
	Chaos   *core.ChaosFlags
	LogHub  *core.LogHub
	Issuer  string
	PrivKey *rsa.PrivateKey
	Kid     string
	JWK     keys.JwkRSA
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	deps := &Dependencies{
		Store:   cfg.Store,
		Config:  cfg.Config,
		Chaos:   cfg.Chaos,
		Issuer:  cfg.Issuer,
		PrivKey: cfg.PrivKey,
		Kid:     cfg.Kid,
	}

	mux.Handle("/", NewRootHandler())
	mux.Handle("/healthz", NewHealthHandler())
	mux.Handle("/jwks.json", NewJWKSHandler(cfg.JWK))
	mux.Handle("/.well-known/openid-configuration", NewDiscoveryHandler(cfg.Issuer, cfg.Config))
	mux.Handle("/authorize", NewAuthorizeHandler(deps))
	mux.Handle("/oauth2/token", NewTokenHandler(deps))

	if cfg.Config.Introspection.Enabled {
		mux.Handle("/oauth2/introspect", NewIntrospectionHandler(deps))
	}

	if cfg.Config.Revocation.Enabled {
		mux.Handle("/oauth2/revoke", NewRevocationHandler(deps))
	}

	if cfg.Config.CallbackServer.Enabled {
		callbackHandler := callback.NewHandler(cfg.Config.CallbackServer, cfg.Issuer)
		mux.Handle(cfg.Config.CallbackServer.Path, callbackHandler)
	}

	middleware := NewLoggingMiddleware(cfg.LogHub, cfg.Chaos)
	return middleware.Wrap(mux)
}
