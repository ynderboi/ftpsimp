package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ynderboi/ftpsimp/internal/config"
	"github.com/ynderboi/ftpsimp/internal/server"
)

type mode int

const (
	modeDashboard mode = iota
	modeEditRoot
	modeEditPIN
	modeConfirmQuit
)

type tickMsg time.Time

type Options struct {
	Server   *server.Server
	Persist  *config.Config
	URLs     []string
	CfgPath  string
	LocalIPs func() []string
}

type Model struct {
	opt     Options
	mode    mode
	input   textinput.Model
	status  string
	errMsg  string
	width   int
	quitting bool
}

func New(opt Options) Model {
	ti := textinput.New()
	ti.CharLimit = 512
	ti.Width = 56
	ti.Prompt = "> "
	return Model{
		opt:   opt,
		mode:  modeDashboard,
		input: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tickMsg:
		return m, tick()
	case tea.KeyMsg:
		if m.mode == modeEditRoot || m.mode == modeEditPIN {
			return m.updateInput(msg)
		}
		if m.mode == modeConfirmQuit {
			return m.updateQuit(msg)
		}
		return m.updateDashboard(msg)
	}
	return m, nil
}

func (m Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.errMsg = ""
	switch msg.String() {
	case "q", "ctrl+c":
		m.mode = modeConfirmQuit
		m.status = "Press Y to quit app, N to cancel"
		return m, nil
	case "s", "S", " ":
		if m.opt.Server.Running() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := m.opt.Server.Stop(ctx)
			cancel()
			if err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = "Server STOPPED — press S to start again"
			}
		} else {
			if err := m.opt.Server.Start(); err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = "Server STARTED"
			}
		}
		return m, nil
	case "1":
		m.mode = modeEditRoot
		m.input.SetValue(m.opt.Server.Root())
		m.input.Focus()
		m.status = "Enter new root folder path, Enter=save, Esc=cancel"
		return m, textinput.Blink
	case "2":
		ro := !m.opt.Server.ReadOnly()
		m.opt.Server.SetReadOnly(ro)
		m.opt.Persist.ReadOnly = ro
		_ = config.Save(*m.opt.Persist)
		if ro {
			m.status = "Mode: READ-ONLY"
		} else {
			m.status = "Mode: READ/WRITE"
		}
		return m, nil
	case "3":
		on := !m.opt.Server.AuthOn()
		pin := m.opt.Server.SetAuth(on)
		m.opt.Persist.Open = !on
		if on {
			m.opt.Persist.PIN = pin
			m.status = fmt.Sprintf("Auth ON · PIN %s", pin)
		} else {
			m.status = "Auth OFF — anyone on LAN can access"
		}
		_ = config.Save(*m.opt.Persist)
		return m, nil
	case "4":
		pin := m.opt.Server.RotatePIN()
		m.opt.Persist.Open = false
		m.opt.Persist.PIN = pin
		_ = config.Save(*m.opt.Persist)
		m.status = fmt.Sprintf("New PIN: %s (sessions cleared)", pin)
		return m, nil
	case "5":
		m.mode = modeEditPIN
		m.input.SetValue("")
		m.input.Placeholder = "6+ digits"
		m.input.Focus()
		m.status = "Enter fixed PIN, Enter=save, Esc=cancel"
		return m, textinput.Blink
	case "6":
		if m.opt.LocalIPs != nil {
			ips := m.opt.LocalIPs()
			m.opt.URLs = nil
			port := m.opt.Persist.Port
			for _, ip := range ips {
				m.opt.URLs = append(m.opt.URLs, fmt.Sprintf("http://%s:%d", ip, port))
			}
		}
		m.status = "LAN addresses refreshed"
		return m, nil
	case "r", "R":
		m.status = "Status refreshed"
		return m, nil
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeDashboard
		m.input.Blur()
		m.status = "Cancelled"
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if m.mode == modeEditRoot {
			if val == "" {
				m.errMsg = "path required"
				return m, nil
			}
			if err := m.opt.Server.SetRoot(val); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.opt.Persist.Root = m.opt.Server.Root()
			_ = config.Save(*m.opt.Persist)
			m.status = "Root updated: " + m.opt.Server.Root()
		} else if m.mode == modeEditPIN {
			if err := m.opt.Server.SetPIN(val); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.opt.Persist.PIN = val
			m.opt.Persist.Open = false
			_ = config.Save(*m.opt.Persist)
			m.status = "PIN set · sessions cleared"
		}
		m.mode = modeDashboard
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.quitting = true
		return m, tea.Quit
	case "n", "N", "esc":
		m.mode = modeDashboard
		m.status = "Cancelled"
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "\n  ftpsimp stopped.\n\n"
	}

	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Width(10)
	value := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	banner := Banner
	if m.width > 0 && m.width < 72 {
		banner = BannerSmall
	}

	srv := m.opt.Server
	running := srv.Running()
	statusStr := warn.Render("STOPPED")
	if running {
		statusStr = accent.Render("RUNNING")
	}
	modeStr := "READ/WRITE"
	if srv.ReadOnly() {
		modeStr = "READ-ONLY"
	}
	authStr := "OFF (open LAN)"
	pinStr := "—"
	if srv.AuthOn() {
		authStr = "ON"
		pinStr = srv.PIN()
	}

	var b strings.Builder
	b.WriteString(accent.Render(strings.TrimRight(banner, "\n")))
	b.WriteString("\n")
	b.WriteString(muted.Render("  " + Tagline))
	b.WriteString("\n\n")

	lines := []string{
		label.Render("STATUS") + statusStr,
		label.Render("ROOT") + value.Render(truncate(srv.Root(), 64)),
		label.Render("PORT") + value.Render(fmt.Sprintf("%d", m.opt.Persist.Port)),
		label.Render("MODE") + value.Render(modeStr),
		label.Render("AUTH") + value.Render(authStr),
		label.Render("PIN") + accent.Render(pinStr),
		label.Render("SESSIONS") + value.Render(fmt.Sprintf("%d", srv.SessionCount())),
	}
	if m.opt.CfgPath != "" {
		lines = append(lines, label.Render("CONFIG")+muted.Render(truncate(m.opt.CfgPath, 64)))
	}
	lines = append(lines, "")
	if !running {
		lines = append(lines, warn.Render("HTTP server is stopped. Press S to start."))
	} else if len(m.opt.URLs) == 0 {
		lines = append(lines, muted.Render("No LAN address. Connect Wi‑Fi and press 6."))
	} else {
		lines = append(lines, muted.Render("Open in browser:"))
		for _, u := range m.opt.URLs {
			lines = append(lines, "  "+accent.Render(u))
		}
	}

	b.WriteString(box.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")

	switch m.mode {
	case modeEditRoot, modeEditPIN:
		title := "New root path"
		if m.mode == modeEditPIN {
			title = "New PIN"
		}
		b.WriteString(accent.Render(title) + "\n")
		b.WriteString(m.input.View() + "\n")
	case modeConfirmQuit:
		b.WriteString(accent.Render("Quit application? [Y/N]") + "\n")
	default:
		startStop := "[S] Start server"
		if running {
			startStop = "[S] Stop server"
		}
		help := []string{
			startStop,
			"[1] Root",
			"[2] Read-only",
			"[3] Auth",
			"[4] Rotate PIN",
			"[5] Set PIN",
			"[6] Refresh IPs",
			"[Q] Quit",
		}
		b.WriteString(muted.Render(strings.Join(help, "  ")))
		b.WriteString("\n")
	}

	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("! " + m.errMsg))
		b.WriteString("\n")
	} else if m.status != "" {
		b.WriteString("\n")
		b.WriteString(muted.Render("· " + m.status))
		b.WriteString("\n")
	}

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 4 {
		return s[:n]
	}
	return "…" + s[len(s)-(n-1):]
}

