package capture

// mentions.go — the ADVISORY half of iss-2609100507421759.
//
// The gate half lives in scripts/check-issue-resolution.sh (RS004): from now on
// a commit message or pull-request title/body that names an iss-N must declare
// its relation to it — `Resolves:` (fixes it) or `Refs:` (touched, not fixed).
// That rule cannot reach backwards. The history a repository already has carries
// bare mentions by the hundred, and four of them, in the field experiment the
// record was written from, were fixes nobody resolved: the ledger said open, the
// default branch said fixed, and finding out cost an hour of hand-diffing before
// any work could be assigned.
//
// This is the listing that reads that history: open records whose ids appear on
// the default branch, with the evidence that named them, ranked by how strong
// that evidence is. It LISTS. It never resolves, never moves a record, never
// writes anything at all — the record itself asks for exactly that ("a mention
// is not a fix, and a human reads the row"), and a lint that quietly closed
// records nobody resolved is the failure its deferral reason named.
//
// Scope note, shared with the gate: this side of the rule reads COMMIT MESSAGES.
// Ids inside RECORD BODIES are a different surface with a different rule
// (`prose_citation_resolves`, in internal/core/lint).

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The strength vocabulary, weakest to strongest. A row carries the strongest
// evidence found for its id; every piece of evidence is still listed under it.
const (
	// StrengthRecord — the id was named by a commit that touched only the record
	// tiers (.abcd/**). Somebody wrote ABOUT the record. That is the weakest
	// signal there is, and it is listed last.
	StrengthRecord = "record"
	// StrengthTree — the id was named by a commit that changed something outside
	// the record tiers. Somebody did work and said which record it was about.
	StrengthTree = "tree"
	// StrengthResolves — a commit DECLARED `Resolves: iss-N` and the record is
	// still in open/. Nothing about this one is ambiguous: somebody said the
	// change fixed it and the ledger never moved.
	StrengthResolves = "resolves"
)

// strengthRank orders the vocabulary for "the strongest evidence on this row".
var strengthRank = map[string]int{StrengthRecord: 1, StrengthTree: 2, StrengthResolves: 3}

