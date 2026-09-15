// Package lint is abcd's record-drift gate: it reads a JSON config and lints
// the markdown design record, returning findings. It performs no I/O beyond
// reading files under a caller-supplied repo root — no printing, no os.Exit —
// so it is fully testable and reusable across surfaces.
package lint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Finding is one lint violation. File is repo-relative; Line is 1-based (0 when
// the finding is not tied to a specific line, e.g. a missing directory README).
type Finding struct {
	File     string
	Line     int
	RuleID   string
	Severity string
	Message  string
}

const (
	severityBlocker = "blocker"
	severityWarn    = "warn"
)

// The two crosscheck depths a receipt can declare (iss-122). "full" runs both
// directions over the whole brief-doc list; "shallow" runs Direction B only. A
// full receipt satisfies any release; a shallow one satisfies only a patch
// (fix/internal) release.
const (
	releaseTierFull    = "full"
	releaseTierShallow = "shallow"
)

// releaseGateManifestPath is the committed input manifest that pins the iss-35
// crosscheck's scope and depth (iss-122). Its PRESENCE in the content tree is
// the era marker: receipts written before the manifest existed are judged by the
// pre-manifest rules, and the manifest-era refusals (manifestHash, tier, and
// findings-disposition) arm only when this file is present at the armed commit.
// The path is fixed rather than config-driven so a committer cannot silence the
// new refusals by blanking a config field.
const releaseGateManifestPath = ".abcd/development/release-gate/manifest.json"

var (
	// Inline markdown link: [text](target). Also matches the link part of an
	// image (![alt](src)), which resolves the same way.
	linkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	// URL scheme prefix (http:, mailto:, ...); such targets are not local paths.
	schemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)
	// Brittle line reference: some-file.md:171 (check D).
	brittleRefRe = regexp.MustCompile(`[A-Za-z0-9_./-]+\.md:\d+`)
	// Intent id embedded in a filename or a superseded_by value.
	intentIDRe   = regexp.MustCompile(`itd-\d+`)
	intentFileRe = regexp.MustCompile(`^itd-\d+.*\.md$`)
	// Issue id embedded in a ledger filename (issue_id_unique).
	issueIDRe   = regexp.MustCompile(`iss-\d+`)
	issueFileRe = regexp.MustCompile(`^iss-\d+.*\.md$`)
	// Splits a matched record id into its prefix and numeric part, so the
	// uniqueness backstop keys on the canonical id rather than the raw spelling.
	recordIDRe = regexp.MustCompile(`^([a-z]+)-0*([0-9]+)$`)
	// Surface registry Command cell: the bare "/abcd" top-level, or "/abcd:<name>".
	surfaceCmdRe = regexp.MustCompile(`^/abcd(?::([a-z0-9-]+))?$`)
	// receipt_gate arming inputs are release-time and become externally supplied
	// (release.yml) — validated as safe path components before use.
	receiptShaRe  = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	receiptGateRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	// gate_lockstep hand-parsers (no YAML library — the repo has none and adds no
	// dependency): a markdown numbered-list item; the `jobs:` line; a 2-space job
	// header (identifier, optional quotes/trailing comment, never a comment line);
	// a job's `steps:` key; a step list-item marker; and a step `name:` key line.
	// A non-empty floor (RuleConfig.MinGates) is the fail-closed backstop that
	// makes any parser under-count block rather than silently pass.
	numberedItemRe = regexp.MustCompile(`^\s*\d+\.\s+(.+?)\s*$`)
	jobsSectionRe  = regexp.MustCompile(`^jobs:\s*(#.*)?$`)
	jobHeaderRe    = regexp.MustCompile(`^  ["']?([A-Za-z0-9_-]+)["']?:\s*(#.*)?$`)
	stepsKeyRe     = regexp.MustCompile(`^\s+steps:\s*(#.*)?$`)
	stepMarkerRe   = regexp.MustCompile(`^\s+-\s+(.*\S)\s*$`)
	stepNameKeyRe  = regexp.MustCompile(`^\s+name:\s*(.+?)\s*$`)
	specIDRe       = regexp.MustCompile(`^spc-`)
	// A superseded intent may be replaced by a later intent OR by an ADR: an ADR
	// that redecides the question an intent rested on retires that intent as surely
	// as a successor intent does (itd-47 → adr-22, itd-49 → adr-26). Restricting
	// this to ^itd- forced such a record to carry no supersession at all (iss-39).
	//
	// The widening makes intent_lifecycle's own target check LOOSER: it reads only
	// the intent tree, so it can resolve an itd- target but not an adr- one, and a
	// repo running intent_lifecycle ALONE therefore accepts `superseded_by:
	// adr-9999`. record_schema is what closes that — it reads every store and
	// resolves cross-store targets — so the two rules are armed together.
	supersededRe = regexp.MustCompile(`^(itd|adr)-\d+`)
	// spec_lifecycle: the anchored filename matcher for the spec store. The
	// well-formed-ID predicates themselves are NOT restated here: they are
	// recordid.ValidIntentID / recordid.ValidSpecID, the same call the intent and
	// spec loaders make, so this gate refuses exactly the set they refuse
	// (iss-2608270500198764).
	specFileRe = regexp.MustCompile(`^spc-\d+.*\.md$`)
	// The lifecycle directories each store declares. They are spelled ONCE, as
	// ordered slices, because record_schema enumerates them to catch an undeclared
	// bucket — a second, divergent copy would let a bucket be "known" to one rule
	// and invisible to the other (iss-39).
	intentBucketNames = []string{"drafts", "planned", "shipped", "disciplines", "superseded"}
	specBucketNames   = []string{"open", "closed"}
	// issueStatusDirs are the issue ledger's status directories (issue_id_unique
	// scans all three for a duplicated iss-N id).
	issueStatusDirs = issueschema.StatusDirs
	intentBuckets   = bucketSet(intentBucketNames)
	// intentImpactValues and issueImpactValues render the legal impact set an
	// error message offers. They are composed from the shared enum's constants
	// (internal/core/changelog) rather than spelled as literals, so a rename there
	// is a compile error here instead of a stale message. The intent list omits
	// internal: an intent is press-release-first, so "invisible to users" is not a
	// judgement it can make.
	intentImpactValues = joinImpacts(changelog.ImpactAdditive, changelog.ImpactBreaking, changelog.ImpactFix)
	issueImpactValues  = joinImpacts(changelog.ImpactAdditive, changelog.ImpactBreaking, changelog.ImpactFix, changelog.ImpactInternal)
)

// bucketSet renders a declared-bucket slice as the membership set the walks test
// against, so the ordered list stays the single declaration of what exists.
func bucketSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

// joinImpacts renders an impact set the way the shared parser's own errors do,
// so every message about the enum reads alike.
func joinImpacts(impacts ...changelog.Impact) string {
	parts := make([]string, 0, len(impacts))
	for _, i := range impacts {
		parts = append(parts, string(i))
	}
	return strings.Join(parts, "|")
}

// Lint runs every enabled check family against the record under repoRoot and
// returns the findings sorted deterministically. An error is returned only for
// malformed configuration (e.g. an uncompilable regexp, or a persona_registry
// roster that is missing, unparsable, or empty); a walkable-but-missing root
// is skipped, not an error.
func Lint(cfg Config, repoRoot string) ([]Finding, error) {
	return LintAt(cfg, repoRoot, time.Now())
}

