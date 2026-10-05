// Package abcdhome is the one place in abcd's Go code that spells the name of
// the user-level home: the folder abcd keeps under the person's home directory
// for its machine-scoped records (the trusted-roots and rules.json
// declarations, the path-entry record, the credential and routing files, the
// transcript, worktree, lab and sources stores, the run logs).
//
// Every other package reaches that folder through Rel, Path or Display, and
// TestOnlyTheHomeResolverNamesTheHome holds them to it, so the folder's name is
// one constant (spc-2610031309233367, step 2), ~/.abcd.noindex since the
// rename (step 3). The same package decides the stop: while the folder's old
// name, ~/.abcd, stands, Check reports it and every entry point writes nothing
// and names the one rename command. The repository tier, a project's own
// `.abcd/` beside its sources, is a different folder that keeps the old name;
// it is not spelled here and nothing here reaches it.
//
// The package is a leaf that imports only the standard library, so
// internal/core, internal/surface and cmd can all import it without an edge
// back. It changes only abcd's own folder: the computer's search settings are
// never named in code (adr-2610030720195401), which
// TestNoCodeNamesTheSearchSettings holds.
package abcdhome

import (
	"os"
	"path/filepath"
	"strings"
)

// name is the home folder abcd keeps under the person's home directory. The
// ".noindex" suffix is what the macOS indexer honours at scan time, for the
// folder and everything beneath it, so creating a worktree, a run log or a
// transcript here sets off no indexing burst (itd-2610030720038073). abcd
// keeps its folders out of indexing only by naming its own folder so, and
// never by touching the computer's search settings (adr-2610030720195401).
const name = ".abcd.noindex"

// oldName is the folder the home was called before. Check reads it, with
// Lstat, and nothing else in abcd does: no reader falls back to it, so there
// is no window in which both names are read (decision 6 of the intent).
const oldName = ".abcd"

// RenameCommand is the one command the stop names: the person's own act,
// which abcd never performs, and the one command the shell guard admits while
// the old folder stands.
const RenameCommand = "mv ~/" + oldName + " ~/" + name

// RepairCommand reconnects the worktrees the rename moved. git records a
// worktree's location in absolute form, so after the rename every worktree in
// the store is listed by its repository as prunable until `git worktree
// repair` runs in it (iss-2610040147016103); the stop prints this command for
// the person to run after the rename, and abcd writes nothing outside its home.
//
// It finds each worktree by its .git file at any depth (iss-2610050728100598):
// a worktree sits at worktrees/<root-sha>/<name>, one named after a slashed
// branch a level or more deeper, and one left from before the root-sha key
// directly under worktrees/. -prune stops the walk at every .git it meets, a
// file or a folder, so the walk never enters a worktree's files, never runs
// git on a nested repository inside one, and never enters a clone someone
// placed in the store; the repair then runs only where .git is a file, a
// linked worktree, so a clone's submodules keep their relative links. It is
// one find with no shell variable and no quote, so it runs as printed from
// sh, bash and zsh, and a path holding a space reaches git as one argument. The "repair: gitdir incorrect" line git prints
// for each worktree is the link it fixed, and the stop lines say so.
const RepairCommand = `find ~/` + name + `/worktrees -type d -exec test -e {}/.git \; -prune -exec test -f {}/.git \; -exec git -C {} worktree repair \;`

// repairFixedNote is what the stop lines add after RepairCommand: git words
// each link it fixes as "repair: gitdir incorrect", which reads like a failure.
const repairFixedNote = " (each `repair: gitdir incorrect` line it prints is a link it fixed)"

// WorktreeRepairCommand is the repair for the one worktree at rel, a slash
// path relative to the person's home directory, in the form a person pastes:
// the tilde outside the quotes, so the shell expands it, and the rest
// single-quoted, so a space or a quote in a worktree's name reaches git as one
// argument. abcd ahoy names it for a worktree the rename left unlinked.
func WorktreeRepairCommand(rel string) string {
	return "git -C ~/'" + strings.ReplaceAll(rel, "'", `'\''`) + "' worktree repair"
}

