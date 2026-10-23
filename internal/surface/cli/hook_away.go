package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// hook_away.go — the away record repeated to the agent and shown to the
// person (spc-2610180914469489 step 2). The session-start hook, on every
// source (`compact` included), and the prompt hook, on every prompt and
// outside the rules loader's dedup ledger, add the away block to the agent's
// context while the record holds; the prompt hook also shows the person one
// line (a systemMessage) saying so, or, once the window has lapsed unclosed
// with anything unread, one line naming `abcd mode back`.
//
// The block's words come from the away record, which lives in the away store
// under the home folder (owner-only, checked as rules.json is), never from a
// tracked file, and every interpolated value is cleaned to one line.

// awayBlock is the agent-facing block for a record that holds.
func awayBlock(rec *autonomy.Record) string {
	var b strings.Builder
	b.WriteString("## AWAY\n\n")
	for _, l := range rec.Terms() {
		b.WriteString(l + "\n")
	}
	b.WriteString("\nThe mode is never permission for anything. While it holds, a turn's end is checked: while work is owed the session keeps working, and it waits only with a scheduled wake that fires soon enough.\n\n")
	b.WriteString("Never ask the person a question while they are away.\n")
	b.WriteString(autonomy.QuestionRoute() + "\n")
	b.WriteString("\nRun `abcd mode back` only on the person's word in this session.\n")
	return b.String()
}

// awayView is what the away record adds to one prompt: the block for the
// agent and the line for the person, either possibly empty.
type awayView struct {
	Block   string
	Message string
}

// readPromptAway reads the away record for the repository whose root commit
// the live drain already looked up for this prompt, so the router still makes
// one root-commit lookup a prompt, and only when abcd manages the checkout at
// root (the loader's root for this prompt). It fails open: a record that cannot be read
// safely is named on stderr and adds nothing; with no root commit (outside a
// checkout, or git not answering in time) it adds nothing silently.
func readPromptAway(stderr io.Writer, drained rootLookup, root string) awayView {
	if !autonomy.Managed(root) || !drained.asked || drained.err != nil || drained.sha == "" {
		return awayView{}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return awayView{}
	}
	st, err := autonomy.StoreAt(home, drained.sha)
	if err != nil {
		return awayView{}
	}
	now := time.Now()
	rec, holds, err := st.Away(now)
	if err != nil {
		diagnosticLine(stderr, "abcd mode: %s; read as not away", fsutil.RedactHome(err.Error()))
		return awayView{}
	}
	switch {
	case holds:
		return awayView{Block: awayBlock(rec), Message: rec.PersonLine()}
	case rec.Lapsed(now):
		return awayView{Message: rec.LapsedLine()}
	}
	return awayView{}
}

// readAwayFailOpen is the hooks' read of the away record: ok false when there
// is nothing to read here (no checkout, no root commit, no home) or the
// record failed the store's checks, which is named on stderr.
func readAwayFailOpen(stderr io.Writer, cwd string) (*autonomy.Record, bool, bool) {
	if cwd == "" {
		return nil, false, false
	}
	_, st, err := autonomy.Resolve(cwd, autonomy.Options{})
	if err != nil {
		return nil, false, false
	}
	rec, holds, err := st.Away(time.Now())
	if err != nil {
		diagnosticLine(stderr, "abcd mode: %s; read as not away", fsutil.RedactHome(err.Error()))
		return nil, false, false
	}
	return rec, holds, true
}

// managedRoot is the root of the checkout cwd stands in, and whether abcd
// manages it (autonomy.Managed). Every away hook and the branch recorder ask
// it first: outside a managed checkout they read nothing, write nothing and
// say nothing, and run no git at all where the checkout's .git is a folder.
func managedRoot(cwd string) (string, bool) {
	if cwd == "" {
		return "", false
	}
	// The marker first, with no git at all: the nearest folder holding a
	// .git entry. A plain .git folder with no .abcd/ beside it is a
	// checkout abcd does not manage, and the hooks leave it untouched; only
	// a managed-looking one, or a linked worktree's .git file, is asked of
	// git.
	shaped := gitutil.RepoShapedRoot(cwd)
	if shaped == "" {
		return "", false
	}
	if fi, err := os.Lstat(filepath.Join(shaped, ".git")); err == nil && fi.IsDir() && !autonomy.Managed(shaped) {
		return "", false
	}
	root, err := gitutil.CheckoutRoot(cwd, autonomy.StoreName)
	if err != nil || !autonomy.Managed(root) {
		return "", false
	}
	return root, true
}

// sessionStartAway notes the session's start copy and returns the away block
// when the record holds. Both happen only in a checkout abcd manages (one with
// `.abcd/` at its root), and the note only the first time for a session, so a
// compaction or resume run from a worktree the session made keeps the start.
func sessionStartAway(stderr io.Writer, in hookInput, cwd string) string {
	root, ok := managedRoot(cwd)
	if !ok {
		return ""
	}
	if autonomy.ValidSessionID(in.SessionID) {
		if _, st, err := autonomy.Resolve(root, autonomy.Options{}); err == nil {
			if _, err := st.NoteSessionCopy(in.SessionID, root, time.Now()); err != nil {
				diagnosticLine(stderr, "abcd mode: the session's start copy was not noted (%s)", fsutil.RedactHome(err.Error()))
			}
		}
	}
	rec, holds, ok := readAwayFailOpen(stderr, root)
	if !ok || !holds {
		return ""
	}
	return awayBlock(rec)
}

// promptOutput is the prompt hook's object when it shows the person a line:
// systemMessage for the person, the additional context for the agent, and
// the session's title when this prompt sets it.
type promptOutput struct {
	SystemMessage      string             `json:"systemMessage,omitempty"`
	HookSpecificOutput promptOutputFields `json:"hookSpecificOutput"`
}

type promptOutputFields struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
	SessionTitle      string `json:"sessionTitle,omitempty"`
}

// emitPrompt writes the prompt hook's plain output: the text alone when there
// is no title and no line for the person, the titled object when only a title
// is owed, and otherwise one object carrying the person's line beside the
// context (and the title). A write that fails is named on stderr; the hook
// still exits 0.
func emitPrompt(cmd *cobra.Command, text, title, message string) {
	switch {
	case message == "" && title == "":
		if text != "" {
			io.WriteString(cmd.OutOrStdout(), text)
		}
		return
	case message == "":
		emitTitled(cmd, text, title)
		return
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(promptOutput{
		SystemMessage: termsafe.CleanProseLine(message, 1024),
		HookSpecificOutput: promptOutputFields{
			HookEventName: "UserPromptSubmit", AdditionalContext: text, SessionTitle: title,
		},
	})
	if err == nil {
		_, err = io.Copy(cmd.OutOrStdout(), &b)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		diagnosticLine(cmd.ErrOrStderr(), "abcd mode: the prompt hook's output was not written (%s)", err.Error())
	}
}
