// Command asking-sync writes the asking rules into the two pages that state
// them: the block of commands/intent.md between the asking-rules markers is
// rendered from question.AskingRules(question.Default) and the knowledge-floor
// pointer, and the block of agents/question-drafter.md between the
// question-drafter markers is rendered from the same rules and the row count
// the check makes, every figure read from question.Default. So the pages, the
// GRILL rule domain and the question check in `abcd guard hook` state every
// limit from the one value
// (spc-2610030944505997, "The interview pages and the glossary floor";
// itd-2610030810350727 criterion A7). The rules are edited in
// internal/core/question/asking.go and the limits in limits.go, never in the
// page.
//
// It mirrors scaffold-sync, run by hand via `make asking-sync` (open question
// 3, decided (a)). Nothing in CI invokes it: the drift is gated by
// TestIntentPageAskingBlockIsGenerated and TestDrafterBlockIsGenerated under
// preflight, which fail when a committed block differs from the rendering.
//
// Exit codes: 0 nothing to do (or, with no -check, the blocks were written);
// 1 a fault; 3 with -check, a block is out of date and nothing was written.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	// pagePath is the planning interview's page carrying the asking-rules
	// block, from the repository root.
	pagePath = "commands/intent.md"
	// beginMarker and endMarker fence the generated block.
	beginMarker = "<!-- generated: asking-rules -->"
	endMarker   = "<!-- /generated -->"
	// drafterPath is the question-drafter agent's prompt, whose block carries
	// the asking rules and the row count.
	drafterPath = "agents/question-drafter.md"
	// drafterBeginMarker opens the drafter's block; endMarker closes it.
	drafterBeginMarker = "<!-- generated: question-drafter -->"
)

// target is one page asking-sync writes: where it is, the marker opening its
// block, and the block's rendering from the limits.
type target struct {
	path   string
	begin  string
	render func(question.Limits) string
}

// targets are the pages asking-sync keeps, in the order it writes them.
var targets = []target{
	{pagePath, beginMarker, render},
	{drafterPath, drafterBeginMarker, renderDrafter},
}

// render is the block between the markers: a note for whoever edits the page,
// the paragraph's lead, every asking rule as its own list item, and the link
// to the knowledge floor. The rules are rendered one per line, unwrapped, so
// no continuation line can be read as markdown of its own.
func render(l question.Limits) string {
	var b strings.Builder
	b.WriteString("\n<!-- Written by `make asking-sync` from internal/core/question; edit asking.go or limits.go there, never this block. -->\n")
	b.WriteString("**How every question is asked (the GRILL rule domain).** These are the rules the GRILL domain carries into every repository abcd manages, and the question check refuses a question that breaks the layout they set:\n\n")
	for _, r := range question.AskingRules(l) {
		b.WriteString("- " + r + "\n")
	}
	fmt.Fprintf(&b, "\nThe knowledge floor every explanation is measured against: [%s](../%s).\n",
		filepath.Base(question.KnowledgeFloorPage), question.KnowledgeFloorPage)
	return b.String()
}

// renderDrafter is the question-drafter's block: a note for whoever edits the
// page, every asking rule as its own list item, and the row count the check
// makes, each figure filled from l. It states the count in the words of
// internal/core/question's estimate (estimateRows, splitRows, blockRows), so
// the agent's arithmetic and the check's read one statement of each figure.
func renderDrafter(l question.Limits) string {
	var b strings.Builder
	b.WriteString("\n<!-- Written by `make asking-sync` from internal/core/question; edit asking.go, limits.go or rows.go there, never this block. -->\n")
	b.WriteString("## The asking rules\n\n")
	b.WriteString("The question you return follows every one of these rules, the rules abcd's question check holds the question to:\n\n")
	for _, r := range question.AskingRules(l) {
		b.WriteString("- " + r + "\n")
	}
	b.WriteString("\n## Counting rows\n\n")
	fmt.Fprintf(&b, "The check estimates the rows one tab takes in the host's question view in a window %d columns wide, and a tab over %d rows is over the limit. Count each tab the same way:\n\n", l.Columns, l.Rows)
	fmt.Fprintf(&b, "1. The header and the host's frame: %d rows, the chip's row included.\n", l.HostChromeRows)
	fmt.Fprintf(&b, "2. The question text: its rows wrapped at %d columns.\n", l.HostTextColumns)
	fmt.Fprintf(&b, "3. Each option: its label's rows wrapped at %d columns, never fewer than 1, plus its description's rows wrapped at %d columns.\n", l.HostOptionColumns, l.HostOptionColumns)
	b.WriteString("\nThe tab's rows are the sum of the three. To wrap a text at a width, take each of its lines on its own and fill it word by word, as a terminal fills a line: a word goes on the current line when it fits beside the words already there, separated by one space, and starts a new line when it does not. Each line of the text takes at least one row, so a blank line is one row; blank lines at the start and the end of the text are dropped; an empty text is 0 rows. A word wider than the width stands alone on its line and the host hard-wraps it, so that line counts its width divided by the width, rounded up. Width is counted in terminal columns: most characters take one, and East Asian wide and fullwidth characters take two.\n")
	return b.String()
}

