# SkillRouter Agent Notes

## Project
- Name: `SkillRouter`
- Version: `0.3.2b`
- Language: `Go`
- Entry point: `main.go`

## Local Run
- Run: `go run .`
- Build output directory: `target/`
- Build command: `go build -o target/skillrouter .`

## Behavior Highlights
- Command-only console mode with in-session command history.
- Interactive menu is enabled only for `/set`.
- Results are printed as separated blocks (blank line before and after).

## Repository Hygiene
- Ignored directories: `!files/`, `out/`, `target/`
- Local IDE data is ignored via `.idea/`

## Notes
- Command history is session-only and is not persisted to config.
