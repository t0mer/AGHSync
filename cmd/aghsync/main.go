package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/term"

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

	// Shutdown on SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
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
	var port int
	if e := os.Getenv("AGHSYNC_PORT"); e != "" {
		p, err := strconv.Atoi(e)
		if err != nil {
			return 0, fmt.Errorf("invalid AGHSYNC_PORT %q: %w", e, err)
		}
		port = p
	} else if flagPort != 0 {
		port = flagPort
	} else {
		p, err := cfg.GetPort()
		if err != nil {
			return 0, err
		}
		port = p
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port %d is out of range (1–65535)", port)
	}
	return port, nil
}

// handleResetPassword prompts for a new password and stores it.
// Full bcrypt hashing is implemented in Plan 2 (auth package);
// this stub stores the plaintext under a temporary key until then.
func handleResetPassword(cfg *config.Config) error {
	fmt.Print("New password: ")
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	fmt.Println() // newline after the hidden input
	pw := string(pwBytes)
	if pw == "" {
		return fmt.Errorf("password cannot be empty")
	}
	return cfg.Set("_ui_password_plain_tmp", pw)
}
