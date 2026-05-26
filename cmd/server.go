package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/container"
)

type server struct {
	log         *slog.Logger
	config      apiv1.Config
	retryPolicy *container.RetryPolicy
}

func newServer(log *slog.Logger, config apiv1.Config, retryPolicy *container.RetryPolicy) *server {
	return &server{
		log:         log,
		config:      config,
		retryPolicy: retryPolicy,
	}
}

func (s *server) mirror() error {
	start := time.Now()
	m := container.New(s.log.WithGroup("mirror"), s.config, s.retryPolicy)
	err := m.Mirror(context.Background())
	if err != nil {
		s.log.Error(fmt.Sprintf("error mirroring images, duration %s", time.Since(start)), "error", err)
		return err
	}
	s.log.Info(fmt.Sprintf("finished mirroring after %s", time.Since(start)))
	return nil
}

func (s *server) purge() error {
	start := time.Now()
	m := container.New(s.log.WithGroup("purge"), s.config, s.retryPolicy)
	err := m.Purge(context.Background())
	if err != nil {
		s.log.Error(fmt.Sprintf("error purging images, duration %s", time.Since(start)), "error", err)
		return err
	}
	s.log.Info(fmt.Sprintf("finished purging after %s", time.Since(start)))
	return nil
}

func (s *server) purgeUnknown() error {
	start := time.Now()
	m := container.New(s.log.WithGroup("purgeunknown"), s.config, s.retryPolicy)
	err := m.PurgeUnknown(context.Background())
	if err != nil {
		s.log.Error(fmt.Sprintf("error purging unknown images, duration %s", time.Since(start)), "error", err)
		return err
	}
	s.log.Info(fmt.Sprintf("finished purging unknown after %s", time.Since(start)))
	return nil
}
