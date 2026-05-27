package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// After TUI exits, print the result URL to the terminal
	if fm, ok := m.(model); ok && fm.resultURL != "" {
		fmt.Printf("One time login link for %s\n%s\n", fm.siteEnv, fm.resultURL)
	}
}
