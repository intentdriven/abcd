package cli

import (
	"errors"
	"strings"

	"github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/spf13/cobra"
)

// refusal_codes.go — the refusal's code, hint and next command at the error
// boundary (itd-2610301428378830, spc-2610301431108175 steps 1 and 2). Run
// reads the code from the error, finds its entry in the registry
// (internal/core/refusal), and adds the hint and the next command to the
// --json envelope and, as a second line, to the text form. The next command
// is filled from the refused call's own command path, never from the
// caller's arguments, which Run withholds.

// refusalUncodedPath lists, from the repository root, the files whose
// exitError literals carry no code yet. The literal test exempts only those,
// and the list only shrinks: the step that codes a file's last refusal
// removes the file from it, and the last step deletes the list.
const refusalUncodedPath = ".abcd/development/release/refusal-uncoded.txt"

// refusalCodeCheck, when set, is told every refusal Run renders, with its
// exit and its code. It is nil in the binary; the package's tests set it so
// the test binary fails on a refusal that leaves with a code the registry
// does not hold (refusal_codes_test.go).
var refusalCodeCheck func(exit int, ref refusal.Code, args []string)

// refusalCodeOf is the code a refusal carries, in the order the spec reads
// it (approach 2): the code an exitError was built with, then the code a core
// error was wrapped with (refusal.Wrap), then the usage classifier's
// (approach 3). A fault (exit 1) carries none, except a usage refusal the
// hook plane moved to exit 1, which keeps its usage code.
func refusalCodeOf(err error, msg string, exit int) refusal.Code {
	var ee *exitError
	if errors.As(err, &ee) && ee.ref != refusal.None {
		return ee.ref
	}
	if c := refusal.CodeOf(err); c != refusal.None {
		return c
	}
	if exit == 2 || exit == 3 || strings.Contains(msg, hookPlaneSkewNote) {
		return usageCode(err, msg)
	}
	return refusal.None
}

// refusalCommandPath is the refused call's command path, which fills a next
// command's slot. A command cobra adds only while it runs (its help verb and
// its completion commands) is not one `abcd explain` answers, so its refusal
// names the root instead.
func refusalCommandPath(called *cobra.Command) string {
	if called == nil {
		return "abcd"
	}
	path := called.CommandPath()
	words := strings.Fields(path)
	if len(words) > 1 && findByPath(NewRootCommand(), words[1:]) == nil {
		return words[0]
	}
	return path
}

// withRefusal adds a refusal's code, hint and next command to the envelope,
// the next command filled from called's command path. A code the registry
// does not hold (internal/uncoded above all) is carried alone, so a reader
// sees it and the tests fail on it.
func (e *errorEnvelope) withRefusal(ref refusal.Code, called *cobra.Command) {
	if ref == refusal.None {
		return
	}
	e.Code = string(ref)
	entry, ok := refusal.Lookup(ref)
	if !ok {
		return
	}
	e.Hint = entry.Summary
	e.Next = entry.Next
	if strings.Contains(e.Next, refusal.CommandSlot) {
		e.Next = refusal.FillNext(e.Next, refusalCommandPath(called))
	}
}

// refusalLine is the text form's second line: the hint, the next command to
// try and the call that explains the code; empty for a refusal with no
// registered code. A next that runs as it stands is offered with "try:", and
// a template, whose <…> words are values the caller supplies, with "fill in
// and run:", so neither is mistaken for the other.
func (e errorEnvelope) refusalLine() string {
	if e.Code == "" || e.Hint == "" {
		return ""
	}
	line := e.Hint
	switch {
	case e.Next == "" || e.Next == refusal.NoNext:
	case refusal.IsTemplate(e.Next):
		line += " — fill in and run: " + e.Next
	default:
		line += " — try: " + e.Next
	}
	return line + " (abcd explain " + e.Code + ")"
}
