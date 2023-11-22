package main

import (
	"context"
	"log/slog"

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
	s.log.Info("run")
	m := mirror.New(s.log, s.config)

	err := m.Mirror(context.Background())
	if err != nil {
		s.log.Error("error synching images", "error", err)
		return err
	}
	return nil
}
