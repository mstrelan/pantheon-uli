package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEscFromDoneReturnsToStateBrowsing(t *testing.T) {
	m := initialModel()
	m.state = stateBrowsing
	m.allSites = []string{"site-a", "site-b", "site-c"}
	m.filter = "site"
	m.applyFilter()
	m.cursor = 1

	// Simulate entering stateRunning then receiving a successful ULI result
	m.state = stateRunning
	m.siteEnv = "site-b.dev"

	result, cmd := m.Update(uliResultMsg{url: "https://example.com/login", err: nil})
	m = result.(model)

	if m.state != stateDone {
		t.Fatalf("expected stateDone after uliResultMsg, got %v", m.state)
	}
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("uliResultMsg handler returned tea.Quit — TUI will exit immediately")
		}
	}

	// Now press Esc
	result2, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := result2.(model)

	if m2.state != stateBrowsing {
		t.Fatalf("expected stateBrowsing after Esc on done screen, got %v", m2.state)
	}
	if m2.cursor != 1 {
		t.Fatalf("expected cursor=1 preserved, got %d", m2.cursor)
	}
	if m2.filter != "site" {
		t.Fatalf("expected filter='site' preserved, got %q", m2.filter)
	}
	if cmd2 != nil {
		msg := cmd2()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("Esc on done screen returned tea.Quit")
		}
	}
}

func TestCtrlCFromDoneQuits(t *testing.T) {
	m := initialModel()
	m.state = stateDone
	m.resultURL = "https://example.com/login"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Fatal("expected tea.Quit cmd from Ctrl+C on done screen, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg from Ctrl+C on done screen, got %T", msg)
	}
}

func TestErrorCaseAnyKeyQuits(t *testing.T) {
	m := initialModel()
	m.state = stateDone
	m.err = fmt.Errorf("some error")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd == nil {
		t.Fatal("expected tea.Quit cmd from any key on error screen, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg from any key on error screen, got %T", msg)
	}
}
