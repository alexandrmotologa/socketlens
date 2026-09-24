package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/alexandrmotologa/socketlens/pkg/codec"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UI styles using lipgloss
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#2563EB")).
			Padding(0, 1)

	badgeConnected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")).
			SetString(" CONNECTED ")

	badgeConnecting = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F59E0B")).
			SetString(" CONNECTING ")

	badgeDisconnected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EF4444")).
			SetString(" DISCONNECTED ")

	inboundTag = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")).
			SetString("IN  ▲")

	outboundTag = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#38BDF8")).
			SetString("OUT ▼")

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1E293B")).
				Foreground(lipgloss.Color("#F8FAFC"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Background(lipgloss.Color("#0F172A")).
			Padding(0, 1)

	detailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3B82F6")).
				Padding(0, 1)
)

// FrameMsg wraps a newly received frame for Bubbletea.
type FrameMsg *client.Frame

// StateMsg wraps a connection state transition.
type StateMsg struct {
	State client.ConnectionState
	Err   error
}

// Model maintains interactive terminal state.
type Model struct {
	client       client.StreamClient
	cfg          client.ConnectionConfig
	frames       []*client.Frame
	selected     int
	paused       bool
	showDetail   bool
	width        int
	height       int
	state        client.ConnectionState
	err          error
	framesChan   chan *client.Frame
	stateChan    chan StateMsg
	mu           sync.Mutex
	program      *tea.Program
}

// NewModel creates an interactive terminal model.
func NewModel(c client.StreamClient, cfg client.ConnectionConfig) *Model {
	return &Model{
		client:     c,
		cfg:        cfg,
		frames:     make([]*client.Frame, 0, 500),
		state:      client.StateConnecting,
		framesChan: make(chan *client.Frame, 256),
		stateChan:  make(chan StateMsg, 16),
	}
}

// SetProgram binds the tea.Program reference.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// OnFrame handles incoming or outgoing frames from the client engine.
func (m *Model) OnFrame(f *client.Frame) {
	if m.paused {
		return
	}

	// In-flight decoding
	fmtName, dec, decErr := codec.DecodeFrame(f.Payload)
	if decErr == nil {
		f.Format = client.PayloadFormat(fmtName)
		f.Decoded = dec
	}

	if m.program != nil {
		m.program.Send(FrameMsg(f))
	}
}

// OnState handles connection state changes.
func (m *Model) OnState(s client.ConnectionState, err error) {
	if m.program != nil {
		m.program.Send(StateMsg{State: s, Err: err})
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles user input and engine events.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			_ = m.client.Close()
			return m, tea.Quit

		case " ":
			m.paused = !m.paused

		case "c":
			m.mu.Lock()
			m.frames = m.frames[:0]
			m.selected = 0
			m.mu.Unlock()

		case "enter":
			if len(m.frames) > 0 {
				m.showDetail = !m.showDetail
			}

		case "esc":
			m.showDetail = false

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < len(m.frames)-1 {
				m.selected++
			}
		}

	case FrameMsg:
		m.mu.Lock()
		m.frames = append(m.frames, msg)
		if len(m.frames) > 1000 {
			m.frames = m.frames[1:]
			if m.selected > 0 {
				m.selected--
			}
		}
		// Auto scroll to bottom unless user is inspecting
		if !m.showDetail {
			m.selected = len(m.frames) - 1
		}
		m.mu.Unlock()

	case StateMsg:
		m.state = msg.State
		m.err = msg.Err
	}

	return m, nil
}