// LintAt is Lint with the clock supplied by the caller. Only the citation
// baseline's staleness arithmetic reads it, but it is a parameter rather than a
// wall-clock read inside the check so the 180- and 365-day boundaries are
// testable exactly, and so a gate's verdict is a function of its inputs.
func LintAt(cfg Config, repoRoot string, now time.Time) ([]Finding, error) {
	var findings []Finding

	tokenChecks, err := NewTokenChecker(cfg.BannedTokens)
	if err != nil {
		return nil, err
	}

	linksCfg, linksOn := cfg.Rules["links_resolve"]
	gitMetaCfg, gitMetaOn := cfg.Rules["no_git_metadata"]
	brittleCfg, brittleOn := cfg.Rules["no_brittle_line_refs"]
	linksOn = linksOn && linksCfg.Enabled
	gitMetaOn = gitMetaOn && gitMetaCfg.Enabled
	brittleOn = brittleOn && brittleCfg.Enabled

	// The citation family shares ONE parse of each page (footnote structure plus
	// the citation corpus), so a page is read once no matter how many of the five
	// rules are armed. citedRefs accumulates across every root because the
	// baseline is a single repo-wide record, checked once after the walk.
	citeFootCfg, citeFootOn := cfg.Rules[ruleCitationFootnotes]
	citeRowsCfg, citeRowsOn := cfg.Rules[ruleCitationCrosswalkRows]
	citeURLCfg, citeURLOn := cfg.Rules[ruleCitationURLSyntax]
	citePolicyCfg, citePolicyOn := cfg.Rules[ruleCitationSourcePolicy]
	citeBaseCfg, citeBaseOn := cfg.Rules[ruleCitationBaseline]
	citeFootOn = citeFootOn && citeFootCfg.Enabled
	citeRowsOn = citeRowsOn && citeRowsCfg.Enabled
	citeURLOn = citeURLOn && citeURLCfg.Enabled
	citePolicyOn = citePolicyOn && citePolicyCfg.Enabled
	citeBaseOn = citeBaseOn && citeBaseCfg.Enabled
	citeParseOn := citeFootOn || citeURLOn || citePolicyOn || citeBaseOn
	var citedRefs []citedRef

	// The harness-leak class rides the same per-file walk. It is NOT a
	// content-authoring rule: a session URL in a superseded record is as much a
	// leak as one in a live page, so it does not consult contentExempt.
	leakCfg, leakOn := cfg.Rules[ruleHarnessLeak]
	leakOn = leakOn && leakCfg.Enabled

	personaCfg, personaOn := cfg.Rules["persona_registry"]
	personaOn = personaOn && personaCfg.Enabled
	var personaRoster map[string]bool
	if personaOn {
		personaRoster, err = loadPersonaRoster(repoRoot, personaCfg.Registry)
		if err != nil {
			return nil, err
		}
	}

	for _, root := range cfg.Roots {
		// The walk reads committed files whose content AND paths a cloned repo
		// controls through its committed .abcd/docs-lint.json / record-lint.json
		// roots. Contain the root and every resolved leaf, and guard the read,
		// exactly as CollectCitedURLs does over this same cfg.Roots — otherwise a
		// `roots` of "../private", or a committed `docs/leak.md -> /etc/passwd`
		// symlink inside a contained root, reads and lints a file the repository
		// does not own (its path and line then printed to stdout/--json/CI), and
		// a `-> /dev/zero` link is followed and read unbounded.
		if err := containedRepoPath(root); err != nil {
			return nil, &configError{"roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		rootAbs := filepath.Join(repoRoot, root)
		if err := resolvedInsideRoot(repoRoot, rootAbs); err != nil {
			return nil, &configError{"roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		// A configured root that does not resolve is misconfiguration, not a clean
		// tree: every per-file family walks this root, so a missing one would report
		// zero findings at exit 0 and silently disarm every blocker for it — the
		// shipped scaffold's `roots: ["docs", …]` does exactly this in an adopter
		// whose docs live elsewhere. Fail loud instead (os.Stat, not IsDir: `roots`
		// legitimately admits files such as README.md) — GitHub #360.
		if _, err := os.Stat(rootAbs); err != nil {
			if os.IsNotExist(err) {
				return nil, &configError{"roots entry " + quote(root) +
					" does not exist; a configured root that does not resolve silently disarms every per-file rule for that tree — fix the roots list or create the tree"}
			}
			return nil, err
		}
		mdFiles, err := markdownFiles(rootAbs)
		if err != nil {
			return nil, err
		}
		for _, fileAbs := range mdFiles {
			realPath, err := containedRealPath(repoRoot, fileAbs)
			if err != nil {
				return nil, &configError{"file " + quote(repoRel(repoRoot, fileAbs)) + " " + err.Error() +
					"; the lint reads only inside the repository"}
			}
			content, err := fsutil.ReadGuarded(realPath, citationPageSizeLimit)
			if err != nil {
				return nil, err
			}
			rel := repoRel(repoRoot, fileAbs)
			lines := strings.Split(string(content), "\n")
			mask := fenceMask(lines)
			// Content-authoring checks (banned_tokens, persona_registry) cover only
			// the forward-looking record — being historical excuses a record from
			// how it is WRITTEN, never from being well-formed. The intent-tree
			// checks (intent_lifecycle, intent_impact_valid) no longer consult the
			// exemption and record_schema never did; the spec-store checks
			// (spec_lifecycle, spec_id_unique) still do (iss-39).
			exempt := contentExempt(rel, frontmatterFields(lines), cfg)

			if tokenChecks.Len() > 0 && !exempt {
				findings = append(findings, tokenChecks.lintLines(rel, lines, mask)...)
			}
			if personaOn && !exempt {
				findings = append(findings, checkPersonaRegistry(rel, lines, mask, personaRoster, personaCfg)...)
			}
			if gitMetaOn {
				findings = append(findings, checkGitMetadata(rel, lines, gitMetaCfg)...)
			}
			if linksOn {
				findings = append(findings, checkLinks(rel, fileAbs, repoRoot, lines, mask, linksCfg)...)
			}
			if brittleOn {
				findings = append(findings, checkBrittleRefs(rel, lines, mask, brittleCfg)...)
			}
			if leakOn {
				findings = append(findings, checkHarnessLeak(rel, lines, mask, leakCfg)...)
			}

			if citeParseOn {
				page := parseCitations(rel, lines, mask)
				if citeFootOn {
					findings = append(findings, checkCitationFootnotes(rel, page, citeFootCfg)...)
				}
				if citeURLOn {
					findings = append(findings, checkCitationURLSyntax(page, citeURLCfg)...)
				}
				if citePolicyOn {
					findings = append(findings, checkCitationSourcePolicy(page, citePolicyCfg)...)
				}
				if citeBaseOn {
					citedRefs = append(citedRefs, page.urls...)
				}
			}
			if citeRowsOn {
				cw, err := checkCitationCrosswalkRows(rel, lines, mask, citeRowsCfg)
				if err != nil {
					return nil, err
				}
				findings = append(findings, cw...)
			}
		}

		if dirCfg, ok := cfg.Rules["directory_coverage"]; ok && dirCfg.Enabled {
			dc, err := checkDirectoryCoverage(repoRoot, rootAbs, dirCfg)
			if err != nil {
				return nil, err
			}
			findings = append(findings, dc...)
		}

		// The intent-tree rules (intent_lifecycle, intent_impact_valid) read the
		// SAME bucket tree, so it is scanned once per distinct intents_dir and each
		// rule validates the records that scan produced. A rule is never a second
		// walk of a tree another rule already read.
		intentTrees := map[string]intentTree{}
		scanIntents := func(dir string) (intentTree, error) {
			if t, ok := intentTrees[dir]; ok {
				return t, nil
			}
			t, err := scanIntentTree(repoRoot, rootAbs, dir)
			if err != nil {
				return intentTree{}, err
			}
			intentTrees[dir] = t
			return t, nil
		}

		if intentCfg, ok := cfg.Rules["intent_lifecycle"]; ok && intentCfg.Enabled {
			tree, err := scanIntents(intentsDirOf(intentCfg))
			if err != nil {
				return nil, err
			}
			findings = append(findings, checkIntentLifecycle(repoRoot, tree, intentCfg)...)
		}

		if impactCfg, ok := cfg.Rules["intent_impact_valid"]; ok && impactCfg.Enabled {
			tree, err := scanIntents(intentsDirOf(impactCfg))
			if err != nil {
				return nil, err
			}
			findings = append(findings, checkIntentImpact(tree, impactCfg)...)
		}

		if specCfg, ok := cfg.Rules["spec_lifecycle"]; ok && specCfg.Enabled {
			sl, err := checkSpecLifecycle(repoRoot, rootAbs, specCfg, cfg)
			if err != nil {
				return nil, err
			}
			findings = append(findings, sl...)
		}

		if suCfg, ok := cfg.Rules["spec_id_unique"]; ok && suCfg.Enabled {
			su, err := checkSpecIDUnique(repoRoot, rootAbs, suCfg, cfg)
			if err != nil {
				return nil, err
			}
			findings = append(findings, su...)
		}

		if fsCfg, ok := cfg.Rules["forbidden_synonyms"]; ok && fsCfg.Enabled {
			fs, err := checkForbiddenSynonyms(repoRoot, rootAbs, fsCfg)
			if err != nil {
				return nil, err
			}
			findings = append(findings, fs...)
		}
	}

	// stray_root_docs is repo-root scoped and non-recursive — independent of
	// cfg.Roots, so it runs once, outside the per-root loop.
	if strayCfg, ok := cfg.Rules["stray_root_docs"]; ok && strayCfg.Enabled {
		sr, err := checkStrayRootDocs(repoRoot, strayCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, sr...)
	}

	// agent_contract walks the agent-prompt tree (agents/ — outside cfg.Roots and
	// outside the docs-lint roots too, which is how the class went undetected),
	// so it runs once here as well.
	if acCfg, ok := cfg.Rules[ruleAgentContract]; ok && acCfg.Enabled {
		ac, err := checkAgentContract(repoRoot, acCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, ac...)
	}

	// context_status_free targets one work-tier file (CONTEXT.md) that lives
	// outside cfg.Roots, so it too runs once, outside the per-root loop.
	if ctxCfg, ok := cfg.Rules["context_status_free"]; ok && ctxCfg.Enabled {
		cs, err := checkContextStatusFree(repoRoot, ctxCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, cs...)
	}

	// context_citation_currency reads the same work-tier CONTEXT.md against the
	// record's stores, both outside cfg.Roots, so it runs once here too.
	if ccCfg, ok := cfg.Rules[ruleContextCitationCurrency]; ok && ccCfg.Enabled {
		cc, err := checkContextCitationCurrency(repoRoot, ccCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, cc...)
	}

	// surface_coverage cross-checks the plugin surface (commands/, skills/ —
	// outside cfg.Roots) against the brief's surface registry (inside a root), so
	// it too runs once, outside the per-root loop.
	if scCfg, ok := cfg.Rules["surface_coverage"]; ok && scCfg.Enabled {
		sc, err := checkSurfaceCoverage(repoRoot, scCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, sc...)
		// The sub-verb pass (spc-27): armed by the snapshot config key, it
		// checks each surface file's `## Sub-verbs` table against the committed
		// command-tree snapshot in both directions.
		sv, err := checkSubVerbCoverage(repoRoot, scCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, sv...)
	}

	// record_schema reasons ACROSS the record's stores (ADRs, intents, specs, and
	// the issue ledger), which straddle cfg.Roots, so its stores are named
	// repo-relative in its own config and it runs once here.
	if rsCfg, ok := cfg.Rules[ruleRecordSchema]; ok && rsCfg.Enabled {
		rs, err := checkRecordSchema(repoRoot, rsCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, rs...)
	}

	// cross_store_id_claim is the other half of the same cross-store question: it
	// walks the markdown OUTSIDE those stores, which is every tree at once, so it
	// too runs once here.
	if csCfg, ok := cfg.Rules[ruleCrossStoreIDClaim]; ok && csCfg.Enabled {
		cs, err := checkCrossStoreIDClaim(repoRoot, cfg, csCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, cs...)
	}

	// index_drift reads documents that live beside the code they index (a package
	// or command README), all outside cfg.Roots, so it runs once here too.
	if idxCfg, ok := cfg.Rules["index_drift"]; ok && idxCfg.Enabled {
		id, err := checkIndexDrift(repoRoot, idxCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, id...)
	}

	// delivery_state reads the repo-root changelog against the intents store, both
	// outside cfg.Roots, so it runs once here as well.
	if dsCfg, ok := cfg.Rules[ruleDeliveryState]; ok && dsCfg.Enabled {
		ds, err := checkDeliveryState(repoRoot, dsCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, ds...)
	}

	// receipt_gate is the release-time verification of the semantic gates. It is
	// disabled for ordinary development (a commit under review has no receipt yet)
	// and armed only at release time with a target commit; it reads sha-keyed
	// receipts outside cfg.Roots, so it also runs once here.
	if rgCfg, ok := cfg.Rules["receipt_gate"]; ok && rgCfg.Enabled {
		rg, err := checkReceiptGate(repoRoot, rgCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, rg...)
	}

	// gate_lockstep keeps the release-gate runbook's deterministic-gate list in
	// lockstep with the CI workflow (both outside cfg.Roots), so it runs once here.
	if glCfg, ok := cfg.Rules["gate_lockstep"]; ok && glCfg.Enabled {
		gl, err := checkGateLockstep(repoRoot, glCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, gl...)
	}

	// The issue-ledger rules (issue_id_unique, issue_impact_valid) read the SAME
	// ledger (.abcd/work/issues/{open,resolved,wontfix} — outside cfg.Roots), so it
	// is scanned once per distinct issues_dir here and each rule validates the
	// records that scan produced.
	issueLedgers := map[string]issueLedger{}
	scanIssues := func(dir string) (issueLedger, error) {
		if l, ok := issueLedgers[dir]; ok {
			return l, nil
		}
		l, err := scanIssueLedger(repoRoot, dir)
		if err != nil {
			return issueLedger{}, err
		}
		issueLedgers[dir] = l
		return l, nil
	}

	if iiCfg, ok := cfg.Rules["issue_id_unique"]; ok && iiCfg.Enabled {
		ledger, err := scanIssues(issuesDirOf(iiCfg))
		if err != nil {
			return nil, err
		}
		findings = append(findings, checkIssueIDUnique(repoRoot, ledger, iiCfg)...)
	}

	// reading_outstanding reads the same ledger root's SIBLING families
	// (readings/, dispositions/), also outside cfg.Roots, so it runs once here.
	// It is a report, never a gate: its severity is pinned in code and its
	// findings are info, so it can never fail a push that has nothing to do with
	// a reading.
	if roCfg, ok := cfg.Rules[ruleReadingOutstanding]; ok && roCfg.Enabled {
		ro, err := checkReadingOutstanding(repoRoot, roCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, ro...)
	}

	// record_provenance reads the same cross-store scan record_schema walks, so
	// it runs once here rather than per root.
	if rpCfg, ok := cfg.Rules[ruleRecordProvenance]; ok && rpCfg.Enabled {
		rp, err := checkRecordProvenance(repoRoot, cfg, rpCfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, rp...)
	}

	if impactCfg, ok := cfg.Rules["issue_impact_valid"]; ok && impactCfg.Enabled {
		ledger, err := scanIssues(issuesDirOf(impactCfg))
		if err != nil {
			return nil, err
		}
		findings = append(findings, checkIssueImpact(ledger, impactCfg)...)
	}

	// citation_baseline is repo-wide: one committed record answers for every
	// citation in every root, so it is loaded once here rather than per page,
	// against the refs the walk collected.
	if citeBaseOn {
		cb, err := checkCitationBaseline(repoRoot, citedRefs, citeBaseCfg, now)
		if err != nil {
			return nil, err
		}
		findings = append(findings, cb...)
	}

	sortFindings(findings)
	return findings, nil
}

// checkStrayRootDocs flags top-level markdown files whose basename stem is not
// in the configured allowlist. It is non-recursive (os.ReadDir on the repo root
// only) — subdirectories such as docs/ and .abcd/ are never touched. A root
// markdown file that is a symlink is judged by its resolved target's stem, so
// bridge links like CLAUDE.md -> AGENTS.md and GEMINI.md -> AGENTS.md are exempt
// without allowlisting the link name; a symlink with an unresolvable target is a
// finding.
func checkStrayRootDocs(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	allow := make(map[string]bool, len(cfg.Allowlist))
	for _, a := range cfg.Allowlist {
		allow[strings.ToUpper(a)] = true
	}

	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Finding
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !hasMarkdownExt(name) {
			continue
		}

		// A root markdown symlink is judged by its resolved target's stem, not
		// its own name — this is what lets the CLAUDE.md / GEMINI.md bridges
		// (symlinks to the allowlisted AGENTS.md) pass. Known tradeoff: a stray
		// name pointed at an allowlisted target (e.g. NOTES.md -> AGENTS.md) is
		// also exempt. Acceptable here — creating such a symlink is a deliberate
		// act with the same trust level as adding the allowlisted file itself.
		stemName := name
		info, err := os.Lstat(filepath.Join(repoRoot, name))
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(filepath.Join(repoRoot, name))
			if err != nil {
				out = append(out, Finding{
					File: name, Line: 0, RuleID: "stray_root_docs",
					Severity: cfg.Severity,
					Message:  "top-level markdown symlink '" + name + "' has an unresolvable target",
				})
				continue
			}
			stemName = filepath.Base(resolved)
		}

		stem := strings.ToUpper(strings.TrimSuffix(stemName, filepath.Ext(stemName)))
		if allow[stem] {
			continue
		}
		out = append(out, Finding{
			File: name, Line: 0, RuleID: "stray_root_docs",
			Severity: cfg.Severity,
			Message:  "top-level markdown '" + name + "' is not in the root allowlist; documentation belongs under docs/",
		})
	}
	return out, nil
}

// contextStatusDefaultPatterns is the fallback line-match set for
// context_status_free when the config supplies no patterns. It targets the
// phase/status idioms an orientation doc drifts into (headings, **Current
// phase**, **Next:**, "Phase N — ...", and a status: frontmatter key).
var contextStatusDefaultPatterns = []string{
	`(?i)^#+\s*current (phase|status)`,
	`(?i)\*\*current phase`,
	`(?i)^\*\*next:`,
	`(?i)\bphase [0-9]+(\.[0-9]+)? — `,
	`(?i)^status:`,
}

const contextStatusMessage = "CONTEXT.md is status-free (DECISIONS.md 2026-07-10): status lives in the live surfaces (CLI, ledger), not in orientation docs"

// checkContextStatusFree scans a single target file (the work-tier CONTEXT.md,
// which sits outside cfg.Roots) line by line, flagging every line that matches
// any configured pattern. Fenced code blocks are masked via the shared
// fenceMask. A missing target is not an error — it yields no findings. This is a
// file-scoped rule: banning "Phase N" corpus-wide would explode, so the ban is
// confined to the one orientation doc that must stay status-free.
func checkContextStatusFree(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	target := cfg.Target
	if target == "" {
		target = filepath.Join(".abcd", "work", "CONTEXT.md")
	}
	rawPatterns := cfg.Patterns
	if len(rawPatterns) == 0 {
		rawPatterns = contextStatusDefaultPatterns
	}
	patterns := make([]*regexp.Regexp, 0, len(rawPatterns))
	for _, p := range rawPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, re)
	}

	fileAbs := filepath.Join(repoRoot, target)
	content, err := os.ReadFile(fileAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	rel := repoRel(repoRoot, fileAbs)
	lines := strings.Split(string(content), "\n")
	mask := fenceMask(lines)

	var out []Finding
	for i, line := range lines {
		if mask[i] {
			continue
		}
		if matchesAny(patterns, line) {
			out = append(out, Finding{
				File: rel, Line: i + 1, RuleID: "context_status_free",
				Severity: cfg.Severity, Message: contextStatusMessage,
			})
		}
	}
	return out, nil
}

// surfaceRow is one parsed row of the brief's surface registry table.
type surfaceRow struct {
	name   string // sub-verb after "/abcd:"; empty for the bare "/abcd" top-level
	status string // lower-cased "shipped" | "staged"; any other value is flagged
	line   int    // 1-based line of the row in the registry file
}

// checkSurfaceCoverage is the deterministic (Direction-B) half of the iss-35
// brief↔surface cross-check. It reads the plugin surface (commands/ + skills/,
// which live outside cfg.Roots) and the brief's surface registry table, then
// asserts three invariants:
//   - coverage: every real surface (a commands/*.md file or a skills/*/
//     directory) has a registry row;
//   - status fidelity: a row marked "shipped" has a backing surface, and a row
//     marked "staged" does not — the bare "/abcd" top-level names no sub-verb,
//     so its command file is the configured BareCommand and it is exempt from
//     the file check;
//   - registry integrity: every row's status is "shipped" or "staged".
//
// The semantic half (a brief claim vs. binary behaviour — flags, exit codes,
// schema fields) stays an agent/release-gate check, not a structural lint. A
// missing registry file is not an error.
func checkSurfaceCoverage(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	if cfg.Registry == "" {
		return nil, nil
	}
	rows, err := parseSurfaceRegistry(repoRoot, cfg.Registry)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil // registry file absent — nothing to cross-check
	}
	real, realPaths, err := realSurfaces(repoRoot, cfg)
	if err != nil {
		return nil, err
	}

	var out []Finding
	rowNames := make(map[string]bool, len(rows))
	for _, r := range rows {
		if r.name != "" {
			rowNames[r.name] = true
		}
		switch r.status {
		case "shipped":
			if r.name != "" && !real[r.name] {
				out = append(out, Finding{
					File: cfg.Registry, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
					Message: "surface row '" + surfaceLabel(r.name) + "' is marked shipped but no commands/ or skills/ surface backs it",
				})
			}
		case "staged":
			if r.name != "" && real[r.name] {
				out = append(out, Finding{
					File: cfg.Registry, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
					Message: "surface row '" + surfaceLabel(r.name) + "' is marked staged but a backing surface exists; mark it shipped",
				})
			}
		default:
			out = append(out, Finding{
				File: cfg.Registry, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: "surface row '" + surfaceLabel(r.name) + "' has unknown status '" + r.status + "' (want shipped|staged)",
			})
		}
	}

	// Coverage: every real surface must resolve to a registry row. Iterated in
	// sorted order so findings are deterministic before the final sort.
	names := make([]string, 0, len(realPaths))
	for name := range realPaths {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !rowNames[name] {
			out = append(out, Finding{
				File: realPaths[name], Line: 0, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: "surface '" + name + "' has no row in the brief surface registry (" + cfg.Registry + ")",
			})
		}
	}
	return out, nil
}

