package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagHost   string
	flagPort   int
	flagIssuer string
)

var rootCmd = &cobra.Command{
	Use:   "jwtea",
	Short: "jwtea – OAuth2/OIDC server with interactive TUI",
	Long:  "jwtea is an OAuth2/OIDC authorization server with an integrated interactive dashboard for testing and development.\n\nRun 'jwtea serve' to start the server with the interactive dashboard.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", defaultHost, "Host interface to bind the HTTP server")
	rootCmd.PersistentFlags().IntVar(&flagPort, "port", defaultPort, "Port to bind the HTTP server")
	rootCmd.PersistentFlags().StringVar(&flagIssuer, "issuer", "", "OIDC issuer URL (optional). If empty, derived from host/port")

	rootCmd.AddCommand(serveCmd)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	cobra.OnInitialize()
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err != nil {
			_, err := fmt.Fprintln(os.Stderr, err)
			if err != nil {
				return err
			}

			if err := cmd.Usage(); err != nil {
				return err
			}
			os.Exit(2)
		}
		return nil
	})
}
