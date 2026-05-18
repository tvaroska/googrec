# PRD: googrec — Google Recorder CLI

| Field        | Value                              |
|--------------|------------------------------------|
| Product      | googrec                            |
| Version      | 1.0                                |
| Status       | Approved                           |
| Last updated | 2026-05-17                         |
| Owner        | Boris Tvaroska                     |

---

## 1. Problem Statement

Google Recorder on Pixel phones automatically transcribes voice recordings and syncs them to the cloud. There is no official API or CLI to export this data. Users who want to archive, search, or process their recordings programmatically must do so manually through the web UI.

A prior TypeScript implementation was delivered by an external agency but failed quality review: no tests, SQL injection patterns, hardcoded values, no pagination, macOS-only features, and duplicated code. It does not meet production standards.

## 2. Objective

Build a clean, tested, single-binary CLI tool in Go that connects to Google Recorder's undocumented gRPC-web API and provides reliable access to recordings, transcripts, and audio files.

### Success Criteria

- Single compiled binary with zero runtime dependencies
- All commands functional with manual cookie-paste authentication
- Test coverage for auth storage, API parsing, hash computation, and formatting
- Clean `go vet` and `go test ./...`
- Concurrent bulk downloads with progress reporting

## 3. Scope

### In Scope

- CLI commands: `auth`, `list`, `info`, `transcript`, `search`, `download`, `audio`, `download-audio`, `config`, `--version`
- Manual cookie-paste authentication (cookies + API key from Chrome DevTools)
- SAPISIDHASH authentication for the gRPC-web API
- Concurrent bulk downloads with configurable parallelism
- JSON and plain text output formats
- Date range filtering (`--since`, `--until`), skip-existing, and limit options for bulk operations
- Input validation (UUID format, positive integers, valid dates)
- Unit tests for core logic

### Out of Scope (Future)

- Gemini AI integration (transcription, translation)
- Automatic Chrome cookie extraction
- Playwright browser-based authentication
- Pagination across multiple API pages (`--all` flag)
- `--quiet` / `--verbose` global flags
- Retry with exponential backoff on transient errors
- Publishing to npm/Homebrew/package registries

## 4. User Personas

### Primary: Developer / Power User

- Has recordings synced to Google Recorder cloud
- Comfortable with CLI tools and Chrome DevTools
- Wants to bulk-export transcripts for archival, search, or downstream processing
- May run the tool on a schedule (cron) to incrementally sync new recordings

## 5. Authentication

### Flow

1. User runs `googrec auth`
2. CLI displays step-by-step instructions for extracting cookies and API key from Chrome DevTools
3. User pastes cookie string — CLI validates that SAPISID cookie is present
4. User pastes API key (or skips to set later via `googrec auth --api-key <key>`)
5. Credentials saved to `~/.config/googrec/auth.json` with `0600` file permissions
6. CLI tests authentication by calling `GetRecordingList` with limit 1
7. Reports success or failure with actionable guidance

### Verification

`googrec auth --check` loads stored credentials and makes a test API call.

### Multi-Account

`--authuser N` flag selects the Google account index (0-based, matching `authuser` in Google URLs).

### Security

- Auth file stored with owner-only read/write (`0600`)
- Config directory created with `0700`
- Auth file contains raw session cookies — documentation warns users to treat it as sensitive
- No credentials logged or printed to stdout

## 6. Commands

### 6.1 `auth` — Set up authentication

| Flag | Description |
|------|-------------|
| `--check` | Test existing authentication |
| `--api-key <key>` | Set/update the API key |
| `--authuser <N>` | Google account index (default: 0) |

### 6.2 `list` — List recent recordings

| Flag | Description |
|------|-------------|
| `-n, --limit <N>` | Maximum recordings (default: 20) |
| `--json` | JSON output |

**Output (text):**
```
Found 3 recording(s):

  bf3451e0-4ea6-424e-8e77-fbef4c0fe17c
    Weekly standup
    Jan 15 2026 9:00 AM | 12:34

  a1b2c3d4-...
    Interview notes
    Jan 14 2026 2:30 PM | 45:01 | Conference Room B
```

### 6.3 `info <id>` — Recording details

| Flag | Description |
|------|-------------|
| `--json` | JSON output |

### 6.4 `transcript <id>` — Download single transcript

| Flag | Description |
|------|-------------|
| `-o, --output <file>` | Output file (default: stdout) |
| `--json` | JSON with speaker segments and timestamps |
| `--plain` | Plain text without speaker labels |

**Output (default):**
```
[Speaker 1] (00:00)
Hello, welcome to today's meeting...

[Speaker 2] (01:23)
Thanks for having me...
```

### 6.5 `search <query>` — Search by title

Client-side title filtering across the most recent 100 recordings.

| Flag | Description |
|------|-------------|
| `-n, --limit <N>` | Maximum results (default: 20) |
| `--json` | JSON output |

### 6.6 `download` — Bulk download transcripts

| Flag | Description |
|------|-------------|
| `-o, --output <dir>` | Output directory (default: `.`) |
| `-n, --limit <N>` | Maximum recordings (default: 50) |
| `--since <date>` | Only recordings after date (YYYY-MM-DD or ISO 8601) |
| `--until <date>` | Only recordings before date (YYYY-MM-DD or ISO 8601) |
| `--format <fmt>` | Output format: `txt` or `json` (default: `txt`) |
| `--json` | Alias for `--format json` |
| `--skip-existing` | Skip recordings with existing output file |
| `--concurrency <N>` | Parallel downloads (default: 3) |

**File naming:** `YYYY-MM-DD Title.{txt,json}`

### 6.7 `audio <id>` — Download single audio file

| Flag | Description |
|------|-------------|
| `-o, --output <file>` | Output file (default: server-provided filename) |

