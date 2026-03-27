package menu

import (
	"testing"
	"time"

	"github.com/liamchampton/copilot-cli-session-monitor/internal/session"
)

func TestSessionDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		session session.CopilotSession
		want    string
	}{
		{
			name: "prefers summary when set",
			session: session.CopilotSession{
				ID:      "abc12345-full-id",
				Summary: "Fix login bug",
				CWD:     "/Users/dev/project",
			},
			want: "Fix login bug",
		},
		{
			name: "falls back to directory basename",
			session: session.CopilotSession{
				ID:  "abc12345-full-id",
				CWD: "/Users/dev/my-project",
			},
			want: "my-project",
		},
		{
			name: "falls back to truncated ID",
			session: session.CopilotSession{
				ID: "abc12345-full-id",
			},
			want: "abc12345",
		},
		{
			name: "short ID returned as-is",
			session: session.CopilotSession{
				ID: "short",
			},
			want: "short",
		},
		{
			name:    "empty session returns empty ID",
			session: session.CopilotSession{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sessionDisplayName(tt.session)
			if got != tt.want {
				t.Errorf("sessionDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	t.Parallel()

	b := &Builder{homePath: "/Users/dev"}

	tests := []struct {
		name   string
		path   string
		maxLen int
		want   string
	}{
		{
			name:   "replaces home dir with tilde",
			path:   "/Users/dev/projects/app",
			maxLen: 50,
			want:   "~/projects/app",
		},
		{
			name:   "short path unchanged",
			path:   "~/src",
			maxLen: 50,
			want:   "~/src",
		},
		{
			name:   "truncates long path",
			path:   "/other/very/long/path/to/some/deep/directory/structure",
			maxLen: 20,
			want:   "/other/very/long/pa…",
		},
		{
			name:   "path exactly at maxLen",
			path:   "1234567890",
			maxLen: 10,
			want:   "1234567890",
		},
		{
			name:   "path one over maxLen is truncated",
			path:   "12345678901",
			maxLen: 10,
			want:   "123456789…",
		},
		{
			name:   "empty home path leaves path unchanged",
			path:   "/Users/dev/code",
			maxLen: 50,
			want:   "~/code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := b.shortenPath(tt.path, tt.maxLen)
			if got != tt.want {
				t.Errorf("shortenPath(%q, %d) = %q, want %q", tt.path, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestShortenPathEmptyHome(t *testing.T) {
	t.Parallel()

	b := &Builder{homePath: ""}

	got := b.shortenPath("/Users/dev/code", 50)
	want := "/Users/dev/code"
	if got != want {
		t.Errorf("shortenPath with empty home = %q, want %q", got, want)
	}
}

func TestSessionDisplayName_Priority(t *testing.T) {
	t.Parallel()

	// Verify that summary takes priority even when all fields are set
	s := session.CopilotSession{
		ID:         "abcdefgh-1234",
		Summary:    "My Summary",
		CWD:        "/some/path",
		Repository: "org/repo",
		UpdatedAt:  time.Now(),
		Status:     session.StatusActive,
		PID:        1234,
	}

	got := sessionDisplayName(s)
	if got != "My Summary" {
		t.Errorf("expected summary to take priority, got %q", got)
	}
}
