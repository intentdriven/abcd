// Command command-facts-sync writes the derived copies of the command facts
// (itd-2610301346085246, spc-2610301431075964 step 3): one JSON file per
// top-level verb under docs/reference/cli/commands/, and the two generated
// blocks of every command page under commands/, the ask-the-binary block
// straight after the page's H1 heading and the command-facts block at its end.
// Each is rendered from cli.CommandFacts, the one source of the facts, by the
// functions in internal/surface/cli/commandfacts.go, which the drift tests
// call too. The facts are changed in the command tree, never in a copy.
//
// It mirrors asking-sync, run by hand via `make command-facts-sync`. Nothing
// in CI invokes it: the drift is gated by the tests in
// internal/surface/cli/commandfacts_drift_test.go, under precheck and
// preflight, which fail when a committed copy differs from the rendering.
//
// It never creates a page: which verbs have one is the menu test's rule
// (TestCommandPagesMatchTheMenu). A visible verb with no page gets its JSON
// file alone. A JSON file left in the directory for a verb the tree no longer
// has is removed.
//
// Exit codes: 0 nothing to do (or, with no -check, the copies were written);
// 1 a fault; 3 with -check, a copy is out of date and nothing was written.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/surface/cli"
)

func main() {
	check := flag.Bool("check", false, "report drift and write nothing; exit 3 when a copy is out of date")
	rootPath := flag.String("root", "", "repo root to sync (default: git toplevel, or cwd)")
	flag.Parse()

	root := *rootPath
	if root == "" {
		root = resolveRoot()
	}
	if err := run(root, cli.CommandFacts(), *check); err != nil {
		if errors.Is(err, errOutOfDate) {
			fmt.Printf("command-facts-sync: %s\n", err)
			fmt.Fprintln(os.Stderr, "command-facts-sync: run `make command-facts-sync` to write the command facts into their copies.")
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, "command-facts-sync:", scrubRoot(err.Error(), root))
		os.Exit(1)
	}
}

// errOutOfDate is -check finding a committed copy that differs from the
// rendering; the error wrapping it names each file.
var errOutOfDate = errors.New("out of date with the command facts")

// change is one file the sync writes, or removes when data is nil.
type change struct {
	path string
	data []byte
}

// plan renders every copy under root from facts and returns the changes that
// would bring the committed files in line, sorted by path. Every page is read
// and rendered before anything is written, so a fault in one leaves all of
// them as they were.
func plan(root string, facts []surface.Facts) ([]change, error) {
	files, err := cli.CommandFactsFiles(facts)
	if err != nil {
		return nil, err
	}
	var changes []change
	for rel, want := range files {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		if err != nil || !bytes.Equal(got, want) {
			changes = append(changes, change{rel, want})
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(cli.CommandFactsDir)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", cli.CommandFactsDir, err)
	}
	for _, e := range entries {
		rel := cli.CommandFactsDir + "/" + e.Name()
		if _, kept := files[rel]; !kept && !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			changes = append(changes, change{rel, nil})
		}
	}

	pages, err := os.ReadDir(filepath.Join(root, cli.CommandPagesDir))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cli.CommandPagesDir, err)
	}
	for _, e := range pages {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if !ok || e.IsDir() {
			continue
		}
		rel := cli.CommandPagesDir + "/" + e.Name()
		page, err := os.ReadFile(filepath.Join(root, cli.CommandPagesDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		synced, err := cli.SyncCommandPage(page, name, facts)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(synced, page) {
			changes = append(changes, change{rel, synced})
		}
	}
	sort.Slice(changes, func(a, b int) bool { return changes[a].path < changes[b].path })
	return changes, nil
}

// run syncs every copy under root from facts, or with check only compares
// them.
func run(root string, facts []surface.Facts, check bool) error {
	changes, err := plan(root, facts)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		fmt.Println("command-facts-sync: every copy already carries the command facts; nothing to do.")
		return nil
	}
	if check {
		names := make([]string, len(changes))
		for i, c := range changes {
			names[i] = c.path
		}
		return fmt.Errorf("%s: %w", strings.Join(names, ", "), errOutOfDate)
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(cli.CommandFactsDir)), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", cli.CommandFactsDir, err)
	}
	for _, c := range changes {
		p := filepath.Join(root, filepath.FromSlash(c.path))
		if c.data == nil {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("remove %s: %w", c.path, err)
			}
			fmt.Printf("command-facts-sync: removed %s, whose verb the tree no longer has\n", c.path)
			continue
		}
		if err := fsutil.WriteFileAtomicPreserveMode(p, c.data); err != nil {
			return fmt.Errorf("write %s: %w", c.path, err)
		}
		fmt.Printf("command-facts-sync: wrote %s\n", c.path)
	}
	return nil
}

// resolveRoot returns the git toplevel, falling back to the working directory,
// as asking-sync does.
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
// asking-sync does: the message names the repo-relative file.
func scrubRoot(msg, root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return msg
	}
	return strings.ReplaceAll(strings.ReplaceAll(msg, abs+string(filepath.Separator), ""), abs, ".")
}
