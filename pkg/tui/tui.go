package tui

import (
	"context"
	"fmt"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	tea "github.com/charmbracelet/bubbletea"
)

// RunTUI launches the interactive terminal TUI.
func RunTUI(cfg client.ConnectionConfig) error {
	var model *Model

	onFrame := func(f *client.Frame) {
		if model != nil {
			model.OnFrame(f)
		}
	}

	onState := func(s client.ConnectionState, err error) {
		if model != nil {
			model.OnState(s, err)
		}
	}

	streamClient, err := client.CreateClient(cfg, onFrame, onState)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	model = NewModel(streamClient, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(p)

	// Connect in background
	go func() {
		_ = streamClient.Connect(context.Background())
	}()

	_, err = p.Run()
	_ = streamClient.Close()
	return err
}
