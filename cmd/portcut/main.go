package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"portcut/internal/app"
	"portcut/internal/platform"
	"portcut/internal/tui"
)

func main() {
	launcher := app.NewLauncher(nil, func(workflow app.Workflow, capabilities platform.Capabilities) app.ProgramRunner {
		return func() error {
			_, err := tea.NewProgram(tui.NewModel(workflow, capabilities)).Run()
			return err
		}
	})

	if err := launcher.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "portcut: %v\n", err)
		os.Exit(1)
	}
}
