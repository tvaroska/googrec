# googrec

CLI tool for downloading transcripts and audio from [Google Recorder](https://recorder.google.com) via its web API.

Single compiled Go binary. No runtime dependencies.

## Install

```bash
go install github.com/boris/googrec@latest
```

Or build from source:

```bash
git clone https://github.com/boris/googrec.git
cd googrec
make build
```

## Authentication

The tool authenticates using cookies from Chrome DevTools.

```bash
googrec auth
```

This prompts you to:
1. Open [recorder.google.com](https://recorder.google.com) in Chrome
2. Open DevTools (F12), go to Network tab, refresh
3. Click any request to `pixelrecorder-pa.clients6.google.com`
4. Copy the `Cookie` header value and paste it
5. Copy the `X-Goog-Api-Key` header value and paste it

Credentials are stored in `~/.config/googrec/auth.json` (owner-only permissions). This file contains session cookies — treat it as sensitive.

### Verify

```bash
googrec auth --check
```

### Multi-account

```bash
googrec auth --authuser 1
```

### Update API key only

```bash
googrec auth --api-key <key>
```

## Commands

### List recordings

```bash
googrec list                # 20 most recent (default)
googrec list -n 50          # more
googrec list --json         # JSON output
```

### Recording details

```bash
googrec info <id>
googrec info <id> --json
```

### Download transcript

```bash
googrec transcript <id>              # stdout
googrec transcript <id> -o out.txt   # file
googrec transcript <id> --json       # JSON with segments
googrec transcript <id> --plain      # no speaker labels
```

### Download audio

```bash
googrec audio <id>
googrec audio <id> -o meeting.m4a
```

### Search by title

```bash
googrec search "meeting"
googrec search "meeting" -n 5 --json
```

### Bulk download transcripts

```bash
googrec download                          # current dir, up to 50
googrec download -o ./transcripts -n 100  # custom dir and limit
googrec download --since 2025-01-01       # date filter
googrec download --skip-existing          # incremental
googrec download --format json            # JSON format
googrec download --concurrency 5          # parallel downloads
```

### Bulk download audio

```bash
googrec download-audio -o ./audio
googrec download-audio --since 2025-06-01 --skip-existing
googrec download-audio --concurrency 5
```

### Show config

```bash
googrec config
```

## Troubleshooting

**"not authenticated"** — Run `googrec auth` to set up credentials.

**"API request failed: 401"** — Cookies expired. Re-run `googrec auth`.

**"API request failed: 403"** — Wrong account index. Try `googrec auth --authuser 1`.

**"no API key configured"** — Run `googrec auth --api-key <key>` with the key from DevTools.

**No recordings** — Verify recordings appear at [recorder.google.com](https://recorder.google.com).

## How it works

Uses two Google APIs authenticated with browser cookies:

- **Transcript/metadata**: gRPC-web at `pixelrecorder-pa.clients6.google.com` with SAPISIDHASH auth
- **Audio download**: HTTP GET at `usercontent.recorder.google.com`

## License

MIT
