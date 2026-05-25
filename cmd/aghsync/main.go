package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/t0mer/aghsync/internal/api"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/logging"
	"github.com/t0mer/aghsync/internal/service"
	"github.com/t0mer/aghsync/internal/store"
)

func main() {
	var (
		flagPort          = flag.Int("port", 0, "listening port (overridden by AGHSYNC_PORT)")
		flagLogLevel      = flag.String("log-level", "", "log level: debug|info|warning|error")
		flagResetPassword = flag.Bool("reset-password", false, "reset the UI password")
		flagService       = flag.String("service", "", "service action: install|uninstall|start|stop|restart")
	)
	flag.Parse()

	// Bootstrap logger at warn level until config is loaded.
	logger := logging.New(slog.LevelWarn, os.Stdout)

	dbPath := resolveDBPath()
	s, err := store.Open(dbPath)
	if err != nil {
		logger.Error("failed to open database", "err", err, "path", dbPath)
		os.Exit(1)
	}
	defer s.Close()

	cfg := config.New(s)

	if _, err := cfg.InstallSecret(); err != nil {
		logger.Error("failed to initialize install secret", "err", err)
		os.Exit(1)
	}

	// Resolve log level: LOG_LEVEL env > --log-level flag > db/default
	logLevel := resolveLogLevel(*flagLogLevel)
	logger = logging.New(logging.LevelFromString(logLevel), os.Stdout)

	if *flagResetPassword {
		if err := handleResetPassword(cfg); err != nil {
			logger.Error("reset-password failed", "err", err)
			os.Exit(1)
		}
		fmt.Println("Password reset successfully.")
		return
	}

	if *flagService != "" {
		if err := service.RunAction(*flagService, func() error { return nil }, nil); err != nil {
			logger.Error("service action failed", "action", *flagService, "err", err)
			os.Exit(1)
		}
		return
	}

	port, err := resolvePort(cfg, *flagPort)
	if err != nil {
		logger.Error("failed to resolve port", "err", err)
		os.Exit(1)
	}

	router := api.NewRouter(logger)
	addr := fmt.Sprintf(":%d", port)
	logger.Info("starting server", "addr", addr, "version", version)

	srv := &http.Server{Addr: addr, Handler: router}
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func resolveDBPath() string {
	if dir := os.Getenv("AGHSYNC_DATA"); dir != "" {
		return dir + "/aghsync.db"
	}
	return "aghsync.db"
}

func resolveLogLevel(flagVal string) string {
	if e := os.Getenv("LOG_LEVEL"); e != "" {
		return e
	}
	if flagVal != "" {
		return flagVal
	}
	return "warning"
}

func resolvePort(cfg *config.Config, flagPort int) (int, error) {
	if e := os.Getenv("AGHSYNC_PORT"); e != "" {
		p, err := strconv.Atoi(e)
		if err != nil {
			return 0, fmt.Errorf("invalid AGHSYNC_PORT %q: %w", e, err)
		}
		return p, nil
	}
	if flagPort != 0 {
		return flagPort, nil
	}
	return cfg.GetPort()
}

// handleResetPassword prompts for a new password and stores it.
// Full bcrypt hashing is implemented in Plan 2 (auth package);
// this stub stores the plaintext under a temporary key until then.
func handleResetPassword(cfg *config.Config) error {
	fmt.Print("New password: ")
	var pw string
	if _, err := fmt.Scanln(&pw); err != nil {
		return err
	}
	return cfg.Set("_ui_password_plain_tmp", pw)
}
