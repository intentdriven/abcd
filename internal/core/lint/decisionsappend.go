package lint

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The decisions-append gate: the deterministic append-only check over
// .abcd/work/DECISIONS.md (iss-2608271804494867), ported from shell into the
// lint engine (iss-2608291814575169). `record-lint decisions-append <base>
// <head>` is its front door; this file holds the rules and returns findings,
// and never writes to a terminal.
//
// The ledger's own header declares "Append-only, one line per decision, newest
// last". The file already carries backwards date steps among its dated bullets
// — historical, committed, and NOT to be repaired: reordering a committed
// append-only log is the one operation append-only forbids. So the gate does
// not check dates and does not read the existing order. A back-dated entry
// appended at the tail is honest (the date names when the decision was taken,
// the append names when it was recorded); what is never legitimate is writing
// INTO the log that is already committed.
//
// THE CONTRACT, in four rules. The first three are scoped BELOW THE HEADER
// REGION; the fourth is about the file's bytes and is scoped to all of it.
//
//	DA001  POSITION. In a single-parent commit, a line ADDED to the ledger must
//	       land after the last line the parent already had. An addition with
//	       surviving pre-existing content below it is an insertion, and is
//	       refused.
//
//	DA002  PRESERVATION. In ANY commit, against EVERY parent, the diff must hold
//	       no REMOVED line below the header region. Position alone is not
//	       append-only: DA001 fires only while pre-existing content SURVIVES
//	       below the addition, so a hunk reaching end-of-file rewrites or erases
//	       committed decisions with nothing beneath it to trip the position rule
//	       — the tail-reaching rewrite, and its two-commit cousin, truncate the
//	       log then restore a forged history, each half positionally innocent.
//	       DA002 is stricter than position, deliberately and in the direction the
//	       ledger's header asks for: correcting an entry means APPENDING a new
//	       dated entry. Amending the file's last line in place is a refusal too.
//
//	DA003  MERGE AUTHORS NOTHING. A merge commit's ledger may hold a line no more
//	       times than the MERGE BASE held it plus what each parent ADDED over it:
//
//	         bound(t) = base(t) + Σ_parents max(0, parent(t) − base(t))
//
//	       Merges cannot be skipped: a hand-forged conflict resolution is authored
//	       content in no single-parent commit. Nor can DA001 be applied to them:
//	       the ledger is `merge=union`, and the union driver concatenates each
//	       conflicting region independently, so a routine merge carries each
//	       side's lines above the other's and extends NEITHER parent's tail. What
//	       a union merge never does is invent a line, or emit one more times than
//	       the base held it plus what the sides added. Set membership was blind to
//	       multiplicity (copy a decision to the head of the log); summing the
//	       parents double-budgeted common history (a complete reordered rendition
//	       of the whole log spliced above the real one passed, +2,150/−0, against
//	       the live ledger). Subtracting what the base held is the correction, and
//	       two branches independently recording the same text still pass
//	       (0 + 1 + 1). With DA002 against every parent, a merge can neither drop,
//	       fabricate nor multiply a decision.
//
//	DA004  THE LEDGER IS TEXT. No commit may introduce a NUL byte. One NUL makes
//	       git call the file binary, which reduces the diff to "Binary files …
//	       differ" with no hunks, and every rule above reads hunks. The diffs are
//	       read with --text, which is the load-bearing fix (it also covers a
//	       `-diff` attribute, which has no NUL to find); DA004 is depth behind it,
//	       because a NUL breaks grep, the site renderer and any editor too. It
//	       fires only on the commit that INTRODUCES one: a NUL already in a parent
//	       is history, and demanding its removal would deadlock against DA002.
//
// Boundaries:
//
//   - The HEADER REGION — everything above the first `- <YYYY-MM-DD>` bullet in
//     the PARENT's copy — is exempt from the first three rules: it is prose about
//     the ledger, and must stay editable, not least to describe this gate. The
//     exemption is anchored on the PARENT's first bullet, so the commit that uses
//     it cannot widen it. It is withdrawn from any hunk that adds a LIST-SHAPED
//     line, because the header's last line and the log's first entry share one
//     insertion point and only the added line's shape tells a paragraph from a
//     smuggled entry. The shape test (decisionsItemRe) is deliberately wider than
//     the canonical bullet, because a forged entry need not be well formed to be
//     read as one, and matching only the canonical shape makes malformity the
//     bypass.
//   - The LAST ENTRY'S interior is not exempt: a continuation appended after the
//     file's last line already extends it, so writing above that line buys
//     nothing and is indistinguishable from any other insertion.
//
// REDACTION AND GRADUATION — the two legitimate operations this gate refuses.
// Redacting a leaked line and the planned graduation to per-file decision
// records are both deletions. Neither gets an escape hatch: a bypass trailer,
// an allow-marker or a skip keyed on a commit message is a lock whose key is
// written on the door, because the commit that wants the bypass is the commit
// that writes it. Both are deliberate gate-edit-and-review changes: the pull
// request that redacts or graduates ALSO adjusts or retires this gate in the same
// diff, and a human reviews the two halves together.
//
// Every untrusted string a finding carries — a commit subject, a ledger line —
// goes through termsafe.Sanitize: the content under check is exactly the content
// an attacker controls, and a raw ESC or CR in it can overprint or recolour the
// verdict the reader relies on.

