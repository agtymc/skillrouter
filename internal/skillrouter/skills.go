package skillrouter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SkillService interface {
	AddPresetInteractive(cfg *Config) error
	SetSkillFlow(cfg Config, sandboxDir string) error
}

type LocalSkillService struct{}

func (LocalSkillService) AddPresetInteractive(cfg *Config) error {
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
	if err := EnsureDir(absPath); err != nil {
		return err
	}

	AddOrUpdatePreset(cfg, name, absPath)
	return nil
}

func (LocalSkillService) SetSkillFlow(cfg Config, sandboxDir string) error {
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
