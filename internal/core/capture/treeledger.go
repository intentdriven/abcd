package capture

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// treeledger.go — the ledger as a commit's tree holds it (spc-2610161044461448
// step 1, iss-2610160957404827). A drain plans from origin's default branch,
// not from the checkout it runs in, which may be behind: its caller reads the
// status folders out of that commit and hands them here, and the plan judges
// each record exactly as the checkout's own reader would (issuerecord.JudgeText,
// the one verdict).

// LedgerFile is one file in a status folder of a commit's ledger.
type LedgerFile struct {
	// Folder is the status folder the file sits in: open, resolved or wontfix.
	Folder string
	// Path is the file's repository-relative path in the commit.
	Path string
	// Text is the file's content. Only a record in open/ is read; a record in
	// a terminal folder is named, not read.
	Text string
	// Unread is why the content could not be read (a link, a file past the
	// cap), "" when it was or was not asked for.
	Unread string
}

// TreeLedger is the issue ledger as one commit holds it.
type TreeLedger struct {
	// Source names the commit as a reader names it: "origin/main at 0123abcd4567".
	Source string
	Files  []LedgerFile
}

// Terminal is the ids the ledger holds in a terminal folder, resolved or
// wontfix, sorted. A file in any other folder, open/ or one that is no status,
// finishes nothing.
func (l *TreeLedger) Terminal() []string {
	var out []string
	for _, f := range l.Files {
		if f.Folder != string(StateResolved) && f.Folder != string(StateWontfix) {
			continue
		}
		if id := recordIDOfFilename(path.Base(f.Path)); id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// listTree reads the open issues of l as List reads the checkout's: one
// status per id across the folders, each open record judged by the ledger's
// reader, the blocked_by projection over open/, and the priority order.
func listTree(l *TreeLedger) ([]Issue, []SkipRecord, error) {
	claims := map[string][]string{}
	var ids []string
	for _, f := range l.Files {
		if !slices.Contains(statusDirs, State(f.Folder)) {
			continue
		}
		id := recordIDOfFilename(path.Base(f.Path))
		if id == "" {
			continue
		}
		if len(claims[id]) == 0 {
			ids = append(ids, id)
		}
		claims[id] = append(claims[id], f.Path)
	}
	sort.Strings(ids)
	var conflicts []string
	for _, id := range ids {
		if paths := claims[id]; len(paths) > 1 {
			conflicts = append(conflicts, fmt.Sprintf("%s is claimed by %s", id, strings.Join(paths, " and ")))
		}
	}
	if len(conflicts) > 0 {
		return nil, nil, fmt.Errorf("%w at %s: %s; move or remove one of each pair there so the ledger says which status the record is in",
			ErrDuplicateIssueID, termsafe.Sanitize(l.Source), termsafe.Sanitize(strings.Join(conflicts, "; and ")))
	}
	var issues []Issue
	var skipped []SkipRecord
	for _, f := range l.Files {
		if f.Folder != string(StateOpen) {
			continue
		}
		var readErr error
		if f.Unread != "" {
			readErr = errors.New(f.Unread)
		}
		fm, body, refusal, claims := issuerecord.JudgeText(f.Path, f.Folder, f.Text, readErr)
		if !claims {
			continue
		}
		if refusal != nil {
			skipped = append(skipped, SkipRecord{Path: termsafe.Sanitize(f.Path), Layer: refusal.Layer, Error: termsafe.Sanitize(refusal.Err.Error())})
			continue
		}
		issues = append(issues, issueFromFrontmatter(fm, StateOpen, f.Path, body))
	}
	sortIssues(issues)
	prioritise(issues, openBlockingIDs(issues, skipped))
	return issues, skipped, nil
}

// RuleOnlyPlan is the plan of a drain move that read no ledger: the rule, its
// record, the floors it loosens and its order, with no disposition.
func RuleOnlyPlan(rule drainrule.Rule) DrainPlan {
	return DrainPlan{Record: rule.Record, Rule: rule, Loosened: rule.Loosened, Order: drainOrder(rule),
		Dispositions: []DrainVerdict{}, Counts: map[DrainOutcome]int{}}
}

// OpenIssue is the open record the plan judged id from: the checkout's, or
// the commit's when the plan was read from a tree.
func (p DrainPlan) OpenIssue(id string) (Issue, bool) {
	iss, ok := p.open[id]
	return iss, ok
}
