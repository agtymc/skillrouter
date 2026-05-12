package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type Preset struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Config struct {
	Presets []Preset `json:"presets"`
}

type SkillItem struct {
	Label string
	Path  string
}

const AppVersion = "0.4.1b"

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	cfgPath, err := configPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve config path: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Presets) == 0 {
		fmt.Println("No presets configured.")
		if err := addPresetInteractive(&cfg); err != nil {
			fmt.Fprintf(os.Stderr, "failed to add first preset: %v\n", err)
			os.Exit(1)
		}
		if err := saveConfig(cfgPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
			os.Exit(1)
		}
	}

	runCommandLoop(&cfg, cfgPath, cwd)
}

func runCommandLoop(cfg *Config, cfgPath, sandboxDir string) {
	oldState, err := stty("-g")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read terminal state: %v\n", err)
		return
	}
	if _, err := stty("raw", "-echo", "opost", "onlcr"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to switch terminal to raw mode: %v\n", err)
		return
	}
	defer func() {
		_, _ = stty(oldState)
		fmt.Print("\r\033[0m")
	}()

	printMOTD(sandboxDir)
	history := make([]string, 0, 64)
	for {
		cmd, err := readCommand("skillrouter> ", history)
		if err != nil {
			fmt.Print("Bye.\r\n")
			return
		}
		if cmd != "" {
			history = append(history, cmd)
		}
		if strings.HasPrefix(cmd, "/delete") {
			name := strings.TrimSpace(strings.TrimPrefix(cmd, "/delete"))
			if name == "" {
				printResult("Usage: /delete <preset-name>")
				continue
			}
			deleted := deletePresetByName(cfg, name)
			if !deleted {
				printResult(fmt.Sprintf("Preset not found: %s", name))
				continue
			}
			if err := saveConfig(cfgPath, *cfg); err != nil {
				printResult(fmt.Sprintf("Error: failed to save config: %v", err))
				continue
			}
			printResult(fmt.Sprintf("Preset deleted: %s", name))
			continue
		}

		switch cmd {
		case "":
			printHelp()
		case "/set":
			if err := setSkillFlow(*cfg, sandboxDir); err != nil {
				printResult(fmt.Sprintf("Error: %v", err))
			}
		case "/addpreset":
			_, _ = stty(oldState)
			if err := addPresetInteractive(cfg); err != nil {
				printResult(fmt.Sprintf("Error: %v", err))
				_, _ = stty("raw", "-echo", "opost", "onlcr")
				continue
			}
			if err := saveConfig(cfgPath, *cfg); err != nil {
				printResult(fmt.Sprintf("Error: failed to save config: %v", err))
			}
			printResult("Preset added.")
			_, _ = stty("raw", "-echo", "opost", "onlcr")
		case "/help":
			printHelp()
		case "/exit":
			fmt.Print("Bye.\r\n")
			return
		default:
			printResult("Only commands are allowed. Use /help")
		}
	}
}

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
		n, err := os.Stdin.Read(buf)
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
					} else {
						if historyIndex >= 0 {
							historyIndex--
						}
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

func addPresetInteractive(cfg *Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter preset name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("preset name cannot be empty")
	}

	fmt.Print("Enter path to skills directory: ")
	path, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("preset path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := ensureDir(absPath); err != nil {
		return err
	}

	for i := range cfg.Presets {
		if strings.EqualFold(cfg.Presets[i].Name, name) {
			cfg.Presets[i].Path = absPath
			return nil
		}
	}

	cfg.Presets = append(cfg.Presets, Preset{Name: name, Path: absPath})
	sort.Slice(cfg.Presets, func(i, j int) bool {
		return strings.ToLower(cfg.Presets[i].Name) < strings.ToLower(cfg.Presets[j].Name)
	})
	return nil
}

