package skillrouter

import (
	"fmt"
	"os"
	"strings"
)

type App struct {
	store  ConfigStore
	skills SkillService
}

func NewApp(store ConfigStore, skills SkillService) *App {
	return &App{store: store, skills: skills}
}

func Run() int {
	app := NewApp(FileConfigStore{}, LocalSkillService{})
	if err := app.run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func (a *App) run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cfgPath, err := a.store.Path()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := a.store.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Presets) == 0 {
		fmt.Println("No presets configured.")
		if err := a.skills.AddPresetInteractive(&cfg); err != nil {
			return fmt.Errorf("failed to add first preset: %w", err)
		}
		if err := a.store.Save(cfgPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if err := withRawTerminal(func() error {
		a.runCommandLoop(&cfg, cfgPath, cwd)
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *App) runCommandLoop(cfg *Config, cfgPath, sandboxDir string) {
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
			deleted := DeletePresetByName(cfg, name)
			if !deleted {
				printResult(fmt.Sprintf("Preset not found: %s", name))
				continue
			}
			if err := a.store.Save(cfgPath, *cfg); err != nil {
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
			if err := a.skills.SetSkillFlow(*cfg, sandboxDir); err != nil {
				printResult(fmt.Sprintf("Error: %v", err))
			}
		case "/addpreset":
			_, _ = stty("sane")
			if err := a.skills.AddPresetInteractive(cfg); err != nil {
				printResult(fmt.Sprintf("Error: %v", err))
				_, _ = stty("raw", "-echo", "opost", "onlcr")
				continue
			}
			if err := a.store.Save(cfgPath, *cfg); err != nil {
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
