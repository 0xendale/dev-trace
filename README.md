# DevTrace

A Go CLI tool that records terminal sessions via PTY wrapping, captures git diffs, and generates a structured Markdown report using an AI API (Gemini or OpenAI).

**Platform:** macOS and Linux only.

---

## Installation

```bash
go install github.com/0xendale/devtrace/cmd/devtrace@latest
```

Or build from source:

```bash
git clone https://github.com/0xendale/devtrace
cd devtrace
go build -o devtrace ./cmd/devtrace
```

---

## Quickstart

```bash
# 1. Run interactive setup (first time only)
devtrace setup

# 2. Start recording your terminal session
devtrace start

# 3. Do your work — write code, run tests, build things

# 4. Stop recording
devtrace stop

# 5. Generate a Markdown report
devtrace gen
```

`REPORT.md` is written to your current directory.

---

## Setup

Run the interactive setup wizard to configure your API key:

```bash
devtrace setup
```

```
devtrace setup

Select provider:
  [1] Gemini  (free tier available — https://aistudio.google.com/apikey)
  [2] OpenAI  (https://platform.openai.com/api-keys)
  [3] Other   (any OpenAI-compatible endpoint)
Choice [1]:
API key: ********
[devtrace] config written to ~/.devtrace/config.toml
```

Config is saved to `~/.devtrace/config.toml` with `0600` permissions.

---

## Configuration

`~/.devtrace/config.toml` (created by `devtrace setup` or on first run):

```toml
provider = "gemini"           # "gemini" or "openai" (default: openai)
api_key  = "AIza..."          # or set DEVTRACE_API_KEY env var
model    = "gemini-2.0-flash" # model to use
hours    = 8                  # git diff time window in hours
endpoint = ""                 # custom endpoint for OpenAI-compatible APIs (Groq, Ollama, etc.)
```

**API key resolution order:** `DEVTRACE_API_KEY` env var → `api_key` in config.

There is no `--api-key` CLI flag by design — passing secrets on the command line exposes them to shell history and `ps`.

### Provider examples

**Gemini (free tier):**
```toml
provider = "gemini"
api_key  = "AIza..."
model    = "gemini-2.0-flash"
```
Get a free key at [aistudio.google.com/apikey](https://aistudio.google.com/apikey).

**OpenAI:**
```toml
provider = "openai"
api_key  = "sk-..."
model    = "gpt-4o"
```

**Groq (free tier, OpenAI-compatible):**
```toml
api_key  = "gsk_..."
model    = "llama-3.1-8b-instant"
endpoint = "https://api.groq.com/openai/v1/chat/completions"
```
Get a free key at [console.groq.com](https://console.groq.com).

### Per-project config

Place a `config.toml` in your project directory to override the global config. It is loaded first if present. Add it to `.gitignore` — it contains your API key.

---

## Commands

### `devtrace start`

Wraps your `$SHELL` in a PTY recorder. All terminal output is captured through a redaction pipeline and appended to `~/.devtrace/session.log`. Stdin (keystrokes) is forwarded to the shell but never logged.

### `devtrace stop`

Sends `SIGTERM` to the recorder. Waits up to 5 seconds for a clean exit, then escalates to `SIGKILL`. Removes the PID file.

### `devtrace gen [--hours=N]`

1. Reads `~/.devtrace/session.log`
2. Fetches the git diff for the last N hours from the current directory
3. Runs a second redaction pass on all content
4. Shows a confirmation dialog (default: **N**)
5. Calls the AI API
6. Writes `REPORT.md` to the current directory

```
About to send to generativelanguage.googleapis.com:
  Session log: 142 lines
  Git diff:    87 lines
  Total:       ~4.1 KB
  Redactions:  5 (across both log write and prompt assembly passes)
Proceed? [y/N]
```

### `devtrace setup`

Interactive wizard to configure provider, API key, and model. Writes `~/.devtrace/config.toml`.

### `devtrace version`

Print the installed version.

---

## Security

- **No `--api-key` flag** — prevents shell history and `ps` exposure
- **`0600` permissions** on `session.log`, `recorder.pid`, and `config.toml`
- **Redaction pipeline** runs twice: at write-time and before the API call
- **Outbound confirmation** required before any data leaves your machine

### Redaction patterns

| Pattern | Matches |
|---|---|
| `sk-[A-Za-z0-9]{20,}` | OpenAI API keys |
| `ghp_[A-Za-z0-9]{36}` | GitHub PATs |
| `AKIA[0-9A-Z]{16}` | AWS access key IDs |
| `Bearer \S+` | Authorization headers |
| `(password\|secret\|token\|api_key)=\S+` | Generic key=value secrets |

Redacted values are replaced with `[REDACTED:<pattern-name>]` and reported to stderr only.

---

## Runtime files

```
~/.devtrace/
  session.log     — append-only output log (0600)
  recorder.pid    — recorder PID, managed by devtrace stop (0600)
  config.toml     — API key and settings (0600)
```

The session log is append-only across multiple `devtrace start` runs. Delete it to start fresh:

```bash
rm ~/.devtrace/session.log
```

---

## Development

```bash
# Build
go build -o devtrace ./cmd/devtrace

# Run all tests
go test ./...

# Run a single package
go test ./internal/recorder/...

# Run a single test
go test -run TestRedactOpenAIKey ./internal/recorder/...

# Lint (requires golangci-lint)
golangci-lint run ./...

# Build with version tag
go build -ldflags "-X main.version=v0.1.0" -o devtrace ./cmd/devtrace
```

---

## Report format

`REPORT.md` is structured with four sections:

- **Summary** — what was worked on
- **Changes** — meaningful code changes
- **Test Results** — build and test output
- **Issues** — errors or problems encountered

The model filters out noise (repetitive errors, trivial commands like `ls`, `cd`, `pwd`) and focuses on meaningful work.
