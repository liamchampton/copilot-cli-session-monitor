package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary SQLite database with the sessions table and
// returns a Reader wired up to it along with a cleanup function.
func setupTestDB(t *testing.T, copilotDir string) *Reader {
	t.Helper()

	dbPath := filepath.Join(copilotDir, "session-store.db")
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=rwc", dbPath))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		cwd TEXT,
		repository TEXT,
		summary TEXT,
		updated_at TEXT
	)`)
	if err != nil {
		t.Fatalf("creating sessions table: %v", err)
	}

	// Reopen read-only as the Reader would
	db.Close()
	roDB, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", dbPath))
	if err != nil {
		t.Fatalf("reopening db read-only: %v", err)
	}

	t.Cleanup(func() { roDB.Close() })

	return &Reader{copilotDir: copilotDir, db: roDB}
}

// insertSession inserts a session into the test database.
func insertSession(t *testing.T, copilotDir string, id, cwd, repo, summary string, updatedAt time.Time) {
	t.Helper()

	dbPath := filepath.Join(copilotDir, "session-store.db")
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=rwc", dbPath))
	if err != nil {
		t.Fatalf("opening db for insert: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(
		`INSERT INTO sessions (id, cwd, repository, summary, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, cwd, repo, summary, updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("inserting session: %v", err)
	}
}

func TestScanLockFiles_NoFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := &Reader{copilotDir: dir}

	pids, err := r.scanLockFiles()
	if err != nil {
		t.Fatalf("scanLockFiles() error: %v", err)
	}
	if len(pids) != 0 {
		t.Errorf("expected no active PIDs, got %d", len(pids))
	}
}

func TestScanLockFiles_ActiveProcess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sessionID := "test-session-123"
	// Use our own PID (guaranteed alive)
	pid := os.Getpid()

	lockDir := filepath.Join(dir, "session-state", sessionID)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("creating lock dir: %v", err)
	}

	lockFile := filepath.Join(lockDir, fmt.Sprintf("inuse.%d.lock", pid))
	if err := os.WriteFile(lockFile, nil, 0o644); err != nil {
		t.Fatalf("writing lock file: %v", err)
	}

	r := &Reader{copilotDir: dir}
	pids, err := r.scanLockFiles()
	if err != nil {
		t.Fatalf("scanLockFiles() error: %v", err)
	}

	if got, ok := pids[sessionID]; !ok {
		t.Error("expected session to be in active PIDs map")
	} else if got != pid {
		t.Errorf("PID = %d, want %d", got, pid)
	}
}

func TestScanLockFiles_DeadProcess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sessionID := "dead-session"
	// Use PID that is very unlikely to exist
	deadPID := 99999999

	lockDir := filepath.Join(dir, "session-state", sessionID)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("creating lock dir: %v", err)
	}

	lockFile := filepath.Join(lockDir, fmt.Sprintf("inuse.%d.lock", deadPID))
	if err := os.WriteFile(lockFile, nil, 0o644); err != nil {
		t.Fatalf("writing lock file: %v", err)
	}

	r := &Reader{copilotDir: dir}
	pids, err := r.scanLockFiles()
	if err != nil {
		t.Fatalf("scanLockFiles() error: %v", err)
	}

	if len(pids) != 0 {
		t.Errorf("expected no active PIDs for dead process, got %v", pids)
	}
}

func TestScanLockFiles_MalformedFileName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sessionID := "malformed-session"

	lockDir := filepath.Join(dir, "session-state", sessionID)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("creating lock dir: %v", err)
	}

	// File with wrong number of dot-separated parts
	badFile := filepath.Join(lockDir, "inuse.lock")
	if err := os.WriteFile(badFile, nil, 0o644); err != nil {
		t.Fatalf("writing bad lock file: %v", err)
	}

	// File with non-numeric PID
	badPIDFile := filepath.Join(lockDir, "inuse.notanumber.lock")
	if err := os.WriteFile(badPIDFile, nil, 0o644); err != nil {
		t.Fatalf("writing bad PID lock file: %v", err)
	}

	r := &Reader{copilotDir: dir}
	pids, err := r.scanLockFiles()
	if err != nil {
		t.Fatalf("scanLockFiles() error: %v", err)
	}

	if len(pids) != 0 {
		t.Errorf("expected no PIDs for malformed files, got %v", pids)
	}
}

func TestQuerySessionsByIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := setupTestDB(t, dir)

	now := time.Now().UTC().Truncate(time.Millisecond)
	insertSession(t, dir, "sess-1", "/home/user/project-a", "org/repo-a", "Fix bug", now)
	insertSession(t, dir, "sess-2", "/home/user/project-b", "org/repo-b", "", now.Add(-time.Minute))

	activePIDs := map[string]int{
		"sess-1": 100,
		"sess-2": 200,
	}

	sessions, err := r.querySessionsByIDs(activePIDs)
	if err != nil {
		t.Fatalf("querySessionsByIDs() error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Build a map for easier assertions
	byID := make(map[string]CopilotSession)
	for _, s := range sessions {
		byID[s.ID] = s
	}

	s1 := byID["sess-1"]
	if s1.CWD != "/home/user/project-a" {
		t.Errorf("sess-1 CWD = %q, want %q", s1.CWD, "/home/user/project-a")
	}
	if s1.Summary != "Fix bug" {
		t.Errorf("sess-1 Summary = %q, want %q", s1.Summary, "Fix bug")
	}
	if s1.PID != 100 {
		t.Errorf("sess-1 PID = %d, want %d", s1.PID, 100)
	}
	if s1.Status != StatusActive {
		t.Errorf("sess-1 Status = %q, want %q", s1.Status, StatusActive)
	}

	s2 := byID["sess-2"]
	if s2.Repository != "org/repo-b" {
		t.Errorf("sess-2 Repository = %q, want %q", s2.Repository, "org/repo-b")
	}
}

func TestQuerySessionsByIDs_NoMatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := setupTestDB(t, dir)

	activePIDs := map[string]int{
		"nonexistent": 999,
	}

	sessions, err := r.querySessionsByIDs(activePIDs)
	if err != nil {
		t.Fatalf("querySessionsByIDs() error: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestReadSessions_NoLockFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := setupTestDB(t, dir)

	sessions, err := r.ReadSessions()
	if err != nil {
		t.Fatalf("ReadSessions() error: %v", err)
	}

	if sessions != nil {
		t.Errorf("expected nil sessions, got %v", sessions)
	}
}

func TestReadSessions_SortedByUpdatedAt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := setupTestDB(t, dir)

	now := time.Now().UTC()
	insertSession(t, dir, "old-session", "/old", "", "Old", now.Add(-10*time.Minute))
	insertSession(t, dir, "new-session", "/new", "", "New", now)

	pid := os.Getpid()

	// Create lock files for both sessions using our own PID
	for _, id := range []string{"old-session", "new-session"} {
		lockDir := filepath.Join(dir, "session-state", id)
		if err := os.MkdirAll(lockDir, 0o755); err != nil {
			t.Fatalf("creating lock dir: %v", err)
		}
		lockFile := filepath.Join(lockDir, fmt.Sprintf("inuse.%d.lock", pid))
		if err := os.WriteFile(lockFile, nil, 0o644); err != nil {
			t.Fatalf("writing lock file: %v", err)
		}
	}

	sessions, err := r.ReadSessions()
	if err != nil {
		t.Fatalf("ReadSessions() error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Most recent first
	if sessions[0].ID != "new-session" {
		t.Errorf("expected newest session first, got %q", sessions[0].ID)
	}
	if sessions[1].ID != "old-session" {
		t.Errorf("expected oldest session second, got %q", sessions[1].ID)
	}
}

func TestIsProcessAlive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pid  int
		want bool
	}{
		{"own process is alive", os.Getpid(), true},
		{"nonexistent PID", 99999999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isProcessAlive(tt.pid)
			if got != tt.want {
				t.Errorf("isProcessAlive(%d) = %v, want %v", tt.pid, got, tt.want)
			}
		})
	}
}

func TestNewReader_MissingDB(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := NewReader(dir)
	if err == nil {
		t.Fatal("expected error for missing database, got nil")
	}
}
