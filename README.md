# Copilot CLI Session Monitor

A lightweight macOS menu bar app that shows your active [GitHub Copilot CLI](https://github.com/features/copilot/cli) sessions at a glance. Click any session to jump straight to its terminal tab.

![Copilot CLI Session Monitor](assets/copilot-cli-session-monitor-screenshot.png)

## Features

- 🖥️ **Live session list** — see all active Copilot CLI sessions in your menu bar
- 🖱️ **One-click tab switching** — click a session to focus its terminal tab
- 🔄 **Auto-refresh** — updates every 30 seconds automatically
- 📁 **Directory context** — shows the working directory for each session
- 🪶 **Lightweight** — single binary, no dock icon, reads local files only (no network)

## How It Works

Copilot CLI stores session metadata locally. This app reads two sources:

1. **`~/.copilot/session-store.db`** — SQLite database with session names, directories, and timestamps
2. **`~/.copilot/session-state/*/inuse.*.lock`** — lock files that indicate which sessions have a live process

The app cross-references lock file PIDs with running processes to determine which sessions are active, then displays them in a native macOS menu bar dropdown.

```mermaid
graph LR
    A["~/.copilot/session-store.db"] -->|SQLite read-only| B["Session Reader"]
    C["~/.copilot/session-state/*/inuse.*.lock"] -->|Glob + PID check| B
    B -->|Active sessions| D["Menu Builder"]
    D -->|systray API| E["macOS Menu Bar"]
    E -->|Click session| F["AppleScript"]
    F -->|Switch tab| G["Terminal.app / Ghostty"]
    H["30s Timer"] -->|Refresh| B

    style A fill:#2d333b,color:#e6edf3,stroke:#444
    style C fill:#2d333b,color:#e6edf3,stroke:#444
    style E fill:#1a7f37,color:#fff,stroke:#444
    style G fill:#2d333b,color:#e6edf3,stroke:#444
```

> **Note:** This app is read-only — it never modifies any Copilot files.

## Prerequisites

- **macOS 26 (Tahoe)** or later
- **Go 1.25+** (for building from source)
- **GitHub Copilot CLI 1.0.12** or later installed and used at least once (so `~/.copilot/` exists)

## Quick Start

```bash
git clone https://github.com/liamchampton/copilot-cli-session-monitor.git
cd copilot-cli-session-monitor
make bundle
open 'Copilot CLI Session Monitor.app'
```

## Installation

### Option 1: Install to Applications (recommended)

Build the `.app` bundle and copy it to `/Applications`:

```bash
make install
```

Then launch from Spotlight or Finder: search for **"Copilot CLI Session Monitor"**.

### Option 2: Run from source

Build and run the binary directly (useful for development):

```bash
make run
```

### Option 3: Build only

Compile the binary without creating an `.app` bundle:

```bash
make build
./copilot-monitor
```

> **Note:** Running the bare binary works, but macOS requires a `.app` bundle for the menu bar icon to render reliably. Use `make bundle` or `make install` for the best experience.

## Usage

Once running, you'll see a small terminal icon in your macOS menu bar:

- **Solid** — at least one Copilot session is active
- **Faded** — no active sessions

Click the icon to see:

```
● Plan Terminal Session Sidebar
  ~/Documents/github/agent-controller

● Azure VS Code Changelog Calendar
  ~/Documents/github/everything-vs-code-shipped

● product-store-demo
  ~/Documents/github/product-store-demo

──────────────────────────
3 active · 13:25
──────────────────────────
Quit
```

**Click any session name** to switch to its terminal tab.

### Terminal Compatibility

Session detection works with **any terminal** — it reads from Copilot CLI's local files, not the terminal itself. Tab switching currently supports macOS default terminal and Ghostty.

| Terminal | Session list | Active/idle status | Click to switch tab |
|----------|:---:|:---:|:---:|
| **macOS default terminal** | ✅ | ✅ | ✅ |
| **Ghostty** | ✅ | ✅ | ✅ |
| **iTerm2** | ✅ | ✅ | ❌ Not yet |
| **Warp** | ✅ | ✅ | ❌ Not yet |
| **Kitty** | ✅ | ✅ | ❌ Not yet |
| **Alacritty** | ✅ | ✅ | ❌ Not yet |
| **tmux** | ✅ | ✅ | ❌ Not yet |
| **VS Code terminal** | ✅ | ✅ | ❌ Not yet |

> **Want support for your terminal?** Contributions welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

### macOS Permissions

On first launch, macOS will ask for **Automation** permission to control Terminal.app (or Ghostty, if detected). This is required for the tab-switching feature. Grant it in:

**System Settings → Privacy & Security → Automation → Copilot CLI Session Monitor → Terminal / Ghostty**

## Uninstall

### If installed to /Applications:

```bash
make uninstall
```

Or manually delete:

```bash
rm -rf '/Applications/Copilot CLI Session Monitor.app'
```

### Remove all traces:

The app itself stores no data. To fully clean up:

```bash
# Remove the app
rm -rf '/Applications/Copilot CLI Session Monitor.app'

# Remove build artifacts (if in the project directory)
make clean
```

> **Note:** This does **not** touch your `~/.copilot/` directory — that belongs to GitHub Copilot CLI.

## Development

### Project Structure

```
copilot-cli-session-monitor/
├── main.go                        # Entry point — systray.Run()
├── internal/
│   ├── menu/
│   │   ├── builder.go             # Systray menu construction
│   │   └── builder_test.go        # Unit tests for menu builder
│   ├── monitor/
│   │   └── monitor.go             # Refresh timer orchestration
│   ├── session/
│   │   ├── types.go               # CopilotSession struct
│   │   ├── reader.go              # SQLite + lock file reader
│   │   └── reader_test.go         # Unit tests for session reader
│   └── terminal/
│       └── focus.go               # Terminal tab switching (Terminal.app + Ghostty)
├── assets/
│   ├── AppIcon.icns               # macOS app icon (icns format)
│   ├── app-icon.png               # App icon source (png)
│   ├── icon-active.png            # Green menu bar icon
│   └── icon-idle.png              # Gray menu bar icon
├── bundle/
│   └── Info.plist                 # macOS app bundle metadata
├── .github/
│   └── agents/
│       └── go-testing-agent.agent.md  # Copilot agent for Go tests
├── Makefile                       # Build, bundle, install targets
├── CONTRIBUTING.md                # Contribution guidelines
├── LICENSE                        # Project license
├── go.mod
└── go.sum
```

### Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile the Go binary |
| `make bundle` | Build + package as `Copilot CLI Session Monitor.app` |
| `make install` | Build + bundle + copy to `/Applications` |
| `make uninstall` | Remove from `/Applications` |
| `make run` | Build and run directly (for development) |
| `make clean` | Remove build artifacts |

### Key Design Decisions

- **Read-only** — never writes to Copilot's files
- **Pure Go SQLite** (`modernc.org/sqlite`) — no CGo dependency for database access
- **Goroutine-safe** — cancel pattern prevents leaked goroutines on menu refresh
- **Persistent DB connection** — opened once, reused across refreshes
- **Efficient queries** — scans lock files first, then queries only active session IDs

## Tech Stack

- [Go](https://go.dev/) — application language
- [fyne.io/systray](https://pkg.go.dev/fyne.io/systray) — cross-platform system tray
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — pure Go SQLite driver
- AppleScript — Terminal.app and Ghostty tab switching

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) and open an issue first to discuss what you'd like to change.