func Run(opt Options) error {
	p := tea.NewProgram(New(opt), tea.WithAltScreen())
	_, err := p.Run()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = opt.Server.Stop(ctx)
	return err
}

func PrintPlain(opt Options) {
	accent := "\033[32m"
	reset := "\033[0m"
	if os.Getenv("NO_COLOR") != "" {
		accent, reset = "", ""
	}
	fmt.Print(accent, strings.TrimLeft(Banner, "\n"), reset)
	fmt.Println("  " + Tagline)
	fmt.Println()
	fmt.Printf("  Folder:   %s\n", opt.Server.Root())
	fmt.Printf("  Port:     %d\n", opt.Persist.Port)
	if opt.Server.ReadOnly() {
		fmt.Println("  Mode:     READ-ONLY")
	} else {
		fmt.Println("  Mode:     read/write")
	}
	if opt.Server.AuthOn() {
		fmt.Printf("  PIN:      %s\n", opt.Server.PIN())
		fmt.Println("  Auth:     ON")
	} else {
		fmt.Println("  Auth:     OFF")
	}
	if opt.CfgPath != "" {
		fmt.Printf("  Settings: %s\n", opt.CfgPath)
	}
	fmt.Println()
	if len(opt.URLs) == 0 {
		fmt.Println("  No LAN address found.")
	} else {
		fmt.Println("  Open in browser:")
		for _, u := range opt.URLs {
			fmt.Printf("    %s\n", u)
		}
	}
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop.")
	fmt.Println()
}
