# aide

A terminal-first AI coding assistant with **Plan** and **Build** agent modes.
Works with any LLM, integrates with your tools, and runs in every terminal.

## Features

- **Plan / Build Agent Modes** — Switch between read-only Plan mode (analyze, research, plan) and full-capability Build mode (edit, write, execute). Cycle with `shift+tab` or via the command palette.
- **Multi-Provider** — Use Anthropic, OpenAI, Gemini, Bedrock, Copilot, Azure, OpenRouter, Vercel, Ollama, LM Studio, llama.cpp, or any OpenAI-/Anthropic-compatible API. Switch models mid-session without losing context.
- **Session-Based** — Maintain multiple work sessions per project. Each session preserves its full conversation history and tool context.
- **LSP-Enhanced** — Connects to your project's language servers for code intelligence, diagnostics, and symbol references.
- **Extensible via MCP** — Add tools, resources, and prompts through MCP servers (`stdio`, `http`, `sse`).
- **Agent Skills** — Supports the [Agent Skills](https://agentskills.io) open standard. Drop a `SKILL.md` in your repo to give the agent reusable capabilities.
- **Project Context Files** — Automatically discovers `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.cursorrules`, `.github/copilot-instructions.md`, and their `.local` variants for project-specific instructions.
- **Hooks System** — Define shell commands in `aide.json` that fire before tool execution. Hooks can allow, deny, rewrite, or halt tool calls — giving you programmable safety guards and custom workflows.
- **Permission Controls** — Granular tool-level permissions with allow-lists, per-session grants, and a `--yolo` mode for fully automated operation.
- **Non-Interactive Mode** — `aide run "prompt"` runs a single prompt and exits. Pipe input from stdin, continue existing sessions, and redirect output to files. Works headlessly.
- **Client/Server Architecture** — Run `aide server` on a machine and connect from anywhere with `aide --host`. The server speaks HTTP/SSE over Unix sockets (or Windows named pipes).
- **Cross-Platform** — Works on macOS, Linux, Windows (PowerShell + WSL), Android, FreeBSD, OpenBSD, and NetBSD.

## Installation

```bash
go install github.com/liamb/aide@latest
```

Or build from source:

```bash
git clone https://github.com/liamb/aide
cd aide
go build .
```

## Quick Start

```bash
# Set your API key
export ANTHROPIC_API_KEY=sk-ant-...

# Launch interactive TUI
aide

# Or run a one-shot prompt non-interactively
aide run "Explain the architecture of this project"
```

## Agent Modes

| Mode | Tools | Use Case |
|------|-------|----------|
| **Build** | All tools (edit, write, bash, view, grep, glob, ls, fetch, search, diagnostics, references, and more) | Implementing changes, running commands, debugging |
| **Plan** | Read-only tools (view, grep, glob, ls, fetch, search, sourcegraph, diagnostics) | Architecture analysis, research, planning, code review |

## Configuration

Place `aide.json` or `.aide.json` in your project root, or globally at `~/.config/aide/aide.json`.

```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    }
  },
  "options": {
    "agent_mode": "build"
  }
}
```

### Common Workflows

```bash
# Run a prompt from stdin
curl https://example.com | aide run "Summarize this"

# Continue the most recent session
aide run --continue "Follow up on your last response"

# Redirect output to a file
aide run "Write a README for this project" > README.md

# Start a server for remote access
aide server

# Connect to a remote server
aide --host /path/to/socket
```

## Safety & Control

- **Hooks**: Define PreToolUse shell commands that can inspect, allow, deny, or rewrite every tool call before it executes. Decisions aggregate across multiple hooks.
- **Permissions**: Per-tool allow-lists, persistent grants per session, and a permission prompt flow.
- **Plan Mode**: Read-only by design — the agent cannot edit files or execute commands.

## Keybindings

| Key | Action |
|-----|--------|
| `shift+tab` | Cycle agent mode (Plan/Build) |
| `ctrl+p` | Command palette |
| `ctrl+l` | Switch model |
| `ctrl+s` | Switch session |
| `ctrl+n` | New session |
| `ctrl+g` | Help |

## Acknowledgments

Built on the [Charm](https://charm.land) ecosystem — Bubble Tea, Lip Gloss, Glamour, Ultraviolet, and more.
