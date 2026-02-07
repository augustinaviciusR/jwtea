package cmd

import (
	"context"
	"errors"
	"fmt"
	"jwtea/internal/core"
	"log"
	"net/http"
	"time"

	"jwtea/internal/config"
	jwthttp "jwtea/internal/http"
	"jwtea/internal/keys"
	"jwtea/internal/tui"

	"github.com/spf13/cobra"
)

const (
	defaultHost      = "127.0.0.1"
	defaultPort      = 8080
	defaultLogBuffer = 500

	shutdownGracePeriod = 100 * time.Millisecond
	shutdownTimeout     = 5 * time.Second
	readHeaderTimeout   = 5 * time.Second
	readTimeout         = 10 * time.Second
	writeTimeout        = 10 * time.Second
	idleTimeout         = 60 * time.Second
)

func applyFlagOverride(cfgVal *string, flagVal, defaultVal string) {
	if flagVal != "" && flagVal != defaultVal {
		*cfgVal = flagVal
	} else if flagVal == defaultVal && *cfgVal == "" {
		*cfgVal = flagVal
	}
}

func applyFlagOverrideInt(cfgVal *int, flagVal, defaultVal int) {
	if flagVal != 0 && flagVal != defaultVal {
		*cfgVal = flagVal
	} else if flagVal == defaultVal && *cfgVal == 0 {
		*cfgVal = flagVal
	}
}

func loadEffectiveConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if flagConfig != "" {
		cfg, err = config.LoadConfig(flagConfig)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		log.Printf("Loaded configuration from %s", flagConfig)
	} else {
		cfg = config.DefaultConfig()
	}

	applyFlagOverride(&cfg.Server.Host, flagHost, defaultHost)
	applyFlagOverrideInt(&cfg.Server.Port, flagPort, defaultPort)
	if flagIssuer != "" {
		cfg.OAuth.Issuer = flagIssuer
	}
	applyFlagOverrideInt(&cfg.Dashboard.LogBufferSize, flagLogBuffer, defaultLogBuffer)
	applyFlagOverrideInt(&cfg.Logging.BufferSize, flagLogBuffer, defaultLogBuffer)

	return cfg, nil
}

func seedStore(s *core.Store, cfg *config.Config) {
	for _, u := range cfg.Users {
		s.AddUser(core.User{
			Email: u.Email,
			Role:  u.Role,
			Dept:  u.Dept,
		})
		log.Printf("Loaded user: %s (%s)", u.Email, u.Role)
	}

	for _, c := range cfg.Clients {
		s.AddClient(c)
		log.Printf("Loaded client: %s", c.ID)
	}
}

func handleShutdown(srv *http.Server, errCh <-chan error, dashboardQuit, dashboardDone chan struct{}) error {
	select {
	case err := <-errCh:
		log.Printf("Server error: %v", err)
		close(dashboardQuit)
		time.Sleep(shutdownGracePeriod)
		return err
	case <-dashboardDone:
		log.Println("Dashboard closed, shutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("Server stopped cleanly")
	return nil
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the OAuth2/OIDC server with interactive dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadEffectiveConfig()
		if err != nil {
			return err
		}

		privKey, kid, jwk := keys.MustGenerateRSA()

		issuer := jwthttp.DeriveIssuer(cfg.OAuth.Issuer, cfg.Server.Host, cfg.Server.Port)
		cfg.OAuth.Issuer = issuer

		logHub := core.NewLogHub(cfg.Logging.BufferSize)
		chaosFlags := core.NewChaosFlags()

		s := core.NewStore()
		seedStore(s, cfg)

		if cfg.CallbackServer.Enabled {
			log.Printf("Registered callback endpoint at %s", cfg.CallbackServer.Path)
		}

		handler := jwthttp.NewRouter(jwthttp.RouterConfig{
			Store:   s,
			Config:  cfg,
			Chaos:   chaosFlags,
			LogHub:  logHub,
			Issuer:  issuer,
			PrivKey: privKey,
			Kid:     kid,
			JWK:     jwk,
		})

		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		srv := &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		}

		errCh := make(chan error, 1)
		go func() {
			log.Printf("HTTP server listening on http://%s\n", addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
			close(errCh)
		}()

		dashboardQuit := make(chan struct{})
		dashboardDone := make(chan struct{})
		tuiCtx := tui.NewContext(tui.ContextConfig{
			PrivKey:       privKey,
			Kid:           kid,
			Issuer:        issuer,
			Store:         s,
			Chaos:         chaosFlags,
			LogHub:        logHub,
			ServerRunning: true,
			Config:        cfg,
			ConfigPath:    flagConfig,
		})
		go func() {
			runDashboardWithContext(tuiCtx, dashboardQuit)
			close(dashboardDone)
		}()

		return handleShutdown(srv, errCh, dashboardQuit, dashboardDone)
	},
}

var (
	flagLogBuffer int
	flagConfig    string
)

func init() {
	serveCmd.Flags().IntVar(&flagLogBuffer, "log-buffer", defaultLogBuffer, "Number of recent log entries to keep for the dashboard")
	serveCmd.Flags().StringVar(&flagConfig, "config", "", "Path to YAML config file for pre-loading clients")
}
