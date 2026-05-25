package service

import (
	"fmt"

	"github.com/kardianos/service"
)

const (
	svcName        = "aghsync"
	svcDisplayName = "AGHSync"
	svcDescription = "AdGuardHome Configuration Sync"
)

type program struct {
	startFn func() error
	stopFn  func() error
}

func (p *program) Start(s service.Service) error {
	if p.startFn == nil {
		return nil
	}
	return p.startFn()
}
func (p *program) Stop(s service.Service) error {
	if p.stopFn != nil {
		return p.stopFn()
	}
	return nil
}

func newService(startFn, stopFn func() error) (service.Service, error) {
	cfg := &service.Config{
		Name:        svcName,
		DisplayName: svcDisplayName,
		Description: svcDescription,
	}
	return service.New(&program{startFn: startFn, stopFn: stopFn}, cfg)
}

// RunAction executes one of install/uninstall/start/stop/restart on the OS service.
func RunAction(action string, startFn, stopFn func() error) error {
	svc, err := newService(startFn, stopFn)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	switch action {
	case "install", "uninstall", "start", "stop", "restart":
		return service.Control(svc, action)
	default:
		return fmt.Errorf("unknown service action %q; valid: install, uninstall, start, stop, restart", action)
	}
}

// Run starts the program under the OS service manager.
func Run(startFn, stopFn func() error) error {
	svc, err := newService(startFn, stopFn)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	return svc.Run()
}
