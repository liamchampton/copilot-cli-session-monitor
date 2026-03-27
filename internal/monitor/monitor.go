package monitor

import (
	"log"
	"time"

	"github.com/liamchampton/copilot-cli-session-monitor/internal/menu"
	"github.com/liamchampton/copilot-cli-session-monitor/internal/session"
)

// Monitor periodically reads session data and updates the menu.
type Monitor struct {
	reader   *session.Reader
	builder  *menu.Builder
	interval time.Duration
	stopCh   chan struct{}
}

// New creates a Monitor that refreshes at the given interval.
func New(reader *session.Reader, builder *menu.Builder, interval time.Duration) *Monitor {
	return &Monitor{
		reader:   reader,
		builder:  builder,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start builds the initial menu and begins the periodic refresh loop.
// Returns a channel that signals when the user clicks Quit.
func (m *Monitor) Start() (quitCh <-chan struct{}) {
	sessions := m.readSessions()
	refreshCh, quit := m.builder.Build(sessions)

	go m.loop(refreshCh)

	return quit
}

// Stop signals the refresh loop to exit and releases resources.
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.reader.Close()
}

func (m *Monitor) loop(refreshCh <-chan struct{}) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refresh()
		case <-refreshCh:
			m.refresh()
			ticker.Reset(m.interval)
		case <-m.stopCh:
			return
		}
	}
}

func (m *Monitor) refresh() {
	sessions := m.readSessions()
	m.builder.Rebuild(sessions)
}

func (m *Monitor) readSessions() []session.CopilotSession {
	sessions, err := m.reader.ReadSessions()
	if err != nil {
		log.Printf("Error reading sessions: %v", err)
		return nil
	}
	return sessions
}
