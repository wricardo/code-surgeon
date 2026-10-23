package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

//go:embed skill/SKILL.md
var skillMD []byte

const skillName = "code-surgeon"

// installSkill writes the embedded skill to dir/SKILL.md, creating dir if needed, and
// reports the file path and whether it already existed (an update replaces its content).
func installSkill(dir string) (path string, existed bool, err error) {
	path = filepath.Join(dir, "SKILL.md")
	if _, statErr := os.Stat(path); statErr == nil {
		existed = true
	} else if !os.IsNotExist(statErr) {
		return path, false, fmt.Errorf("could not check existing skill file: %w", statErr)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return path, existed, fmt.Errorf("could not create skill directory: %w", err)
	}
	if err := os.WriteFile(path, skillMD, 0o644); err != nil {
		return path, existed, fmt.Errorf("could not write skill file: %w", err)
	}
	return path, existed, nil
}

// installSkillCommand returns `install-skill`, which installs or updates the code-surgeon
// skill for Claude Code at ~/.claude/skills/code-surgeon/.
func installSkillCommand() *cli.Command {
	return &cli.Command{
		Name:  "install-skill",
		Usage: "install or update the code-surgeon skill for Claude Code at ~/.claude/skills/code-surgeon/",
		Description: "Installs the embedded skill so that Claude knows how to use code-surgeon " +
			"(parse, list-tests and the graph commands). If the skill is already present it is " +
			"overwritten with the version embedded in this binary.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "dir", Usage: "install into this directory instead of ~/.claude/skills/" + skillName},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("could not determine home directory: %w", err)
				}
				dir = filepath.Join(home, ".claude", "skills", skillName)
			}
			path, existed, err := installSkill(dir)
			if err != nil {
				return err
			}
			if existed {
				fmt.Println("skill updated at", path)
			} else {
				fmt.Println("skill installed at", path)
			}
			return nil
		},
	}
}
