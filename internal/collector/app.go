package collector

import (
	"context"
	"log"
)

type App struct {
	cfg Config
}

func New(cfg Config) *App {
	return &App{cfg: cfg}
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("collector starting, listen=%s", a.cfg.ListenAddr)

	srv, err := newGRPCServer(a.cfg)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	select {
	case <-ctx.Done():
		log.Printf("collector shutting down (context cancelled)")
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
