# aide

A terminal-first AI coding assistant with **Plan** and **Build** agent modes.

## Features

- **Plan/Build Agent Modes** — Switch between read-only Plan mode (analyze, research, plan) and full-capability Build mode (edit, write, execute). Cycle with `shift+tab` or via the command palette.
- **Three TUI Layouts** — The interface adapts to your terminal size: a full sidebar layout for wide terminals, a compact header layout for medium widths, and a minimal topbar layout for narrow/compact views.
- **Thinking Display Toggle** — Show or hide the model's thinking/reasoning output. Toggle from the command palette. Persisted across restarts.
- **Traveling Slash Animation** — The "Thinking" spinner uses a smooth traveling `///` pattern instead of scrambled random characters.
- **Multi-Model** — Choose from a wide range of LLMs or add your own via OpenAI- or Anthropic-compatible APIs.
- **Session-Based** — Maintain multiple work sessions and contexts per project.
- **LSP-Enhanced** — Uses LSPs for additional context and code intelligence.
- **Extensible** — Add capabilities via MCP servers (`stdio`, `http`, `sse`).
- **Skills** — Supports the Agent Skills open standard for reusable skill packages.
- **Dark Theme** — Built on the Charmtone Pantera dark palette with orange accents, green accent borders, and a clean `>_ AIDE` logo.

## Installation

```bash
go install github.com/liamb/opencode/aide@latest
```

Or build from source:

```bash
git clone https://github.com/liamb/opencode/aide
cd aide
go build .
```

## Quick Start

```bash
# Set your API key
export ANTHROPIC_API_KEY=sk-ant-...

# Launch aide
./aide
```

## Agent Modes

aide ships with two agent modes accessible via `shift+tab` or the command palette:

| Mode | Tools | Use Case |
|------|-------|----------|
| **Build** | All tools (edit, write, bash, view, grep, etc.) | Implementing changes, running commands |
| **Plan** | Read-only (view, grep, glob, ls, fetch, search) | Architecture analysis, research, planning |

## TUI Layouts

aide automatically adapts its layout to your terminal width:

- **Layout 1 (wide)** — Full sidebar with session info, model/provider details, file list, LSP/MCP status, and skills.
- **Layout 2 (medium)** — Compact header with gradient logo, working directory, context percentage, and keystroke hints.
- **Layout 3 (narrow)** — Minimal topbar showing `>_ AIDE`, working directory, context percentage, and keystroke hints, with a green accent border.

## Configuration

Configuration can be placed in `aide.json` or `.aide.json` in your project root, or globally in `$HOME/.config/aide/aide.json`.

```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    }
  },
  "options": {
    "tui": {
      "show_thinking": true
    },
    "agent_mode": "build"
  }
}
```

## Keybindings

| Key | Action |
|-----|--------|
| `shift+tab` | Cycle agent mode (Plan/Build) |
| `ctrl+p` | Open command palette |
| `ctrl+l` | Switch model |
| `ctrl+s` | Switch session |
| `ctrl+n` | New session |
| `ctrl+g` | Help |

## Acknowledgments

Built on the [Charm](https://charm.land) ecosystem — Bubble Tea, Lip Gloss, Glamour, Ultraviolet, and more.
