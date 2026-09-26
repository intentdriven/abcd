package lifeboat

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// graveyard_archaeology.go — Layer 1 of the graveyard (M4, adr-35): the Tier-0,
// git-only, deterministic, EVIDENCE-ONLY dig. It reads ONLY through the cached,
// capped SourceContext git surface (ctx.Git/ctx.GitLines/ctx.ReadFile) and never
// writes. Every human string is run through gvText; every id is built by the
// namespaced helpers in graveyard.go; findings are grouped in signalRank order so
// the assembled file is byte-identical across re-plans of an unchanged repo. No
// wall-clock time enters any output — commit dates used for branch ranking are
// stable repository data, not the clock.

// buildArchaeology runs every Tier-0 signal over the source and returns the
// deterministic dig. Findings are grouped in signalRank order (reverts, unmerged
// branches, deleted paths, removed dependencies, wholesale rewrites); each
// signal's own natural order is preserved within its group. A non-git source, an
// empty repo, or a repo history recording nothing abandoned yields an empty (but
// non-nil) Findings slice, so the marshalled file always carries "findings": [],
// never null.
func buildArchaeology(ctx *SourceContext) Archaeology {
	var fs []Finding
	fs = append(fs, gvReverts(ctx)...)
	fs = append(fs, gvUnmergedBranches(ctx)...)
	fs = append(fs, gvDeletedPaths(ctx)...)
	fs = append(fs, gvRemovedDependencies(ctx)...)
	fs = append(fs, gvRewrites(ctx)...)
	if fs == nil {
		fs = []Finding{}
	}
	return Archaeology{SchemaVersion: GraveyardSchemaVersion, Findings: fs}
}

// isRevertSubject recognises a reverting commit's subject. `git revert` writes a
// `Revert "..."` subject; some tools write `Revert:`. This is the single predicate
// both the graveyard signal and the gitReverts coverage adapter share.
func isRevertSubject(s string) bool {
	return strings.HasPrefix(s, "Revert \"") || strings.HasPrefix(s, "Revert:")
}

// gvReverts reports every reverted commit in HEAD's history — a deliberate,
// explicit abandonment written into the log — keyed by the reverting commit's
// SHA, in git-log order (reverse chronological, deterministic).
func gvReverts(ctx *SourceContext) []Finding {
	var out []Finding
	for _, row := range ctx.GitLines("log", "--format=%H%x00%s") {
		i := strings.IndexByte(row, 0)
		if i < 0 {
			continue
		}
		sha, subject := row[:i], row[i+1:]
		if !isRevertSubject(subject) {
			continue
		}
		out = append(out, Finding{
			ID:       revID(sha),
			Signal:   SignalRevert,
			Summary:  "reverted commit",
			Evidence: []string{gvText(subject)},
		})
	}
	return capSignalFindings(out)
}

// refIsSafe reports whether a repo-derived ref name is safe to hand git as a
// positional argument — gitutil.RefIsSafe, the one definition. The lifeboat
// probe is designed to run over hostile/archived repositories, so a crafted ref
// such as "-x" (a branch or an origin/HEAD target written straight into
// .git/refs) must never reach git where it would parse as a flag — argument
// injection.
func refIsSafe(ref string) bool {
	return gitutil.RefIsSafe(ref)
}

// defaultBranch resolves the branch unmerged work is measured against, without
// touching the network: origin/HEAD → the first of {main,master,trunk,develop}
// that exists → the branch HEAD points at → "" (detached / no branches, so the
// unmerged-branch signal is skipped entirely).
func defaultBranch(ctx *SourceContext) string {
	if !ctx.IsGit() {
		return ""
	}
	const originPrefix = "refs/remotes/origin/"
	if out, err := ctx.Git("symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil {
		if ref := strings.TrimSpace(out); strings.HasPrefix(ref, originPrefix) {
			if name := strings.TrimPrefix(ref, originPrefix); refIsSafe(name) {
				return name
			}
		}
	}
	for _, cand := range []string{"main", "master", "trunk", "develop"} {
		if _, err := ctx.Git("rev-parse", "--verify", "--quiet", "refs/heads/"+cand); err == nil {
			return cand
		}
	}
	if out, err := ctx.Git("symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		if name := strings.TrimSpace(out); refIsSafe(name) {
			return name
		}
	}
	return ""
}

