# googrec

CLI tool for downloading transcripts and audio from [Google Recorder](https://recorder.google.com) via its web API.

Single compiled Go binary. No runtime dependencies.

## Install

### Download binary

Grab the latest release for your platform from [GitHub Releases](https://github.com/tvaroska/googrec/releases):

```bash
# Example: Linux amd64
curl -Lo googrec.tar.gz https://github.com/tvaroska/googrec/releases/latest/download/googrec_linux_amd64.tar.gz
tar xzf googrec.tar.gz
sudo mv googrec /usr/local/bin/
```

### Go install

```bash
go install github.com/tvaroska/googrec@latest
```

### Build from source

```bash
git clone https://github.com/tvaroska/googrec.git
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

### Environment variables

For non-interactive use (agents, CI, scripts), set credentials via environment variables instead of `googrec auth`:

```bash
export GOOGREC_COOKIES="<cookie header value>"
export GOOGREC_API_KEY="<api key>"
export GOOGREC_AUTHUSER=0  # optional, default 0
```

Environment variables override the auth file. No auth file is needed when all required env vars are set.

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
googrec download --since 2025-01-01       # after date
googrec download --until 2025-06-01       # before date
googrec download --since 2025-01-01 --until 2025-06-01  # date range
googrec download --skip-existing          # incremental
googrec download --format json            # JSON format
googrec download --concurrency 5          # parallel downloads
```

### Bulk download audio

```bash
googrec download-audio -o ./audio
googrec download-audio --since 2025-06-01 --until 2025-12-01  # date range
googrec download-audio --skip-existing
googrec download-audio --concurrency 5
```

### Show config

```bash
googrec config
```

### Version

```bash
googrec --version
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