// surfaceLabel renders a surface name as its slash-command spelling.
func surfaceLabel(name string) string {
	if name == "" {
		return "/abcd"
	}
	return "/abcd:" + name
}

// realSurfaces enumerates the shipped plugin surface as a name set plus a
// name→repo-relative-path map. Command surfaces are the *.md files directly under
// CommandsDir (README and the bare top-level file excepted); skill surfaces are
// the immediate subdirectories of SkillsDir. A missing directory contributes
// nothing and is not an error.
//
// Only the files DIRECTLY under CommandsDir count, because that is what the
// harness registers as /<plugin>:<verb>: a subdirectory becomes an extra
// namespace segment, so anything nested is a different command name than the
// registry describes and must not be read as backing a row.
func realSurfaces(repoRoot string, cfg RuleConfig) (map[string]bool, map[string]string, error) {
	set := map[string]bool{}
	paths := map[string]string{}

	if cfg.CommandsDir != "" {
		entries, err := os.ReadDir(filepath.Join(repoRoot, cfg.CommandsDir))
		if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			if strings.EqualFold(name, "README") {
				continue
			}
			if cfg.BareCommand != "" && name == cfg.BareCommand {
				continue
			}
			set[name] = true
			paths[name] = filepath.Join(cfg.CommandsDir, e.Name())
		}
	}

	if cfg.SkillsDir != "" {
		entries, err := os.ReadDir(filepath.Join(repoRoot, cfg.SkillsDir))
		if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			set[e.Name()] = true
			paths[e.Name()] = filepath.Join(cfg.SkillsDir, e.Name())
		}
	}
	return set, paths, nil
}

// parseSurfaceRegistry reads the surface registry markdown and returns its table
// rows. It locates the one pipe-table whose header names both a Command and a
// Status column, then reads each data row's Command and Status cells (keyed by
// header position, so column order is free). Fenced code blocks are masked (the
// house convention, per fenceMask) so a table shown as a markdown *example* — a
// real risk in a doc whose subject is the surface table itself — is never
// mistaken for the registry. A missing file yields (nil, nil); a present file
// with no such table yields an empty, non-nil slice.
func parseSurfaceRegistry(repoRoot, registry string) ([]surfaceRow, error) {
	content, err := os.ReadFile(filepath.Join(repoRoot, registry))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	mask := fenceMask(lines)

	cmdCol, statusCol, headerIdx := -1, -1, -1
	for i, line := range lines {
		if mask[i] || !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		c, s := -1, -1
		for j, cell := range tableCells(line) {
			switch strings.ToLower(cell) {
			case "command":
				c = j
			case "status":
				s = j
			}
		}
		if c >= 0 && s >= 0 {
			cmdCol, statusCol, headerIdx = c, s, i
			break
		}
	}
	rows := []surfaceRow{}
	if headerIdx < 0 {
		return rows, nil
	}

	for i := headerIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if mask[i] || !strings.HasPrefix(trimmed, "|") {
			break // table ended (a fence closes it too)
		}
		if isTableSeparator(trimmed) {
			continue
		}
		// A row whose Command/Status cell is missing or malformed is skipped
		// here (status-fidelity unchecked) but still caught by the coverage
		// direction if it names a real backing surface — fail-loud, not silent.
		cells := tableCells(lines[i])
		if cmdCol >= len(cells) || statusCol >= len(cells) {
			continue
		}
		name, ok := parseSurfaceCommand(cells[cmdCol])
		if !ok {
			continue // not a /abcd surface row
		}
		rows = append(rows, surfaceRow{
			name:   name,
			status: strings.ToLower(strings.TrimSpace(cells[statusCol])),
			line:   i + 1,
		})
	}
	return rows, nil
}

// tableCells splits a markdown table row into trimmed cell strings, dropping the
// empty cells the leading and trailing border pipes produce.
func tableCells(line string) []string {
	parts := strings.Split(strings.TrimSpace(line), "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	if len(cells) > 0 && cells[0] == "" {
		cells = cells[1:]
	}
	if len(cells) > 0 && cells[len(cells)-1] == "" {
		cells = cells[:len(cells)-1]
	}
	return cells
}

// isTableSeparator reports whether a table row is the header separator (only
// pipes, dashes, colons, and whitespace).
func isTableSeparator(line string) bool {
	for _, r := range line {
		switch r {
		case '|', '-', ':', ' ', '\t':
		default:
			return false
		}
	}
	return true
}

// parseSurfaceCommand extracts the surface name from a Command cell. The bare
// "/abcd" yields ("", true); "/abcd:<name>" yields (name, true); anything else
// yields ("", false) so non-command rows are skipped rather than misread.
func parseSurfaceCommand(cell string) (string, bool) {
	c := strings.TrimSpace(strings.Trim(strings.TrimSpace(cell), "`"))
	m := surfaceCmdRe.FindStringSubmatch(c)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// receipt is the parsed shape of a semantic-pass receipt — a Verification
// Summary Attestation (VSA): only the fields the release gate checks.
type receipt struct {
	Subject struct {
		Digest struct {
			GitCommit string `json:"gitCommit"`
		} `json:"digest"`
	} `json:"subject"`
	VerificationResult string `json:"verificationResult"`
	JudgeModel         string `json:"judgeModel"`
	// Policy.Detector names the gate this receipt attests. The gate binding is the
	// receipt's content, not its filename — so a genuine receipt for one detector
	// cannot be copied to another gate's path and satisfy it.
	Policy struct {
		Detector string `json:"detector"`
	} `json:"policy"`
	// Tier and ManifestHash are the two manifest-era fields (iss-122), read only
	// when the committed release-gate manifest exists. Tier is the crosscheck depth
	// the run used ("full" or "shallow"); ManifestHash is the sha256 the run pinned
	// its inputs against, which must equal the committed manifest's actual hash.
	Tier         string `json:"tier"`
	ManifestHash string `json:"manifestHash"`
	// Failing is the findings the run recorded. It already existed in the receipt
	// schema (an informational VSA field); the manifest-era gate now READS it to
	// require that each finding carries a disposition. Only the disposition is
	// inspected — the gate never judges a finding's content or severity.
	Failing []struct {
		Disposition string `json:"disposition"`
	} `json:"failing"`
}

// checkReceiptGate is the fail-closed, release-time verification of the semantic
// gates (the LLM passes CI cannot run). For the target commit it asserts that
// every required gate has a receipt whose subject digest is that commit, whose
// verdict is PROMOTE, and which pins a judge model. A missing, mismatched,
// malformed, HOLD, or model-less receipt BLOCKS — an un-run semantic pass is
// never a silent pass. The rule is release-time only: it stays disabled for
// ordinary development (a commit under review has no receipt yet), and
// release.yml supplies the target commit when it arms the rule. An enabled rule
// with no configured commit fails closed rather than passing vacuously.
func checkReceiptGate(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	dir := cfg.ReceiptsDir
	if dir == "" {
		dir = filepath.Join(".abcd", "work", "reviews")
	}
	// Every arming-input defect fails closed with a single finding — an armed gate
	// that has nothing valid to check is never a pass. Guards are symmetric so no
	// missing/blank/malformed arming input can slip through to a vacuous success.
	failClosed := func(msg string) []Finding {
		return []Finding{{File: dir, Line: 0, RuleID: "receipt_gate", Severity: cfg.Severity, Message: msg}}
	}
	if cfg.Commit == "" {
		return failClosed("receipt_gate is enabled but no target commit is configured; the release gate fails closed"), nil
	}
	if !receiptShaRe.MatchString(cfg.Commit) {
		return failClosed("receipt_gate target commit '" + cfg.Commit + "' is not a valid commit sha; the release gate fails closed"), nil
	}
	if len(cfg.RequiredGates) == 0 {
		return failClosed("receipt_gate is enabled but lists no required gates; the release gate fails closed"), nil
	}

	// Manifest era (iss-122): the committed input manifest pins the crosscheck's
	// scope and depth, so once it exists a receipt must echo its hash and a
	// sufficient tier, and any recorded finding must carry a disposition. Its
	// presence in the content tree is the era marker — a receipt/commit that
	// predates the manifest is judged by the pre-manifest rules only. Read it from
	// repoRoot, the checked-out content tree the gate is armed against.
	manifestBytes, manifestErr := os.ReadFile(filepath.Join(repoRoot, releaseGateManifestPath))
	var manifestEra bool
	var expectedManifestHash, requiredTier string
	switch {
	case manifestErr == nil:
		manifestEra = true
		expectedManifestHash = hashManifest(manifestBytes)
		requiredTier = requiredReleaseTier(repoRoot)
	case os.IsNotExist(manifestErr):
		manifestEra = false
	default:
		// The manifest exists but cannot be read — never fall through to the
		// pre-manifest rules on an I/O error, which would silently drop the new
		// refusals; fail closed.
		return failClosed("receipt_gate cannot read the release-gate manifest " + releaseGateManifestPath + ": " + manifestErr.Error()), nil
	}

	var out []Finding
	add := func(rel, msg string) {
		out = append(out, Finding{
			File: rel, Line: 0, RuleID: "receipt_gate", Severity: cfg.Severity, Message: msg,
		})
	}
	for _, gate := range cfg.RequiredGates {
		if !receiptGateRe.MatchString(gate) {
			add(dir, "receipt_gate required gate name '"+gate+"' is not a safe path component; the release gate fails closed")
			continue
		}
		rel := filepath.Join(dir, cfg.Commit, gate+".json")
		data, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			if os.IsNotExist(err) {
				add(rel, "no '"+gate+"' receipt for commit "+cfg.Commit+"; the semantic gate has not run (fail-closed)")
				continue
			}
			return nil, err
		}
		var r receipt
		if err := json.Unmarshal(data, &r); err != nil {
			add(rel, "'"+gate+"' receipt is malformed JSON: "+err.Error())
			continue
		}
		if r.Subject.Digest.GitCommit != cfg.Commit {
			add(rel, "'"+gate+"' receipt subject '"+r.Subject.Digest.GitCommit+"' does not match the target commit "+cfg.Commit)
			continue
		}
		if r.VerificationResult != "PROMOTE" {
			add(rel, "'"+gate+"' receipt verdict is '"+r.VerificationResult+"', not PROMOTE")
			continue
		}
		// Bind the receipt to the gate it attests: a receipt whose policy.detector
		// names a different gate (or none) does not satisfy this one, even if it is a
		// genuine PROMOTE for the target commit. Without this, one receipt copied
		// across every gate's path would satisfy them all.
		if strings.TrimSpace(r.Policy.Detector) != gate {
			add(rel, "'"+gate+"' receipt attests detector '"+r.Policy.Detector+"', not this gate; a receipt is bound to its detector, not its filename")
			continue
		}
		if strings.TrimSpace(r.JudgeModel) == "" {
			add(rel, "'"+gate+"' receipt pins no judge model; a floating judge is not auditable")
			continue
		}
		if why := floatingJudgeModel(r.JudgeModel); why != "" {
			add(rel, "'"+gate+"' receipt judgeModel '"+r.JudgeModel+"' is "+why+", not a pinned snapshot; a floating judge is not auditable")
			continue
		}
		if !manifestEra {
			// Pre-manifest era: the receipt is judged by the rules above only, so a
			// historical receipt that carries no tier/manifestHash/disposition stays
			// valid for its own commit.
			continue
		}
		// Manifest-era refusals — PROCEDURAL only (iss-122). None of these judge a
		// finding's content or severity: confirmed findings route to the maintainer,
		// whose PROMOTE with recorded dispositions is the gate.
		if r.ManifestHash != expectedManifestHash {
			add(rel, "'"+gate+"' receipt manifestHash '"+r.ManifestHash+"' does not match the committed release-gate manifest ("+expectedManifestHash+"); the run's pinned inputs are not this release's")
			continue
		}
		if !tierSufficient(r.Tier, requiredTier) {
			add(rel, "'"+gate+"' receipt tier '"+r.Tier+"' is insufficient for a "+requiredTier+"-tier release; run the crosscheck at "+requiredTier+" depth")
			continue
		}
		for _, f := range r.Failing {
			if strings.TrimSpace(f.Disposition) == "" {
				add(rel, "'"+gate+"' receipt carries a finding with no recorded disposition; every finding must record a disposition before release")
				break
			}
		}
	}
	return out, nil
}

