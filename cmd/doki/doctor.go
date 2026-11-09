package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	dockerdapter "github.com/zpol/doki/internal/infrastructure/docker"
	"github.com/zpol/doki/internal/version"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check Docker connectivity and environment compatibility",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("DOKI doctor")
			fmt.Println("-----------")
			fmt.Printf("Version:     %s\n", version.String())
			fmt.Printf("Go:          %s\n", runtime.Version())
			fmt.Printf("OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)

			if _, set := os.LookupEnv("NO_COLOR"); set {
				fmt.Println("NO_COLOR:    set (colors disabled)")
			} else {
				fmt.Println("NO_COLOR:    unset")
			}

			term := os.Getenv("TERM")
			if term == "" {
				term = "(empty)"
			}
			fmt.Printf("TERM:        %s\n", term)
			fmt.Printf("Colorterm:   %s\n", envOr("COLORTERM", "(unset)"))
			fmt.Printf("DOCKER_HOST: %s\n", envOr("DOCKER_HOST", "(default socket)"))

			cli, err := dockerdapter.New(dockerdapter.Options{})
			if err != nil {
				fmt.Printf("Docker client: FAIL (%v)\n", err)
				return nil
			}
			defer cli.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := cli.Ping(ctx); err != nil {
				fmt.Printf("Docker ping:   FAIL (%v)\n", err)
				fmt.Println("\nSuggestions:")
				fmt.Println("  • Ensure the Docker daemon is running")
				fmt.Println("  • Check socket permissions / docker group membership")
				fmt.Println("  • Verify DOCKER_HOST / Docker context configuration")
				return nil
			}
			fmt.Println("Docker ping:   OK")

			info, err := cli.HostInfo(ctx)
			if err != nil {
				fmt.Printf("Docker info:   FAIL (%v)\n", err)
				return nil
			}
			fmt.Printf("Docker info:   OK\n")
			fmt.Printf("  Server:      %s\n", info.ServerVersion)
			fmt.Printf("  API:         %s\n", info.APIVersion)
			fmt.Printf("  Host:        %s\n", info.Name)
			fmt.Printf("  Containers:  %d running / %d total\n", info.ContainersRunning, info.Containers)
			fmt.Println("\nAll basic checks passed.")
			return nil
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
