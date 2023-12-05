package main

import (
	"context"
	"log/slog"
	"time"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/mirror"
)

type server struct {
	log    *slog.Logger
	config apiv1.Config
}

func newServer(log *slog.Logger, config apiv1.Config) *server {
	return &server{
		log:    log,
		config: config,
	}
}

func (s *server) run() error {
	start := time.Now()
	m := mirror.New(s.log, s.config)
	err := m.Mirror(context.Background())
	if err != nil {
		s.log.Error("error mirroring images", "error", err)
		return err
	}
	s.log.Info("finished mirroring after", "duration", time.Since(start))
	return nil
}
