package cli

// The SubagentStart hook: what a sub-agent is told as it starts
// (spc-2611251757228271).
//
// The rules loader runs when a person sends a prompt, and a sub-agent is sent
// none; the conventions file reaches it only where the host loads it. So the
// hook tells the sub-agent where the conventions file is, which two commands
// print abcd's rules, and the names of the domains in force, and it carries no
// rule's text and no word of the file. The host delivers the hook's
// additionalContext to the sub-agent itself, and nothing else of its stdout.
//
// abcd's four cold-reading agents are told nothing: a reading's context holds
// its assembled input and no repository path. A repository abcd does not
// manage, or one whose rules are switched off, hears nothing.
//
// A sub-agent's start never waits on it. The work runs beside a bound of five
// seconds, under the ten the manifest declares, and stdout is written whole or
// not at all. A failure exits 1 with one line on stderr: never 0, which would
// hide it, and never 2, the host's blocking status, which a Go panic would
// give.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// subagentStartBound is the hook's own deadline, set before stdin is read and
// under the timeout hooks/hooks.json declares for the event, so the hook
// answers for itself before the host kills it.
var subagentStartBound = 5 * time.Second

// newSubagentStartCommand builds `abcd hook subagent-start`.
func newSubagentStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "subagent-start",
		Short: "SubagentStart: tell a starting sub-agent where the conventions are and which commands print the rules",
		Args:  cobra.NoArgs,
		RunE:  runHookSubagentStart,
	}
}

// errSubagentStartSpent is the work finding the hook's bound spent; the hook,
// which holds the bound, says how long it was.
var errSubagentStartSpent = errors.New("the bound was spent")

// subagentStartAnswer is what the work beside the bound came to: the stdout
// document (empty for silence), the diagnostics to write beside it, or the
// failure.
type subagentStartAnswer struct {
	in    hookInput
	out   []byte
	told  *rules.SubagentRecord // the told fact to keep before out is written, nil for none
	notes []byte
	err   error
}

// runHookSubagentStart is the hook. Only this goroutine writes to the host.
func runHookSubagentStart(cmd *cobra.Command, _ []string) (err error) {
	// The bound is read once, here: the work below may outlive this call, and
	// reads none of the hook's settings once it has started.
	bound := subagentStartBound
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()
	var in hookInput
	defer func() {
		if r := recover(); r != nil {
			err = subagentStartFailed(cmd, in, fmt.Errorf("an internal fault (%T)", r))
		}
	}()
	read := make(chan hookInput, 1)
	done := make(chan subagentStartAnswer, 1)
	go func() {
		var a subagentStartAnswer
		defer func() {
			if r := recover(); r != nil {
				a.out, a.err = nil, fmt.Errorf("an internal fault (%T)", r)
			}
			done <- a
		}()
		if a.in, a.err = readHookInput(cmd); a.err != nil {
			a.err = errors.New("the hook payload could not be read")
			return
		}
		read <- a.in
		var notes bytes.Buffer
		a.out, a.told, a.err = subagentStartMessage(ctx, a.in, &notes)
		a.notes = notes.Bytes()
	}()

	var a subagentStartAnswer
	select {
	case a = <-done:
	case <-ctx.Done():
		// The work and the bound may end together: an answer that is there
		// is the work's, and it is delivered as if it had come first.
		select {
		case a = <-done:
			if !errors.Is(a.err, errSubagentStartSpent) {
				return deliverSubagentStart(cmd, a)
			}
			return subagentStartFailed(cmd, a.in, fmt.Errorf("it did not finish within %s", bound))
		default:
		}
		// Stdin may never end, and the work may be past any cancelling. What
		// was read of the payload says whose failure this is.
		select {
		case in = <-read:
		default:
		}
		if !looksManaged(orProcessDir(in.Cwd)) {
			return nil
		}
		return subagentStartFailed(cmd, in, fmt.Errorf("it did not finish within %s", bound))
	}
	if errors.Is(a.err, errSubagentStartSpent) {
		return subagentStartFailed(cmd, a.in, fmt.Errorf("it did not finish within %s", bound))
	}
	return deliverSubagentStart(cmd, a)
}

// deliverSubagentStart writes a finished answer. An answer that cannot be
// written told the sub-agent nothing, so the told fact the work kept for it
// is taken back, and the sub-agent reads as never told rather than told.
func deliverSubagentStart(cmd *cobra.Command, a subagentStartAnswer) error {
	switch {
	case a.err != nil:
		return subagentStartFailed(cmd, a.in, a.err)
	case len(a.out) == 0:
		return nil
	}
	// The note is kept here, by the one goroutine that writes to the host, and
	// before the answer: a note that cannot be kept sends nothing, so every
	// sub-agent that was told has one, and a bound spent while the work ran
	// leaves none, since the work keeps no fact itself.
	if a.told != nil {
		if err := rules.WriteSubagentFact(a.in.SessionID, a.in.AgentID, rules.SubagentTold, *a.told); err != nil {
			return subagentStartFailed(cmd, a.in, errors.New("the note that it was told could not be kept"))
		}
	}
	if _, werr := cmd.OutOrStdout().Write(a.out); werr != nil {
		if a.told != nil {
			if rec, found, err := rules.ReadSubagentFact(a.in.SessionID, a.in.AgentID, rules.SubagentTold); err == nil && found && rec.By == rules.SubagentToldByStart {
				_ = rules.RemoveSubagentFact(a.in.SessionID, a.in.AgentID, rules.SubagentTold)
			}
		}
		return subagentStartFailed(cmd, a.in, errors.New("its answer could not be written"))
	}
	_, _ = cmd.ErrOrStderr().Write(a.notes)
	return nil
}

