# SkillRouter (v0.4.1b)

SkillRouter is a small command-line tool that makes it faster to access and reuse skill files.
It lets you register directories with skills as presets, browse them in an interactive menu, and copy the selected skill file into the current working directory.

## Why SkillRouter

When you work with many skill collections, finding and copying the right file becomes repetitive.
SkillRouter reduces this to a short flow:

1. Choose a preset (a named directory with skills).
2. Choose a skill from that preset.
3. SkillRouter copies the selected `.md` file into the directory where you launched the program.

This keeps your workflow focused and avoids manual navigation and copy/paste.

## Core Concept

- `Preset`: A user-defined name + path to a directory that contains skills.
- `Skill`: A markdown file (`.md`) discovered inside preset subdirectories.
- `Sandbox`: The current directory where SkillRouter was started. Selected skills are copied here.

## How It Works

At startup, SkillRouter:

1. Detects your current working directory (copy destination).
2. Loads config from `~/.skillrouter/config.json`.
3. If no presets exist, asks you to create one.
4. Starts a command-only prompt.

Main flow:

1. Run `/set`.
2. Select a preset in the interactive menu.
3. Select a skill in the interactive menu.
4. SkillRouter copies the chosen file to your current directory.
5. You get a clear result message in the console.

## Commands

- `/set` - open interactive preset/skill selection and copy selected skill
- `/addpreset` - add or update a preset (name + path)
- `/delete` - delete preset by name, usage: /delete <preset-name>
- `/help` - show command help
- `/exit` - exit the program

Notes:

- Interactive navigation (`Up/Down`, `Enter`, `Esc`) is enabled only inside menus.
- Command history is available per session only (not persisted in config).

## Skill Discovery Rules

SkillRouter scans each preset directory and:

- Treats each immediate subdirectory as a skill group.
- Collects markdown files (`.md`) inside those subdirectories.
- If a group has one markdown file, menu label is the directory name.
- If a group has multiple markdown files, labels are shown as `group/file.md`.

## Installation and Build

Requirements:

- Go 1.21+ (or compatible modern Go version)

Build:

```bash
mkdir -p target
go build -o target/skillrouter .
```

Run:

```bash
go run .
```

or run the built binary:

```bash
./target/skillrouter
```

## Configuration

Config file location:

```text
~/.skillrouter/config.json
```

Example:

```json
{
  "presets": [
    {
      "name": "agent-skills",
      "path": "/home/user/skills/agent-skills"
    }
  ]
}
```

## Typical Example

1. Start SkillRouter in your project directory.
2. Run `/set`.
3. Choose preset `agent-skills`.
4. Choose skill `frontend-ui-engineering`.
5. The selected `.md` file is copied into your current project directory.

## Project Status

Current app version in CLI MOTD: `v0.4.1b`.