// DecisionsLedger is the repo-relative path of the append-only decision log.
const DecisionsLedger = ".abcd/work/DECISIONS.md"

// decisionsMaxBytes caps any one git answer the gate reads: a ledger copy, a
// diff, a rev-list. The live ledger is around half a megabyte; an answer past
// the cap is refused rather than truncated, because a truncated ledger is a
// different ledger.
const decisionsMaxBytes = 64 << 20

var (
	// decisionsBulletRe is a dated bullet: it opens an entry, and the first one
	// ends the header region.
	decisionsBulletRe = regexp.MustCompile(`^- [0-9]{4}-[0-9]{2}-[0-9]{2}`)

	// decisionsItemRe is a LIST-SHAPED line: what a forged entry looks like when
	// it is not well formed. Declared ONCE and used by both rules that test the
	// shape — DA001's header exemption and DA003's in-scope test — because a
	// second copy is how "malformity is the bypass" comes back: the copies drift,
	// and the shape one rule refuses becomes the shape the other lets through.
	// A bullet with two spaces, `*`/`+`/`>`, an ordered item, a bolded date, a
	// dash-led line (ASCII, en or em), or a bare date-led line.
	decisionsItemRe = regexp.MustCompile(`^[[:space:]]*(?:[-*+>]|[0-9]+[.)]|[0-9]{4}-[0-9]{2}-[0-9]{2}|—|–)`)
)

// DecisionsAppendReport is one run of the gate over a range.
type DecisionsAppendReport struct {
	// Base and Head are the refs the caller named, for the report's wording.
	Base, Head string
	// Skipped is non-empty when there was no usable base for this event (an empty
	// value, or the all-zeroes placeholder): nothing was checked, and the front
	// door says so rather than reading the empty range as a clean one.
	Skipped string
	// Checked is how many commits base..head held, merges included.
	Checked int
	// Findings are the violations, every one a blocker.
	Findings []Finding
}

