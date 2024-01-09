package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
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

	mirrorCmd = &cli.Command{
		Name:  "mirror",
		Usage: "mirror images as specified in configuration",
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

			s := newServer(log, config)
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

			s := newServer(log, config)
			if err := s.purge(); err != nil {
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
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Error in cli: %v", err)
	}

}
