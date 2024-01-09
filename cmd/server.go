package main

import (
	"context"
	"fmt"
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
		s.log.Error(fmt.Sprintf("error mirroring images, duration %s", time.Since(start)), "error", err)
		return err
	}
	s.log.Info(fmt.Sprintf("finished mirroring after %s", time.Since(start)))

	start = time.Now()
	err = m.Purge(context.Background())
	if err != nil {
		s.log.Error(fmt.Sprintf("error purging images, duration %s", time.Since(start)), "error", err)
		return err
	}
	s.log.Info(fmt.Sprintf("finished purging after %s", time.Since(start)))
	return nil
}