// CheckDecisionsAppend runs DA001–DA004 over every commit in base..head of the
// repository at root, merges included.
//
// base is resolved once, through gitutil.ResolveRangeBase: an empty value or
// the null object name is a skip (Report.Skipped), and anything that names no
// commit is an error. Every error is "the gate could not answer" — a repository
// git cannot read, a shallow checkout, a ref that resolves to nothing or is
// option-shaped, a git probe that failed, a merge whose bound cannot be
// anchored — and a front door reports it as its own polarity (exit 2), never as
// a pass and never as a rule violation.
func CheckDecisionsAppend(root, base, head string) (DecisionsAppendReport, error) {
	rep := DecisionsAppendReport{Base: base, Head: head}

	// Every path is resolved from the repository root: the ledger pathspec is
	// matched against git's working directory, so from a subdirectory the diff
	// would match nothing and the gate would pass having scanned nothing.
	top, err := gitutil.Run(root, "rev-parse", "--show-toplevel")
	if err != nil || top == "" {
		return rep, fmt.Errorf("not a readable git repository — refusing rather than reporting a vacuous pass: %v", err)
	}

	baseSHA, usable, err := gitutil.ResolveRangeBase(top, base)
	if err != nil {
		return rep, fmt.Errorf("the base: %w", err)
	}
	if !usable {
		rep.Skipped = fmt.Sprintf("no usable base ref for this event (%q) — the range check is skipped", termsafe.Sanitize(base))
		return rep, nil
	}
	headSHA, err := gitutil.ResolveCommit(top, head)
	if err != nil {
		return rep, fmt.Errorf("the head: %w", err)
	}
	if err := gitutil.RequireFullHistory(top); err != nil {
		return rep, fmt.Errorf("the rules compare each commit against its parents: %w", err)
	}

	// The empty tree, for diffing a root commit. `hash-object` without -w
	// computes the id and writes nothing; it follows the repository's hash.
	emptyTree, err := gitutil.Run(top, "hash-object", "-t", "tree", "/dev/null")
	if err != nil || emptyTree == "" {
		return rep, fmt.Errorf("git hash-object failed — refusing rather than reporting a vacuous pass: %v", err)
	}

	// Merges are NOT excluded: a hand-forged conflict resolution is authored
	// content that appears in no single-parent commit.
	out, err := gitutil.RunCapped(top, decisionsMaxBytes, "rev-list", baseSHA+".."+headSHA)
	if err != nil {
		return rep, fmt.Errorf("git rev-list failed for %s..%s — refusing rather than reporting a vacuous pass: %w", base, head, err)
	}
	g := &decisionsGate{root: top, emptyTree: emptyTree}
	for _, sha := range strings.Fields(out) {
		if err := g.checkCommit(sha); err != nil {
			return rep, err
		}
		rep.Checked++
	}
	rep.Findings = g.findings
	return rep, nil
}

type decisionsGate struct {
	root      string
	emptyTree string
	findings  []Finding
}

// ledgerCopy is the ledger as one tree-ish holds it.
type ledgerCopy struct {
	present bool
	data    []byte
	lines   []string
	// headerEnd is the line above the first dated bullet; with no bullet at all
	// the whole file is header, which makes the very first entry an append.
	headerEnd int
}

