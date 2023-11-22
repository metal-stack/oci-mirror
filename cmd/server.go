package main

import (
	"context"
	"log/slog"

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

	err := syncher.Sync(context.Background())
	if err != nil {
		s.log.Error("error synching images", "error", err)
		return err
	}
	return nil
}