// orProcessDir is a payload's working directory, else the process's.
func orProcessDir(cwd string) string {
	if cwd != "" {
		return cwd
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

// subagentStartFailed is the hook failing: one line on stderr, exit 1, and the
// failure counted in the lane's marker. It runs no git: the store is the one
// the session-start hook noted for the session.
func subagentStartFailed(cmd *cobra.Command, in hookInput, why error) error {
	counted := ""
	if err := countSubagentStartFailure(in); err != nil {
		counted = "; the failure was not counted"
	}
	diagnosticLine(cmd.ErrOrStderr(), "abcd: this sub-agent was not told where the conventions and the rules are (%s)%s", scrubPaths(why), counted)
	return exitWith(1, refusal.None, "")
}

func countSubagentStartFailure(in hookInput) error {
	sha, err := history.SessionRepo(in.SessionID)
	if err != nil {
		return err
	}
	agent := ""
	if idScalarRe.MatchString(in.AgentID) {
		agent = in.AgentID
	}
	return history.NoteSubagentLine("", sha, history.SubagentLine{
		AgentType: safeTextScalar(in.AgentType), AgentID: agent, Kind: history.SubagentStartFailed,
	}, history.SubagentLineLock)
}

// coldReader reports whether the host names one of abcd's cold-reading agents,
// by its own name or the plugin's.
func coldReader(agentType string) bool {
	return slices.Contains(oracle.SelfContained(), strings.TrimPrefix(agentType, "abcd:"))
}

// subagentKey reports whether a payload's ids can key a sub-agent's facts.
func subagentKey(session, agent string) bool {
	return idScalarRe.MatchString(session) && idScalarRe.MatchString(agent)
}

// subagentStartMessage is the stdout document for one payload, nil where the
// sub-agent is told nothing, and the told fact the hook keeps before it writes
// the document, nil where it keeps none. It writes no fact itself: the hook
// may give up on it at its bound, and a fact kept by work given up on would
// say the sub-agent was told when nothing reached it. Anything the helpers
// have to say goes to notes.
func subagentStartMessage(ctx context.Context, in hookInput, notes io.Writer) ([]byte, *rules.SubagentRecord, error) {
	if coldReader(in.AgentType) {
		return nil, nil, nil
	}
	c, err := subagentConventions(ctx, orProcessDir(in.Cwd), notes)
	if err != nil || !c.Applies {
		return nil, nil, err
	}
	doc, err := composeSubagentStart(c, subagentAbcd(), notes)
	if err != nil {
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, errSubagentStartSpent
	}
	var told *rules.SubagentRecord
	if subagentKey(in.SessionID, in.AgentID) {
		told = &rules.SubagentRecord{File: c.File, Root: c.Root, By: rules.SubagentToldByStart}
	}
	return doc, told, nil
}

// conventionsScope is where a sub-agent's conventions come from: the checkout
// root looked in, the conventions file's name there ("" for none), the rules
// in force, and whether any of it applies.
type conventionsScope struct {
	Root    string
	File    string
	Rules   rules.RuleSet
	Applies bool
}

// subagentConventions answers where the conventions apply for a sub-agent in
// cwd. The start hook and the guard's hold both ask it, so the two cannot
// disagree. Nothing applies in a checkout abcd does not manage, or where the
// rules are switched off. An error is that it could not be learned: git did not
// answer before ctx ended, or a rules file will not load; a ctx spent in a
// checkout that does not look managed is nothing applying, as the other hooks
// read it. The conventions file is looked at and never opened, so nothing in it
// can reach an answer.
func subagentConventions(ctx context.Context, cwd string, errOut io.Writer) (conventionsScope, error) {
	unanswered := func() (conventionsScope, error) {
		if !looksManaged(cwd) {
			return conventionsScope{}, nil
		}
		return conventionsScope{}, errors.New("git did not answer in time which checkout this is")
	}
	root, managed := managedRootContext(ctx, cwd)
	if !managed {
		if ctx.Err() != nil {
			return unanswered()
		}
		return conventionsScope{}, nil
	}
	rulesAt, err := rulesRootContext(ctx, cwd, errOut)
	if err != nil {
		return unanswered()
	}
	rs, err := rulesForHook(rulesAt, errOut)
	if err != nil {
		return conventionsScope{}, fmt.Errorf("the rules did not load: %w", err)
	}
	if rs.Disabled {
		return conventionsScope{}, nil
	}
	c := conventionsScope{Root: root, Rules: rs, Applies: true}
	for _, name := range rules.SubagentConventionsFiles {
		if fi, err := os.Stat(filepath.Join(root, name)); err == nil && fi.Mode().IsRegular() {
			c.File = name
			break
		}
	}
	return c, nil
}

// plainPath is p as a message writes it. ok is false where it holds any byte
// outside letters, digits and / . _ - + @ ~ , : =, the space included, and it
// is then left out: no word of a directory's name is written into a message.
func plainPath(p string) (string, bool) {
	if p == "" {
		return "", false
	}
	for i := range len(p) {
		if !plainByte(p[i]) {
			return "", false
		}
	}
	return p, true
}

// plainByte reports whether c is written into a message and a command as it is.
func plainByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("/._-+@~,:=", c) >= 0
}

// shellPath is p as a shell word: as it is where every byte is plain, and else
// one single-quoted word (shellquote.Single). ok is false where p is not valid
// UTF-8 or holds a rune that does not print, and it is then left out.
func shellPath(p string) (string, bool) {
	if p == "" || !utf8.ValidString(p) {
		return "", false
	}
	plain := true
	for _, r := range p {
		if !unicode.IsPrint(r) {
			return "", false
		}
		plain = plain && r < utf8.RuneSelf && plainByte(byte(r))
	}
	if plain {
		return p, true
	}
	return shellquote.Single(p), true
}

// subagentAbcd is the abcd a sub-agent is told to run, in every checkout: this
// hook's own executable, so it exists and is the version that named the
// domains. It is "" where that path cannot be printed, and the message then
// names no command.
func subagentAbcd() string {
	if exe, err := osExecutable(); err == nil && filepath.IsAbs(exe) {
		if p, ok := shellPath(exe); ok {
			return p
		}
	}
	return ""
}

// The two commands a sub-agent is told, each after the abcd that runs it.
const (
	subagentRulesFor    = ` rules --prompt "<your task in one sentence>"`
	subagentRulesDomain = " rules <DOMAIN>"
)

// composeSubagentStart is the stdout document, whole and within
// rules.InjectionBudgetBytes; abcd is the one the commands run. A domain an
// override names carries the label it has wherever a domain is printed. A list
// of domains that does not fit is replaced whole by its count and the command
// that prints it, said on notes; no name is ever cut.
func composeSubagentStart(c conventionsScope, abcd string, notes io.Writer) ([]byte, error) {
	var names []string
	for _, d := range c.Rules.Active() {
		names = append(names, rules.Label(d.Name, d.Source))
	}
	root, _ := plainPath(c.Root)
	list := fmt.Sprintf("Domains in force (%d): %s", len(names), strings.Join(names, ", "))
	if len(names) == 0 {
		list = "Domains in force (0)."
	}
	doc, err := subagentStartDoc(c.File, root, abcd, list)
	if err != nil || len(doc) <= rules.InjectionBudgetBytes {
		return doc, err
	}
	counted := func(abcd string) string {
		if abcd == "" {
			return fmt.Sprintf("Domains in force: %d, too many to name here.", len(names))
		}
		return fmt.Sprintf("Domains in force: %d, too many to name here; %s rules lists them.", len(names), abcd)
	}
	diagnosticLine(notes, "abcd: the %d rules domains in force do not fit what a sub-agent can be told, so it is told their count and the command that lists them", len(names))
	doc, err = subagentStartDoc(c.File, root, abcd, counted(abcd))
	if err != nil || len(doc) <= rules.InjectionBudgetBytes {
		return doc, err
	}
	// Paths this long name nothing a message has room for.
	return subagentStartDoc(c.File, "", "", counted(""))
}

// subagentStartDoc is the host's output object for one message.
func subagentStartDoc(file, root, abcd, domains string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("abcd: for a sub-agent starting in this repository.\n")
	from := "the root of the worktree you work in"
	if file != "" {
		fmt.Fprintf(&b, "Conventions: read %s at the root of the worktree you work in first.", file)
		if root != "" {
			fmt.Fprintf(&b, " This hook looked in %s; if you were given another worktree, read the copy at its root.", root)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "Writes: the guard refuses your file writes, and the git commands that record or send your work, until a call of your own has read %s.\n", file)
		from = "that root"
	}
	if abcd == "" {
		b.WriteString("Rules: abcd's rules are not sent to you, and the path of the abcd that runs this hook cannot be printed here.\n")
	} else {
		fmt.Fprintf(&b, "Rules: abcd's rules are not sent to you; fetch them from %s.\n", from)
		fmt.Fprintf(&b, "  %s%s   names the domains your task recalls\n", abcd, subagentRulesFor)
		fmt.Fprintf(&b, "  %s%s   prints one domain's rules\n", abcd, subagentRulesDomain)
	}
	b.WriteString(domains)

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	err := enc.Encode(promptOutput{HookSpecificOutput: promptOutputFields{
		HookEventName: "SubagentStart", AdditionalContext: b.String(),
	}})
	if err != nil {
		return nil, errors.New("its answer could not be encoded")
	}
	return out.Bytes(), nil
}