// ledgerLines splits a blob into lines the way awk counts records: a trailing
// newline ends the last line rather than opening an empty one, and an empty
// blob has no lines. A count taken any other way refuses the commit that seeds
// an empty ledger, or miscounts a file ending in blank lines.
func ledgerLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	s := string(data)
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func (g *decisionsGate) read(ref string) (ledgerCopy, error) {
	if ref == g.emptyTree {
		return ledgerCopy{}, nil
	}
	// ls-tree tells "no ledger here" apart from "git could not answer", which a
	// bare existence probe folds together — and a swallowed git error reads
	// exactly like an untouched ledger.
	entry, err := gitutil.RunCapped(g.root, 1<<16, "ls-tree", "--full-tree", "-z", ref, "--", DecisionsLedger)
	if err != nil {
		return ledgerCopy{}, fmt.Errorf("reading %s at %s failed — refusing rather than reporting a vacuous pass: %w", DecisionsLedger, short(ref), err)
	}
	entry = strings.TrimRight(entry, "\x00")
	if entry == "" {
		return ledgerCopy{}, nil
	}
	// "<mode> SP <type> SP <object> TAB <path>"
	meta, _, _ := strings.Cut(entry, "\t")
	f := strings.Fields(meta)
	if len(f) != 3 || f[1] != "blob" {
		return ledgerCopy{}, fmt.Errorf("%s at %s is not a file (%q) — refusing rather than guessing what to read", DecisionsLedger, short(ref), meta)
	}
	data, err := gitutil.RunCappedBytes(g.root, decisionsMaxBytes, "cat-file", "blob", f[2])
	if err != nil {
		return ledgerCopy{}, fmt.Errorf("reading %s at %s failed — refusing rather than reporting a vacuous pass: %w", DecisionsLedger, short(ref), err)
	}
	c := ledgerCopy{present: true, data: data, lines: ledgerLines(data)}
	c.headerEnd = len(c.lines)
	for i, l := range c.lines {
		if decisionsBulletRe.MatchString(l) {
			c.headerEnd = i
			break
		}
	}
	return c, nil
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

func (g *decisionsGate) fail(rule string, line int, msg string) {
	g.findings = append(g.findings, Finding{
		File: DecisionsLedger, Line: line, RuleID: rule, Severity: severityBlocker, Message: msg,
	})
}

func (g *decisionsGate) checkCommit(sha string) error {
	line, err := gitutil.Run(g.root, "rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return fmt.Errorf("git rev-list --parents failed for %s — refusing rather than reporting a vacuous pass: %w", short(sha), err)
	}
	// "<sha> <parent1> <parent2> …" — drop the commit's own sha, then
	// DE-DUPLICATE. A commit object may list one parent twice: commit-tree
	// refuses to write one, but `hash-object -t commit -w` validates nothing, and
	// the object is fsck-clean and read back with the parent repeated. DA003
	// credits each parent with what it ADDED, so a repeated parent is a forged
	// allowance. De-duplicating HERE fixes every consumer at once: the
	// merge-versus-single-parent decision (so [X, X] is judged as the
	// single-parent commit it is, DA001 included), the DA002 loop, and DA003.
	var parents []string
	seen := map[string]bool{}
	for _, p := range strings.Fields(line)[1:] {
		if !seen[p] {
			seen[p] = true
			parents = append(parents, p)
		}
	}

	subject, err := gitutil.Run(g.root, "log", "-1", "--format=%s", sha)
	if err != nil {
		return fmt.Errorf("reading the subject of %s failed: %w", short(sha), err)
	}
	subject = termsafe.Sanitize(subject)

	mine, err := g.read(sha)
	if err != nil {
		return err
	}
	parentCopies := make([]ledgerCopy, len(parents))
	for i, p := range parents {
		if parentCopies[i], err = g.read(p); err != nil {
			return err
		}
	}

	// DA004 first, on every shape of commit: a NUL is what makes git call the
	// ledger binary, and a binary ledger is what makes the rules below see nothing.
	g.checkNUL(sha, subject, mine, parentCopies)

	if len(parents) <= 1 {
		// A root commit has no parent; it is diffed against the empty tree, so the
		// whole file reads as an append into an empty ledger.
		parent, pc := g.emptyTree, ledgerCopy{}
		if len(parents) == 1 {
			parent, pc = parents[0], parentCopies[0]
		}
		raws, err := g.analyse(sha, parent, pc)
		if err != nil {
			return err
		}
		g.report(sha, parent, subject, len(pc.lines), raws, false)
		return nil
	}

	// A merge. DA002 against every parent — no side's decisions may disappear
	// through a conflict resolution. DA001 is NOT applied (a union merge extends
	// neither parent's tail); DA003 carries the other half.
	for i, p := range parents {
		raws, err := g.analyse(sha, p, parentCopies[i])
		if err != nil {
			return err
		}
		g.report(sha, p, subject, len(parentCopies[i].lines), raws, true)
	}
	return g.checkMergeAuthored(sha, subject, mine, parents, parentCopies)
}

// rawFinding is one analysed hunk verdict before it is worded.
type rawFinding struct {
	rule   string
	at     int // DA001: the old line the addition sits after; DA002: the first removed old line
	count  int
	sample string
}

// analyse reads the ledger diff between parent and commit and returns both
// DA001 and DA002 verdicts; the caller decides which apply, because a merge's
// parents are judged by DA002 individually and by DA001 not at all.
func (g *decisionsGate) analyse(commit, parent string, pc ledgerCopy) ([]rawFinding, error) {
	// --unified=0 so each hunk's own line numbers locate the change exactly;
	// --no-renames so a rename into the path is a plain add rather than a
	// similarity header with no hunks.
	//
	// --text is LOAD-BEARING. Without it git decides for itself whether the
	// ledger is text, and it decides "binary" on a single NUL anywhere in the file
	// or on a `-diff` attribute — either reduces the diff to no hunks at all, and
	// the gate passes, permanently.
	//
	// --no-ext-diff and --no-textconv keep the answer git's own: an external diff
	// driver or a textconv filter configured for the path would otherwise decide
	// what the gate sees. --diff-algorithm=myers pins the hunk shape the rules and
	// their cases were written against, whatever a repository's config prefers.
	//
	// --inter-hunk-context=0 and --indent-heuristic pin the two other hunk-shaping
	// settings a repository's config supplies (diff.interHunkContext,
	// diff.indentHeuristic), each at git's own default. The old-line arithmetic
	// below assumes every hunk holds only removed and added lines: a non-zero
	// inter-hunk context folds neighbouring hunks into one with context lines
	// between them, which merges two findings and miscounts both spans, and an
	// indent heuristic switched off slides an ambiguous hunk to another line.
	// diff.context is overridden by --unified=0; diff.relative cannot narrow a
	// diff run from the top level; the prefix settings (diff.noprefix,
	// diff.mnemonicPrefix, diff.srcPrefix, diff.dstPrefix) change only the
	// file headers, which the reader never parses; and the colour settings are
	// switched off wholesale by --no-color.
	out, err := gitutil.RunCappedBytes(g.root, decisionsMaxBytes, "diff", "--unified=0", "--no-color",
		"--no-ext-diff", "--no-textconv", "--no-renames", "--text", "--diff-algorithm=myers",
		"--inter-hunk-context=0", "--indent-heuristic",
		parent, commit, "--", DecisionsLedger)
	if err != nil {
		return nil, fmt.Errorf("git diff failed for %s against %s — refusing rather than reporting a vacuous pass: %w", short(commit), short(parent), err)
	}
	if len(out) == 0 {
		return nil, nil
	}

	oldTotal, headerEnd := len(pc.lines), pc.headerEnd
	var res []rawFinding
	var (
		pending             bool
		a, b                int
		adds                int
		addSample           string
		hasItem             bool
		seenDels, delsBelow int
		delFirstLine        int
		delSample           string
	)
	// For a hunk `@@ -a,b +c,d @@` under --unified=0 the hunk removes old lines
	// a .. a+b-1 and places its additions among them. With b == 0 nothing is
	// removed and the addition sits after old line a, so the next surviving old
	// line is a + 1; with b > 0 it is a + b. An addition is an INSERTION exactly
	// when that next surviving old line exists.
	flush := func() {
		if !pending {
			return
		}
		if adds > 0 {
			firstAfter, lastOld := a+b, a+b-1
			if b == 0 {
				firstAfter, lastOld = a+1, a
			}
			// Header prose is exempt — unless the addition is list-shaped, and so
			// an entry smuggled in at the boundary the header shares with the log.
			if !(lastOld <= headerEnd && !hasItem) && firstAfter <= oldTotal {
				res = append(res, rawFinding{"DA001", a, adds, addSample})
			}
		}
		// Every removed line below the header region is a committed decision being
		// erased or rewritten. Reported once per hunk, naming the first.
		if delsBelow > 0 {
			res = append(res, rawFinding{"DA002", delFirstLine, delsBelow, delSample})
		}
	}
	for _, l := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(l, "@@ "):
			flush()
			a, b = parseHunkOld(l)
			pending = true
			adds, addSample, hasItem = 0, "", false
			seenDels, delsBelow, delFirstLine, delSample = 0, 0, 0, ""
		case pending && strings.HasPrefix(l, "+"):
			adds++
			content := l[1:]
			if addSample == "" {
				addSample = content
			}
			if decisionsItemRe.MatchString(content) {
				hasItem = true
			}
		case pending && strings.HasPrefix(l, "-"):
			// The k-th removed line of the hunk is old line a + k - 1.
			oldno := a + seenDels
			seenDels++
			if oldno > headerEnd {
				delsBelow++
				if delFirstLine == 0 {
					delFirstLine, delSample = oldno, l[1:]
				}
			}
		}
	}
	flush()
	return res, nil
}

