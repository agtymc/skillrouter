package skillrouter

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

func printMOTD(dir string) {
	title := fmt.Sprintf("SkillRouter (v%s)", AppVersion)
	mode := "mode:      command-only"
	directory := "directory: " + dir
	lines := []string{title, "", mode, directory}
	innerWidth := 0
	for _, line := range lines {
		if w := utf8.RuneCountInString(line); w > innerWidth {
			innerWidth = w
		}
	}

	fmt.Printf("╔%s╗\n", strings.Repeat("═", innerWidth+2))
	fmt.Printf("║ %s ║\n", padRight(title, innerWidth))
	fmt.Printf("║ %s ║\n", padRight("", innerWidth))
	fmt.Printf("║ %s ║\n", padRight(mode, innerWidth))
	fmt.Printf("║ %s ║\n", padRight(directory, innerWidth))
	fmt.Printf("╚%s╝\n\n", strings.Repeat("═", innerWidth+2))

	fmt.Println("  Tip: Use /set to select preset, /help for help, /exit to quit.")
	fmt.Println()
}

func padRight(s string, width int) string {
	diff := width - utf8.RuneCountInString(s)
	if diff <= 0 {
		return s
	}
	return s + strings.Repeat(" ", diff)
}

func printHelp() {
	printResult(
		"Available commands:",
		"  /set       - choose preset and skill, then copy selected .md to current directory",
		"  /addpreset - add a new preset (name + path)",
		"  /delete    - delete preset by name, usage: /delete <preset-name>",
		"  /help      - show this help",
		"  /exit      - exit program",
	)
}

func printResult(lines ...string) {
	fmt.Print("\r\n")
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Print("\r\n")
}

func readCommand(prompt string, history []string) (string, error) {
	fmt.Print(prompt)
	var line []byte
	escSeen := false
	csiSeen := false
	historyActive := false
	historyPrefix := ""
	historyMatches := make([]string, 0, len(history))
	historyIndex := -1
	buf := make([]byte, 1)

	for {
		n, err := readByte(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", io.EOF
			}
			return "", err
		}
		if n == 0 {
			continue
		}

		b := buf[0]
		if escSeen {
			escSeen = false
			if b == '[' || b == 'O' {
				csiSeen = true
			}
			continue
		}
		if csiSeen {
			if b == 'A' || b == 'B' {
				if !historyActive {
					historyPrefix = string(line)
					historyMatches = filterHistory(history, historyPrefix)
					historyActive = true
					historyIndex = -1
				}
				if len(historyMatches) > 0 {
					if b == 'A' {
						if historyIndex < len(historyMatches)-1 {
							historyIndex++
						}
					} else if historyIndex >= 0 {
						historyIndex--
					}
					if historyIndex >= 0 {
						line = []byte(historyMatches[historyIndex])
					} else {
						line = []byte(historyPrefix)
					}
					redrawLine(prompt, line)
				}
			}
			if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
				csiSeen = false
			}
			continue
		}

		switch b {
		case 3, 4:
			fmt.Print("\r\n")
			return "", io.EOF
		case 13, 10:
			fmt.Print("\r\n")
			return strings.TrimSpace(string(line)), nil
		case 8, 127:
			if len(line) > 0 {
				line = line[:len(line)-1]
				fmt.Print("\b \b")
			}
			historyActive = false
			historyMatches = historyMatches[:0]
			historyIndex = -1
		case 27:
			escSeen = true
		default:
			if b < 32 {
				continue
			}
			line = append(line, b)
			fmt.Printf("%c", b)
			historyActive = false
			historyMatches = historyMatches[:0]
			historyIndex = -1
		}
	}
}

func redrawLine(prompt string, line []byte) {
	fmt.Print("\r\033[2K")
	fmt.Print(prompt)
	fmt.Print(string(line))
}

func filterHistory(history []string, prefix string) []string {
	matches := make([]string, 0, len(history))
	seen := make(map[string]struct{}, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		cmd := history[i]
		if prefix != "" && !strings.HasPrefix(cmd, prefix) {
			continue
		}
		if _, ok := seen[cmd]; ok {
			continue
		}
		seen[cmd] = struct{}{}
		matches = append(matches, cmd)
	}
	return matches
}
