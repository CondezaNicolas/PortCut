package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"portcut/internal/resident"
	residentwindows "portcut/internal/resident/windows"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", resident.FormatLaunchError(err))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	executable, err := os.Executable()
	if err != nil {
		return residentwindows.BootstrapError{Op: "locate executable", Err: err}
	}

	mode, err := residentwindows.EnsureDetached(residentwindows.BootstrapConfig{
		GOOS:       runtime.GOOS,
		Args:       os.Args,
		Env:        os.Environ(),
		Executable: executable,
	})
	if err != nil {
		return err
	}
	if mode == residentwindows.BootstrapRelaunched {
		return nil
	}

	return resident.Run(ctx, resident.RuntimeConfig{})
}
