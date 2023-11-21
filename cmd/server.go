package main

import (
	"context"
	"log/slog"
	"time"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/ocisync"
)

type server struct {
	log    *slog.Logger
	config apiv1.SyncConfig
}

func newServer(log *slog.Logger, config apiv1.SyncConfig) *server {
	return &server{
		log:    log,
		config: config,
	}
}

func (s *server) run() error {
	s.log.Info("run")
	syncher := ocisync.New(s.log, s.config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// TODO do this in a loop
	return syncher.Sync(ctx)
}