// View renders the terminal UI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing SocketLens..."
	}

	var b strings.Builder

	// 1. Status Bar
	statusBadge := badgeConnected.String()
	if m.state == client.StateConnecting || m.state == client.StateReconnecting {
		statusBadge = badgeConnecting.String()
	} else if m.state == client.StateDisconnected || m.state == client.StateError {
		statusBadge = badgeDisconnected.String()
	}

	stats := m.client.Stats()
	pauseStr := ""
	if m.paused {
		pauseStr = " [PAUSED]"
	}

	header := fmt.Sprintf(" SocketLens  %s %s | Target: %s | In: %d (%d B) | Out: %d (%d B)%s",
		statusBadge,
		strings.ToUpper(string(m.cfg.Protocol)),
		m.cfg.URL,
		stats.FramesReceived,
		stats.BytesReceived,
		stats.FramesSent,
		stats.BytesSent,
		pauseStr,
	)
	b.WriteString(headerStyle.Width(m.width).Render(header))
	b.WriteString("\n")

	// 2. Timeline List or Split Detail
	availableHeight := m.height - 4
	if availableHeight < 5 {
		availableHeight = 5
	}

	m.mu.Lock()
	frameCount := len(m.frames)
	m.mu.Unlock()

	if frameCount == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Padding(2, 2).Render("Waiting for streaming frames..."))
		b.WriteString("\n")
	} else if m.showDetail && m.selected >= 0 && m.selected < frameCount {
		// Detail view
		f := m.frames[m.selected]
		b.WriteString(m.renderDetailView(f, availableHeight))
	} else {
		// Timeline list view
		start := 0
		if frameCount > availableHeight {
			start = frameCount - availableHeight
		}

		m.mu.Lock()
		for i := start; i < frameCount; i++ {
			f := m.frames[i]
			line := m.renderFrameRow(f, i == m.selected)
			b.WriteString(line)
			b.WriteString("\n")
		}
		m.mu.Unlock()
	}

	// 3. Footer Keybindings
	footer := " [q] Quit  [Space] Pause/Resume  [c] Clear  [↑/↓] Select  [Enter] Inspect Payload "
	b.WriteString(footerStyle.Width(m.width).Render(footer))

	return b.String()
}

func (m *Model) renderFrameRow(f *client.Frame, isSelected bool) string {
	tag := inboundTag.String()
	if f.Direction == client.DirectionOutbound {
		tag = outboundTag.String()
	}

	ts := f.Timestamp.Format("15:04:05.000")
	fmtLabel := fmt.Sprintf("[%s]", strings.ToUpper(string(f.Format)))
	if f.Format == "" || f.Format == client.FormatRaw {
		fmtLabel = fmt.Sprintf("[%s]", strings.ToUpper(string(f.OpCode)))
	}

	summary := f.Summary()
	maxLen := m.width - 45
	if maxLen > 10 && len(summary) > maxLen {
		summary = summary[:maxLen-3] + "..."
	}

	row := fmt.Sprintf(" %s  %s  %-10s %6d B  %s",
		tag,
		ts,
		fmtLabel,
		f.Length,
		summary,
	)

	if isSelected {
		return selectedRowStyle.Width(m.width).Render(row)
	}
	return row
}

func (m *Model) renderDetailView(f *client.Frame, height int) string {
	var b strings.Builder
	dir := "INBOUND"
	if f.Direction == client.DirectionOutbound {
		dir = "OUTBOUND"
	}

	meta := fmt.Sprintf("Frame #%d | %s | Opcode: %s | Format: %s | Length: %d bytes | Timestamp: %s",
		f.Sequence,
		dir,
		f.OpCode,
		f.Format,
		f.Length,
		f.Timestamp.Format(time.RFC3339Nano),
	)
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#38BDF8")).Render(meta))
	b.WriteString("\n\n")

	payloadStr := f.Decoded
	if payloadStr == "" {
		payloadStr = string(f.Payload)
	}
	if f.Format == client.FormatRaw || !isPrintable(f.Payload) {
		payloadStr = codec.HexDump(f.Payload)
	}

	lines := strings.Split(payloadStr, "\n")
	maxLines := height - 4
	if maxLines < 5 {
		maxLines = 5
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "... [TRUNCATED]")
	}

	b.WriteString(strings.Join(lines, "\n"))
	return detailPanelStyle.Width(m.width - 2).Render(b.String())
}

func isPrintable(data []byte) bool {
	for _, b := range data {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}