// gvUnmergedBranches reports local branches never merged into the default branch,
// ranked by divergence age: the branch whose merge-base commit is oldest comes
// first, ties broken by name. The merge-base commit's date is stable repository
// data, so the ranking survives re-plans; it is not the wall clock.
func gvUnmergedBranches(ctx *SourceContext) []Finding {
	db := defaultBranch(ctx)
	if !refIsSafe(db) {
		return nil
	}
	type branch struct {
		name     string
		base     string
		ahead    int
		baseTime int64
	}
	var branches []branch
	// Bound the branches probed BEFORE the per-branch fan-out (merge-base +
	// rev-list + log, three execs each). The list arrives refname-sorted and
	// deterministic, so truncating it keeps a stable, reproducible subset. The
	// count dropped here is the ONLY place this signal's truncation is visible:
	// findings are built from the bounded probe, so the findings-level cap can
	// never see it — hence noteFindingsOmitted below.
	all := ctx.GitLines("branch", "--format=%(refname:short)", "--no-merged", db)
	probe := boundProbeList(all, maxGraveyardFindingsPerSignal)
	unprobed := len(all) - len(probe)
	for _, name := range probe {
		name = strings.TrimSpace(name)
		if name == db || !refIsSafe(name) {
			continue // empty, the base branch itself, or an option-like hostile ref
		}
		base, err := ctx.Git("merge-base", db, name)
		if err != nil {
			continue
		}
		base = strings.TrimSpace(base)
		ahead := 0
		if s, err := ctx.Git("rev-list", "--count", db+".."+name); err == nil {
			ahead, _ = strconv.Atoi(strings.TrimSpace(s))
		}
		var baseTime int64
		if s, err := ctx.Git("log", "-1", "--format=%ct", base); err == nil {
			baseTime, _ = strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		}
		branches = append(branches, branch{name: name, base: base, ahead: ahead, baseTime: baseTime})
	}
	sort.SliceStable(branches, func(i, j int) bool {
		if branches[i].baseTime != branches[j].baseTime {
			return branches[i].baseTime < branches[j].baseTime
		}
		return branches[i].name < branches[j].name
	})
	var out []Finding
	for _, b := range branches {
		out = append(out, Finding{
			ID:      branchID(b.name),
			Signal:  SignalUnmergedBranch,
			Summary: fmt.Sprintf("branch never merged into %s; diverged %d commits ago", gvText(db), b.ahead),
			Evidence: []string{
				fmt.Sprintf("%d commits ahead of %s", b.ahead, gvText(db)),
				"merge-base " + shortHex(b.base),
			},
		})
	}
	// The findings-level cap still runs (it holds the invariant if the probe bound
	// ever changes); the probe-level notice announces what it cannot see.
	return noteFindingsOmitted(capSignalFindings(out), unprobed)
}

// gvDeletedPaths reports paths deleted after substantial history — sustained
// investment retired, not a scratch file swept. A path qualifies only when at
// least substantialHistoryCommits commits touched it AND it is absent at HEAD, so
// a deleted-then-re-added live file is never falsely reported. Sorted by path
// (gitDeletedPaths is already sorted and deduped).
func gvDeletedPaths(ctx *SourceContext) []Finding {
	candidates := gitDeletedPaths(ctx)
	// Defence in depth: bound the candidate set before iterating, so a hostile
	// history with hundreds of thousands of deleted paths cannot make the loop
	// (or its findings) balloon. gitDeletedPaths is already sorted, so the bound
	// keeps the alphabetically-first paths deterministically.
	candidates = boundProbeList(candidates, maxGraveyardCandidatePaths)

	// One walk yields the per-path commit-touch counts for the WHOLE history, and
	// one listing yields the HEAD tracked-file set — so the whole signal costs a
	// constant number of git execs regardless of how many paths were deleted (the
	// old code fanned out one `git log -- <p>` and one `cat-file -e` per path).
	counts := gvPathTouchCounts(ctx)
	tracked := map[string]bool{}
	for _, p := range ctx.GitLines("ls-files") {
		tracked[p] = true
	}

	var out []Finding
	for _, p := range candidates {
		n := counts[p]
		if n < substantialHistoryCommits {
			continue
		}
		if tracked[p] {
			continue
		}
		out = append(out, Finding{
			ID:       deletedPathID(p),
			Signal:   SignalDeletedPath,
			Summary:  "path deleted after substantial history",
			Evidence: []string{fmt.Sprintf("deleted; %d commits touched it before deletion", n)},
		})
	}
	return capSignalFindings(out)
}

