package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	containersvc "github.com/zpol/doki/internal/app/containers"
	diagsvc "github.com/zpol/doki/internal/app/diagnostics"
	hostsvc "github.com/zpol/doki/internal/app/host"
	logsvc "github.com/zpol/doki/internal/app/logs"
	metricssvc "github.com/zpol/doki/internal/app/metrics"
	resourcessvc "github.com/zpol/doki/internal/app/resources"
	secsvc "github.com/zpol/doki/internal/app/security"
	shellsvc "github.com/zpol/doki/internal/app/shell"
	"github.com/zpol/doki/internal/config"
	dockerdapter "github.com/zpol/doki/internal/infrastructure/docker"
	tuiapp "github.com/zpol/doki/internal/tui/app"
	"github.com/zpol/doki/internal/version"
)

func newRootCmd() *cobra.Command {
	var (
		themeFlag   string
		contextFlag string
		hostFlag    string
	)

	cmd := &cobra.Command{
		Use:   "doki",
		Short: "DOKI — Docker Operations Cockpit for the terminal",
		Long: `DOKI is a terminal-native Docker operations cockpit.

Your Containers. One Terminal. Total Control.

Connects to a real Docker Engine and provides keyboard-driven
container management with a btop-inspired visual interface.`,
		Version: version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if themeFlag != "" {
				cfg.Theme = themeFlag
			}

			dockerClient, err := dockerdapter.New(dockerdapter.Options{
				Host:    hostFlag,
				Context: contextFlag,
			})
			if err != nil {
				return fmt.Errorf("docker client: %w", err)
			}
			defer dockerClient.Close()

			deps := tuiapp.Dependencies{
				Containers:  containersvc.New(dockerClient),
				Host:        hostsvc.New(dockerClient),
				Logs:        logsvc.New(dockerClient),
				Metrics:     metricssvc.New(dockerClient),
				Resources:   resourcessvc.New(dockerClient),
				Shell:       shellsvc.New(dockerClient),
				Security:    secsvc.New(dockerClient),
				Diagnostics: diagsvc.New(dockerClient),
				Docker:      dockerClient,
				Config:      cfg,
			}
			return tuiapp.Run(deps)
		},
	}

	cmd.Flags().StringVar(&themeFlag, "theme", "", "UI theme (cyberpunk|matrix|nord|monochrome)")
	cmd.Flags().StringVar(&contextFlag, "context", "", "Docker context name (display label for now)")
	cmd.Flags().StringVar(&hostFlag, "host", "", "Docker host URL (overrides DOCKER_HOST)")
	cmd.SetVersionTemplate("doki {{.Version}}\n")

	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newVersionCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stdout, "doki", version.String())
		},
	}
}