// floatingJudgeModel reports why a receipt's judgeModel is a floating alias
// rather than a pinned snapshot, or "" when it is pinned. The runbook's rule is
// that a receipt names the judge that produced it so the pass can be re-run
// against the same judge; an id that resolves to whatever the vendor serves
// today cannot be. Pinned means the id carries a version or date component —
// claude-opus-4-8, Claude Fable 5, a dated suffix — which is what the documented
// example and every committed receipt satisfy. Refused: a bare family name with
// no digit anywhere (opus, claude-sonnet), and a rolling alias, which names
// "latest" whether or not a version fragment precedes it (claude-opus-4-latest
// floats within the 4 line exactly as claude-opus-latest floats across lines).
// The check is deliberately shape-only — it knows no vendor's catalogue.
func floatingJudgeModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(m, "latest") {
		return "a rolling alias"
	}
	if !strings.ContainsAny(m, "0123456789") {
		return "a bare family alias with no version or date"
	}
	return ""
}

// hashManifest renders the release-gate manifest's canonical hash — sha256 over
// its exact committed bytes, the same convention the receipt's promptHash and
// the binary provenance use. Rendered as "sha256:<hex>" so a receipt's
// manifestHash is reproducible with `shasum -a 256`.
func hashManifest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// tierSufficient reports whether a receipt's declared crosscheck depth covers
// the depth this release requires. A full run satisfies any release; a shallow
// run satisfies only a shallow (patch) release; an empty or unrecognised tier is
// never sufficient (fail closed — an unpinned depth is not auditable).
func tierSufficient(receiptTier, requiredTier string) bool {
	switch receiptTier {
	case releaseTierFull:
		return true
	case releaseTierShallow:
		return requiredTier == releaseTierShallow
	default:
		return false
	}
}

// requiredReleaseTier is the crosscheck depth this release must have run at,
// derived from the release's impact class: an additive or breaking release
// requires the full crosscheck; a fix/internal (patch) release is covered by the
// shallow Direction-B pass. When the impact cannot be classified at gate time,
// it returns the strict tier (full) so only a full receipt passes — fail closed,
// never a silent shallow pass.
func requiredReleaseTier(repoRoot string) string {
	impact, ok := releaseImpact(repoRoot)
	if !ok {
		return releaseTierFull
	}
	if impact == changelog.ImpactAdditive || impact == changelog.ImpactBreaking {
		return releaseTierFull
	}
	return releaseTierShallow
}

// releaseImpact classifies the release currently at HEAD by the strongest impact
// among the records it shipped — the same signal the version itself is derived
// from (changelog-driven versioning, adr-37), read at gate time from git state
// release.yml already provides. The version NUMBER alone cannot carry this
// pre-1.0: an additive feature and a fix both bump the patch component while the
// repo is 0.x, so only the shipped records' impact frontmatter distinguishes a
// feature release from a patch release. The base is the newest release tag
// strictly older than the current CHANGELOG version, so the cut is exactly this
// release. ok=false means the impact could not be read (no version, no prior
// tag, or git unavailable) and the caller must fall back to the strict tier.
func releaseImpact(repoRoot string) (changelog.Impact, bool) {
	cur, ok, err := changelog.LatestChangelogVersion(repoRoot)
	if err != nil || !ok {
		return "", false
	}
	tags, err := launch.GitExistingTags(repoRoot)
	if err != nil {
		return "", false
	}
	var base launch.Semver
	haveBase := false
	for _, t := range tags {
		if !launch.CoreGreater(cur, t) {
			continue // t is the current release or newer, not a prior base
		}
		if !haveBase || launch.CoreGreater(t, base) {
			base, haveBase = t, true
		}
	}
	if !haveBase {
		return "", false
	}
	records, err := changelog.ShippedSince(repoRoot, base.Tag())
	if err != nil {
		return "", false
	}
	// An added record with a missing or malformed impact ranks below every real
	// impact, so it would silently pull the cut's max down to internal and let a
	// shallow receipt through. The version derivation refuses such a cut upstream
	// (changelog.Derive, RefusalUnlabelledRecord); honour the same invariant here
	// so the tier gate is fail-closed on its own — an unclassifiable cut falls back
	// to the strict full tier rather than being read as a patch.
	if len(records.UnlabelledAdded()) > 0 {
		return "", false
	}
	return records.Impact(), true
}

// checkGateLockstep asserts the release-gate runbook's deterministic-gate list
// is in lockstep with the CI workflow's gate steps — neither is a projection of
// the other (the runbook carries prose the workflow can't, the workflow carries
// setup the runbook shouldn't), so the anti-drift shape is a consistency check,
// not generate-from-source. A gate in one but not the other BLOCKS. Both are
// hand-parsed (no YAML dependency): the runbook's numbered "Deterministic gates"
// list, and the named steps of the workflow's target job minus the configured
// setup steps.
func checkGateLockstep(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	if cfg.Runbook == "" || cfg.Workflow == "" {
		return nil, nil
	}
	minGates := cfg.MinGates
	if minGates < 1 {
		minGates = 1 // enabled ⇒ at least a non-empty floor, never a vacuous pass
	}

	var out []Finding
	block := func(file, msg string) {
		out = append(out, Finding{File: file, Line: 0, RuleID: "gate_lockstep", Severity: cfg.Severity, Message: msg})
	}

	// A configured path that does not resolve is drift/misconfig, not "clean" —
	// fail loud rather than parsing an empty list that would pass vacuously.
	runbookExists, err := fsutil.Exists(filepath.Join(repoRoot, cfg.Runbook))
	if err != nil {
		return nil, err
	}
	workflowExists, err := fsutil.Exists(filepath.Join(repoRoot, cfg.Workflow))
	if err != nil {
		return nil, err
	}
	if !runbookExists {
		block(cfg.Runbook, "gate_lockstep runbook '"+cfg.Runbook+"' does not exist; the release gate fails closed")
	}
	if !workflowExists {
		block(cfg.Workflow, "gate_lockstep workflow '"+cfg.Workflow+"' does not exist; the release gate fails closed")
	}
	if !runbookExists || !workflowExists {
		return out, nil
	}

	runbookGates, err := runbookGateList(repoRoot, cfg.Runbook)
	if err != nil {
		return nil, err
	}
	workflowGates, err := workflowStepNames(repoRoot, cfg.Workflow, cfg.Job, cfg.IgnoreSteps)
	if err != nil {
		return nil, err
	}

	// Non-empty floor: an under-count means a heading/job rename, a missed step
	// form, or a mis-scoped job silently dropped gates. This is the fail-closed
	// backstop that turns every such under-count loud instead of a silent pass.
	if len(runbookGates) < minGates {
		block(cfg.Runbook, "gate_lockstep parsed "+strconv.Itoa(len(runbookGates))+" runbook gate(s), below the expected minimum "+strconv.Itoa(minGates)+"; the release gate fails closed")
	}
	if len(workflowGates) < minGates {
		block(cfg.Workflow, "gate_lockstep parsed "+strconv.Itoa(len(workflowGates))+" '"+cfg.Job+"' gate(s) from "+cfg.Workflow+", below the expected minimum "+strconv.Itoa(minGates)+"; the release gate fails closed")
	}

	inWorkflow := make(map[string]bool, len(workflowGates))
	for _, g := range workflowGates {
		inWorkflow[g] = true
	}
	inRunbook := make(map[string]bool, len(runbookGates))
	for _, g := range runbookGates {
		inRunbook[g] = true
	}
	for _, g := range runbookGates {
		if !inWorkflow[g] {
			block(cfg.Runbook, "runbook lists deterministic gate '"+g+"' that is not a '"+cfg.Job+"' step in "+cfg.Workflow)
		}
	}
	for _, g := range workflowGates {
		if !inRunbook[g] {
			block(cfg.Workflow, cfg.Workflow+" '"+cfg.Job+"' step '"+g+"' is missing from the runbook's deterministic gates ("+cfg.Runbook+")")
		}
	}
	return out, nil
}

// runbookGateList extracts the numbered items under the runbook's "Deterministic
// gates" heading (case-insensitive), stopping at the next heading, with the same
// surrounding-quote normalization the workflow side uses so identical gates never
// look different. A missing file yields nil — the caller has already failed it
// closed.
func runbookGateList(repoRoot, rel string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var gates []string
	inSection := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			inSection = strings.Contains(strings.ToLower(trimmed), "deterministic gate")
			continue
		}
		if !inSection {
			continue
		}
		if m := numberedItemRe.FindStringSubmatch(line); m != nil {
			gates = append(gates, strings.Trim(strings.TrimSpace(m[1]), `"'`))
		}
	}
	return gates, nil
}