### 6.8 `download-audio` — Bulk download audio

| Flag | Description |
|------|-------------|
| `-o, --output <dir>` | Output directory (default: `.`) |
| `-n, --limit <N>` | Maximum recordings (default: 50) |
| `--since <date>` | Only recordings after date |
| `--until <date>` | Only recordings before date |
| `--skip-existing` | Skip existing files |
| `--concurrency <N>` | Parallel downloads (default: 3) |

**File naming:** `YYYY-MM-DD Title.m4a`

### 6.9 `config` — Show configuration

Displays auth file path, existence, and stored credential summary (no secrets printed).

## 7. API Reference

### 7.1 Transcript / Metadata Endpoint (gRPC-web)

| Property | Value |
|----------|-------|
| Base URL | `https://pixelrecorder-pa.clients6.google.com/$rpc/java.com.google.wireless.android.pixel.recorder.protos.PlaybackService` |
| Content-Type | `application/json+protobuf` |
| Methods | `GetRecordingList`, `GetTranscription` |

**Required headers:**
- `Authorization: SAPISIDHASH <timestamp>_<sha1(timestamp + " " + SAPISID + " " + origin)>`
- `X-Goog-Api-Key: <api_key>`
- `X-Goog-AuthUser: <N>`
- `Cookie: <browser_cookies>`
- `Origin: https://recorder.google.com`

**GetRecordingList request body:**
```json
[[{"1": <unix_timestamp>}], <page_size>]
```

**Recording item array indices:**

| Index | Field |
|-------|-------|
| 0 | Device ID (string) |
| 1 | Title (string) |
| 2 | Created timestamp `[seconds_string, nanoseconds]` |
| 3 | Duration `[seconds_string, nanoseconds]` |
| 4 | Latitude |
| 5 | Longitude |
| 6 | Location name (string) |
| 13 | Web UUID (string) — used for transcript/audio API calls |

**GetTranscription request body:**
```json
["<recording-uuid>"]
```

**Transcript response structure:**
```
[[[segments], ...]]
Each segment: [[words], speaker_id, language_code]
Each word: [text, formatted_text, start_ms_str, end_ms_str, ...]
```

### 7.2 Audio Download Endpoint (HTTP)

| Property | Value |
|----------|-------|
| URL | `https://usercontent.recorder.google.com/download/playback/{id}?authuser={N}&download=true` |
| Method | GET |
| Auth | Cookie header |
| Response | m4a audio, filename in `Content-Disposition` header |

## 8. Architecture

```
googrec/
├── main.go                  # entrypoint
├── cmd/
│   ├── root.go              # cobra root command
│   ├── auth.go              # auth command
│   ├── config.go            # config command
│   ├── list.go              # list command
│   ├── info.go              # info command
│   ├── transcript.go        # transcript command
│   ├── search.go            # search command
│   ├── audio.go             # audio command
│   ├── bulk.go              # shared bulk download logic
│   ├── bulk_test.go         # unit tests for bulk utilities
│   ├── download.go          # bulk transcript download
│   └── download_audio.go    # bulk audio download
├── internal/
│   ├── api/
│   │   ├── client.go        # HTTP client, SAPISIDHASH, API methods
│   │   ├── types.go         # Recording, Transcript, AudioResult types
│   │   └── client_test.go   # unit tests
│   └── auth/
│       ├── store.go         # load/save/validate credentials
│       └── store_test.go    # unit tests
├── doc/
│   └── PRD.md               # this document
├── README.md
├── Makefile
├── go.mod
└── go.sum
```

### Design Principles

1. **Standard library first** — only external dependency is `cobra` for CLI framework
2. **No protobuf library** — the API uses JSON-encoded protobuf; parsed with `encoding/json`
3. **Internal packages** — `internal/` prevents accidental import by external code
4. **Shared bulk logic** — `cmd/bulk.go` provides concurrent download infrastructure used by both transcript and audio bulk commands
5. **Goroutine-based concurrency** — worker pool pattern for parallel downloads

## 9. Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| Go standard library | Everything else (net/http, crypto/sha1, encoding/json, os, sync) |

## 10. Testing

### Unit Tests

| Package | Coverage |
|---------|----------|
| `internal/auth` | Save/Load round-trip, SAPISID extraction, file permissions, API key preservation, env overrides |
| `internal/api` | SAPISIDHASH format, UUID validation, duration formatting, time formatting, recording parsing, transcript parsing (multi-speaker, same-speaker merge, empty response, API errors), audio streaming (Content-Disposition, fallback filename, byte counting), RPC header verification |
| `cmd` | Filename sanitization, date range filtering (since/until/both/RFC3339/invalid/reversed), date parsing |

### Manual Verification

1. `go build` produces a binary
2. `go test ./...` passes
3. `go vet ./...` clean
4. `googrec auth` → paste credentials → `auth --check` succeeds
5. `googrec list --limit 3` shows recordings
6. `googrec transcript <id>` outputs transcript
7. `googrec audio <id>` downloads audio file
8. `googrec download -o ./out --limit 5 --concurrency 2` bulk downloads

## 11. Future Work

| Priority | Feature | Description |
|----------|---------|-------------|
| P1 | Pagination (`--all`) | Use API pagination token to fetch all recordings across multiple pages |
| P1 | Retry with backoff | Automatic retry on 429/5xx with exponential backoff, 3 attempts max |
| P2 | `--quiet` / `--verbose` | Global flags for output verbosity control |
| P2 | `hasTranscript` accuracy | Derive from API response field instead of hardcoding `true` |
| P3 | Gemini integration | Transcription and translation via Vertex AI or Developer API |
| P3 | Transcript search | Search within transcript content, not just titles |
