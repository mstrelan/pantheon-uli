# pantheon-uli

A terminal-based interactive UI (TUI) for generating [Drupal one-time login](https://www.drush.org/latest/commands/user_login/) (ULI) links for sites hosted on [Pantheon](https://pantheon.io).

## Features

- Filterable, keyboard-navigable list of all your Pantheon sites
- Choose environment: `dev`, `test`, or `live`
- Automatically resolves vanity/custom domains for live environments
- Injects HTTP Basic Auth credentials into the URL when a site lock is active
- Opens the URL in your browser and prints it to the terminal
- Caches the site list locally for fast startup (30-day TTL, background refresh)

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
| `Enter` | Generate the ULI link |
| `Ctrl+R` | Refresh site cache in the background |
| `Esc` / `Ctrl+C` | Quit |

## Caching

The site list is cached to `pantheon-uli/sites.txt` inside the OS user cache directory (`~/.cache` on Linux, `~/Library/Caches` on macOS, `%LOCALAPPDATA%` on Windows) with a 30-day TTL. On startup, a stale cache triggers a background refresh so you can browse immediately. If no cache exists, a blocking refresh runs first.