// workflowStepNames extracts the step names of the named job, dropping configured
// setup steps. It scopes to the `jobs:` block (so 2-space keys under on:/
// permissions:/concurrency: never masquerade as jobs), identifies a job only by a
// strict 2-space header regex (so a comment such as `  # NOTE:` cannot close a
// job early), and captures each step's name wherever it appears in the step block
// — inline `- name:` OR a following `name:` line after `- uses:` — so the
// alternate step form is not invisible. A missing file yields nil; the caller has
// already failed it closed.
func workflowStepNames(repoRoot, rel, job string, ignore []string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ignored := make(map[string]bool, len(ignore))
	for _, s := range ignore {
		ignored[s] = true
	}

	var names []string
	seenJobs, inJob, inSteps, inStep := false, false, false, false
	var cur string
	// Column where the current step's own mapping keys sit (the marker's content
	// column). A `name:` deeper than this belongs to a nested mapping (e.g.
	// `with: name:`) and is NOT the step's name.
	stepKeyCol := -1
	flush := func() {
		if inStep {
			name := strings.Trim(strings.TrimSpace(cur), `"'`)
			if name != "" && !ignored[name] {
				names = append(names, name)
			}
		}
		cur, inStep = "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !seenJobs {
			if jobsSectionRe.MatchString(line) {
				seenJobs = true
			}
			continue
		}
		if m := jobHeaderRe.FindStringSubmatch(line); m != nil {
			flush()
			inJob, inSteps = m[1] == job, false
			continue
		}
		if !inJob {
			continue
		}
		if stepsKeyRe.MatchString(line) {
			inSteps = true
			continue
		}
		if !inSteps {
			continue
		}
		if m := stepMarkerRe.FindStringSubmatch(line); m != nil {
			flush()
			inStep = true
			// The step's keys align at the marker's content column, so a later
			// `name:` at that exact column is the step's name; anything deeper is nested.
			stepKeyCol = strings.Index(line, m[1])
			if content := m[1]; strings.HasPrefix(content, "name:") {
				cur = strings.TrimSpace(strings.TrimPrefix(content, "name:"))
			}
			continue
		}
		if inStep && cur == "" {
			if nm := stepNameKeyRe.FindStringSubmatch(line); nm != nil {
				if indent := len(line) - len(strings.TrimLeft(line, " ")); indent == stepKeyCol {
					cur = strings.TrimSpace(nm[1])
				}
			}
		}
	}
	flush()
	return names, nil
}

// tokenCheck is a compiled BannedToken ready for line matching.
type tokenCheck struct {
	token   BannedToken
	pattern *regexp.Regexp
	allow   []*regexp.Regexp
}

// ValidateBannedToken reports whether one banned-token entry is loadable by THIS
// linter — the engine that enforces the public banlist layer. It exists so a writer
// of that config (`abcd banlist add --public`) validates through the same compile
// the gate itself performs, on the exact string it is about to store, rather than
// through a second reading that could accept an entry the gate then refuses. The
// error is returned for the caller to discard or wrap: it quotes the expression,
// which is safe for a public pattern and never safe for a private one.
func ValidateBannedToken(t BannedToken) error {
	_, err := compileTokens([]BannedToken{t})
	return err
}

func compileTokens(tokens []BannedToken) ([]tokenCheck, error) {
	checks := make([]tokenCheck, 0, len(tokens))
	for _, t := range tokens {
		pat, err := regexp.Compile(t.Pattern)
		if err != nil {
			return nil, err
		}
		allow := make([]*regexp.Regexp, 0, len(t.AllowContext))
		for _, a := range t.AllowContext {
			re, err := regexp.Compile(a)
			if err != nil {
				return nil, err
			}
			allow = append(allow, re)
		}
		checks = append(checks, tokenCheck{token: t, pattern: pat, allow: allow})
	}
	return checks, nil
}

// bannedTokenMessage renders a banned-token finding message, auto-citing the
// entry's machine-readable successor (iss-51 decision c) so the finding always
// tells the reader what to use instead. LoadConfig guarantees a non-empty
// successor, so the citation is always present.
func bannedTokenMessage(t BannedToken) string {
	return t.Message + " (successor: " + t.Successor + ")"
}

// checkBannedTokens implements check family A.
func checkBannedTokens(rel string, lines []string, mask []bool, checks []tokenCheck) []Finding {
	var out []Finding
	for i, line := range lines {
		for _, c := range checks {
			if c.token.skipFences() && mask[i] {
				continue
			}
			if !c.pattern.MatchString(line) {
				continue
			}
			if matchesAny(c.allow, line) {
				continue
			}
			out = append(out, Finding{
				File: rel, Line: i + 1, RuleID: c.token.ID,
				Severity: c.token.Severity, Message: bannedTokenMessage(c.token),
			})
		}
	}
	return out
}

// checkGitMetadata implements check family B.
func checkGitMetadata(rel string, lines []string, cfg RuleConfig) []Finding {
	banned := make(map[string]bool, len(cfg.Fields))
	for _, f := range cfg.Fields {
		banned[f] = true
	}
	var out []Finding
	for key, field := range frontmatterFields(lines) {
		if banned[key] {
			out = append(out, Finding{
				File: rel, Line: field.line, RuleID: "no_git_metadata",
				Severity: cfg.Severity,
				Message:  "git-inferable metadata key '" + key + "' in frontmatter; git log/blame is canonical",
			})
		}
	}
	return out
}

// checkLinks implements check family C.
func checkLinks(rel, fileAbs, repoRoot string, lines []string, mask []bool, cfg RuleConfig) []Finding {
	fileDir := filepath.Dir(fileAbs)
	var out []Finding
	for i, line := range lines {
		if mask[i] {
			continue
		}
		for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
			target := strings.TrimSpace(m[1])
			if target == "" || strings.HasPrefix(target, "#") ||
				strings.HasPrefix(target, "//") || schemeRe.MatchString(target) {
				continue
			}
			// Strip anchor/query before resolving to a filesystem path.
			if idx := strings.IndexAny(target, "#?"); idx >= 0 {
				target = target[:idx]
			}
			if target == "" {
				continue
			}
			resolved := filepath.Join(fileDir, target)
			relTo, err := filepath.Rel(repoRoot, resolved)
			if err != nil || relTo == ".." || strings.HasPrefix(relTo, ".."+string(filepath.Separator)) {
				out = append(out, Finding{
					File: rel, Line: i + 1, RuleID: "links_resolve",
					Severity: cfg.Severity,
					Message:  "link target escapes repo root: " + m[1],
				})
				continue
			}
			if _, err := os.Stat(resolved); err != nil {
				out = append(out, Finding{
					File: rel, Line: i + 1, RuleID: "links_resolve",
					Severity: cfg.Severity,
					Message:  "link target does not resolve: " + m[1],
				})
			}
		}
	}
	return out
}

// checkBrittleRefs implements check family D.
func checkBrittleRefs(rel string, lines []string, mask []bool, cfg RuleConfig) []Finding {
	var out []Finding
	for i, line := range lines {
		if mask[i] {
			continue
		}
		for _, m := range brittleRefRe.FindAllString(line, -1) {
			out = append(out, Finding{
				File: rel, Line: i + 1, RuleID: "no_brittle_line_refs",
				Severity: cfg.Severity,
				Message:  "brittle line reference '" + m + "'; use a section anchor instead",
			})
		}
	}
	return out
}

// checkDirectoryCoverage implements check family E.
func checkDirectoryCoverage(repoRoot, rootAbs string, cfg RuleConfig) ([]Finding, error) {
	var out []Finding
	err := filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		rel := repoRel(repoRoot, path)
		if matchesGlob(cfg.Exempt, rel) {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, "README.md")); err != nil {
			out = append(out, Finding{
				File: rel, Line: 0, RuleID: "directory_coverage",
				Severity: cfg.Severity,
				Message:  "directory has no README.md",
			})
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

// intentRecord is one intent file as the intent-tree rules see it: its
// repo-relative path, the bucket directory that IS its lifecycle state, its
// basename (which carries the id), and its parsed frontmatter.
type intentRecord struct {
	rel    string
	bucket string
	name   string
	fields map[string]fmField
	// preamble is the 1-based line the leading `---` sits on when something
	// precedes it, 0 otherwise — the extraction half of the loader contract.
	preamble int
}

// intentTree is ONE scan of the intent buckets, shared by every rule that reads
// the tree (intent_lifecycle, intent_impact_valid). records holds EVERY intent
// file in a declared bucket: both rules over this tree are schema rules (which
// bucket demands which fields, which impact values are legal), and a schema rule
// is not excused by the content exemption — that is what let two intents sit in
// superseded/ carrying no supersession at all (iss-39). known and idFiles index
// every intent file found anywhere beneath the tree, because a superseded_by
// target and an id collision must resolve against the whole corpus.
type intentTree struct {
	records []intentRecord
	known   map[string]bool
	idFiles map[string][]string
}

// intentsDirOf resolves a rule's intents subdirectory (relative to a record
// root), defaulting to "intents". It is shared so every intent-tree rule spells
// the default once and two rules configured alike scan one tree, not two.
func intentsDirOf(cfg RuleConfig) string {
	if cfg.IntentsDir == "" {
		return "intents"
	}
	return cfg.IntentsDir
}

// scanIntentTree reads the intent buckets once. A missing intents/ directory is
// soft (no records, no error), mirroring the rest of the record lint.
func scanIntentTree(repoRoot, rootAbs, intentsDir string) (intentTree, error) {
	intentsRoot := filepath.Join(rootAbs, intentsDir)
	if _, err := os.Stat(intentsRoot); err != nil {
		return intentTree{}, nil
	}

	// Collect every intent id that exists as a file in any bucket, so the
	// superseded_by target can be checked for existence. Track the files each id
	// claims: ids are the intent's identity across the record, and parallel
	// branches each allocating "the next free id" collide silently otherwise.
	known := map[string]bool{}
	idFiles := map[string][]string{}
	_ = filepath.WalkDir(intentsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if intentFileRe.MatchString(d.Name()) {
			id := canonRecordID(intentIDRe.FindString(d.Name()))
			known[id] = true
			idFiles[id] = append(idFiles[id], path)
		}
		return nil
	})

	buckets, err := os.ReadDir(intentsRoot)
	if err != nil {
		return intentTree{}, err
	}
	tree := intentTree{known: known, idFiles: idFiles}
	for _, b := range buckets {
		if !b.IsDir() || !intentBuckets[b.Name()] {
			continue
		}
		bucket := b.Name()
		bucketDir := filepath.Join(intentsRoot, bucket)
		entries, err := os.ReadDir(bucketDir)
		if err != nil {
			return intentTree{}, err
		}
		for _, e := range entries {
			if e.IsDir() || !intentFileRe.MatchString(e.Name()) {
				continue
			}
			fileAbs := filepath.Join(bucketDir, e.Name())
			content, err := os.ReadFile(fileAbs)
			if err != nil {
				return intentTree{}, err
			}
			rel := repoRel(repoRoot, fileAbs)
			lines := strings.Split(string(content), "\n")
			fields := frontmatterFields(lines)
			tree.records = append(tree.records, intentRecord{
				rel: rel, bucket: bucket, name: e.Name(), fields: fields,
				preamble: preambleLine(lines),
			})
		}
	}
	return tree, nil
}

// checkIntentLifecycle implements check family F over an already-scanned tree.
func checkIntentLifecycle(repoRoot string, tree intentTree, cfg RuleConfig) []Finding {
	var out []Finding
	for _, r := range tree.records {
		out = append(out, validateIntent(r.rel, r.bucket, r.fields, tree.known, r.preamble, cfg.Severity)...)
		out = append(out, validateIntentIDUnique(repoRoot, r.rel, r.name, r.fields, tree.idFiles, cfg.Severity)...)
	}
	return out
}

// checkIntentImpact implements intent_impact_valid: an intent may not sit in
// shipped/ without a valid, non-internal impact (spc-10, plan outcome 8).
//
// WHY the gate is the move into shipped/ and nothing earlier: the judgement is
// made when the intent is shaped, but it only becomes load-bearing when the
// intent enters the release set the version is derived from, and a draft nobody
// has designed yet cannot answer it honestly. Requiring it in drafts/ would
// trade a real signal for ceremony.
//
// WHY a value present OUTSIDE shipped/ is still validated: absence in drafts/ is
// "not decided yet", but `impact: braking` is not an undecided judgement, it is
// a wrong one. Caught in the bucket where it was written it is a typo the author
// fixes; caught only at the shipped/ gate it is archaeology. The two failures
// are therefore split — presence is gated at the terminal bucket, legality
// everywhere.
//
// WHY internal fails on an intent although it is legal on an issue: an intent is
// press-release-first — it exists because a user can be told about it — so
// "invisible to users" is a category error on an intent rather than a valid
// judgement. Plumbing belongs on an issue, which may be internal.
//
// `impact: null` reads as absent, matching the rest of the intent schema, where
// null is how a record says "this field does not apply yet" (kind, spec_id).
func checkIntentImpact(tree intentTree, cfg RuleConfig) []Finding {
	var out []Finding
	for _, r := range tree.records {
		f := r.fields["impact"]
		line := f.line
		if line == 0 {
			line = 1
		}
		add := func(msg string) {
			out = append(out, Finding{
				File: r.rel, Line: line, RuleID: "intent_impact_valid",
				Severity: cfg.Severity, Message: msg,
			})
		}
		if isNull(f.value) {
			if r.bucket == "shipped" {
				add("shipped: impact must be set explicitly to one of " + intentImpactValues +
					" — it decides the derived version, and there is no default")
			}
			continue
		}
		impact, err := changelog.ParseImpact(f.value)
		if err != nil {
			add("invalid impact '" + f.value + "': an intent must declare exactly one of " +
				intentImpactValues + " (lower-case, no surrounding whitespace)")
			continue
		}
		if impact == changelog.ImpactInternal {
			add("impact must not be internal on an intent: a press-release-first intent is " +
				"user-facing by definition — declare one of " + intentImpactValues +
				", or record the work as an issue instead")
		}
	}
	return out
}

