package tui

import (
	"crypto/rsa"
	"jwtea/internal/core"

	"jwtea/internal/config"
)

type Context struct {
	PrivKey       *rsa.PrivateKey
	Kid           string
	Issuer        string
	Store         *core.Store
	Chaos         *core.ChaosFlags
	LogHub        *core.LogHub
	ServerRunning bool
	Config        *config.Config
	ConfigPath    string
}

type ContextConfig struct {
	Config        *config.Config
	Chaos         *core.ChaosFlags
	LogHub        *core.LogHub
	Store         *core.Store
	PrivKey       *rsa.PrivateKey
	Kid           string
	Issuer        string
	ServerRunning bool
	ConfigPath    string
}

func NewContext(cfg ContextConfig) *Context {
	return &Context{
		PrivKey:       cfg.PrivKey,
		Kid:           cfg.Kid,
		Issuer:        cfg.Issuer,
		Store:         cfg.Store,
		Chaos:         cfg.Chaos,
		LogHub:        cfg.LogHub,
		ServerRunning: cfg.ServerRunning,
		Config:        cfg.Config,
		ConfigPath:    cfg.ConfigPath,
	}
}

func (ctx *Context) AutoSave() error {
	if ctx.ConfigPath == "" || ctx.Config == nil {
		return nil
	}

	ctx.Config.SyncFromStore(ctx.Store)
	return config.SaveConfig(ctx.Config, ctx.ConfigPath)
}