// parseHunkOld reads the old-side "a,b" of a hunk header; a lone "a" means b = 1.
func parseHunkOld(header string) (int, int) {
	f := strings.Fields(header)
	if len(f) < 2 {
		return 0, 0
	}
	old := strings.TrimPrefix(f[1], "-")
	if x, y, ok := strings.Cut(old, ","); ok {
		a, _ := strconv.Atoi(x)
		b, _ := strconv.Atoi(y)
		return a, b
	}
	a, _ := strconv.Atoi(old)
	return a, 1
}

// report words the raw verdicts that apply: DA001 and DA002 for a
// single-parent commit, DA002 alone for one parent of a merge.
func (g *decisionsGate) report(commit, parent, subject string, oldTotal int, raws []rawFinding, mergeParent bool) {
	for _, r := range raws {
		sample := termsafe.Sanitize(r.sample)
		switch r.rule {
		case "DA001":
			if mergeParent {
				continue
			}
			g.fail("DA001", r.at+1, fmt.Sprintf("commit %s (%s) inserts %d line(s) into %s above line %d of its parent's copy, which has %d lines. The ledger is append-only, newest last: append the entry at the end of the file instead. First inserted line: %s",
				short(commit), subject, r.count, DecisionsLedger, r.at+1, oldTotal, sample))
		case "DA002":
			g.fail("DA002", r.at, fmt.Sprintf("commit %s (%s) removes %d committed line(s) from %s, starting at line %d of %s's copy. The ledger is append-only: correct an entry by appending a new dated one, never by editing or deleting the old one. First removed line: %s",
				short(commit), subject, r.count, DecisionsLedger, r.at, short(parent), sample))
		}
	}
}