// validateIntentIDUnique flags a duplicated intent id, delegating to the shared
// validateIDUnique primitive (the issue-id rule uses the same logic).
func validateIntentIDUnique(repoRoot, rel, name string, fields map[string]fmField, idFiles map[string][]string, severity string) []Finding {
	id := canonRecordID(intentIDRe.FindString(name))
	return validateIDUnique(repoRoot, rel, id, "intent", "intent_lifecycle", severity, fields, idFiles)
}

// validateIDUnique flags every file in a colliding id set, not just one: the
// linter cannot know which claimant is authoritative, and flagging a single file
// would imply the others are fine. It is the one primitive behind both the
// intent-id (intent_lifecycle) and issue-id (issue_id_unique) uniqueness rules —
// an id is a record's identity across its register and must be unique within it.
// noun names the record kind in the message; ruleID and severity tag the emitted
// Finding. idFiles maps each id to the repo-absolute paths that claim it.
func validateIDUnique(repoRoot, rel, id, noun, ruleID, severity string, fields map[string]fmField, idFiles map[string][]string) []Finding {
	claimants := idFiles[id]
	if len(claimants) < 2 {
		return nil
	}
	others := make([]string, 0, len(claimants)-1)
	for _, p := range claimants {
		if r := repoRel(repoRoot, p); r != rel {
			others = append(others, r)
		}
	}
	sort.Strings(others)

	line := 1
	if f, ok := fields["id"]; ok && f.line > 0 {
		line = f.line
	}
	return []Finding{{
		File: rel, Line: line, RuleID: ruleID, Severity: severity,
		Message: "duplicate " + noun + " id " + id + "; also claimed by " + strings.Join(others, ", ") +
			" — an id is the " + noun + "'s identity across the record and must be unique",
	}}
}

// issueRecord is one issue file as the ledger rules see it: its repo-relative
// path, the status directory that IS its lifecycle state, its iss-N id, and its
// parsed frontmatter.
type issueRecord struct {
	rel    string
	status string
	id     string
	fields map[string]fmField
}

// issueLedger is ONE scan of the issue ledger's status directories, shared by
// every rule that reads it (issue_id_unique, issue_impact_valid). idFiles maps
// each iss-N id to the repo-absolute paths claiming it.
type issueLedger struct {
	records []issueRecord
	idFiles map[string][]string
}

// issuesDirOf resolves a rule's issue-ledger root (repo-relative), defaulting to
// .abcd/work/issues. It is shared so every ledger rule spells the default once
// and two rules configured alike scan one ledger, not two.
func issuesDirOf(cfg RuleConfig) string {
	if cfg.IssuesDir == "" {
		return ".abcd/work/issues"
	}
	return cfg.IssuesDir
}

// scanIssueLedger reads the ledger's status directories (open/, resolved/,
// wontfix/) once. The ledger lives outside cfg.Roots, so its rules run once, not
// per-root. A missing ledger, or a missing status directory within it, is not an
// error — it simply contributes no records.
func scanIssueLedger(repoRoot, issuesDir string) (issueLedger, error) {
	issuesRoot := filepath.Join(repoRoot, issuesDir)
	if _, err := os.Stat(issuesRoot); err != nil {
		if os.IsNotExist(err) {
			return issueLedger{}, nil
		}
		return issueLedger{}, err
	}

	// Track every file each iss-N id claims across the three status dirs: an id is
	// the issue's identity across the ledger, and a bypassed-allocator file (or two
	// parallel branches) can land the same id silently otherwise.
	ledger := issueLedger{idFiles: map[string][]string{}}
	type pending struct{ status, abs string }
	var files []pending
	for _, sub := range issueStatusDirs {
		entries, err := os.ReadDir(filepath.Join(issuesRoot, sub))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return issueLedger{}, err
		}
		for _, e := range entries {
			if e.IsDir() || !issueFileRe.MatchString(e.Name()) {
				continue
			}
			fileAbs := filepath.Join(issuesRoot, sub, e.Name())
			id := canonRecordID(issueIDRe.FindString(e.Name()))
			ledger.idFiles[id] = append(ledger.idFiles[id], fileAbs)
			files = append(files, pending{status: sub, abs: fileAbs})
		}
	}

	for _, p := range files {
		content, err := os.ReadFile(p.abs)
		if err != nil {
			return issueLedger{}, err
		}
		ledger.records = append(ledger.records, issueRecord{
			rel:    repoRel(repoRoot, p.abs),
			status: p.status,
			id:     canonRecordID(issueIDRe.FindString(filepath.Base(p.abs))),
			fields: frontmatterFields(strings.Split(string(content), "\n")),
		})
	}
	return ledger, nil
}

// canonRecordID normalises a matched record id to its canonical spelling by
// stripping leading zeros from the numeric part (iss-0100 → iss-100), matching
// recordid.fileID. It governs every record-id keyspace the record-lint rules
// build or query — the uniqueness backstops (issue, intent, spec), the
// delivery-state bucket map and its CHANGELOG citations, and the intent
// existence/supersession lookups — so a zero-padded spelling and its canonical
// twin are one handle on both the key and the lookup side. Without it a padded
// file or citation keys separately: a uniqueness backstop fails open on the
// twin, and a lookup gate either misses (fail-open) or false-blocks (iss-392).
func canonRecordID(match string) string {
	m := recordIDRe.FindStringSubmatch(match)
	if m == nil {
		return match
	}
	return m[1] + "-" + m[2]
}

// checkIssueIDUnique flags any iss-N id claimed by two or more files across the
// issue ledger's status directories (open/, resolved/, wontfix/). The capture
// allocator rejects a duplicate on the reservation path, but a hand-added issue
// file that bypassed it — how a past iss-56 collision arose — slips straight
// through; this is the record-lint backstop that catches it. It is the issue-side
// mirror of the intent-id uniqueness rule and shares the same validateIDUnique
// primitive.
func checkIssueIDUnique(repoRoot string, ledger issueLedger, cfg RuleConfig) []Finding {
	var out []Finding
	for _, r := range ledger.records {
		out = append(out, validateIDUnique(repoRoot, r.rel, r.id, "issue", "issue_id_unique", cfg.Severity, r.fields, ledger.idFiles)...)
	}
	return out
}

// checkIssueImpact implements issue_impact_valid: an issue may not sit in
// resolved/ without a valid impact (spc-10, plan outcome 8).
//
// It is the issue-side mirror of intent_impact_valid, with one deliberate
// difference: internal is a LEGAL judgement here. Most resolved issues are
// plumbing — a TOCTOU fix, an atomic-write hardening, a lint internal — that no
// user can be told about, and a rule that refused internal would either force
// that work into a user-facing changelog or push authors into a mislabel.
//
// Absence gates only the terminal bucket (resolved/), while legality is checked
// wherever a value is written, for the same reason as on the intent side: an
// open issue has not been judged yet, but a misspelt judgement is wrong the
// moment it is typed. `impact: null` reads as absent, matching the schema
// convention that null means "does not apply yet". The message for a malformed
// value is the shared parser's own, so the lint and the derivation describe the
// same defect in the same words.
func checkIssueImpact(ledger issueLedger, cfg RuleConfig) []Finding {
	var out []Finding
	for _, r := range ledger.records {
		f := r.fields["impact"]
		line := f.line
		if line == 0 {
			line = 1
		}
		add := func(msg string) {
			out = append(out, Finding{
				File: r.rel, Line: line, RuleID: "issue_impact_valid",
				Severity: cfg.Severity, Message: msg,
			})
		}
		if isNull(f.value) {
			if r.status == "resolved" {
				add("resolved: impact must be set explicitly to one of " + issueImpactValues +
					" — it decides the derived version and the changelog line, and there is no default")
			}
			continue
		}
		if _, err := changelog.ParseImpact(f.value); err != nil {
			add(err.Error())
		}
	}
	return out
}

func validateIntent(rel, bucket string, fields map[string]fmField, known map[string]bool, preamble int, severity string) []Finding {
	var out []Finding
	add := func(line int, msg string) {
		if line == 0 {
			line = 1
		}
		out = append(out, Finding{
			File: rel, Line: line, RuleID: "intent_lifecycle",
			Severity: severity, Message: msg,
		})
	}

	// ANY bucket: the frontmatter must open on line 1, because that is the only
	// place intent.Load looks for it. This rule reads the record through the
	// tolerant scanner, so without this check a preamble-led record hands a
	// perfect id to the gate and nothing at all to the loader
	// (iss-2608270926031827).
	if preamble > 0 {
		add(1, preambleMessage("intent", preamble))
	}

	// ANY bucket: the id must be the one the loader will accept. intent.Load
	// fail-closes the WHOLE corpus on a malformed id, so a record this rule passed
	// with an absent/empty/null id merged green and then broke every intent verb —
	// `abcd spec close` included — for everyone who pulled it. The predicate is
	// recordid.ValidIntentID, the same call intent.Validate makes, never a second
	// pattern that could drift (iss-2608270500198764). The filename↔id agreement
	// check in record_schema does NOT cover this: it returns early on an absent
	// value, a tolerance that is load-bearing for filename-derived ADR ids.
	idField := fields["id"]
	if !recordid.ValidIntentID(idField.value) {
		add(idField.line, "intent id must be present and match ^itd-\\d+$ (got '"+idField.value+"')")
	}

	// ANY bucket: lifecycle state is the directory, never a status: field.
	if st, ok := fields["status"]; ok {
		add(st.line, "status: key forbidden; lifecycle state is the bucket directory, not a field")
	}

	kind, kindOK := fields["kind"]
	spec, specOK := fields["spec_id"]
	kindNull := !kindOK || isNull(kind.value)
	specNull := !specOK || isNull(spec.value)
	kindVal := kind.value

	switch bucket {
	case "drafts":
		if !(kindNull || kindVal == "standalone" || kindVal == "bundle-member") {
			add(kind.line, "drafts: kind must be null, standalone, or bundle-member (got '"+kindVal+"')")
		}
		if !specNull {
			add(spec.line, "drafts: spec_id must be null")
		}
	case "planned":
		if kindNull || !(kindVal == "standalone" || kindVal == "bundle-member") {
			add(kind.line, "planned: kind must be standalone or bundle-member (non-null)")
		}
		// Re-baselined for the Go build: a planned intent is committed to build, but
		// its native spec_id is assigned only when the spec layer schedules it
		// (Phase 4). So spec_id may be null (unscheduled) or a spc- id — never other.
		if !specNull && !specIDRe.MatchString(spec.value) {
			add(spec.line, "planned: spec_id must be null (unscheduled) or match ^spc- (got '"+spec.value+"')")
		}
	case "shipped":
		if kindNull || !(kindVal == "standalone" || kindVal == "bundle-member") {
			add(kind.line, "shipped: kind must be standalone or bundle-member (non-null)")
		}
		if specNull {
			add(spec.line, "shipped: spec_id must be non-null")
		}
	case "disciplines":
		if kindVal != "discipline" {
			add(kind.line, "disciplines: kind must be discipline (got '"+kindVal+"')")
		}
		if !specNull {
			add(spec.line, "disciplines: spec_id must be null")
		}
	case "superseded":
		// The successor may be a later intent or the ADR that redecided the
		// question (iss-39). Only an intent target is resolved here — the intent
		// tree is the only corpus this rule reads; a cross-store target's existence
		// is record_schema's, which reads every store.
		sup, supOK := fields["superseded_by"]
		if !supOK || !supersededRe.MatchString(sup.value) {
			add(sup.line, "superseded: superseded_by must be present and match ^(itd|adr)-\\d+")
		} else if id := canonRecordID(intentIDRe.FindString(sup.value)); id != "" && !known[id] {
			add(sup.line, "superseded: superseded_by target '"+id+"' does not exist in any bucket")
		}
		if kindNull {
			add(kind.line, "superseded: kind must be non-null")
		}
	}
	return out
}

