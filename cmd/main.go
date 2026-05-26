package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/container"
	"github.com/metal-stack/v"

	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

var (
	configMapFlag = &cli.StringFlag{
		Name:  "mirror-config",
		Usage: "path to mirror-config-map",
		Value: "oci-mirror.yaml",
	}
	debugFlag = &cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug logging",
		Value: false,
	}
	retryMaxAttemptsFlag = &cli.IntFlag{
		Name:  "retry.max-attempts",
		Usage: "maximum retry attempts for transient mirror errors",
		Value: 10,
	}
	retryInitialDelayFlag = &cli.DurationFlag{
		Name:  "retry.initial-delay",
		Usage: "initial delay between retry attempts",
		Value: 10 * time.Second,
	}
	retryMaxDelayFlag = &cli.DurationFlag{
		Name:  "retry.max-delay",
		Usage: "maximum delay between retry attempts",
		Value: 5 * time.Minute,
	}

	mirrorCmd = &cli.Command{
		Name:  "mirror",
		Usage: "mirror images as specified in configuration",
		Flags: []cli.Flag{
			debugFlag,
			configMapFlag,
			retryMaxAttemptsFlag,
			retryInitialDelayFlag,
			retryMaxDelayFlag,
		},
		Action: func(ctx *cli.Context) error {
			level := slog.LevelInfo
			if ctx.Bool(debugFlag.Name) {
				level = slog.LevelDebug
			}
			jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
			log := slog.New(jsonHandler)

			log.Info("start mirror", "version", v.V.String())
			raw, err := os.ReadFile(ctx.String(configMapFlag.Name))
			if err != nil {
				return fmt.Errorf("unable to read config file:%w", err)
			}
			var config apiv1.Config
			err = yaml.Unmarshal(raw, &config)
			if err != nil {
				return fmt.Errorf("unable to parse config file:%w", err)
			}

			err = config.Validate()
			if err != nil {
				return fmt.Errorf("config invalid:%w", err)
			}

			s := newServer(log, config, &container.RetryPolicy{
				MaxAttempts:  ctx.Int(retryMaxAttemptsFlag.Name),
				InitialDelay: ctx.Duration(retryInitialDelayFlag.Name),
				MaxDelay:     ctx.Duration(retryMaxDelayFlag.Name),
			})
			if err := s.mirror(); err != nil {
				log.Error("error during mirror", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
	purgeCmd = &cli.Command{
		Name:  "purge",
		Usage: "purge images as specified in configuration",
		Flags: []cli.Flag{
			debugFlag,
			configMapFlag,
		},
		Action: func(ctx *cli.Context) error {
			level := slog.LevelInfo
			if ctx.Bool(debugFlag.Name) {
				level = slog.LevelDebug
			}
			jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
			log := slog.New(jsonHandler)

			log.Info("start purge", "version", v.V.String())
			raw, err := os.ReadFile(ctx.String(configMapFlag.Name))
			if err != nil {
				return fmt.Errorf("unable to read config file:%w", err)
			}
			var config apiv1.Config
			err = yaml.Unmarshal(raw, &config)
			if err != nil {
				return fmt.Errorf("unable to parse config file:%w", err)
			}

			err = config.Validate()
			if err != nil {
				return fmt.Errorf("config invalid:%w", err)
			}

			s := newServer(log, config, nil)
			if err := s.purge(); err != nil {
				log.Error("error during purge", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
	purgeUnknownCmd = &cli.Command{
		Name:  "purge-unknown",
		Usage: "purge unknown images according to the configuration",
		Flags: []cli.Flag{
			debugFlag,
			configMapFlag,
		},
		Action: func(ctx *cli.Context) error {
			level := slog.LevelInfo
			if ctx.Bool(debugFlag.Name) {
				level = slog.LevelDebug
			}
			jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
			log := slog.New(jsonHandler)

			log.Info("start purge unknown", "version", v.V.String())
			raw, err := os.ReadFile(ctx.String(configMapFlag.Name))
			if err != nil {
				return fmt.Errorf("unable to read config file:%w", err)
			}
			var config apiv1.Config
			err = yaml.Unmarshal(raw, &config)
			if err != nil {
				return fmt.Errorf("unable to parse config file:%w", err)
			}

			err = config.Validate()
			if err != nil {
				return fmt.Errorf("config invalid:%w", err)
			}

			s := newServer(log, config, nil)
			if err := s.purgeUnknown(); err != nil {
				log.Error("error during purge", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
)

func main() {
	app := &cli.App{
		Name:  "oci-mirror",
		Usage: "oci mirror server",
		Commands: []*cli.Command{
			mirrorCmd,
			purgeCmd,
			purgeUnknownCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Error in cli: %v", err)
	}

}