// mentionRe finds an iss-N anywhere in free prose. The leading guard is the Go
// twin of MENTION_RE in scripts/check-issue-resolution.sh, and it exists for the
// same reason: `xiss-1` inside a longer token is not a mention of iss-1, and
// `iss-12` is not a mention of iss-1 either (the greedy digit run takes the
// whole number). Keeping the two spellings side by side is deliberate — the gate
// refuses what this listing would otherwise have to report.
var mentionRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9])(iss-[0-9]+)`)

// declareRe is the Go twin of DECLARE_RE: the whole declaration vocabulary,
// nothing but the declaration on the line. A closed set of two spellings, for the
// reason the script states — admitting `Ref:`/`See:`/`Related:` would reopen the
// omission the rule closes. The id half is a COMMA-SEPARATED LIST because that is
// the conventional trailer shape (`Refs: iss-1, iss-2`); refusing it made an
// author write the trailer twice, or drop the second id, which is the omission
// the rule exists to close. `\r?` is here because a message may arrive
// CRLF-terminated from a forge-composed squash.
var declareRe = regexp.MustCompile(`(?m)^(Resolves|Refs):[ \t]+(iss-[0-9]+(?:[ \t]*,[ \t]*iss-[0-9]+)*)[ \t]*\r?$`)

// declaredIDRe takes the ids back out of a declaration line's id list.
var declaredIDRe = regexp.MustCompile(`iss-[0-9]+`)

// recordTierPrefix is the working-tree tier a mention can be written in without
// anything being fixed: the record itself, the brief, the decision log. A commit
// confined to it is narrating, not repairing.
const recordTierPrefix = ".abcd/"

// mentionsLogCap bounds the history buffered by the walk. `git log` over a
// hostile or merely enormous repository emits unbounded output, and this is a
// read-only advisory that must never be able to exhaust memory. The cap is
// enforced with RunCapped rather than RunLimited on purpose: a truncated history
// is not a shorter one, it is a WRONG one, and here the wrong answer is silence
// — an operator reading a short listing concludes there is nothing to do.
const mentionsLogCap = 32 << 20

// MentionsRequest asks for the listing over one repository.
type MentionsRequest struct {
	RepoRoot   string
	IssuesRoot string
	// Ref is the history to walk. Empty means the repository's default branch,
	// resolved without touching the network.
	Ref string
}

// MentionEvidence is one commit that named a record, and how it named it.
type MentionEvidence struct {
	Commit   string `json:"commit"`
	Date     string `json:"date"`
	Subject  string `json:"subject"`
	Strength string `json:"strength"`
}

// MentionRow is one open record the walk found named, with every mention that
// named it. A row with no evidence is never emitted: a row nobody can check is
// an accusation, not a listing.
type MentionRow struct {
	ID       string            `json:"id"`
	Severity Severity          `json:"severity,omitempty"`
	Path     string            `json:"path,omitempty"`
	Strength string            `json:"strength"`
	Evidence []MentionEvidence `json:"evidence"`
}

// MentionsResult is the advisory listing. Ref and Commits say what was walked,
// so a reader can tell an empty listing from an unwalked history.
type MentionsResult struct {
	Ref         string       `json:"ref"`
	Commits     int          `json:"commits"`
	OpenRecords int          `json:"open_records"`
	Rows        []MentionRow `json:"rows"`
	Skipped     []SkipRecord `json:"skipped"`
}

// Mentions lists open records whose ids are named by the history of Ref with no
// resolution behind them. Strictly read-only: it runs read-only git commands and
// the ledger's own List, and writes nothing anywhere.
func Mentions(req MentionsRequest) (MentionsResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return MentionsResult{}, err
	}
	open, err := List(ListRequest{RepoRoot: repoRoot, IssuesRoot: issuesRoot, State: StateOpen})
	if err != nil {
		return MentionsResult{}, err
	}
	byID := make(map[string]Issue, len(open.Issues))
	for _, iss := range open.Issues {
		byID[iss.ID] = iss
	}

	ref, err := resolveMentionsRef(repoRoot, req.Ref)
	if err != nil {
		return MentionsResult{}, err
	}
	commits, err := walkMentionCommits(repoRoot, ref)
	if err != nil {
		return MentionsResult{}, err
	}

	evidence := map[string][]MentionEvidence{}
	for _, c := range commits {
		for id, strength := range c.judge(byID) {
			evidence[id] = append(evidence[id], MentionEvidence{
				Commit: c.sha, Date: c.date, Subject: c.subject(), Strength: strength,
			})
		}
	}

	rows := make([]MentionRow, 0, len(evidence))
	for id, ev := range evidence {
		// Evidence arrives in git log order — newest first — because the walk
		// appends per commit in that order and each id owns its own slice. It is
		// deliberately NOT re-sorted on Date: %cI carries a timezone offset, so
		// comparing two of them as strings orders by wall-clock text rather than
		// by instant, which is a wrong answer dressed as a tidy one.
		//
		// What IS re-ordered is strength, and it is done HERE rather than in a
		// renderer so every front door reads the same first row. A stable sort on
		// the rank alone keeps the log order inside each rank, so the head of the
		// slice is the strongest evidence and, among equals, the newest. The
		// listing exists to surface the `Resolves:` commit nobody acted on; with
		// the raw log order that commit hid behind a later docs commit that
		// merely named the record, which is the one row an operator must see.
		sort.SliceStable(ev, func(i, j int) bool {
			return strengthRank[ev[i].Strength] > strengthRank[ev[j].Strength]
		})
		row := MentionRow{ID: id, Evidence: ev, Strength: ev[0].Strength,
			Severity: byID[id].Severity, Path: byID[id].Path}
		rows = append(rows, row)
	}
	// Strongest evidence first — the rows an operator should read are the ones
	// somebody already said were fixed — then by id, so the listing is stable.
	sort.Slice(rows, func(i, j int) bool {
		if a, b := strengthRank[rows[i].Strength], strengthRank[rows[j].Strength]; a != b {
			return a > b
		}
		return issNumber(rows[i].ID) < issNumber(rows[j].ID)
	})

	skipped := open.Skipped
	if skipped == nil {
		skipped = []SkipRecord{}
	}
	return MentionsResult{
		Ref: ref, Commits: len(commits), OpenRecords: len(open.Issues),
		Rows: rows, Skipped: skipped,
	}, nil
}

// resolveMentionsRef settles which history is walked, and proves it exists.
//
// A ref the caller named is verified rather than assumed: `git log` on a missing
// ref fails loudly, but a caller that mistyped a branch must be told the NAME it
// got wrong — an empty listing reads as "nothing to do", which is the exact
// wrong answer for an advisory whose whole purpose is to break a silence.
func resolveMentionsRef(repoRoot, want string) (string, error) {
	ref := strings.TrimSpace(want)
	if ref == "" {
		ref = detectDefaultBranch(repoRoot)
	}
	// A ref beginning with '-' would reach git as a flag: argument injection. No
	// legitimate ref name starts with one.
	if ref == "" || strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("cannot resolve a history to walk: no usable ref (%q)", ref)
	}
	// The probe's own error is deliberately not wrapped: `--quiet` suppresses
	// git's stderr, so the cause is a bare "exit status 1" that adds nothing to
	// a message already naming the ref and the repository.
	// Every git call in this file that takes a ref POSITIONALLY ends with `--`.
	// Without it a ref whose name equals a tracked path — a branch `docs` beside
	// the `docs/` tree, which this repository would have today — is refused with
	// "ambiguous argument", and the advisory dies on the repositories most likely
	// to need it.
	if _, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--"); err != nil {
		return "", fmt.Errorf("cannot walk %q: no such commit-ish in this repository", ref)
	}
	return ref, nil
}

// detectDefaultBranch resolves the branch the listing walks when the caller names
// none, without touching the network: origin/HEAD, then the conventional names,
// then whatever HEAD points at. It mirrors the lifeboat probe's resolution — the
// question is the same one and the answer must not differ between two readers of
// one repository.
func detectDefaultBranch(repoRoot string) string {
	const originPrefix = "refs/remotes/origin/"
	if out, err := gitutil.Run(repoRoot, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil {
		if name := strings.TrimPrefix(strings.TrimSpace(out), originPrefix); name != "" && name != out {
			return name
		}
	}
	for _, cand := range []string{"main", "master", "trunk", "develop"} {
		if _, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+cand, "--"); err == nil {
			return cand
		}
	}
	if out, err := gitutil.Run(repoRoot, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		if name := strings.TrimSpace(out); name != "" {
			return name
		}
	}
	return ""
}

// mentionCommit is one commit of the walked history: its message and the paths it
// changed, which is everything the classification needs.
type mentionCommit struct {
	sha   string
	date  string
	body  string
	added []string // paths this commit ADDED
	paths []string // every path it touched
}

func (c mentionCommit) subject() string {
	line, _, _ := strings.Cut(strings.TrimSpace(c.body), "\n")
	return strings.TrimSpace(line)
}

// judge applies the evidence rules to one commit and returns the strength it
// contributes per open id. An id it says nothing usable about is absent.
func (c mentionCommit) judge(open map[string]Issue) map[string]string {
	mentioned := map[string]bool{}
	for _, m := range mentionRe.FindAllStringSubmatch(c.body, -1) {
		mentioned[m[1]] = true
	}
	if len(mentioned) == 0 {
		return nil
	}
	declared := map[string]string{}
	for _, m := range declareRe.FindAllStringSubmatch(c.body, -1) {
		// One line declares the same relation for every id it lists.
		for _, id := range declaredIDRe.FindAllString(m[2], -1) {
			// `Resolves:` outranks `Refs:` if a message carries both for one id:
			// the stronger claim is the one its author is answerable for.
			if declared[id] != "Resolves" {
				declared[id] = m[1]
			}
		}
	}

	out := map[string]string{}
	for id := range mentioned {
		if _, isOpen := open[id]; !isOpen {
			continue
		}
		switch declared[id] {
		case "Refs":
			// The operator SAID this is not a fix. Taking them at their word is
			// what makes the declaration worth writing; a listing that reported
			// it anyway would teach people to stop declaring.
			continue
		case "Resolves":
			out[id] = StrengthResolves
			continue
		}
		// PROVENANCE, not evidence: the commit that FILED the record names the id
		// it is filing. It is the commonest mention in any ledger's history and
		// it says nothing about whether anything was fixed.
		if namesRecordFile(c.added, id) {
			continue
		}
		out[id] = StrengthRecord
		for _, p := range c.paths {
			if !strings.HasPrefix(p, recordTierPrefix) {
				out[id] = StrengthTree
				break
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// namesRecordFile reports whether any of paths is the ledger file of id — by the
// filename the minting convention guarantees (`<id>-<slug>.md`), so a record that
// has since been renamed or refiled in another status folder is still recognised.
func namesRecordFile(paths []string, id string) bool {
	for _, p := range paths {
		base := p
		if i := strings.LastIndexByte(p, '/'); i >= 0 {
			base = p[i+1:]
		}
		if base == id+".md" || strings.HasPrefix(base, id+"-") {
			return true
		}
	}
	return false
}

// walkMentionCommits reads the history of ref: for every non-merge commit, its
// sha, commit date, full message and the paths it changed.
//
// Merge commits are skipped for the same reason RS004 exempts them — the message
// is composed by the forge, and every commit it brings in is reachable from ref
// and read on its own. A squash merge is not a merge commit, so it is read here
// carrying its branch's own declarations.
func walkMentionCommits(repoRoot, ref string) ([]mentionCommit, error) {
	known, err := walkedShas(repoRoot, ref)
	if err != nil {
		return nil, err
	}
	// \x1e opens a commit record and \x1f separates its fields; neither can occur
	// in a path, because git quotes control characters in pathnames by default.
	out, err := gitutil.RunCapped(repoRoot, mentionsLogCap,
		"log", ref, "--root", "--no-merges", "--name-status",
		"--format=%x1e%H%x1f%cI%x1f%B%x1f", "--")
	if err != nil {
		return nil, fmt.Errorf("walking %q: %w", ref, err)
	}

	// First pass: re-assemble the records. A commit MESSAGE can contain \x1e, and
	// a message is attacker-controlled text — `--cleanup=verbatim` will write
	// whatever an author hands it. A chunk that merely LOOKS like a boundary
	// (\x1e, 40 hex, \x1f) would then be read as a commit of its own: the real
	// commit's sha vanishes from the listing and a sha nobody ever made appears
	// in its place, shape-valid for `capture resolve --commit`. So a boundary is
	// only a boundary when rev-list — which reads the commit graph, not the
	// message — named that sha. Anything else is text belonging to the record
	// before it and is given back verbatim, mark included; the body/name-status
	// split happens after re-assembly, so a forged sequence costs the walk
	// nothing but the honesty of quoting the message in full.
	var records []string
	for _, chunk := range strings.Split(out, "\x1e") {
		if chunk == "" {
			continue
		}
		if sha, _, ok := strings.Cut(chunk, "\x1f"); ok && known[sha] {
			records = append(records, chunk)
			continue
		}
		if n := len(records); n > 0 {
			records[n-1] += "\x1e" + chunk
		}
	}

	var commits []mentionCommit
	for _, rec := range records {
		sha, rest, _ := strings.Cut(rec, "\x1f")
		date, rest, _ := strings.Cut(rest, "\x1f")
		// The LAST separator ends the message: the file list that follows can
		// contain no \x1f, while a message in principle can.
		body, names := rest, ""
		if i := strings.LastIndex(rest, "\x1f"); i >= 0 {
			body, names = rest[:i], rest[i+1:]
		}
		c := mentionCommit{sha: sha, date: date, body: body}
		for _, line := range strings.Split(names, "\n") {
			// `--name-status` rows are "<status>\t<path>" — and for a rename or a
			// copy, "<status>\t<old>\t<new>". Every path on the row counts as
			// touched; only an add counts as filing.
			fields := strings.Split(strings.TrimRight(line, "\r"), "\t")
			if len(fields) < 2 || fields[0] == "" {
				continue
			}
			for _, p := range fields[1:] {
				if p == "" {
					continue
				}
				c.paths = append(c.paths, p)
				if strings.HasPrefix(fields[0], "A") {
					c.added = append(c.added, p)
				}
			}
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// walkedShas is the set of commits the walk is allowed to see as boundaries: the
// same history `git log` is about to print, read from the commit graph where no
// commit message can reach. It is capped like the log for the same reason.
func walkedShas(repoRoot, ref string) (map[string]bool, error) {
	out, err := gitutil.RunCapped(repoRoot, mentionsLogCap, "rev-list", "--no-merges", ref, "--")
	if err != nil {
		return nil, fmt.Errorf("walking %q: %w", ref, err)
	}
	known := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if sha := strings.TrimSpace(line); sha != "" {
			known[sha] = true
		}
	}
	return known, nil
}