// checkSpecLifecycle is the spec-side mirror of checkIntentLifecycle (check
// family G). It validates that each spec under specs/{open,closed}/ has a
// well-formed id/slug/intent link, that the named intent EXISTS in the corpus,
// and — the load-bearing cross-check — that the intent points back at this spec
// (bidirectional agreement). A missing specs/ directory is soft.
//
// The two trees are read by ScanSpecLinks, the one traversal of the intent
// buckets and the spec store, so this rule and the release cut's stale-intent
// refusal can never disagree about what the record says.
func checkSpecLifecycle(repoRoot, rootAbs string, cfg RuleConfig, top Config) ([]Finding, error) {
	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "specs"
	}
	if _, err := os.Stat(filepath.Join(rootAbs, specsDir)); err != nil {
		return nil, nil // missing specs/ is soft, mirroring intent_lifecycle
	}

	rootRel, err := filepath.Rel(repoRoot, rootAbs)
	if err != nil {
		return nil, err
	}
	idx, err := ScanSpecLinks(repoRoot,
		filepath.ToSlash(filepath.Join(rootRel, intentsDirOf(cfg))),
		filepath.ToSlash(filepath.Join(rootRel, specsDir)), top)
	if err != nil {
		return nil, err
	}
	knownIntent := idx.KnownIntents()
	intentSpecID := idx.IntentSpecID()

	var out []Finding
	for _, spec := range idx.Specs {
		if spec.exempt {
			// A content-exempt spec is excused from how it is WRITTEN, never from
			// being well-formed: spec.Load validates id and intent on every file and
			// aborts the whole store on one failure, with no exemption concept at all
			// (iss-2608270500207987).
			out = append(out, validateSpecWellFormed(spec.Path, spec.fields, spec.preamble, cfg.Severity)...)
			continue
		}
		out = append(out, validateSpec(spec.Path, spec.fields, knownIntent, intentSpecID, spec.preamble, cfg.Severity)...)
	}
	return out, nil
}

// checkSpecIDUnique flags any spc-N id claimed by two or more spec-store files
// across specs/{open,closed}/. The spec mint is timestamp-numeric and consults
// no maximum (adr-45), so two branches minting in the same window differ by
// entropy; what remains is the accepted same-second, same-suffix residue and a
// hand-added spec file, which bypasses the mint entirely. This is the armed
// scheme assertion (adr-45 ruling 5) that CI runs on the merged PR's union,
// mirroring issue_id_unique/intent_lifecycle and sharing the one
// validateIDUnique primitive. A malformed or absent id is spec_lifecycle's
// concern, not this rule's, so only well-formed spc-N ids are compared; a
// content-exempt spec is skipped exactly as spec_lifecycle skips it.
func checkSpecIDUnique(repoRoot, rootAbs string, cfg RuleConfig, top Config) ([]Finding, error) {
	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "specs"
	}
	if _, err := os.Stat(filepath.Join(rootAbs, specsDir)); err != nil {
		return nil, nil // missing specs/ is soft, mirroring spec_lifecycle
	}
	rootRel, err := filepath.Rel(repoRoot, rootAbs)
	if err != nil {
		return nil, err
	}
	idx, err := ScanSpecLinks(repoRoot,
		filepath.ToSlash(filepath.Join(rootRel, intentsDirOf(cfg))),
		filepath.ToSlash(filepath.Join(rootRel, specsDir)), top)
	if err != nil {
		return nil, err
	}

	// Track every file each well-formed spc-N id claims, then flag every member of
	// a set of size > 1 via the shared primitive.
	idFiles := map[string][]string{}
	for _, spec := range idx.Specs {
		if spec.exempt {
			continue
		}
		raw := spec.fields["id"].value
		if !recordid.ValidSpecID(raw) {
			continue
		}
		// Canonical key so spc-005 and spc-5 collide as one id (iss-392's
		// keyspace) — else the uniqueness backstop fails open on a padded twin.
		id := canonRecordID(raw)
		idFiles[id] = append(idFiles[id], filepath.Join(repoRoot, spec.Path))
	}

	var out []Finding
	for _, spec := range idx.Specs {
		if spec.exempt {
			continue
		}
		raw := spec.fields["id"].value
		if !recordid.ValidSpecID(raw) {
			continue
		}
		out = append(out, validateIDUnique(repoRoot, spec.Path, canonRecordID(raw), "spec", "spec_id_unique", cfg.Severity, spec.fields, idFiles)...)
	}
	return out, nil
}

// validateSpecWellFormed is the loader-parity subset of spec_lifecycle: exactly
// what spec.Validate enforces before spec.Load will trust the record — a
// well-formed id and a well-formed intent back-link — and nothing about how the
// record is WRITTEN.
//
// It is split out because it is the one part of this rule the content exemption
// must not reach. spec.Load has no exemption concept: it validates every spec's
// id and intent unconditionally and aborts the WHOLE store on one failure, so a
// `status: superseded` spec that skipped this rule entirely merged green and
// then broke `abcd <spc-N>` dispatch, spec minting and every intent verb that
// loads specs (iss-2608270500207987). Being historical excuses a record from how
// it is written, never from being well-formed (iss-39).
func validateSpecWellFormed(rel string, fields map[string]fmField, preamble int, severity string) []Finding {
	var out []Finding
	add := func(line int, msg string) {
		if line == 0 {
			line = 1
		}
		out = append(out, Finding{
			File: rel, Line: line, RuleID: "spec_lifecycle",
			Severity: severity, Message: msg,
		})
	}
	if preamble > 0 {
		add(1, preambleMessage("spec", preamble))
	}
	id := fields["id"]
	if !recordid.ValidSpecID(id.value) {
		add(id.line, "spec id must be present and match ^spc-\\d+$ (got '"+id.value+"')")
	}
	intent := fields["intent"]
	if !recordid.ValidIntentID(intent.value) {
		add(intent.line, "spec intent link must be present and match ^itd-\\d+$ (got '"+intent.value+"')")
	}
	return out
}

func validateSpec(rel string, fields map[string]fmField, knownIntent map[string]bool, intentSpecID map[string]string, preamble int, severity string) []Finding {
	// The well-formedness subset first, so the id/intent patterns and the
	// frontmatter-placement rule are stated once for both the exempt and the
	// non-exempt path.
	out := validateSpecWellFormed(rel, fields, preamble, severity)
	add := func(line int, msg string) {
		if line == 0 {
			line = 1
		}
		out = append(out, Finding{
			File: rel, Line: line, RuleID: "spec_lifecycle",
			Severity: severity, Message: msg,
		})
	}

	// Status is the bucket directory (open/closed), never a field.
	if st, ok := fields["status"]; ok {
		add(st.line, "status: key forbidden; spec status is the bucket directory (open/closed), not a field")
	}

	id := fields["id"]
	idValid := recordid.ValidSpecID(id.value)
	if slug, ok := fields["slug"]; !ok || isNull(slug.value) {
		add(slug.line, "spec slug must be present")
	}

	intent := fields["intent"]
	if !recordid.ValidIntentID(intent.value) {
		return out // no existence/agreement check possible without a well-formed link
	}
	if !knownIntent[canonRecordID(intent.value)] {
		add(intent.line, "spec intent '"+intent.value+"' does not exist in any bucket")
		return out
	}
	// Bidirectional agreement: the named intent must carry spec_id == this spec's
	// id. Drift either way (the intent points elsewhere, or at null) is flagged.
	if idValid {
		back := intentSpecID[canonRecordID(intent.value)]
		if specNum(back) != specNum(id.value) {
			add(intent.line, "bidirectional drift: spec '"+id.value+"' names intent '"+intent.value+"' but that intent's spec_id is '"+back+"'")
		}
	}
	return out
}

// specNum extracts the numeric N from a spec id or spec_id value (which may carry
// a trailing slug, e.g. spc-1-thing), or -1 when there is no spc- id at all. It
// lets the bidirectional check compare spc-1 against a reserved spec_id: spc-1
// written with or without a slug suffix.
func specNum(v string) int {
	if !strings.HasPrefix(v, "spc-") {
		return -1
	}
	rest := v[len("spc-"):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return -1
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return -1
	}
	return n
}

// fmField is a frontmatter key's value and 1-based source line.
// checkForbiddenSynonyms implements the GL002 forbidden-synonym family. It reads
// the glossary term files under cfg.GlossaryDir (the single source of truth for
// what a forbidden synonym is), then flags live prose that uses an *enforced*
// synonym as a standalone word. Enforcement is scoped to cfg.Enforce because most
// forbidden synonyms are common English words; each enforced entry must be a
// declared forbidden_synonym or the config is rejected (an error), so the config
// can never gate a word the glossary does not forbid.
//
// Matching is case-insensitive with explicit Unicode word boundaries — Go's
// regexp \b is ASCII-only, so a run of unicode letters adjacent to the synonym
// (é, а, …) would otherwise read as a boundary and leak a false hit. Code spans
// (fenced and inline single-backtick), YAML frontmatter, exempt path prefixes, the
// glossary directory itself, and any line matching an allow_context regexp are all
// out of scope: a term file names its own forbidden synonyms legitimately, and a
// code span or external token (`epic-review`) is a mention, not a substitution.
func checkForbiddenSynonyms(repoRoot, rootAbs string, cfg RuleConfig) ([]Finding, error) {
	glossaryDir := cfg.GlossaryDir
	if glossaryDir == "" {
		glossaryDir = ".abcd/development/brief/glossary"
	}
	forbidden, canonical, err := loadForbiddenSynonyms(repoRoot, glossaryDir)
	if err != nil {
		return nil, err
	}

	// Compile one boundary-aware matcher per enforced synonym, rejecting any the
	// glossary does not actually forbid (the glossary is the source of truth).
	type synMatcher struct {
		word string
		re   *regexp.Regexp
	}
	var matchers []synMatcher
	for _, s := range cfg.Enforce {
		key := strings.ToLower(strings.TrimSpace(s))
		if key == "" {
			continue
		}
		if !forbidden[key] {
			return nil, &configError{"forbidden_synonyms: enforced word " + strconv.Quote(s) +
				" is not declared as a forbidden_synonym by any glossary term under " + glossaryDir}
		}
		matchers = append(matchers, synMatcher{word: key, re: regexp.MustCompile("(?i)" + regexp.QuoteMeta(key))})
	}
	if len(matchers) == 0 {
		return nil, nil
	}

	allow := make([]*regexp.Regexp, 0, len(cfg.AllowContext))
	for _, a := range cfg.AllowContext {
		re, err := regexp.Compile(a)
		if err != nil {
			return nil, err
		}
		allow = append(allow, re)
	}

	glossaryPrefix := filepath.ToSlash(glossaryDir)
	files, err := markdownFiles(rootAbs)
	if err != nil {
		return nil, err
	}

	var out []Finding
	for _, fileAbs := range files {
		rel := repoRel(repoRoot, fileAbs)
		relSlash := filepath.ToSlash(rel)
		if strings.HasPrefix(relSlash, glossaryPrefix) || hasAnyPrefix(relSlash, cfg.ExemptPrefixes) {
			continue
		}
		content, err := os.ReadFile(fileAbs)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(content), "\n")
		mask := fenceMask(lines)
		bodyStart := frontmatterBodyStart(lines)
		for i, line := range lines {
			if i < bodyStart || mask[i] {
				continue // YAML frontmatter and fenced code are not prose
			}
			if matchesAny(allow, line) {
				continue
			}
			stripped := stripInlineCode(line)
			for _, m := range matchers {
				for _, loc := range m.re.FindAllStringIndex(stripped, -1) {
					if !wordBoundaryAt(stripped, loc[0], loc[1]) {
						continue
					}
					term := canonical[m.word]
					out = append(out, Finding{
						File: rel, Line: i + 1, RuleID: "GL002", Severity: cfg.Severity,
						Message: "forbidden synonym '" + m.word + "' for glossary term '" + term +
							"' in live prose; use '" + term + "' (itd-43). If this is a mention (not a substitution), quote it in a code span.",
					})
					break // one finding per enforced word per line is enough signal
				}
			}
		}
	}
	return out, nil
}

// configError is a malformed-configuration error (an enforced synonym the glossary
// does not forbid), surfaced through Lint's error return like an uncompilable regexp.
type configError struct{ msg string }

func (e *configError) Error() string { return e.msg }

