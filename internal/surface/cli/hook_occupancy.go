package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// noteOccupancy is the session-start hook's record of the checkout this
// session starts in (iss-2610240859590333), so a record-writing verb can tell
// a main working tree another live session also works in. It never fails the
// hook: a session it cannot record is one the count cannot see, which is said
// on stderr whatever the reason.
func noteOccupancy(cmd *cobra.Command, in hookInput, cwd string) {
	if in.SessionID == "" || in.AgentID != "" {
		return
	}
	top, err := gitutil.Toplevel(cwd)
	if err != nil {
		return // not a checkout: there is no working tree to share
	}
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return
	}
	unseen := func(why string) {
		diagnosticLine(cmd.ErrOrStderr(), "abcd: could not record which checkout this session works in (%s), so a record verb's count of sessions sharing it cannot see this one.",
			termsafe.Sanitize(fsutil.RedactHome(why)))
	}
	tab, err := m.ReadTable()
	if err != nil {
		unseen(err.Error())
		return
	}
	host, err := sessionname.FindHost(tab, m.From)
	if err != nil {
		unseen(err.Error())
		return
	}
	if err := occupancy.Note(m.Home, in.SessionID, top, host, tab); err != nil {
		unseen(err.Error())
	}
}

// forgetOccupancy is the session-end hook's removal of that record: one file
// remove, so it fits the host's short budget for the hook. A session that ends
// without it stops counting when its host process does.
func forgetOccupancy(in hookInput) {
	if in.SessionID == "" || in.AgentID != "" {
		return
	}
	if m, ok := sessionNameMachine(); ok && m.Home != "" {
		_ = occupancy.Forget(m.Home, in.SessionID)
	}
}

// refuseSharedMainCheckout refuses a record write into root when root is a
// main working tree that two or more live sessions started in
// (iss-2610240859590333): a record written there lands under every one of
// them, in the tree, the index and the indexes a peer regenerates. A linked
// worktree is never refused, nor is a checkout one session or none works in,
// so a person working alone is not held up. Exit 3 is abcd's "back off",
// as a held reservation is. A count that cannot be made (no process table on
// this platform) refuses nothing and says so.
func refuseSharedMainCheckout(cmd *cobra.Command, verb, root string) error {
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return nil
	}
	err := occupancy.Check(m.Home, root, m.ReadTable)
	var shared *occupancy.SharedError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &shared):
		return &exitError{Code: 3, Msg: verb + ": " + shared.Error()}
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s: could not count the sessions working in this checkout (%s), so a shared main checkout is not refused this time\n",
		verb, termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	return nil
}

// fileWriteTools are the host tool names that write a file. The hook
// manifest's PreToolUse matcher names exactly these beside the shell and
// question tools (TestGuardHookIsInstalledForBashCalls holds the two
// together), so the guard hook sees every write the host's own tools make.
var fileWriteTools = []string{"Write", "Edit", "MultiEdit", "NotebookEdit"}

// isFileWriteTool reports whether a hook payload's tool writes a file.
func isFileWriteTool(name string) bool {
	for _, f := range fileWriteTools {
		if strings.EqualFold(name, f) {
			return true
		}
	}
	return false
}

// fileWriteFailOpen is the guard's fail-open-loud contract on a file tool's
// call: the write runs, and exit 1 keeps the warning in front of a human.
func fileWriteFailOpen(w io.Writer, format string, a ...any) error {
	diagnosticLine(w, "abcd guard: NOT CHECKED — "+format+". This file write runs UNGUARDED.", a...)
	return &exitError{Code: 1}
}

// fileWriteGate is the guard hook's judgement of a file the host's own tools
// are about to write (iss-2610280051498465): the record verbs refuse a main
// checkout two or more live sessions share, and a file written there through
// Write or Edit lands under all of them just the same. The file's working
// tree is found by walking up to its .git, with no git subprocess, and judged
// by occupancy.Check, so a linked worktree, a checkout one session or none
// works in, and a path in no checkout are allowed silently. A shared one is
// the host's deny with the verbs' own refusal, naming the worktree store. A
// call naming no file, or a count that could not be made, fails open loud.
func fileWriteGate(cmd *cobra.Command, tool, cwd, filePath, notebookPath string) error {
	path := filePath
	if path == "" {
		path = notebookPath
	}
	if strings.TrimSpace(path) == "" {
		return fileWriteFailOpen(cmd.ErrOrStderr(), "the %s call names no file to check", termsafe.Sanitize(tool))
	}
	if !filepath.IsAbs(path) {
		base := cwd
		if base == "" {
			wd, err := os.Getwd()
			if err != nil {
				return fileWriteFailOpen(cmd.ErrOrStderr(), "the %s call names a relative path and the session directory is unknown", termsafe.Sanitize(tool))
			}
			base = wd
		}
		path = filepath.Join(base, path)
	}
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return nil
	}
	err := occupancy.CheckPath(m.Home, path, m.ReadTable)
	var shared *occupancy.SharedError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &shared):
		return denyCall("Blocked by the abcd guard (" + termsafe.Sanitize(tool) + " into a shared main checkout): " + shared.Error())
	}
	return fileWriteFailOpen(cmd.ErrOrStderr(), "could not count the sessions working in the checkout this file is in (%s)",
		termsafe.Sanitize(fsutil.RedactHome(err.Error())))
}
