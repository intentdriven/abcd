// Command asking-sync writes the asking rules into the planning interview's
// page: the block of commands/intent.md between the asking-rules markers is
// rendered from question.AskingRules(question.Default) and the knowledge-floor
// pointer, so the page, the GRILL rule domain and the question check in
// `abcd guard hook` state every limit from the one value
// (spc-2610030944505997, "The interview pages and the glossary floor";
// itd-2610030810350727 criterion A7). The rules are edited in
// internal/core/question/asking.go and the limits in limits.go, never in the
// page.
//
// It mirrors scaffold-sync, run by hand via `make asking-sync` (open question
// 3, decided (a)). Nothing in CI invokes it: the drift is gated by
// TestIntentPageAskingBlockIsGenerated under preflight, which fails when the
// committed block differs from the rendering.
//
// Exit codes: 0 nothing to do (or, with no -check, the block was written);
// 1 a fault; 3 with -check, the block is out of date and nothing was written.
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
	// pagePath is the page carrying the block, from the repository root.
	pagePath = "commands/intent.md"
	// beginMarker and endMarker fence the generated block.
	beginMarker = "<!-- generated: asking-rules -->"
	endMarker   = "<!-- /generated -->"
)

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

// sync returns page with its one asking-rules block rendered from l, every
// byte outside the block kept. A page with no block, an unclosed one, or more
// than one is a fault: the command never guesses where the block belongs.
func sync(page []byte, l question.Limits) ([]byte, error) {
	begin, end := []byte(beginMarker), []byte(endMarker)
	switch n := bytes.Count(page, begin); n {
	case 0:
		return nil, fmt.Errorf("%s carries no %s marker", pagePath, beginMarker)
	case 1:
	default:
		return nil, fmt.Errorf("%s carries %d %s markers; it must carry one", pagePath, n, beginMarker)
	}
	start := bytes.Index(page, begin) + len(begin)
	stop := bytes.Index(page[start:], end)
	if stop < 0 {
		return nil, fmt.Errorf("%s opens %s and never closes it with %s", pagePath, beginMarker, endMarker)
	}
	out := make([]byte, 0, len(page))
	out = append(out, page[:start]...)
	out = append(out, render(l)...)
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
			fmt.Printf("asking-sync: %s is out of date with the asking rules\n", pagePath)
			fmt.Fprintln(os.Stderr, "asking-sync: run `make asking-sync` to write the asking rules into the page.")
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, "asking-sync:", scrubRoot(err.Error(), root))
		os.Exit(1)
	}
}

// errOutOfDate is -check finding the committed block differs from the
// rendering.
var errOutOfDate = errors.New("out of date")

// run syncs the page under root, or with check only compares it.
func run(root string, check bool) error {
	p := filepath.Join(root, filepath.FromSlash(pagePath))
	page, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("read %s: %w", pagePath, err)
	}
	synced, err := sync(page, question.Default)
	if err != nil {
		return err
	}
	if bytes.Equal(synced, page) {
		fmt.Printf("asking-sync: %s already carries the asking rules; nothing to do.\n", pagePath)
		return nil
	}
	if check {
		return errOutOfDate
	}
	if err := fsutil.WriteFileAtomicPreserveMode(p, synced); err != nil {
		return fmt.Errorf("write %s: %w", pagePath, err)
	}
	fmt.Printf("asking-sync: wrote the asking rules into %s\n", pagePath)
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
