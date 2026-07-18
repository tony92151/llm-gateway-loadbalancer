package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/app"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/config"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/store"
)

var version = "dev"

func main() {
	if err := rootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "lgl",
		Short:         "OpenAI-compatible LLM gateway load balancer",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(serveCommand(), migrateCommand(), versionCommand())
	return cmd
}

func serveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve <config>",
		Short: "Start the proxy and admin servers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(args[0])
			if err != nil {
				return err
			}
			setupLogger(cfg)
			gateway, err := app.New(cfg)
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			log.Info().Int("proxy_port", cfg.Server.Port).Int("admin_port", cfg.Admin.Port).Msg("starting llm gateway")
			return gateway.Run(ctx)
		},
	}
}

func migrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate <config>",
		Short: "Apply SQLite migrations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(args[0])
			if err != nil {
				return err
			}
			db, err := store.Open(cfg.Database.Path, cfg.Database.WALMode, cfg.Database.MaxOpenConn, cfg.Database.MaxIdleConn)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.Migrate()
		},
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}

func setupLogger(cfg config.Config) {
	level, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	if cfg.Logging.Format == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	_ = context.Background()
}
