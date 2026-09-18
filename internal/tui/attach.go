// Package tui is mcm's interactive per-server dashboard (`mcm attach`,
// M16): a live composition of work already proven out individually in
// earlier milestones, calling the same fleet functions as every other
// front end — a scrolling log pane (mcm logs -f), a stats header (mcm
// top), and a command line that sends RCON commands (mcm exec). No new
// business logic lives here, only presentation.
package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

const (
	statsInterval = 2 * time.Second
	maxLogLines   = 2000
)

type logLineMsg string
type logDoneMsg struct{ err error }
type statsMsg fleet.ServerStats
type statsErrMsg struct{ err error }
type execResultMsg struct {
	output string
	err    error
}
type tickMsg struct{}

// Attach runs the full-screen dashboard for name until the user quits
// (Ctrl+C) or the underlying program exits on its own.
func Attach(ctx context.Context, f *fleet.Fleet, name string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logCh := make(chan string, 512)
	go streamLogs(ctx, f, name, logCh)

	ti := textinput.New()
	ti.Placeholder = "console command — Enter to run via RCON, Ctrl+C to quit"
	ti.Focus()
	ti.CharLimit = 512
	ti.Prompt = "> "

	m := &model{
		f:     f,
		ctx:   ctx,
		name:  name,
		input: ti,
		logCh: logCh,
	}

	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// streamLogs tails name's container output into ch, one line per
// message, until ctx is canceled or the stream ends on its own.
func streamLogs(ctx context.Context, f *fleet.Fleet, name string, ch chan<- string) {
	w := &lineChanWriter{ch: ch}
	err := f.Logs(ctx, name, true, "100", w)
	select {
	case ch <- fmt.Sprintf("[log stream ended: %v]", err):
	default:
	}
	close(ch)
}

// lineChanWriter adapts an io.Writer expecting arbitrary byte chunks
// (fleet.Logs) into one channel send per complete line.
type lineChanWriter struct {
	ch  chan<- string
	buf []byte
}

func (w *lineChanWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:i]), "\r")
		w.buf = w.buf[i+1:]
		w.ch <- line
	}
	return len(p), nil
}

type model struct {
	f    *fleet.Fleet
	ctx  context.Context
	name string

	viewport viewport.Model
	input    textinput.Model
	ready    bool
	width    int
	height   int

	lines []string

	stats    fleet.ServerStats
	statsErr error

	logCh <-chan string
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(waitForLogLine(m.logCh), fetchStats(m.f, m.ctx, m.name), tickCmd())
}

func waitForLogLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logLineMsg(line)
	}
}

func fetchStats(f *fleet.Fleet, ctx context.Context, name string) tea.Cmd {
	return func() tea.Msg {
		s, err := f.SingleStats(ctx, name)
		if err != nil {
			return statsErrMsg{err}
		}
		return statsMsg(s)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(statsInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func execCommand(f *fleet.Fleet, ctx context.Context, name, command string) tea.Cmd {
	return func() tea.Msg {
		out, err := f.Exec(ctx, name, command)
		return execResultMsg{output: out, err: err}
	}
}

func (m *model) appendLine(line string) {
	m.lines = append(m.lines, line)
	if len(m.lines) > maxLogLines {
		m.lines = m.lines[len(m.lines)-maxLogLines:]
	}
	if !m.ready {
		return
	}
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	if atBottom {
		m.viewport.GotoBottom()
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// header (1 row) + input box (1 row + 2 border) + the viewport's
		// own border (2 rows) all come out of the terminal height; the
		// viewport's border also takes 2 columns.
		const chromeRows, chromeCols = 1 + 3 + 2, 2
		vpHeight := m.height - chromeRows
		if vpHeight < 1 {
			vpHeight = 1
		}
		vpWidth := m.width - chromeCols
		if vpWidth < 1 {
			vpWidth = 1
		}
		if !m.ready {
			m.viewport = viewport.New(vpWidth, vpHeight)
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.viewport.GotoBottom()
			m.ready = true
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = vpHeight
		}
		m.input.Width = m.width - chromeCols - len(m.input.Prompt) - 1
		return m, nil

	case logLineMsg:
		m.appendLine(string(msg))
		return m, waitForLogLine(m.logCh)

	case logDoneMsg:
		return m, nil

	case statsMsg:
		m.stats = fleet.ServerStats(msg)
		m.statsErr = nil
		return m, nil

	case statsErrMsg:
		m.statsErr = msg.err
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchStats(m.f, m.ctx, m.name), tickCmd())

	case execResultMsg:
		if msg.err != nil {
			m.appendLine("[error] " + msg.err.Error())
		} else if out := strings.TrimSpace(msg.output); out != "" {
			m.appendLine("> " + out)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			command := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if command == "" {
				return m, nil
			}
			m.appendLine("$ " + command)
			return m, execCommand(m.f, m.ctx, m.name, command)
		}
	}

	var cmds [2]tea.Cmd
	m.input, cmds[0] = m.input.Update(msg)
	m.viewport, cmds[1] = m.viewport.Update(msg)
	return m, tea.Batch(cmds[0], cmds[1])
}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

func (m *model) View() string {
	if !m.ready {
		return "loading…"
	}

	status := m.stats.Status
	if status == "" {
		status = "unknown"
	}
	statsLine := fmt.Sprintf("%s  status: %s  cpu: %.1f%%  mem: %.0fMiB  players: %s",
		m.name, status, m.stats.CPUPercent, mib(m.stats.MemUsageBytes), m.stats.Players)
	if m.statsErr != nil {
		statsLine = fmt.Sprintf("%s  (stats unavailable: %v)", m.name, m.statsErr)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(m.width).Render(statsLine),
		paneStyle.Width(m.width-2).Render(m.viewport.View()),
		paneStyle.Width(m.width-2).Render(m.input.View()),
	)
}

func mib(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}