func setSkillFlow(cfg Config, sandboxDir string) error {
	if len(cfg.Presets) == 0 {
		return errors.New("no presets configured, use /addpreset")
	}

	presetNames := make([]string, 0, len(cfg.Presets))
	for _, p := range cfg.Presets {
		presetNames = append(presetNames, p.Name)
	}

	presetIndex, action, err := selectFromMenu("Select preset", presetNames)
	if err != nil {
		return err
	}
	if action == "esc" {
		return nil
	}

	selectedPreset := cfg.Presets[presetIndex]
	skills, err := collectSkills(selectedPreset.Path)
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		return fmt.Errorf("no markdown skills found in preset %q", selectedPreset.Name)
	}

	skillLabels := make([]string, 0, len(skills))
	for _, s := range skills {
		skillLabels = append(skillLabels, s.Label)
	}

	for {
		title := fmt.Sprintf("%s: select skill", selectedPreset.Name)
		skillIndex, skillAction, err := selectFromMenu(title, skillLabels)
		if err != nil {
			return err
		}
		if skillAction == "esc" {
			presetIndex, action, err = selectFromMenu("Select preset", presetNames)
			if err != nil {
				return err
			}
			if action == "esc" {
				return nil
			}
			selectedPreset = cfg.Presets[presetIndex]
			skills, err = collectSkills(selectedPreset.Path)
			if err != nil {
				return err
			}
			if len(skills) == 0 {
				return fmt.Errorf("no markdown skills found in preset %q", selectedPreset.Name)
			}
			skillLabels = skillLabels[:0]
			for _, s := range skills {
				skillLabels = append(skillLabels, s.Label)
			}
			continue
		}

		src := skills[skillIndex].Path
		dst := filepath.Join(sandboxDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return err
		}
		printResult(
			"Copied",
			fmt.Sprintf("   From: %s", src),
			fmt.Sprintf("   To: %s", dst),
		)
		return nil
	}
}

func selectFromMenu(title string, items []string) (int, string, error) {
	if len(items) == 0 {
		return -1, "", errors.New("empty menu")
	}

	oldState, err := stty("-g")
	if err != nil {
		return -1, "", fmt.Errorf("failed to read terminal state: %w", err)
	}
	if _, err := stty("raw", "-echo"); err != nil {
		return -1, "", fmt.Errorf("failed to switch terminal to raw mode: %w", err)
	}
	defer func() {
		_, _ = stty(oldState)
		fmt.Print("\r\033[0m")
	}()

	index := 0
	renderMenu(title, items, index)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return -1, "esc", nil
			}
			return -1, "", err
		}
		if n == 0 {
			continue
		}

		b := buf[:n]
		switch {
		case len(b) == 1 && b[0] == 3:
			return -1, "esc", nil
		case len(b) == 1 && b[0] == 4:
			return -1, "esc", nil
		case len(b) == 1 && b[0] == 13:
			return index, "enter", nil
		case len(b) == 1 && b[0] == 27:
			return -1, "esc", nil
		case len(b) >= 3 && b[0] == 27 && b[1] == 91 && b[2] == 65:
			if index > 0 {
				index--
			}
			renderMenu(title, items, index)
		case len(b) >= 3 && b[0] == 27 && b[1] == 91 && b[2] == 66:
			if index < len(items)-1 {
				index++
			}
			renderMenu(title, items, index)
		}
	}
}

func renderMenu(title string, items []string, index int) {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\r\n\r\n", title)
	for i, item := range items {
		if i == index {
			fmt.Printf("\033[1;36m> %s\033[0m\r\n", item)
		} else {
			fmt.Printf("  %s\r\n", item)
		}
	}
	fmt.Print("\r\nUse Up/Down, Enter to select, Esc to go back\r\n")
}

func collectSkills(root string) ([]SkillItem, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read preset path %q: %w", root, err)
	}

	var out []SkillItem
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDir := filepath.Join(root, e.Name())
		files, err := os.ReadDir(skillDir)
		if err != nil {
			continue
		}

		var mdFiles []string
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if strings.EqualFold(filepath.Ext(f.Name()), ".md") {
				mdFiles = append(mdFiles, f.Name())
			}
		}
		if len(mdFiles) == 0 {
			continue
		}

		sort.Strings(mdFiles)
		if len(mdFiles) == 1 {
			out = append(out, SkillItem{
				Label: e.Name(),
				Path:  filepath.Join(skillDir, mdFiles[0]),
			})
			continue
		}

		for _, md := range mdFiles {
			out = append(out, SkillItem{
				Label: filepath.ToSlash(filepath.Join(e.Name(), md)),
				Path:  filepath.Join(skillDir, md),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label)
	})
	return out, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("failed to flush target file: %w", err)
	}
	return nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".skillrouter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func saveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ensureDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}

func deletePresetByName(cfg *Config, name string) bool {
	for i := range cfg.Presets {
		if strings.EqualFold(cfg.Presets[i].Name, name) {
			cfg.Presets = append(cfg.Presets[:i], cfg.Presets[i+1:]...)
			return true
		}
	}
	return false
}

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