// gvPathTouchCounts returns, for every path named across HEAD's history, the
// number of commits that touched it — derived in ONE git walk. `git log
// --name-only` with an empty format emits each changed path once per (non-merge)
// commit; counting occurrences reproduces the old per-path `git log -- <p>` count
// for the linear histories the graveyard reasons over, without a per-path exec.
func gvPathTouchCounts(ctx *SourceContext) map[string]int {
	counts := map[string]int{}
	for _, p := range ctx.GitLines("log", "--name-only", "--format=") {
		counts[p]++
	}
	return counts
}

// gvRemovedDependencies reports manifests that carried a dependency (or a whole
// ecosystem) in history but not at HEAD. For each manifest with history it diffs
// the tokens of its earliest revision against HEAD (empty when the manifest is
// gone), and reports the set difference. depTokens is a conservative,
// ecosystem-agnostic extractor — it may under-report, but never fabricates a
// name not literally in the file. Sorted by manifest, removed tokens sorted, the
// citation list capped at maxDependencyTokens.
func gvRemovedDependencies(ctx *SourceContext) []Finding {
	var out []Finding
	for _, m := range gitManifestFiles {
		if len(ctx.GitLines("log", "--format=%h", "--", m)) == 0 {
			continue
		}
		var head map[string]bool
		if data, ok := ctx.ReadFile(m); ok {
			head = depTokens(data)
		} else {
			head = map[string]bool{}
		}
		revs := ctx.GitLines("log", "--format=%H", "--reverse", "--", m)
		if len(revs) == 0 {
			continue
		}
		show, err := ctx.Git("show", revs[0]+":"+m)
		if err != nil {
			continue
		}
		earliest := depTokens([]byte(show))
		var removed []string
		for tok := range earliest {
			if !head[tok] {
				removed = append(removed, tok)
			}
		}
		if len(removed) == 0 {
			continue
		}
		sort.Strings(removed)
		if len(removed) > maxDependencyTokens {
			removed = removed[:maxDependencyTokens]
		}
		evidence := make([]string, 0, len(removed))
		for _, tok := range removed {
			evidence = append(evidence, "removed: "+gvText(tok))
		}
		out = append(out, Finding{
			ID:       dependencyID(m),
			Signal:   SignalRemovedDependency,
			Summary:  "dependencies present in history but absent at HEAD",
			Evidence: evidence,
		})
	}
	return capSignalFindings(out)
}

// depTokenRe is the conservative dependency-name shape: an identifier-ish run of
// characters common to manifest ecosystems. The FIRST match on a candidate line
// is taken as that line's dependency token.
var depTokenRe = regexp.MustCompile(`[A-Za-z0-9._/@-]+`)

// goModRequireDirectives are the go.mod directive keywords whose SECOND token is
// the dependency (module) path. When a line leads with one of these AND a second
// identifier follows, depTokens takes the second token, so the single-line form
// `require github.com/x/y v1` and the block form (an indented `github.com/x/y v1`
// line) yield the SAME token. Without this, `go mod edit -fmt`/tidy collapsing a
// one-dependency block into a single line would fabricate a removed dependency.
// module and go are deliberately excluded: their second token is the module's own
// name or a version, not a dependency, and taking it would invent false removals
// on a rename or a go-version bump.
var goModRequireDirectives = map[string]bool{
	"require": true, "exclude": true, "replace": true, "retract": true,
}

