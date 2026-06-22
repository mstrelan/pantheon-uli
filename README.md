# pantheon-uli

A terminal-based interactive UI (TUI) for generating [Drupal one-time login](https://www.drush.org/latest/commands/user_login/) (ULI) links for sites hosted on [Pantheon](https://pantheon.io).

## Features

- Filterable, keyboard-navigable list of all your Pantheon sites
- Choose environment: `dev`, `test`, or `live`
- Automatically resolves vanity/custom domains for live environments
- Injects HTTP Basic Auth credentials into the URL when a site lock is active
- Opens the URL in your browser and prints it to the terminal
- Caches the site list locally for fast startup (30-day TTL, background refresh)
- Open a site without logging in using `Alt+Enter` (uses cached vanity domain and lock credentials)

## Prerequisites

- Go 1.25+
- [`terminus`](https://github.com/pantheon-systems/terminus) installed and authenticated (`terminus auth:login`)
## Install

```bash
go install github.com/mstrelan/pantheon-uli@latest
```

Or build from source:

```bash
git clone https://github.com/mstrelan/pantheon-uli.git
cd pantheon-uli
go build -o pantheon-uli .
```

## Usage

```bash
pantheon-uli
```

## Keyboard Controls

| Key | Action |
|---|---|
| Type characters | Filter the site list |
| `Backspace` | Delete last filter character |
| `Up` / `Down` | Move cursor |
| `Tab` / `Shift+Tab` | Cycle environment: `dev` → `test` → `live` |
| `Enter` | Generate a one-time login link and open it in the browser |
| `Alt+Enter` | Open the site in the browser without logging in |
| `Ctrl+R` | Refresh site cache in the background |
| `Esc` / `Ctrl+C` | Quit |

## Caching

The site list is cached to `pantheon-uli/sites.json` inside the OS user cache directory (`~/.cache` on Linux, `~/Library/Caches` on macOS, `%LOCALAPPDATA%` on Windows) with a 30-day TTL. On startup, a stale cache triggers a background refresh so you can browse immediately. If no cache exists, a blocking refresh runs first.

Per-environment metadata (vanity domain and HTTP Basic Auth credentials) is cached separately with a 30-day TTL. It is populated lazily the first time you generate a ULI for a given site and environment. Subsequent `Enter` presses reuse the cached values without calling terminus, and `Alt+Enter` uses them to construct the base URL without generating a login link at all.
