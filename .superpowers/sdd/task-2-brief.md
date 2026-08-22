### Task 2: Configuration Package

- [x] **Step 1:** Create `internal/config/config.go` — `Config` struct with all 12 fields, `Load()` function using `flag` + ENV, `splitHostPort` helper, `flagWasSet` helper
- [x] **Step 2:** Create `internal/config/config_test.go` — table-driven tests: defaults, ENV override, CLI override priority, JSON custom headers parsing
- [x] **Step 3:** `go test ./internal/config/ -v` — verify tests pass
- [x] **Step 4:** Commit: `feat: add configuration package with ENV + CLI flag support`

