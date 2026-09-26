package lint

// The cross-store family (cross_store_id_claim): a record id claimed by a
// document that is not in the store that id belongs to. A second arm, below,
// names a reframe record sitting outside its store, which the cold reading
// would otherwise receive (spc-2609020626048705).
//
// record_schema reasons across the stores, but only INSIDE them: a file outside
// every configured store is not a malformed record to the engine, it is not a
// record at all. So a markdown file can head itself `# ADR-23`, declare a status,
// reuse an id a real ADR already holds, and pass the whole gate at exit 0 with
// zero findings. That was not hypothetical — it was measured by probe, and the
// known instance survived a full architecture change (iss-2608230752354926).
//
// The rule fires on a PAIR of signals, never on one:
//
//   - the document's H1 OPENS with a record handle, which is a claim on that id
//     rather than a mention of it; and
//   - the body carries the decision shape (`## Status`, `Status: Accepted`); and
//   - the id is already HELD by a real record in the corresponding store.
//
// The pair is what grandfathers the undated Phase 0 notes without an allowlist of
// them. A filename that reads like an ordinal (`01-harness-interface.md`) claims
// nothing; a reading note that names a decision in prose claims nothing; a
// meeting note with a Status block claims no id. None of them fires, and none of
// them had to be enumerated to be spared — which matters, because an allowlist of
// grandfathered files is a list that grows silently.
//
// It is a STRUCTURAL rule and so does not consult contentExempt, for the reason
// record_schema does not: an id collision is not a question of how a document is
// WRITTEN. The exempt paths excuse the historical record from the
// content-authoring rules; being historical cannot excuse a document from
// claiming an id another record answers to, and the known instance lives in
// exactly such a tree — exempting it would disarm the rule over the corpus that
// motivated it.

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const ruleCrossStoreIDClaim = "cross_store_id_claim"

var (
	// A record handle OPENING a heading: `# ADR-23: …`, `# itd-7 …`. Anchored,
	// because the anchor is what parts a CLAIM from a mention — "# How adr-23 was
	// decided" is a document about a decision, not a second copy of it.
	claimedHandleRe = regexp.MustCompile(`(?i)^(adr|itd|iss|spc)-(\d+)\b`)
	// The decision shape: a `Status` heading (the ADR template's own), or a
	// `Status:` key line in either the bold or the bare spelling. Frontmatter is
	// not read for this — a `status:` field is ordinary record metadata on plenty
	// of documents, while a Status SECTION is how a decision announces itself.
	decisionStatusHeadingRe = regexp.MustCompile(`(?i)^\s*#{1,6}\s+status\s*$`)
	decisionStatusKeyRe     = regexp.MustCompile(`(?i)^\s*\**status\**\s*:(.*)$`)
	// The status VALUE has to be a record LIFECYCLE state, not merely a line that
	// says "status". That distinction is what parts the two populations the corpus
	// actually contains. A design plan heads itself with the record it plans for
	// (`# itd-3 modular rules loader`) and carries a status of its own
	// (`**Status:** design recorded 2026-07-11`, `**Status:** SIGNED OFF`): it is a
	// document ABOUT a record, and there are dozens of them. A document DECLARING
	// itself accepted or superseded under a handle another record answers to is a
	// second copy of that record. Only the second is this rule's finding.
	decisionStatusValueRe = regexp.MustCompile(`(?i)\b(accepted|proposed|rejected|superseded|deprecated)\b`)
)

// maxCrossStoreBytes caps a candidate read. The rule reads ordinary markdown, so
// the cap is the citation walk's own page limit rather than a second number.
const maxCrossStoreBytes = citationPageSizeLimit

