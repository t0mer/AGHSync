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
	"github.com/t0mer/aghsync/internal/auth"
	"github.com/t0mer/aghsync/internal/config"
	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
	"github.com/t0mer/aghsync/internal/logging"
	"github.com/t0mer/aghsync/internal/service"
	internalsync "github.com/t0mer/aghsync/internal/sync"
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

	installSecret, err := cfg.InstallSecret()
	if err != nil {
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

	instanceRepo := instance.NewRepository(s.DB(), installSecret)
	historyStore := history.New(s.DB())
	engine := internalsync.NewEngine(instanceRepo, historyStore)
	dispatcher := internalsync.NewDispatcher(engine)

	ctx, cancel := context.WithCancel(context.Background())
	dispatcherDone := dispatcher.Start(ctx)

	scheduler := internalsync.NewScheduler(dispatcher)
	scheduler.Start()

	// Restore saved schedule (if any).
	if expr, err := cfg.GetSchedulerCron(); err == nil && expr != "" {
		if err := scheduler.SetSchedule(expr); err != nil {
			logger.Warn("saved scheduler cron is invalid", "expr", expr, "err", err)
		}
	}

	deps := api.Deps{
		Store:      s,
		Config:     cfg,
		Logger:     logger,
		Instances:  instanceRepo,
		History:    historyStore,
		Dispatcher: dispatcher,
		Scheduler:  scheduler,
	}
	router := api.NewRouter(deps)
	addr := fmt.Sprintf(":%d", port)
	logger.Info("starting server", "addr", addr, "version", version)

	srv := &http.Server{Addr: addr, Handler: router}

	// Shutdown on SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	srvErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srvErr <- err
		}
	}()

	select {
	case <-quit:
		logger.Info("shutting down")
	case err := <-srvErr:
		logger.Error("server error", "err", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}

	// Ordered graceful shutdown: stop scheduler → cancel dispatcher → wait for in-flight sync.
	scheduler.Stop()
	cancel()
	<-dispatcherDone
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

// handleResetPassword prompts for a new password, hashes it with bcrypt, and persists the hash.
// It also enables UI auth so the new credentials are immediately active.
func handleResetPassword(cfg *config.Config) error {
	fmt.Print("New password: ")
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	fmt.Println()
	pw := string(pwBytes)
	if pw == "" {
		return fmt.Errorf("password cannot be empty")
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := cfg.SetUIPasswordHash(hash); err != nil {
		return fmt.Errorf("save password hash: %w", err)
	}
	return cfg.SetUIAuthEnabled(true)
}
