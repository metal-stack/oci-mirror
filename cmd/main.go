package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

var (
	configMapFlag = &cli.StringFlag{
		Name:  "sync-config",
		Usage: "path to sync-config-map",
		Value: "oci-mirror.yaml",
	}

	syncCmd = &cli.Command{
		Name:  "sync",
		Usage: "sync images as specified in configuration",
		Flags: []cli.Flag{
			configMapFlag,
		},
		Action: func(ctx *cli.Context) error {

			jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
			log := slog.New(jsonHandler)

			raw, err := os.ReadFile(ctx.String(configMapFlag.Name))
			if err != nil {
				return fmt.Errorf("unable to read config file:%w", err)
			}
			var config apiv1.SyncConfig
			yaml.Unmarshal(raw, &config)

			s := newServer(log, config)
			if err := s.run(); err != nil {
				log.Error("unable to start server", "error", err)
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
			syncCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Error in cli: %v", err)
	}

}
