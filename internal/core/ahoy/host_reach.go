package ahoy

// The host-reach warnings (itd-2610030814013772, spc-2610031156364295 A5):
// what keeps AGENTS.md from Claude Code that is not one of the project's own
// conventions files. Claude Code reads AGENTS.md on its own only when no
// CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md sits in the folder it runs in
// or any folder above it, and only from a release on. Each is a warning, never
// a refusal (decision 3), and each is found by presence alone:
//
//   - CLAUDE.local.md at the project root is the person's own, usually kept out
//     of git, so it is named and never read, classified or offered for removal;
//   - from the root's parent up to the file-system root, each folder is asked,
//     with lstat alone, whether a file of those names exists. In the person's
//     home folder the user-level .claude/CLAUDE.md is skipped, since it does not
//     switch AGENTS.md off; a CLAUDE.md directly in the home folder is still
//     named. A folder that cannot be searched ends the walk quietly;
//   - the claude command on PATH older than the release that reads AGENTS.md
//     on its own (host_version.go) is named, with no version number.
//
// Install checks only: Detect, which the status board and the hooks call, and
// Managed, which the status line calls on every refresh, make neither.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// HostReachGapID is a file above the project, or a personal file at its root,
// that keeps Claude Code from reading AGENTS.md: a warning, never resolvable.
const HostReachGapID = "conventions.host_reach"

// HostVersionGapID is a Claude Code on PATH older than the release that reads
// AGENTS.md on its own: a warning, never resolvable.
const HostVersionGapID = "conventions.host_version"

// hostReachWhy is the one fixed sentence each presence warning carries: why a
// check above the project is not a read of settings from above it. abcd's
// rule that no .abcd/ is read above the working tree bounds where abcd takes
// configuration that changes its own behaviour; a presence check takes
// nothing, and the most a hostile file above the tree can cause is a warning.
const hostReachWhy = "abcd reads no settings from folders above this project. " +
	"This check only asks whether a file of this name exists there; it reads nothing in it " +
	"and changes nothing abcd does, because the agent tool itself reads that folder."

// hostReachAbove is each file Claude Code reads, in a folder above the one it
// runs in, in place of AGENTS.md, in the order a folder is asked for them.
var hostReachAbove = []string{"CLAUDE.md", filepath.Join(".claude", "CLAUDE.md"), "CLAUDE.local.md"}

// userLevelInstructions is the file in the home folder that is Claude Code's
// own user-level instructions, which leave AGENTS.md read.
var userLevelInstructions = filepath.Join(".claude", "CLAUDE.md")

// hostVersionWarning is the warning for a Claude Code older than the floor. It
// names no version, the host's or the floor's.
const hostVersionWarning = "The Claude Code on this computer is older than the release that reads AGENTS.md on its own, " +
	"so it loads none of abcd's rules in this project until it is updated."

// detectHostReach raises the host-reach warnings for the project at root, in
// order: the personal file at the root, the files above it from the nearest
// folder up, then the host's version.
func detectHostReach(root string) []Gap {
	start := root
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		start = resolved
	}
	var gaps []Gap
	personal := filepath.Join(start, "CLAUDE.local.md")
	if _, err := os.Lstat(personal); err == nil {
		gaps = append(gaps, hostReachGap(personal,
			"your personal Claude Code file in this project, which Claude Code reads and then not AGENTS.md, "+
				"so abcd's rules stay hidden from it here while the file is there; abcd never reads, edits or removes it."))
	}
	home := homeFolder()
	for dir := filepath.Dir(start); ; {
		ended := false
		for _, name := range hostReachAbove {
			if name == userLevelInstructions && home != "" && dir == home {
				continue
			}
			p := filepath.Join(dir, name)
			_, err := os.Lstat(p)
			switch {
			case err == nil:
				gaps = append(gaps, hostReachGap(p,
					"Claude Code reads this file, in a folder above this project, and then not AGENTS.md, "+
						"so abcd's rules stay hidden from it here while the file is there."))
			case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			default:
				ended = true // a folder that cannot be searched ends the walk
			}
			if ended {
				break
			}
		}
		parent := filepath.Dir(dir)
		if ended || parent == dir {
			break
		}
		dir = parent
	}
	if v, err := readClaudeVersion(root); err == nil && v.less(claudeCodeAgentsFloor) {
		gaps = append(gaps, Gap{
			ID: HostVersionGapID, Category: ConventionsFile, Scope: "machine",
			Title: "Claude Code is older than the release that reads AGENTS.md", Detail: hostVersionWarning,
			FixHint: "Update Claude Code; abcd changes nothing of it.", Required: false, Resolvable: false,
		})
	}
	return gaps
}

// hostReachGap is the warning naming the file at path, shown with the home
// folder as ~, then what it does, then the fixed sentence.
func hostReachGap(path, what string) Gap {
	shown := fsutil.RedactHome(path)
	return Gap{
		ID: HostReachGapID, Category: ConventionsFile, Scope: "machine",
		Title:    shown + " hides AGENTS.md from Claude Code",
		Detail:   shown + ": " + what + " " + hostReachWhy,
		FixHint:  "Move what " + shown + " says into AGENTS.md, or remove it, if you want Claude Code to read AGENTS.md here.",
		Required: false, Resolvable: false,
	}
}

// homeFolder is the person's home folder with its links resolved, or "".
func homeFolder() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(home); err == nil {
		return resolved
	}
	return filepath.Clean(home)
}