// checkCrossStoreIDClaim walks the markdown OUTSIDE the configured record stores
// and reports every decision-shaped document that claims an id a real record
// already holds. A repo with no stores configured has no baseline to weigh a
// claim against, so it contributes nothing and is not an error.
func checkCrossStoreIDClaim(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	stores := rc.RecordStores
	if len(stores) == 0 {
		stores = cfg.Rules[ruleRecordSchema].RecordStores
	}
	if len(stores) == 0 {
		return nil, nil
	}

	// The taken-id set comes from the record graph — the same scan record_schema
	// trusts for its own high-water marks — so this detector cannot drift from the
	// stores' own view of which ids are taken.
	graphCfg := cfg
	graphCfg.Rules = map[string]RuleConfig{ruleRecordSchema: {RecordStores: stores}}
	graph, err := LoadRecordGraph(graphCfg, repoRoot)
	if err != nil {
		return nil, err
	}
	taken := make(map[string]bool, len(graph.Nodes))
	for _, n := range graph.Nodes {
		taken[canonRecordID(strings.ToLower(n.ID))] = true
	}
	reframeStore := strings.TrimSuffix(filepath.ToSlash(stores[issueschema.ReframeFamily]), "/")
	if len(taken) == 0 && reframeStore == "" {
		return nil, nil
	}

	storeDirs := make([]string, 0, len(stores))
	for _, dir := range stores {
		if dir == "" {
			continue
		}
		if err := containedRepoPath(dir); err != nil {
			return nil, &configError{ruleCrossStoreIDClaim + ": record store " + quote(dir) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		storeDirs = append(storeDirs, filepath.ToSlash(strings.TrimSuffix(filepath.ToSlash(dir), "/"))+"/")
	}

	candidates, err := crossStoreCandidates(repoRoot, storeDirs)
	if err != nil {
		return nil, err
	}

	var out []Finding
	for _, rel := range candidates {
		// The rule reads files whose content AND paths a cloned repo controls, so
		// each leaf is contained and guarded the way the roots walk contains and
		// guards its own: a `docs/leak.md -> /etc/passwd` link must not be read, and
		// a `-> /dev/zero` one must not be read unbounded.
		realPath, err := containedRealPath(repoRoot, filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			continue // a leaf resolving outside the repo is not this rule's finding
		}
		content, err := fsutil.ReadGuarded(realPath, maxCrossStoreBytes)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		if f, ok := reframeOutsideStore(rel, lines, reframeStore, rc.Severity); ok {
			out = append(out, f)
			continue
		}
		if f, ok := crossStoreClaim(rel, lines, taken, rc.Severity); ok {
			out = append(out, f)
		}
	}
	sortFindings(out)
	return out, nil
}

// crossStoreCandidates lists the markdown OUTSIDE the record stores that the rule
// judges, repo-relative and slash-separated.
//
// In a git repository the candidate set is the TRACKED files, the same scope
// `abcd lint`'s privacy rule uses. This rule is the only one in the package that
// reasons about a whole tree rather than a configured root, and a bare walk of
// the root was wrong in both directions. It read the gitignored local tier —
// which AGENTS.md tells agents to default to for oracle output, traces and
// intermediate analysis — so a scratch copy of a record turned the gate into a
// blocker over a file that is not in the repository; and it read a `git worktree`
// made INSIDE the checkout, which AGENTS.md's concurrent-sessions guidance
// contemplates, turning every record in the sibling checkout into a finding. An
// untracked file is also not yet a claim on an id: nothing outside this working
// tree can resolve to it.
//
// Outside a repository (a fixture tree, an unpacked archive) there is no tracked
// set to ask for, so the walk is the fallback. It skips `.git` and the stores.
func crossStoreCandidates(repoRoot string, storeDirs []string) ([]string, error) {
	if gitutil.InRepo(repoRoot) {
		tracked, err := gitutil.TrackedFiles(repoRoot)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, rel := range tracked {
			rel = filepath.ToSlash(rel)
			if hasMarkdownExt(rel) && !hasAnyPrefix(rel, storeDirs) {
				out = append(out, rel)
			}
		}
		sort.Strings(out)
		return out, nil
	}

	var out []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// One unreadable directory must not abort the whole lint: this rule
			// walks the entire tree, so a permission fault anywhere in it would
			// take every other rule down with it. Skip the subtree instead.
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel := filepath.ToSlash(repoRel(repoRoot, path))
		if d.IsDir() {
			// .git is machinery, never content; a record store's own files are
			// judged by record_schema, and this rule is about what sits OUTSIDE
			// them. A symlinked directory is not followed by WalkDir at all, so a
			// committed link cannot redirect the walk out of the repository.
			if d.Name() == ".git" || hasAnyPrefix(rel+"/", storeDirs) {
				return filepath.SkipDir
			}
			return nil
		}
		if hasMarkdownExt(d.Name()) && !hasAnyPrefix(rel, storeDirs) {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// crossStoreClaim judges one candidate document: does its H1 claim a taken id,
// and does its body carry the decision shape.
func crossStoreClaim(rel string, lines []string, taken map[string]bool, severity string) (Finding, bool) {
	heading, line := recordH1(lines)
	m := claimedHandleRe.FindStringSubmatch(strings.TrimSpace(heading))
	if m == nil {
		return Finding{}, false
	}
	num, err := strconv.Atoi(m[2])
	if err != nil {
		return Finding{}, false
	}
	id := canonRecordID(strings.ToLower(m[1]) + "-" + strconv.Itoa(num))
	if !taken[id] {
		return Finding{}, false
	}
	if !hasDecisionShape(lines) {
		return Finding{}, false
	}
	return Finding{
		File: rel, Line: line, RuleID: ruleCrossStoreIDClaim, Severity: severity,
		Message: "document claims the record id '" + id + "' but does not sit in that record's store; " +
			id + " is already held by a real record, so this file is a second document answering to one handle — " +
			"every cross-reference and index that keys on it resolves to the other. Give the document its own " +
			"identity (a title that claims no handle), or file it as a record in the store and let the id be minted",
	}, true
}

// The reframe arm. A reframe record is warm at every reading position, and the
// assembler keeps it out by the PATH of its store (spc-2609020626048705): a
// reframe-shaped file anywhere else is a file like any other to the include
// table, so a copy planted as a brief chapter reaches every cold reading with its
// grounds and its body, and nothing else names it. Unlike the id-claim arm this
// one fires on ONE signal, because the hazard is the content travelling, not a
// collision with a held id: an rfm-N file name, an rfm id in the frontmatter, or
// the reframe's own pair of keys — the occasion and a before fingerprint — which
// no other record carries together.
var (
	reframeFileNameRe = regexp.MustCompile(`(?i)^rfm-\d+(?:[-.]|$)`)
	reframeIDRe       = regexp.MustCompile(`(?i)^rfm-\d+$`)
)

// reframeOutsideStore judges one candidate outside every store. A file inside
// a directory ending in the reframe store's own path is a nested tree's store —
// a fixture repository, which every reading denies by its `.abcd` segment — and
// is its own record, not a stray one.
func reframeOutsideStore(rel string, lines []string, store, severity string) (Finding, bool) {
	if store == "" || strings.HasSuffix(path.Dir(rel), "/"+store) {
		return Finding{}, false
	}
	fields := frontmatterFields(lines)
	var why string
	var line int
	switch {
	case reframeFileNameRe.MatchString(path.Base(rel)):
		why, line = "its name is a reframe record's ("+path.Base(rel)+")", 1
	case reframeIDRe.MatchString(fieldValue(fields, "id")):
		why, line = "its frontmatter claims the reframe id "+strings.ToLower(fieldValue(fields, "id")), fields["id"].line
	default:
		occ, hasOcc := fields["occasioned_by"]
		if !hasOcc {
			return Finding{}, false
		}
		for _, n := range issueschema.FrameSurfaceNames {
			if _, ok := fields[n+"_before"]; ok {
				why, line = "its frontmatter carries a reframe record's occasioned_by and "+n+"_before", occ.line
				break
			}
		}
		if why == "" {
			return Finding{}, false
		}
	}
	if line == 0 {
		line = 1
	}
	return Finding{
		File: rel, Line: line, RuleID: ruleCrossStoreIDClaim, Severity: severity,
		Message: "a reframe record outside its store: " + why + ", but it does not sit in " + store + "/. " +
			"A reframe is warm, and the cold reading keeps it out by that store's path alone, so a copy anywhere " +
			"else reaches every cold reading with its grounds and body. Move it into the store, or remove it",
	}, true
}

// hasDecisionShape reports whether the body DECLARES a record lifecycle status —
// a `## Status` section or a `Status:` line whose value is one of the record's
// own lifecycle states. Fenced blocks are masked by the shared fenceMask: a page
// QUOTING the shape (which is how this rule gets documented) is an example, not a
// decision.
func hasDecisionShape(lines []string) bool {
	mask := fenceMask(lines)
	// Start at the BODY. The comment on decisionStatusKeyRe says frontmatter is
	// not read for this, and it has to be true: a `status:` frontmatter key is
	// ordinary record metadata on a great many documents, so reading it made the
	// fire condition "an H1 claiming a taken id, plus almost any record-shaped
	// file" — wide enough that a verbatim copy of an ADR in a scratch directory
	// fired, with no Status section anywhere in it. A decision SECTION is how a
	// decision announces itself; a metadata key is not.
	for i := recordBodyStart(lines); i < len(lines); i++ {
		line := lines[i]
		if mask[i] {
			continue
		}
		if m := decisionStatusKeyRe.FindStringSubmatch(line); m != nil {
			if decisionStatusValueRe.MatchString(m[1]) {
				return true
			}
			continue
		}
		// A `## Status` heading carries its value in the section beneath it, so
		// the first non-blank line after the heading is the value.
		if decisionStatusHeadingRe.MatchString(line) {
			for j := i + 1; j < len(lines); j++ {
				if mask[j] || strings.TrimSpace(lines[j]) == "" {
					continue
				}
				if decisionStatusValueRe.MatchString(lines[j]) {
					return true
				}
				break
			}
		}
	}
	return false
}