// The two stop lines and their status-line short forms, written once here
// (spc-2610031309233367, "The stop", amended by its open question 6). The
// line for both folders never names RenameCommand: with ~/.abcd.noindex
// standing, that command would move the old folder into the new one.
const (
	oldStandsLine = "abcd's folder is now ~/" + name + ", a name the Mac's search indexer passes over, and ~/" + oldName +
		" still stands, so abcd has written nothing. Rename it with `" + RenameCommand +
		"`, reconnect the working copies kept there with `" + RepairCommand + "`" + repairFixedNote + ", then run abcd again."
	bothStandLine = "Both ~/" + oldName + " and ~/" + name + " exist, so abcd has written nothing and moves neither." +
		" Keep the one you want, named ~/" + name + ", and take the other out of your home folder, reconnect the" +
		" working copies kept there with `" + RepairCommand + "`" + repairFixedNote + ", then run abcd again."
	oldStandsShort = "abcd stopped: rename ~/" + oldName + " to ~/" + name
	bothStandShort = "abcd stopped: both ~/" + oldName + " and ~/" + name + " exist"
)

// Stop is the state that stops abcd before it writes anything: an old
// ~/.abcd standing, alone or beside ~/.abcd.noindex.
type Stop struct {
	// Both is set when ~/.abcd.noindex stands beside the old folder.
	Both bool
	// Line is the one line every entry point says, in its own form.
	Line string
	// Short is the status line's form of it.
	Short string
}

// Check reports the stop for home, the person's home directory, or nil. It
// reads the two names with os.Lstat and nothing else: an entry of any kind at
// the old name stops, a folder, a file or a link alike, and a link is never
// followed, because a link at the old path is still the old home
// (spc-2610031309233367, open question 2). An entry Lstat cannot see reads as
// absent, as a home this account cannot search reads as absent everywhere. An
// empty or relative home is no stop: the refusals for that shape stand where
// they are, and a relative name would be judged against whatever directory
// the caller happens to run in.
func Check(home string) *Stop {
	if home == "" || !filepath.IsAbs(home) {
		return nil
	}
	if _, err := os.Lstat(filepath.Join(home, oldName)); err != nil {
		return nil
	}
	if _, err := os.Lstat(filepath.Join(home, name)); err == nil {
		return &Stop{Both: true, Line: bothStandLine, Short: bothStandShort}
	}
	return &Stop{Line: oldStandsLine, Short: oldStandsShort}
}

// DirMode is the mode abcd's home and every folder abcd makes in it are
// created at: the account's alone. Every writer hands it to
// fsutil.EnsureHomeScope, so the home is private whichever command creates it
// first (iss-2610032205304585); TestEveryHomeWriterMakesTheHomePrivate holds
// them to it.
const DirMode os.FileMode = 0o700

// FileMode is the mode of a record abcd writes into its home: read and written
// by the account alone.
const FileMode os.FileMode = 0o600

// Rel is the slash path, relative to the person's home directory, of leaf
// below abcd's home: Rel("trusted-roots") is ".abcd.noindex/trusted-roots",
// and Rel() is the home folder itself. It is the form the home-scope
// primitives in internal/fsutil take as their rel, so every level of it, the
// home folder included, is judged and a symlinked level refused as before. The
// leaves are joined as written and never cleaned, so a leaf that is not clean
// ("../x", "") reaches fsutil.ValidRelPath unaltered and is refused there:
// every guard a reader applies judges exactly the bytes it is handed.
func Rel(leaf ...string) string {
	if len(leaf) == 0 {
		return name
	}
	return name + "/" + strings.Join(leaf, "/")
}

// Path is the filesystem path of leaf below abcd's home inside home, the
// person's home directory: filepath.Join(home, ".abcd.noindex", leaf...) in the
// platform's separators, cleaned as filepath.Join cleans.
func Path(home string, leaf ...string) string {
	return filepath.Join(home, filepath.FromSlash(Rel(leaf...)))
}

// Display is leaf below abcd's home in the tilde form a message names it by,
// "~/.abcd.noindex/trusted-roots", so a line a person pastes into a shell works and no
// message carries the caller's home path. Like Rel it joins the leaves as
// written, so a placeholder or a trailing slash a message shows survives:
// Display("worktrees/<root-sha>/<name>/") keeps its final slash, and Display()
// is the home folder itself.
func Display(leaf ...string) string {
	return "~/" + Rel(leaf...)
}
