package notification

import (
	"context"
	"log/slog"

	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
)

// Service dispatches notifications after a sync run completes.
type Service struct {
	repo      *Repository
	instances *instance.Repository
	logger    *slog.Logger
}

// NewService creates a Service.
func NewService(repo *Repository, instances *instance.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, instances: instances, logger: logger}
}

// Notify sends notifications for the completed run to all matching enabled channels.
// It is best-effort: errors are logged but never returned.
func (s *Service) Notify(ctx context.Context, run *history.Run, results []*history.Result) {
	isSuccess := run.Status == "success"
	channels, err := s.repo.ListEnabled(ctx, isSuccess)
	if err != nil {
		s.logger.Warn("notification: list channels failed", "err", err)
		return
	}
	if len(channels) == 0 {
		return
	}

	// Build instance name map for the message.
	insts, err := s.instances.List(ctx)
	if err != nil {
		s.logger.Warn("notification: list instances for message failed", "err", err)
		insts = nil
	}
	names := instanceNamesFromList(insts)
	message := BuildMessage(run, results, names)

	for _, ch := range channels {
		sender, err := NewSender(ch)
		if err != nil {
			s.logger.Warn("notification: build sender failed", "channel", ch.Name, "err", err)
			continue
		}
		if err := sender.Send(ctx, message); err != nil {
			s.logger.Warn("notification: send failed", "channel", ch.Name, "err", err)
		} else {
			s.logger.Info("notification: sent", "channel", ch.Name, "run", run.ID)
		}
	}
}
