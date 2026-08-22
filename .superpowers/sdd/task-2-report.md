### Task 2: Configuration Package — Report

**Status:** DONE

**Commit:** `d779a38` — `feat: add configuration package with ENV + CLI flag support`

**Files created:**
- `internal/config/config.go` — Config struct (12 fields), `Load()` function, `splitHostPort`, `flagWasSet`, transport flag helpers
- `internal/config/config_test.go` — 7 test cases covering all scenarios

**Test summary:** 7/7 PASS — splitHostPort (7 subtests), defaults, ENV override, CLI override precedence, custom headers JSON, empty custom headers

**Implementation notes:**
- All CLI flags registered in `init()` so they're available before the Go test framework's `flag.Parse()` call — this is critical for subprocess-based config testing
- Priority order enforced: CLI flag > ENV > default value, using `flagWasSet()` helper
- `--http :8080` transport flag splits host:port via `splitHostPort` helper
- Custom headers parsed from JSON string to `map[string]string`
- `TransportFlags()` exported for main.go to access `--stdio` and `--http` flag values

**Design decisions:**
- Used package-level `init()` for flag registration instead of inside `Load()` — required because Go test binaries call `flag.Parse()` before any test runs, so custom flags must be registered at that point
- Subprocess-based testing: each test scenario spawns a fresh test binary via `os.Executable()` to isolate `flag.CommandLine` state and prevent flag pollution between scenarios