// checkNUL implements DA004. It fires only on the commit that INTRODUCES a NUL:
// a parent already carrying one makes this commit an inheritor, and demanding
// its removal would deadlock against DA002.
func (g *decisionsGate) checkNUL(commit, subject string, mine ledgerCopy, parents []ledgerCopy) {
	at := bytes.IndexByte(mine.data, 0)
	if !mine.present || at < 0 {
		return
	}
	for _, p := range parents {
		if p.present && bytes.IndexByte(p.data, 0) >= 0 {
			return
		}
	}
	line := bytes.Count(mine.data[:at], []byte("\n")) + 1
	g.fail("DA004", line, fmt.Sprintf("commit %s (%s) introduces a NUL byte into %s. The ledger is prose; a NUL is never honest content, and it makes git treat the file as binary — every hunk-reading rule here would see nothing at all.",
		short(commit), subject, DecisionsLedger))
}

// countLines is a multiset of a copy's lines.
func countLines(lines []string) map[string]int {
	m := make(map[string]int, len(lines))
	for _, l := range lines {
		m[l]++
	}
	return m
}

// checkMergeAuthored implements DA003: for every line, the merge's ledger may
// hold it no more times than the BASE held it plus what each parent ADDED.
//
// Counting rather than a diff, because the union driver's interleaving defeats
// every position-based comparison while leaving this invariant intact. Written
// as "base plus each parent's additions" rather than "Σ parents − base" because
// only this form generalises past two parents.
//
// count_base(t) IS THE MAXIMUM OVER THE PAIRWISE MERGE-BASES of the parents
// that carry the ledger — never `git merge-base --octopus`. The octopus base is
// chosen by data the merge author controls: ONE extra parent older than the
// ledger (the repository's own initial commit, or an unrelated orphan root that
// makes the octopus base fail to resolve at all) drags it below the point the
// ledger existed, count_base collapses to 0, and the full-log splice passes. Max
// over pairwise bases is monotone the other way: an extra parent only adds
// pairs, a maximum over a superset only rises, so a bolted-on parent can only
// tighten the bound against its author.
//
// Degradation is LOUD. When two or more parents carry the ledger and no pairwise
// base carries it, this returns an error rather than falling back to a weaker
// bound: the attack never has to beat the bound, only to reach the branch where
// it stops applying.
//
// Residual, accepted: if the union driver ever pulls a line both sides SHARE
// into a conflicting region, it emits it once per side and the merge holds one
// more copy than this bound allows. Two tail appends do not produce that shape,
// and the replay over this repository's history reports no such merge.
func (g *decisionsGate) checkMergeAuthored(commit, subject string, mine ledgerCopy, parents []string, copies []ledgerCopy) error {
	// A merge that does not carry the ledger cannot have authored anything in it.
	if !mine.present {
		return nil
	}

	// Only the parents that CARRY the ledger take part: a ledgerless parent is
	// precisely what drags a common ancestor below the point the ledger existed.
	// The parent list is already de-duplicated; this function's result is a
	// per-parent SUM, so the property is asserted again where it is load-bearing.
	var lparents []string
	var lcopies []ledgerCopy
	seen := map[string]bool{}
	for i, p := range parents {
		if seen[p] || !copies[i].present {
			continue
		}
		seen[p] = true
		lparents = append(lparents, p)
		lcopies = append(lcopies, copies[i])
	}

	base := map[string]int{}
	basesFound, ledgerBases := 0, 0
	for i := range lparents {
		for j := range lparents {
			a, b := lparents[i], lparents[j]
			if !(a < b) { // each unordered pair once
				continue
			}
			// A pair with no common ancestor is not a fault on its own; the
			// aggregate verdict below decides.
			pb, err := gitutil.Run(g.root, "merge-base", a, b)
			if err != nil {
				continue
			}
			pb, _, _ = strings.Cut(pb, "\n")
			if pb = strings.TrimSpace(pb); pb == "" {
				continue
			}
			basesFound++
			bc, err := g.read(pb)
			if err != nil {
				return err
			}
			if !bc.present {
				continue
			}
			ledgerBases++
			for t, c := range countLines(bc.lines) {
				if c > base[t] {
					base[t] = c
				}
			}
		}
	}

	if len(lparents) >= 2 && ledgerBases == 0 {
		why := "no pairwise merge-base carries the ledger"
		if basesFound == 0 {
			why = "no pairwise merge-base resolves between them"
		}
		return fmt.Errorf("merge commit %s has %d parents carrying %s, but %s. DA003's bound is anchored on that base; without it the rule would silently weaken to one a merge can satisfy by duplicating all of common history. Refusing rather than reporting a vacuous pass. "+
			"This is reachable honestly — two histories that each seeded their own %s (two repositories joined with --allow-unrelated-histories, or the ledger created independently on two branches) share no ledger history for a bound to rest on. "+
			"Either seed the ledger once on the default branch and branch from there, so every side descends from one ledger history; or, if the join itself is the intent, treat it as the deliberate gate-edit-and-review change the gate's contract describes for redaction and graduation — the pull request that performs the join also adjusts or retires this gate in the same diff, and a human reviews both halves together",
			short(commit), len(lparents), DecisionsLedger, why, DecisionsLedger)
	}

	// Each parent contributes only what it ADDED over the base.
	allow := map[string]int{}
	for _, pc := range lcopies {
		for t, c := range countLines(pc.lines) {
			if d := c - base[t]; d > 0 {
				allow[t] += d
			}
		}
	}

	// In scope: anything below the merge's own header, plus any list-shaped line
	// wherever it sits — a forged entry does not become header prose by being
	// planted above the first well-formed bullet.
	scoped := map[string]int{}
	firstAt := map[string]int{}
	for i, t := range mine.lines {
		if i+1 > mine.headerEnd || decisionsItemRe.MatchString(t) {
			scoped[t]++
			if scoped[t] == 1 {
				firstAt[t] = i + 1
			}
		}
	}
	type over struct {
		line, got, bound int
		text             string
	}
	var authored []over
	for t, got := range scoped {
		if bound := base[t] + allow[t]; got > bound {
			authored = append(authored, over{firstAt[t], got, bound, t})
		}
	}
	if len(authored) == 0 {
		return nil
	}
	sort.Slice(authored, func(i, j int) bool { return authored[i].line < authored[j].line })
	first := authored[0]
	g.fail("DA003", first.line, fmt.Sprintf("merge commit %s (%s) holds %d distinct line(s) in %s more times than the merge base held them plus what the parents added — a union merge concatenates both sides and neither invents nor duplicates, so this content was authored in the conflict resolution. First such line appears %d time(s) in the merge against a bound of %d (line %d of the merge's copy): %s",
		short(commit), subject, len(authored), DecisionsLedger, first.got, first.bound, first.line, termsafe.Sanitize(first.text)))
	return nil
}
