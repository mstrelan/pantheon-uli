package main

import (
	"fmt"
	"testing"
	"time"

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

// --- Alt+Enter ---

func TestAltEnterWarmCacheOpensImmediately(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{Sites: []string{"site-a"}, RefreshedAt: time.Now()})
	if err := SaveEnvMeta("site-a.dev", "", "user", "pass"); err != nil {
		t.Fatalf("SaveEnvMeta: %v", err)
	}

	m := initialModel()
	m.state = stateBrowsing
	m.allSites = []string{"site-a"}
	m.applyFilter()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m2 := result.(model)

	// Cache was warm: must stay in stateBrowsing and fire openBrowser immediately.
	if m2.state != stateBrowsing {
		t.Fatalf("expected stateBrowsing (warm cache), got %v", m2.state)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (openBrowser), got nil")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("alt+enter warm path returned tea.Quit")
		}
	}
}

func TestAltEnterColdCacheTransitionsToStateRunning(t *testing.T) {
	overrideCacheDir(t) // empty cache

	m := initialModel()
	m.state = stateBrowsing
	m.allSites = []string{"site-a"}
	m.applyFilter()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m2 := result.(model)

	// Cache was cold: must transition to stateRunning to fetch from terminus.
	if m2.state != stateRunning {
		t.Fatalf("expected stateRunning (cold cache), got %v", m2.state)
	}
}

func TestAltEnterOnEmptyListIsNoOp(t *testing.T) {
	m := initialModel()
	m.state = stateBrowsing
	m.allSites = []string{}
	m.applyFilter()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m2 := result.(model)

	if m2.state != stateBrowsing {
		t.Fatalf("expected stateBrowsing after alt+enter on empty list, got %v", m2.state)
	}
	if cmd != nil {
		t.Fatal("expected nil cmd from alt+enter on empty list, got non-nil")
	}
}

func TestOpenURLMsgReturnsToStateBrowsing(t *testing.T) {
	m := initialModel()
	m.state = stateRunning

	result, cmd := m.Update(openURLMsg{url: "https://example.com/"})
	m2 := result.(model)

	if m2.state != stateBrowsing {
		t.Fatalf("expected stateBrowsing after openURLMsg, got %v", m2.state)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (openBrowser)")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("openURLMsg handler returned tea.Quit")
		}
	}
}

func TestEnterStillTransitionsToStateRunning(t *testing.T) {
	m := initialModel()
	m.state = stateBrowsing
	m.allSites = []string{"site-a"}
	m.applyFilter()
	m.cursor = 0

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := result.(model)

	if m2.state != stateRunning {
		t.Fatalf("expected stateRunning after enter, got %v", m2.state)
	}
}

// --- buildBaseURL ---

func TestBuildBaseURLNoCreds(t *testing.T) {
	got := buildBaseURL("my-site", "dev", "", "", "")
	want := "https://dev-my-site.pantheonsite.io/"
	if got != want {
		t.Fatalf("buildBaseURL: got %q, want %q", got, want)
	}
}

func TestBuildBaseURLWithVanityAndCreds(t *testing.T) {
	got := buildBaseURL("my-site", "live", "mysite.com", "admin", "s3cr3t")
	want := "https://admin:s3cr3t@mysite.com/"
	if got != want {
		t.Fatalf("buildBaseURL: got %q, want %q", got, want)
	}
}

func TestBuildBaseURLCredsNoVanity(t *testing.T) {
	got := buildBaseURL("my-site", "dev", "", "devuser", "devpass")
	want := "https://devuser:devpass@dev-my-site.pantheonsite.io/"
	if got != want {
		t.Fatalf("buildBaseURL: got %q, want %q", got, want)
	}
}
