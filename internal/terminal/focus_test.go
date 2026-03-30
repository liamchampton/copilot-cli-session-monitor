package terminal

import (
	"os"
	"testing"
)

func TestDetectTerminal_Self(t *testing.T) {
	t.Parallel()

	// detectTerminal walks up the process tree and may or may not find
	// a recognised terminal depending on the environment running the tests.
	// We only verify it returns a valid value and does not panic.
	known := map[string]bool{"": true, "Ghostty": true}
	got := detectTerminal(os.Getpid())
	if !known[got] {
		t.Errorf("detectTerminal(self) = %q, want one of %v", got, known)
	}
}

func TestDetectTerminal_InvalidPID(t *testing.T) {
	t.Parallel()

	got := detectTerminal(99999999)
	if got != "" {
		t.Errorf("detectTerminal(99999999) = %q, want empty string", got)
	}
}
