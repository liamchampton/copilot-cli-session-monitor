package menu

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/systray"
	"github.com/liamchampton/copilot-cli-session-monitor/internal/session"
	"github.com/liamchampton/copilot-cli-session-monitor/internal/terminal"
)

// Builder manages the systray menu lifecycle.
type Builder struct {
	activeIcon []byte
	idleIcon   []byte
	homePath   string
	refreshCh  chan struct{}
	quitCh     chan struct{}
	cancelCh   chan struct{} // signals old click-handler goroutines to exit
}

// NewBuilder creates a menu Builder with the given icons.
func NewBuilder(activeIcon, idleIcon []byte) *Builder {
	home, _ := os.UserHomeDir()
	return &Builder{
		activeIcon: activeIcon,
		idleIcon:   idleIcon,
		homePath:   home,
		refreshCh:  make(chan struct{}, 1),
		quitCh:     make(chan struct{}, 1),
	}
}

// Build sets up the systray menu from the given sessions.
// Returns channels for refresh and quit actions.
func (b *Builder) Build(sessions []session.CopilotSession) (refresh <-chan struct{}, quit <-chan struct{}) {
	b.cancelCh = make(chan struct{})
	b.setIcon(sessions)
	b.buildMenuItems(sessions)
	return b.refreshCh, b.quitCh
}

// Rebuild updates the menu with fresh session data,
// cancelling any goroutines from the previous build.
func (b *Builder) Rebuild(sessions []session.CopilotSession) {
	// Signal all goroutines from the previous build to exit
	close(b.cancelCh)
	b.cancelCh = make(chan struct{})

	b.setIcon(sessions)
	systray.ResetMenu()
	b.buildMenuItems(sessions)
}

func (b *Builder) setIcon(sessions []session.CopilotSession) {
	if len(sessions) > 0 {
		systray.SetIcon(b.activeIcon)
		systray.SetTooltip(fmt.Sprintf("Copilot: %d active session(s)", len(sessions)))
	} else {
		systray.SetIcon(b.idleIcon)
		systray.SetTooltip("Copilot: no active sessions")
	}
}

func (b *Builder) buildMenuItems(sessions []session.CopilotSession) {
	done := b.cancelCh

	if len(sessions) == 0 {
		item := systray.AddMenuItem("No active sessions", "")
		item.Disable()
	} else {
		for _, s := range sessions {
			name := sessionDisplayName(s)
			dir := b.shortenPath(s.CWD, 50)

			item := systray.AddMenuItem(
				fmt.Sprintf("● %s", name),
				fmt.Sprintf("Open terminal: %s", s.CWD),
			)

			sub := systray.AddMenuItem(fmt.Sprintf("  %s", dir), "")
			sub.Disable()

			pid := s.PID
			go func() {
				for {
					select {
					case <-done:
						return
					case <-item.ClickedCh:
						if err := terminal.FocusTabByPID(pid); err != nil {
							log.Printf("Failed to focus tab for PID %d: %v", pid, err)
						}
					}
				}
			}()
		}
	}

	systray.AddSeparator()

	summary := systray.AddMenuItem(
		fmt.Sprintf("%d active · %s", len(sessions), time.Now().Format("15:04")),
		"Last refreshed",
	)
	summary.Disable()

	mRefresh := systray.AddMenuItem("↻ Refresh Now", "Refresh session list")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Copilot CLI Session Monitor")

	go func() {
		for {
			select {
			case <-done:
				return
			case <-mRefresh.ClickedCh:
				select {
				case b.refreshCh <- struct{}{}:
				default:
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-mQuit.ClickedCh:
				select {
				case b.quitCh <- struct{}{}:
				default:
				}
			}
		}
	}()
}

func sessionDisplayName(s session.CopilotSession) string {
	if s.Summary != "" {
		return s.Summary
	}
	if s.CWD != "" {
		return filepath.Base(s.CWD)
	}
	if len(s.ID) >= 8 {
		return s.ID[:8]
	}
	return s.ID
}

func (b *Builder) shortenPath(p string, maxLen int) string {
	if b.homePath != "" {
		p = strings.Replace(p, b.homePath, "~", 1)
	}
	if len(p) <= maxLen {
		return p
	}
	return p[:maxLen-1] + "…"
}