// loadForbiddenSynonyms walks the glossary directory and returns the set of
// forbidden synonyms (lower-cased) and a map from each synonym to its canonical
// term. A missing glossary directory yields empty maps, not an error.
func loadForbiddenSynonyms(repoRoot, glossaryDir string) (map[string]bool, map[string]string, error) {
	forbidden := map[string]bool{}
	canonical := map[string]string{}
	// glossary_dir is a cloned-repo-controlled config field, so this walk is
	// contained and guarded exactly like the main lint walk and CollectCitedURLs:
	// otherwise a "../secret" glossary_dir, or a committed symlink leaf inside it,
	// reads a file the repository does not own (or a "-> /dev/zero" link unbounded).
	if err := containedRepoPath(glossaryDir); err != nil {
		return nil, nil, &configError{"glossary_dir " + quote(glossaryDir) + " " + err.Error() +
			"; the lint reads only inside the repository"}
	}
	dirAbs := filepath.Join(repoRoot, glossaryDir)
	if err := resolvedInsideRoot(repoRoot, dirAbs); err != nil {
		return nil, nil, &configError{"glossary_dir " + quote(glossaryDir) + " " + err.Error() +
			"; the lint reads only inside the repository"}
	}
	files, err := markdownFiles(dirAbs)
	if err != nil {
		return nil, nil, err
	}
	for _, fileAbs := range files {
		realPath, err := containedRealPath(repoRoot, fileAbs)
		if err != nil {
			return nil, nil, &configError{"file " + quote(repoRel(repoRoot, fileAbs)) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		content, err := fsutil.ReadGuarded(realPath, citationPageSizeLimit)
		if err != nil {
			return nil, nil, err
		}
		lines := strings.Split(string(content), "\n")
		// frontmatterFields tolerates a glossary term file's leading attribution
		// comment (the mattpocock template) before the `---` block, so no manual
		// slice is needed here.
		fields := frontmatterFields(lines)
		term, ok := fields["term"]
		if !ok {
			continue
		}
		syns, ok := fields["forbidden_synonyms"]
		if !ok {
			continue
		}
		for _, s := range parseYAMLStringList(syns.value) {
			key := strings.ToLower(strings.TrimSpace(s))
			if key == "" {
				continue
			}
			forbidden[key] = true
			if _, seen := canonical[key]; !seen {
				canonical[key] = term.value
			}
		}
	}
	return forbidden, canonical, nil
}

// parseYAMLStringList parses an inline YAML flow sequence of strings, e.g.
// `["sprint", "milestone", "epic"]`, into its members.
//
// It is `frontmatter.StringList` under this package's own name: the glossary's
// index reads the same `aliases`/`forbidden_synonyms` fields this rule reads,
// and one reader of a field is how the two cannot come to disagree about what a
// term declares.
func parseYAMLStringList(v string) []string { return frontmatter.StringList(v) }

// frontmatterOpen returns the index of the opening `---` frontmatter delimiter,
// skipping leading blank lines and HTML comments (single- and multi-line); -1
// when the leading block is not frontmatter. It lets a term file carry an
// attribution comment above its `---`. The first line is BOM-stripped before
// every comparison: a UTF-8 byte-order mark is invisible to TrimSpace, so an
// untrimmed BOM ahead of the `---` (or ahead of a leading comment) would make a
// well-formed record read as having no frontmatter and slip every
// frontmatter-keyed blocker.
func frontmatterOpen(lines []string) int {
	norm := func(idx int) string {
		s := lines[idx]
		if idx == 0 {
			s = frontmatter.TrimBOM(s)
		}
		return strings.TrimSpace(s)
	}
	inComment := false
	for i := 0; i < len(lines); i++ {
		t := norm(i)
		switch {
		case inComment:
			// Inside a multi-line comment: consume lines until its close.
			if strings.Contains(t, "-->") {
				inComment = false
			}
		case t == "":
			// blank line: skip.
		case strings.HasPrefix(t, "<!--") && strings.HasSuffix(t, "-->"):
			// a complete single-line comment: skip.
		case strings.HasPrefix(t, "<!--"):
			// a multi-line comment opens here and does not close on this line.
			inComment = true
		case t == "---":
			return i
		default:
			return -1
		}
	}
	return -1
}

// frontmatterBodyStart returns the index of the first body line after a leading
// YAML frontmatter block (the line after the closing `---`); 0 when there is none.
// It shares frontmatterOpen's tolerance of a leading attribution comment, so a
// file whose frontmatter carries a `core/epic` term reference is never scanned as
// prose just because a comment precedes its `---`.
func frontmatterBodyStart(lines []string) int {
	open := frontmatterOpen(lines)
	if open < 0 {
		return 0
	}
	for j := open + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "---" {
			return j + 1
		}
	}
	return 0 // unterminated frontmatter: treat all as body rather than swallow the file
}

// stripInlineCode blanks the contents of single-backtick inline code spans (and
// their delimiters) so a forbidden synonym named inside a code span is a mention,
// not a match. It blanks matched backtick pairs only; a trailing unpaired backtick
// and its tail are left literal so an earlier, correctly-closed span stays blanked
// (double-backtick spans are out of scope — this masks single-backtick pairs).
func stripInlineCode(line string) string {
	b := []rune(line)
	out := make([]rune, len(b))
	copy(out, b)
	// open tracks the index of an as-yet-unclosed opening backtick; -1 when none.
	open := -1
	for i, r := range b {
		if r != '`' {
			continue
		}
		if open < 0 {
			open = i // provisional opener; blanked only once its pair closes
			continue
		}
		// Closing backtick: blank the delimiters and the span between them.
		for j := open; j <= i; j++ {
			out[j] = ' '
		}
		open = -1
	}
	// A leftover unpaired backtick (open >= 0) and its tail stay literal, so the
	// earlier paired spans that were already blanked are preserved.
	return string(out)
}

// wordBoundaryAt reports whether [start,end) in s is bounded by non-word runes on
// both sides. Word runes are Unicode letters, digits, and underscore — explicit
// because Go's regexp \b is ASCII-only and would treat a unicode-letter neighbour
// as a boundary, leaking "épic"/"epicа" style false hits past an ASCII \b.
func wordBoundaryAt(s string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(s[:start])
		if isWordRune(r) {
			return false
		}
	}
	if end < len(s) {
		r, _ := utf8.DecodeRuneInString(s[end:])
		if isWordRune(r) {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(s, filepath.ToSlash(p)) {
			return true
		}
	}
	return false
}

type fmField struct {
	value string
	line  int
}

// frontmatterFields returns the top-level keys of the leading YAML frontmatter
// (the block between the first two `---` lines) in lint's local fmField shape. It
// adapts the canonical scanner in internal/core/frontmatter, so the frontmatter
// line-scanner (and its delimiter handling) lives in ONE place, not a divergent
// copy here.
func frontmatterFields(lines []string) map[string]fmField {
	fields := map[string]fmField{}
	// A record file may carry a leading attribution comment before its `---`
	// block (the mattpocock glossary template), and the shared scanner requires
	// `---` on line 0 — so a comment-led file would otherwise yield no fields and
	// slip every frontmatter check (checkGitMetadata's no_git_metadata blocker,
	// contentExempt, scanRecordStores). Slice from the opening delimiter, as
	// loadForbiddenSynonyms and the glossary reader already do, and add the
	// skipped-line offset back so reported line numbers stay absolute.
	offset := 0
	if start := frontmatterOpen(lines); start > 0 {
		offset = start
		lines = lines[start:]
	}
	for key, f := range frontmatter.Fields(lines) {
		fields[key] = fmField{value: f.Value, line: f.Line + offset}
	}
	return fields
}

// preambleLine reports the 1-based line the leading `---` sits on when anything
// precedes it (blank lines, an attribution comment), and 0 when the delimiter
// opens the file or is absent altogether.
//
// It is the ONE place the extraction half of the lint↔loader contract is
// expressed. frontmatterFields deliberately tolerates a preamble so a
// comment-led file still yields its fields to every rule — but the intent and
// spec loaders call frontmatter.Fields on the RAW lines, which requires the
// delimiter on line 0 and otherwise returns NO fields at all. Reading the id
// through the tolerant scanner and never noticing the loader could not is how a
// record with a perfectly good id stayed lint-green and fail-closed the whole
// store (iss-2608270926031827). The two stores whose loaders fail closed refuse
// the shape; every other family keeps the tolerance, which is deliberate there.
func preambleLine(lines []string) int {
	if start := frontmatterOpen(lines); start > 0 {
		return start + 1
	}
	return 0
}

// preambleMessage is the finding text for a record whose frontmatter does not
// open on line 1, in the two stores whose loaders refuse it. It names the fix.
func preambleMessage(store string, at int) string {
	return store + " frontmatter must open on line 1: the `---` is on line " +
		strconv.Itoa(at) + ", and the loader reads the delimiter at the top of the " +
		"file only — it sees NO fields here and fails closed over the whole store. " +
		"Move the `---` to line 1 (delete the leading blank lines or comment)."
}

// frontmatterDuplicates reports the duplicated top-level frontmatter keys in a
// record, applying the same leading-comment offset frontmatterFields does so the
// reported line stays absolute. It is the record-lint side of the strict/lenient
// agreement (GitHub #357): the lenient scanner keeps the FIRST occurrence of a
// duplicated key and drops the rest silently, while the strict ledger parser the
// record's consumers use refuses the file outright — so a second `impact:` line
// can silence the value the armed issue_impact_valid blocker is set to reject, or
// a merge that lands the same key twice drops the record out of capture's corpus.
// The gate reports the duplicate so its parse agrees with its consumers'.
func frontmatterDuplicates(lines []string) []frontmatter.Dup {
	offset := 0
	if start := frontmatterOpen(lines); start > 0 {
		offset = start
		lines = lines[start:]
	}
	dups := frontmatter.Duplicates(lines)
	for i := range dups {
		dups[i].Line += offset
	}
	return dups
}

// contentExempt reports whether a file is excused from the content-AUTHORING
// checks (banned_tokens, persona_registry) because it is part of the historical
// record: its repo-relative path begins with a configured prefix, or its leading
// frontmatter status: value is listed.
//
// Neither intent-tree check consults this — intent_lifecycle and
// intent_impact_valid share one scan of the tree, so both stopped — and
// record_schema never did; the spec-store checks (spec_lifecycle,
// spec_id_unique) still do. The exemption used to reach
// the intent-lifecycle schema too, which is how two intents came to sit in
// superseded/ with no supersession field at all — the one rule that would have
// caught it was the rule the bucket exempted them from (iss-39). Being historical
// excuses a record from how it is WRITTEN, never from being well-formed.
func contentExempt(rel string, fields map[string]fmField, cfg Config) bool {
	for _, p := range cfg.ExemptPaths {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}
	if st, ok := fields["status"]; ok {
		for _, s := range cfg.ExemptIfStatus {
			if st.value == s {
				return true
			}
		}
	}
	return false
}

// isNull delegates to the one null predicate. It stays as a name here only
// because the call sites read better for it; the SET of null spellings lives in
// internal/core/frontmatter and is not restated (iss-287). The private copy this
// replaces recognised only the lower-case spelling, so `impact: NULL` was null
// to capture and a malformed impact to this lint.
func isNull(v string) bool { return frontmatter.IsNull(v) }

// fenceMask marks lines that are inside (or are a marker for) a triple-backtick
// fenced code block.
func fenceMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	inFence := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			mask[i] = true
			inFence = !inFence
			continue
		}
		mask[i] = inFence
	}
	return mask
}

// hasMarkdownExt reports whether name ends in the markdown extension, folding
// case. A record renamed to `.MD`/`.Md`/`.mD` is the same record to every
// discovery filter — otherwise a casing slip makes the file INVISIBLE to every
// blocking rule (which then all report clean) while the lifeboat still packs it
// as a live record (GitHub #333). The filename-WELLFORMEDNESS matchers
// (store.fileNumRe, intentFileRe) stay strict lowercase `.md`, so a `.MD` file is
// DISCOVERED and then reported as a malformed filename rather than silently
// skipped — the file is gated, not invisible.
func hasMarkdownExt(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".md")
}

func markdownFiles(rootAbs string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// A genuinely-absent root is "unpopulated, not a fault" for the
			// optional-directory callers (forbidden_synonyms, context currency),
			// whose trees need not exist; the roots loop stats the root itself and
			// fails loud (GitHub #360) before reaching here. A file that vanishes
			// MID-walk — a git checkout/clean racing the gate — is a different
			// event: swallowing it would discard the whole root's accumulated
			// results silently, so only the root's OWN ENOENT is tolerated.
			if os.IsNotExist(err) && path == rootAbs {
				return nil
			}
			return err
		}
		if !d.IsDir() && hasMarkdownExt(d.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func matchesAny(res []*regexp.Regexp, s string) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func matchesGlob(globs []string, rel string) bool {
	for _, g := range globs {
		if ok, _ := filepath.Match(g, rel); ok {
			return true
		}
	}
	return false
}

func repoRel(repoRoot, abs string) string {
	if rel, err := filepath.Rel(repoRoot, abs); err == nil {
		return rel
	}
	return abs
}

func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].File != f[j].File {
			return f[i].File < f[j].File
		}
		if f[i].Line != f[j].Line {
			return f[i].Line < f[j].Line
		}
		if f[i].RuleID != f[j].RuleID {
			return f[i].RuleID < f[j].RuleID
		}
		return f[i].Message < f[j].Message
	})
}
