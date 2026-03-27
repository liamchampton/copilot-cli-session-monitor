package terminal

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FocusTabByPID detects which terminal emulator owns the given PID and
// brings the corresponding tab/window to the front.
// The cwd parameter is used by terminals (like Ghostty) whose AppleScript
// API does not expose a TTY property — matching falls back to working directory.
func FocusTabByPID(pid int, cwd string) error {
	term := detectTerminal(pid)
	switch term {
	case "Ghostty":
		return focusGhostty(cwd)
	default:
		return focusTerminalApp(pid)
	}
}

// detectTerminal walks up the process tree from pid and returns the name of
// the first recognised terminal emulator it finds. Returns "" if none matched.
func detectTerminal(pid int) string {
	current := pid
	for {
		out, err := exec.Command("ps", "-p", strconv.Itoa(current), "-o", "ppid=,comm=").Output()
		if err != nil {
			break
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) < 2 {
			break
		}
		ppid, err := strconv.Atoi(fields[0])
		if err != nil {
			break
		}
		comm := filepath.Base(strings.Join(fields[1:], " "))
		switch strings.ToLower(comm) {
		case "ghostty":
			return "Ghostty"
		}
		if ppid <= 1 {
			break
		}
		current = ppid
	}
	return ""
}

// focusTerminalApp brings the macOS Terminal.app tab running the given PID to front.
func focusTerminalApp(pid int) error {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "tty=").Output()
	if err != nil {
		return fmt.Errorf("looking up TTY for PID %d: %w", pid, err)
	}

	tty := strings.TrimSpace(string(out))
	if tty == "" || tty == "??" {
		return fmt.Errorf("no TTY found for PID %d", pid)
	}

	if !strings.HasPrefix(tty, "/dev/") {
		tty = "/dev/" + tty
	}

	script := fmt.Sprintf(`
tell application "Terminal"
	activate
	set targetTTY to %q
	repeat with w in windows
		set tabIndex to 0
		repeat with t in tabs of w
			set tabIndex to tabIndex + 1
			if tty of t is targetTTY then
				set selected tab of w to t
				set index of w to 1
				return
			end if
		end repeat
	end repeat
end tell
`, tty)

	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("activating terminal tab: %w (%s)", err, string(out))
	}
	return nil
}

// focusGhostty brings the Ghostty terminal whose working directory matches
// cwd to the front. It iterates windows → tabs → terminals and uses
// Ghostty's AppleScript commands (activate window, select tab, focus).
func focusGhostty(cwd string) error {
	script := fmt.Sprintf(`
tell application "Ghostty"
	activate
	set targetCWD to %q
	repeat with w in windows
		repeat with tb in tabs of w
			repeat with trm in terminals of tb
				if working directory of trm is targetCWD then
					select tab tb
					activate window w
					focus trm
					return
				end if
			end repeat
		end repeat
	end repeat
end tell
`, cwd)

	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("activating Ghostty terminal: %w (%s)", err, string(out))
	}
	return nil
}
