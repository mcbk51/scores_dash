# Scores Dashboard

A terminal-based live sports dashboard for the four major North American pro leagues: NFL, NBA, NHL, and MLB.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)

## Features

- **Live game tracking** - Real-time scores with clock, period/quarter/inning display
- **Upcoming games** - Shows the next scheduled game for each league when no live games are available
- **Betting odds** - Spread and over/under lines via ESPN's odds API
- **Auto-refresh** - Updates every 30 seconds
- **Color-coded leagues** - NFL (red), NBA (blue), NHL (orange), MLB (green)

## Installation

```bash
go install github.com/mcbk51/scores_dash@latest
```

Or clone and build:

```bash
git clone https://github.com/mcbk51/scores_dash.git
cd scores_dash
go build -o scores_dash .
```

## Usage

```bash
./scores_dash
```

Press `q` or `Esc` to quit.

## Dependencies

- [tview](https://github.com/rivo/tview) - Terminal UI framework
- [tcell](https://github.com/gdamore/tcell) - Terminal cell library

## Data Source

All game data and betting odds are fetched from ESPN's public API.

## License

MIT