// sync returns page with its one asking-rules block rendered from l, every
// byte outside the block kept. A page with no block, an unclosed one, or more
// than one is a fault: the command never guesses where the block belongs.
func sync(page []byte, l question.Limits) ([]byte, error) {
	return syncTarget(page, targets[0], l)
}

// syncTarget is sync for any target: its one block rendered from l.
func syncTarget(page []byte, t target, l question.Limits) ([]byte, error) {
	begin, end := []byte(t.begin), []byte(endMarker)
	switch n := bytes.Count(page, begin); n {
	case 0:
		return nil, fmt.Errorf("%s carries no %s marker", t.path, t.begin)
	case 1:
	default:
		return nil, fmt.Errorf("%s carries %d %s markers; it must carry one", t.path, n, t.begin)
	}
	start := bytes.Index(page, begin) + len(begin)
	stop := bytes.Index(page[start:], end)
	if stop < 0 {
		return nil, fmt.Errorf("%s opens %s and never closes it with %s", t.path, t.begin, endMarker)
	}
	out := make([]byte, 0, len(page))
	out = append(out, page[:start]...)
	out = append(out, t.render(l)...)
	out = append(out, page[start+stop:]...)
	return out, nil
}

func main() {
	check := flag.Bool("check", false, "report drift and write nothing; exit 3 when the block is out of date")
	rootPath := flag.String("root", "", "repo root to sync (default: git toplevel, or cwd)")
	flag.Parse()

	root := *rootPath
	if root == "" {
		root = resolveRoot()
	}
	if err := run(root, *check); err != nil {
		if errors.Is(err, errOutOfDate) {
			fmt.Printf("asking-sync: %s\n", err)
			fmt.Fprintln(os.Stderr, "asking-sync: run `make asking-sync` to write the asking rules into the pages.")
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, "asking-sync:", scrubRoot(err.Error(), root))
		os.Exit(1)
	}
}

// errOutOfDate is -check finding a committed block differs from the
// rendering; the error wrapping it names each page.
var errOutOfDate = errors.New("out of date with the asking rules")

// run syncs every target's page under root, or with check only compares them.
// Every page is read and rendered before any is written, so a fault in one
// leaves all of them as they were.
func run(root string, check bool) error {
	type pending struct {
		path   string
		synced []byte
	}
	var stale []pending
	for _, t := range targets {
		p := filepath.Join(root, filepath.FromSlash(t.path))
		page, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", t.path, err)
		}
		synced, err := syncTarget(page, t, question.Default)
		if err != nil {
			return err
		}
		if bytes.Equal(synced, page) {
			fmt.Printf("asking-sync: %s already carries the asking rules; nothing to do.\n", t.path)
			continue
		}
		stale = append(stale, pending{t.path, synced})
	}
	if len(stale) == 0 {
		return nil
	}
	if check {
		names := make([]string, len(stale))
		for i, s := range stale {
			names[i] = s.path
		}
		return fmt.Errorf("%s: %w", strings.Join(names, ", "), errOutOfDate)
	}
	for _, s := range stale {
		if err := fsutil.WriteFileAtomicPreserveMode(filepath.Join(root, filepath.FromSlash(s.path)), s.synced); err != nil {
			return fmt.Errorf("write %s: %w", s.path, err)
		}
		fmt.Printf("asking-sync: wrote the asking rules into %s\n", s.path)
	}
	return nil
}

// resolveRoot returns the git toplevel, falling back to the working directory,
// as scaffold-sync does.
func resolveRoot() string {
	if wd, err := os.Getwd(); err == nil {
		if top, err := gitutil.Toplevel(wd); err == nil {
			return top
		}
		return wd
	}
	return "."
}

// scrubRoot keeps an absolute checkout path out of a reported fault, as
// scaffold-sync does: the message names the repo-relative file.
func scrubRoot(msg, root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return msg
	}
	return strings.ReplaceAll(strings.ReplaceAll(msg, abs+string(filepath.Separator), ""), abs, ".")
}
