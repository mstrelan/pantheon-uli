package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}

// --- Styles ---

var (
	filterPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	filterTextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	normalStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	selectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	envActiveStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true)
	envInactiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	statusStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	resultStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	dimStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// --- Messages ---

type sitesLoadedMsg struct{ sites []string }
type cacheRefreshedMsg struct {
	sites []string
	err   error
}
type uliResultMsg struct {
	url string
	err error
}

// --- Model ---

type state int

const (
	stateLoading state = iota
	stateBrowsing
	stateRunning
	stateDone
)

var envs = []string{"dev", "test", "live"}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinnerTickMsg struct{}

type model struct {
	allSites      []string
	filtered      []string
	filter        string
	cursor        int
	envIndex      int // 0=dev, 1=test, 2=live
	state         state
	siteEnv       string
	resultURL     string
	err           error
	statusMsg     string
	spinnerFrame  int
	windowHeight  int
	cancelRunning context.CancelFunc
}

func initialModel() model {
	return model{
		state:    stateLoading,
		envIndex: 0,
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		sites, err := LoadSites()
		if err != nil || len(sites) == 0 {
			// No cache, must refresh
			sites, err := RefreshCache()
			if err != nil {
				return cacheRefreshedMsg{nil, err}
			}
			return sitesLoadedMsg{sites}
		}
		return sitesLoadedMsg{sites}
	}
}

func (m *model) applyFilter() {
	if m.filter == "" {
		m.filtered = m.allSites
	} else {
		q := strings.ToLower(m.filter)
		m.filtered = nil
		for _, s := range m.allSites {
			if strings.Contains(strings.ToLower(s), q) {
				m.filtered = append(m.filtered, s)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		return m, nil

	case sitesLoadedMsg:
		m.allSites = msg.sites
		m.state = stateBrowsing
		m.applyFilter()
		// If cache is stale, kick off a background refresh
		if CacheIsStale() {
			return m, func() tea.Msg {
				sites, err := RefreshCache()
				return cacheRefreshedMsg{sites, err}
			}
		}
		return m, nil

	case cacheRefreshedMsg:
		if msg.err != nil {
			if m.state == stateLoading {
				m.err = msg.err
				m.state = stateDone
			}
			// If browsing, just ignore the error silently
			return m, nil
		}
		m.allSites = msg.sites
		if m.state == stateLoading {
			m.state = stateBrowsing
		}
		m.applyFilter()
		return m, nil

	case uliResultMsg:
		if m.state != stateRunning {
			// Stale result after cancellation; discard.
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err
			m.state = stateDone
			return m, nil
		}
		m.resultURL = msg.url
		m.state = stateDone
		// Open in browser then quit
		openBrowser(msg.url)
		return m, tea.Quit

	case spinnerTickMsg:
		if m.state == stateRunning {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, func() tea.Msg {
				time.Sleep(80 * time.Millisecond)
				return spinnerTickMsg{}
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.state == stateDone {
			return m, tea.Quit
		}
		if m.state == stateRunning {
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				if m.cancelRunning != nil {
					m.cancelRunning()
					m.cancelRunning = nil
				}
				m.state = stateBrowsing
				m.siteEnv = ""
			}
			return m, nil
		}
		if m.state != stateBrowsing {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyCtrlR:
			m.statusMsg = "Refreshing cache..."
			return m, func() tea.Msg {
				sites, err := RefreshCache()
				return cacheRefreshedMsg{sites, err}
			}

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}

		case tea.KeyTab, tea.KeyShiftTab, tea.KeyLeft, tea.KeyRight:
			if msg.Type == tea.KeyShiftTab || msg.Type == tea.KeyLeft {
				m.envIndex = (m.envIndex + 2) % 3
			} else {
				m.envIndex = (m.envIndex + 1) % 3
			}

		case tea.KeyEnter:
			if len(m.filtered) == 0 {
				return m, nil
			}
			site := m.filtered[m.cursor]
			env := envs[m.envIndex]
			m.siteEnv = site + "." + env
			m.state = stateRunning
			m.spinnerFrame = 0
			siteEnv := m.siteEnv
			ctx, cancel := context.WithCancel(context.Background())
			m.cancelRunning = cancel
			return m, tea.Batch(
				func() tea.Msg { return runTerminus(ctx, siteEnv, env) },
				func() tea.Msg {
					time.Sleep(80 * time.Millisecond)
					return spinnerTickMsg{}
				},
			)

		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}

		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.applyFilter()
		}
	}
	return m, nil
}

func runTerminus(ctx context.Context, siteEnv, env string) tea.Msg {
	url, err := GetULI(ctx, siteEnv)
	if err != nil {
		return uliResultMsg{"", err}
	}

	url = EnforceHTTPS(url)

	if env == "live" {
		if vanity := GetVanityDomain(ctx, siteEnv); vanity != "" {
			url = SwapDomain(url, vanity)
		}
	}

	user, pass := GetLockCreds(ctx, siteEnv)
	url = InjectCreds(url, user, pass)

	return uliResultMsg{url, nil}
}

func (m model) View() string {
	var b strings.Builder

	switch m.state {
	case stateLoading:
		b.WriteString(statusStyle.Render("Loading sites..."))
		return b.String()

	case stateRunning:
		frame := spinnerFrames[m.spinnerFrame]
		b.WriteString(statusStyle.Render(fmt.Sprintf("  %s Generating login link for %s...", frame, m.siteEnv)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  esc: cancel"))
		return b.String()

	case stateDone:
		if m.err != nil {
			b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		} else {
			b.WriteString(resultStyle.Render(m.resultURL))
		}
		return b.String()

	case stateBrowsing:
		// Filter input line
		b.WriteString(filterPromptStyle.Render("> "))
		b.WriteString(filterTextStyle.Render(m.filter))
		b.WriteString(dimStyle.Render("_"))
		b.WriteString("\n")

		if len(m.filtered) == 0 {
			b.WriteString(dimStyle.Render("  No matches"))
			b.WriteString("\n")
		} else {
			// Calculate visible window
			maxVisible := m.windowHeight - 3 // filter line + help line + padding
			if maxVisible < 1 {
				maxVisible = 20
			}
			start := 0
			if m.cursor >= maxVisible {
				start = m.cursor - maxVisible + 1
			}
			end := start + maxVisible
			if end > len(m.filtered) {
				end = len(m.filtered)
			}

			for i := start; i < end; i++ {
				site := m.filtered[i]
				if i == m.cursor {
					// Selected row: show site name + env selector
					b.WriteString(selectedStyle.Render("  " + site + "  "))
					for j, e := range envs {
						if j == m.envIndex {
							b.WriteString(envActiveStyle.Render(" " + e + " "))
						} else {
							b.WriteString(envInactiveStyle.Render(" " + e + " "))
						}
					}
				} else {
					b.WriteString(normalStyle.Render("  " + site))
				}
				b.WriteString("\n")
			}

			if len(m.filtered) > maxVisible {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d/%d sites", len(m.filtered), len(m.allSites))))
				b.WriteString("\n")
			}
		}

		// Help line
		b.WriteString(dimStyle.Render("  tab/arrows: env  enter: go  ctrl+r: refresh  esc: quit"))

		if m.statusMsg != "" {
			b.WriteString("\n")
			b.WriteString(statusStyle.Render("  " + m.statusMsg))
		}
	}

	return b.String()
}