// depTokens extracts a conservative set of dependency-ish tokens from a manifest:
// the first identifier-ish token of each non-blank, non-comment, non-bracket
// line. It over-includes structural keys (which cancel in the earliest-vs-HEAD
// set difference) and under-includes multi-token lines, but it never invents a
// name not literally present in the file — the honesty the removed-dependency
// signal needs.
func depTokens(data []byte) map[string]bool {
	toks := map[string]bool{}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, ";") || strings.HasPrefix(line, "/*") ||
			strings.HasPrefix(line, "*") {
			continue
		}
		switch line[0] {
		case '{', '}', '[', ']', '(', ')', '<', '>':
			continue
		}
		matches := depTokenRe.FindAllString(line, 2)
		if len(matches) == 0 {
			continue
		}
		tok := matches[0]
		if goModRequireDirectives[tok] {
			if len(matches) < 2 {
				// A bare directive line (e.g. `require (`) names no dependency.
				continue
			}
			tok = matches[1]
		}
		toks[tok] = true
	}
	return toks
}

// gvTreeSize is the number of files tracked at HEAD — the deterministic, cheap
// denominator for the wholesale-rewrite fraction. It is effectively cached: the
// underlying `git ls-files` runs once and the SourceContext memoises it.
func gvTreeSize(ctx *SourceContext) int {
	return len(ctx.GitLines("ls-files"))
}

// gvRewrites reports single non-merge commits that replaced a large fraction of
// the tree — a restructure rather than incremental work. A commit qualifies when
// it changed at least wholesaleRewriteMinFiles files AND at least
// wholesaleRewriteTreeFraction of the HEAD tree. Git-log order.
func gvRewrites(ctx *SourceContext) []Finding {
	size := gvTreeSize(ctx)
	if size <= 0 {
		return nil
	}
	var out []Finding
	var curSHA, curSubject string
	curFiles := 0
	curHasParent := false
	have := false
	flush := func() {
		if !have {
			return
		}
		// The parentless root commit diffs against the empty tree, so its shortstat
		// counts the whole initial import — a "rewrite" of nothing. Skip it: a
		// wholesale rewrite is only meaningful against something it replaced.
		if !curHasParent {
			return
		}
		if curFiles >= wholesaleRewriteMinFiles && float64(curFiles) >= wholesaleRewriteTreeFraction*float64(size) {
			out = append(out, Finding{
				ID:      rewriteID(curSHA),
				Signal:  SignalWholesaleRewrite,
				Summary: "single commit replaced a large fraction of the tree",
				Evidence: []string{
					"rewrite: " + gvText(curSubject),
					fmt.Sprintf("%d files changed of %d tracked", curFiles, size),
				},
			})
		}
	}
	for _, line := range ctx.GitLines("log", "--no-merges", "--format=%H%x00%P%x00%s", "--shortstat") {
		if strings.IndexByte(line, 0) >= 0 {
			// A commit header (SHA \0 parents \0 subject). Flush the previous commit
			// (whose file count may still be 0 — an empty commit emits no shortstat
			// line) before starting this one. An empty parents field marks the root.
			flush()
			parts := strings.SplitN(line, "\x00", 3)
			curSHA = parts[0]
			curHasParent = len(parts) > 1 && strings.TrimSpace(parts[1]) != ""
			curSubject = ""
			if len(parts) > 2 {
				curSubject = parts[2]
			}
			curFiles = 0
			have = true
			continue
		}
		if n, ok := shortstatFiles(line); ok {
			curFiles = n
		}
	}
	flush()
	return capSignalFindings(out)
}

// shortstatFiles parses the leading "N files changed" count from a
// `git log --shortstat` summary line (e.g. "3 files changed, 10 insertions(+)").
// A line that is not a shortstat summary returns ok=false.
func shortstatFiles(line string) (int, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil || !strings.HasPrefix(fields[1], "file") {
		return 0, false
	}
	return n, true
}

// boundProbeList truncates a deterministically-ordered probe list to at most n
// entries BEFORE a per-entry git fan-out, so a hostile repo cannot turn one probe
// into an unbounded number of git execs. It preserves order and leaves a list
// already within the bound (or a non-positive bound treated as "no truncation
// needed") untouched. A pure helper so the bound is unit-testable without a
// fixture of hundreds of branches or paths.
func boundProbeList(names []string, n int) []string {
	if n >= 0 && len(names) > n {
		return names[:n]
	}
	return names
}
