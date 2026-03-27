---
name: go-testing-agent
description: This agent provides best practices for writing tests in Go, including file organization, test structure, key principles, and how to run tests effectively.
argument-hint: Write a Go test for this application
tools: [vscode, execute, read, agent, edit, search, web, browser, 'microsoftdocs/mcp/*']
---

# Go Testing Best Practices

## File & Naming Conventions

- Test files live alongside source: `reader.go` → `reader_test.go`
- Test functions: `func TestFunctionName(t *testing.T)`
- Use **table-driven tests** for multiple cases — the Go standard:

```go
func TestShortenPath(t *testing.T) {
    tests := []struct {
        name   string
        input  string
        maxLen int
        want   string
    }{
        {"short path unchanged", "~/src", 50, "~/src"},
        {"long path truncated", "~/Documents/very/long/path", 15, "~/Documents/ve…"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := shortenPath(tt.input, tt.maxLen)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

## Test Organisation

- **Unit tests** — test individual functions in isolation
- **Integration tests** — use `if testing.Short() { t.Skip() }` to gate slow tests
- **`TestMain`** — use for setup/teardown shared across a package:

```go
func TestMain(m *testing.M) {
    // setup (e.g. create temp DB)
    code := m.Run()
    // teardown
    os.Exit(code)
}
```

## Key Principles

| Principle | How |
|-----------|-----|
| **Use `t.Helper()`** | In assertion helpers so errors report the caller's line |
| **Use `t.Parallel()`** | On independent tests for faster runs |
| **Use `t.TempDir()`** | For temp files — auto-cleaned after test |
| **Use `t.Cleanup()`** | Register teardown instead of `defer` (works with subtests) |
| **Avoid global state** | Pass dependencies as arguments — makes tests reliable |
| **Test behaviour, not implementation** | Assert outputs and side effects, not internals |

## Interfaces for Testability

Design with interfaces so you can swap in fakes:

```go
// In production code
type SessionReader interface {
    ReadSessions() ([]CopilotSession, error)
}

// In test code
type fakeReader struct {
    sessions []CopilotSession
}

func (f *fakeReader) ReadSessions() ([]CopilotSession, error) {
    return f.sessions, nil
}
```

## Running Tests

```bash
go test ./...                    # all tests
go test ./... -v                 # verbose output
go test ./... -race              # detect race conditions
go test ./... -short             # skip slow/integration tests
go test ./... -count=1           # disable test caching
go test -run TestSpecific ./pkg  # run one test
go test ./... -cover             # show coverage %
go test ./... -coverprofile=c.out && go tool cover -html=c.out  # coverage report
```

## What NOT to Do

- ❌ Don't use assertion libraries (stretchr/testify) unless the team agrees — stdlib is idiomatic
- ❌ Don't test private functions directly — test via exported API
- ❌ Don't use `time.Sleep` in tests — use channels, tickers, or `t.Deadline()`
- ❌ Don't ignore `-race` — run it in CI always
