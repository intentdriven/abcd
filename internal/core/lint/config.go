package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxLintConfigBytes caps the docs-lint/record-lint config read (trust
// boundary). The shipped configs are a few kilobytes; the cap is many times
// larger, so it bounds a hostile /dev/zero-symlinked config without ever
// constraining a real one.
const maxLintConfigBytes = 256 * 1024

// Config is the on-disk record-lint configuration (.abcd/record-lint.json).
type Config struct {
	// Roots are repo-relative directories the lint walks (markdown record).
	Roots []string `json:"roots"`
	// BannedTokens are line-level substring/regex bans (check family A).
	BannedTokens []BannedToken `json:"banned_tokens"`
	// Rules holds the per-check configuration for the remaining families,
	// keyed by rule id (no_git_metadata, links_resolve, ...).
	Rules map[string]RuleConfig `json:"rules"`
	// ExemptPaths are repo-relative path prefixes whose files skip the
	// content-AUTHORING checks (banned_tokens, persona_registry) — the historical,
	// non-forward-looking part of the record, which is excused from how it is
	// written but never from being well-formed. Both intent-tree checks
	// (intent_lifecycle, intent_impact_valid — they share one scan) stay universal
	// (iss-39); the spec-store checks (spec_lifecycle, spec_id_unique) still skip an
	// exempt file. record_schema is cross-store and never consults this at all.
	ExemptPaths []string `json:"exempt_paths"`
	// NameRoots are repo-relative directories or files the name gate — the
	// banned_tokens whose id carries the `names/` prefix, the public banlist
	// layer — reads in addition to Roots. Every text file there is read, not only
	// markdown (a script names a project as readily as a page does), and only the
	// `names/` family runs: the rest of the family is a writing rule for the
	// documentation, and a name ban is about the whole public surface (iss-279).
	// exempt_paths and exempt_if_status apply as they do under Roots.
	NameRoots []string `json:"name_roots"`
	// ExemptIfStatus lists leading-frontmatter status: values that likewise
	// exempt a file from the content-authoring checks (e.g. superseded records).
	ExemptIfStatus []string `json:"exempt_if_status"`
}

// BannedToken is one entry in the banned_tokens family (check A).
type BannedToken struct {
	ID       string `json:"id"`
	Pattern  string `json:"pattern"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	// Successor is the machine-readable old->new mapping: what to use instead of
	// the banned token. It is REQUIRED (a ban with no successor left its
	// replacement in prose only, iss-51) and is auto-cited in the finding message.
	Successor string `json:"successor"`
	// AllowContext lists regexps that, if any matches the same line, suppress
	// the finding (the token is legitimate in that context). It is REQUIRED to be
	// non-empty: every ban must declare where its token is legitimately allowed.
	AllowContext []string `json:"allow_context"`
	// SkipCodeFences omits fenced-code lines from scanning. A nil pointer means
	// the family's default: true for a documentation token, whose fenced example
	// is not prose; false for a `names/` token, the name gate, which reaches the
	// whole public surface, where a fence is published as readily as prose
	// (iss-2609252251320133). Set it to override either default.
	SkipCodeFences *bool `json:"skip_code_fences"`
}

// skipFences resolves the SkipCodeFences pointer to its effective value.
func (t BannedToken) skipFences() bool {
	if t.SkipCodeFences == nil {
		return !strings.HasPrefix(t.ID, nameTokenPrefix)
	}
	return *t.SkipCodeFences
}

// RuleConfig is the shared shape for the non-token check families. Only the
// fields relevant to a given rule are populated.
type RuleConfig struct {
	Enabled  bool   `json:"enabled"`
	Severity string `json:"severity"`
	// Fields is the no_git_metadata banned frontmatter key list.
	Fields []string `json:"fields"`
	// Exempt is a glob allowlist of repo-relative paths (filepath.Match, so `*`
	// stays inside one directory). directory_coverage reads it for directories
	// excused a README; links_resolve reads it for files whose links are not
	// checked — a tool-mandated mirror of a root file, whose relative links
	// resolve from the root and not from the mirror's directory.
	Exempt []string `json:"exempt"`
	// IntentsDir is the intents subdirectory (relative to a root) read by the
	// intent-tree rules, intent_lifecycle and intent_impact_valid. Rules that name
	// the same directory share one scan of it. spec_lifecycle also reads it to
	// resolve the intent corpus its specs link to.
	IntentsDir string `json:"intents_dir"`
	// SpecsDir is the spec_lifecycle specs subdirectory (relative to a root),
	// mirroring IntentsDir. Default "specs".
	SpecsDir string `json:"specs_dir"`
	// IssuesDir is the issue-ledger root (repo-relative) read by the ledger rules,
	// issue_id_unique and issue_impact_valid; it holds the open/, resolved/, and
	// wontfix/ status directories. Default .abcd/work/issues. It lies outside Roots
	// — the rules read the ledger and run once, sharing one scan of it.
	// reading_outstanding reads the same root's SIBLING families (readings/,
	// dispositions/), which is why they are named relative to it rather than
	// configured twice.
	IssuesDir string `json:"issues_dir"`
	// Allowlist is the stray_root_docs permitted basename-stem list (upper-cased,
	// extension-stripped) for top-level markdown files.
	Allowlist []string `json:"allowlist"`
	// Registry is a rule's registry file, repo-relative. For persona_registry it
	// is the persona roster (.abcd/development/personas.json); for
	// surface_coverage it is the brief surface table
	// (.abcd/development/brief/04-surfaces/README.md).
	Registry string `json:"registry"`
	// CommandsDir is the surface_coverage plugin-command directory (commands);
	// each *.md file (README and BareCommand excepted) is a shipped command
	// surface. It lies outside Roots — the rule reads the surface tree and
	// cross-checks the brief. The directory is flat: a harness maps a
	// subdirectory of it to an extra namespace segment, so a file one level down
	// registers as /<plugin>:<dir>:<verb> rather than the documented
	// /<plugin>:<verb>.
	CommandsDir string `json:"commands_dir"`
	// BareCommand is the CommandsDir file stem that backs the plugin's BARE
	// top-level command rather than a /<plugin>:<verb> surface — for abcd, the
	// `abcd.md` whose registry row is the bare `/abcd`. It is excluded from the
	// command-surface set the way README is, because the registry names it
	// without a sub-verb and matching it by name would demand a row that cannot
	// exist. Empty means the plugin has no such file.
	BareCommand string `json:"bare_command"`
	// SkillsDir is the surface_coverage skills directory (skills); each immediate
	// subdirectory is a shipped skill surface. Also outside Roots.
	SkillsDir string `json:"skills_dir"`
	// Snapshot is the committed command-tree snapshot surface_coverage's
	// sub-verb pass checks table rows against (spc-27), repo-relative — for
	// abcd, .abcd/development/release/surface.json, whose bytes the surface
	// drift test keeps equal to the live cobra tree. Empty leaves the sub-verb
	// grain unarmed (the pre-spc-27 surface-grain check only).
	Snapshot string `json:"snapshot"`
	// HostDelegated lists surfaces whose workflow runs in the host agent with
	// no Go verb (consult, ingest, prepare-this-repo): their sub-verb tables
	// are format-checked only, never compared to the cobra tree. Explicit
	// config, never a hard-coded skip — an unlisted surface gets the full
	// comparison.
	HostDelegated []string `json:"host_delegated"`
	// OperatorInternal lists top-level verbs absent from the surface registry
	// by design (spec, rules, hook, completion — operator plumbing, not
	// product surface): the sub-verb reverse sweep does not demand a surface
	// file for them. Explicit config, same rationale as HostDelegated.
	OperatorInternal []string `json:"operator_internal"`
	// Target is the context_status_free single-file target, repo-relative
	// (.abcd/work/CONTEXT.md). The rule runs even though the target lies outside
	// Roots; a missing target is not an error.
	Target string `json:"target"`
	// Patterns is the context_status_free line-match regexp list; when empty the
	// rule falls back to contextStatusDefaultPatterns.
	Patterns []string `json:"patterns"`
	// Section is the context_citation_currency heading matcher: the rule reads only
	// the section of Target whose heading matches (the sharp-edges list), because a
	// citation to a terminal record is legitimate everywhere else. Empty falls back
	// to defaultContextSection.
	Section string `json:"section"`
	// ReceiptsDir is the receipt_gate directory of sha-keyed semantic-pass
	// receipts (VSA-shaped JSON), repo-relative (default .abcd/work/reviews).
	// Outside Roots.
	ReceiptsDir string `json:"receipts_dir"`
	// RequiredGates lists the semantic gates that must each have a PROMOTE receipt
	// for the target commit before a release (e.g. docs-currency-reviewer,
	// iss35-brief-surface-crosscheck).
	RequiredGates []string `json:"required_gates"`
	// Commit is the receipt_gate target commit sha whose receipts are verified.
	// Release-time input (release.yml supplies the tagged commit); empty while the
	// rule is disabled for ordinary development.
	Commit string `json:"commit"`
	// Runbook is the gate_lockstep runbook path (its numbered "Deterministic
	// gates" list), repo-relative.
	Runbook string `json:"runbook"`
	// Workflow is the gate_lockstep CI workflow path — the source of truth for the
	// deterministic gate list, repo-relative.
	Workflow string `json:"workflow"`
	// Job is the gate_lockstep workflow job whose step names are the gate list.
	Job string `json:"job"`
	// IgnoreSteps are workflow step names that are setup, not gates, and so are
	// excluded from the lockstep comparison.
	IgnoreSteps []string `json:"ignore_steps"`
	// MinGates is the gate_lockstep non-empty floor: each side must parse at least
	// this many gates or the rule fails closed (an under-count means the parser or
	// a heading/job rename silently dropped gates). It is the safety net that makes
	// the hand-parse fail-closed. Enforced as at least 1 when the rule is enabled.
	MinGates int `json:"min_gates"`
	// GlossaryDir is the forbidden_synonyms (GL002) glossary directory, repo-relative
	// (default .abcd/development/brief/glossary). The rule walks it for term files and
	// reads each term's forbidden_synonyms frontmatter list — the glossary is the
	// single source of truth for what a forbidden synonym is.
	GlossaryDir string `json:"glossary_dir"`
	// Enforce is the forbidden_synonyms subset that GL002 mechanically gates. Each
	// entry MUST be declared as a forbidden_synonym by some glossary term (the rule
	// errors otherwise, so the config can never gate a word the glossary does not
	// forbid). Enforcement is a deliberate subset because most forbidden synonyms
	// ("user", "release", "project", "feature", ...) are common English words whose
	// live-prose false-positive rate blows the detector's budget; "epic" is the
	// mechanically-clean member (itd-43). Promotion path: add a synonym here once the
	// corpus is swept clean of its non-substituting uses.
	Enforce []string `json:"enforce"`
	// ExemptPrefixes are repo-relative path prefixes whose files GL002 skips — the
	// historical, git-tracked records the rename intent (itd-43 AC1) exempts:
	// research/, decisions/ (dated ADRs), plans/ (dated), shipped/superseded intents,
	// the issue ledger, and review records. The glossary directory itself is always
	// exempt (a term file names its own forbidden synonyms legitimately).
	ExemptPrefixes []string `json:"exempt_prefixes"`
	// AllowContext lists regexps that, if any matches a line, suppress every GL002
	// finding on that line — the legitimate-mention escape (naming the old token in an
	// external reference like `epic-review`, or the rename itself `epic->spec`).
	AllowContext []string `json:"allow_context"`
	// CrosswalkHeading is the citation_crosswalk_rows heading matcher: a table is
	// judged a crosswalk only when its nearest preceding heading matches this
	// regexp. Default defaultCrosswalkHeading. Keying on the heading rather than on
	// table shape is what keeps the rule off every ordinary table in the corpus.
	CrosswalkHeading string `json:"crosswalk_heading"`
	// RefusedDomains is the citation_source_policy list of aggregator domains a
	// citation may not point at. Matching is host-normalised (case-folded, www.
	// dropped) and covers subdomains. It ships EMPTY: naming a domain is a
	// project's editorial policy, never something the gate invents for it.
	RefusedDomains []string `json:"refused_domains"`
	// Baseline is the citation_baseline record's repo-relative path. Default
	// DefaultBaselinePath.
	Baseline string `json:"baseline"`
	// WarnAfterDays is the citation_baseline staleness warn threshold in days
	// (spc-17: 180). Zero means the default.
	WarnAfterDays int `json:"warn_after_days"`
	// BlockAfterDays is the age at which an entry becomes OVERDUE and is reported
	// under the citation_baseline_overdue rule id (spc-17: 365). Zero means the
	// default.
	BlockAfterDays int `json:"block_after_days"`
	// OverdueSeverity is the severity of a citation_baseline_overdue finding. It
	// defaults to warn because the COMMIT gate never calendar-blocks (spc-17);
	// the release gate is what promotes it to a blocker.
	OverdueSeverity string `json:"overdue_severity"`
	// RecordStores are the record_schema stores: the repo-relative directory of
	// each identified record store, keyed by id prefix (adr, itd, spc, iss). The
	// rule reasons ACROSS the stores, which straddle Roots (the issue ledger is
	// working-tier, the rest is the design record), so each is named repo-relative
	// here. An unnamed or absent store contributes nothing. Only the LOCATIONS are
	// configurable — which lifecycle buckets each store declares is the record's
	// schema and lives in code, so a config can never quietly add or hide one.
	RecordStores map[string]string `json:"record_stores"`
	// Indexes are the index_drift pairs: each names a marked region in a document
	// and the directory that region enumerates. The rule reads documents outside
	// Roots (a package or command README lives beside its code), so the pairs are
	// declared here rather than discovered by the walk.
	Indexes []IndexSpec `json:"indexes"`
	// Changelog is the delivery_state changelog, repo-relative. Default
	// "CHANGELOG.md". It sits at the repo root, outside Roots.
	Changelog string `json:"changelog"`
	// IntentsRoot is the delivery_state intents store, repo-relative — the whole
	// store, not a subdirectory of a root, because the rule resolves a cited id to
	// the lifecycle bucket holding it and the changelog it reads is repo-scoped.
	// Default .abcd/development/intents.
	IntentsRoot string `json:"intents_root"`
	// DeliverySections are additional changelog change-type headings delivery_state
	// reads as delivery claims. They are UNIONED with deliveryStateSections, never
	// substituted for them: a repo names its own headings here to widen the gate,
	// and a config that could narrow it could silently disarm it.
	DeliverySections []string `json:"delivery_sections"`
	// AgentsDir is the agent_contract prompt tree, repo-relative (default
	// "agents"). It lies outside Roots — an agent prompt is not a record, so the
	// rule walks the tree itself rather than the tree being added to Roots and
	// judged by the record stores' schema.
	AgentsDir string `json:"agents_dir"`
	// DiffRange is the agent_contract changelog sub-check's git revision range.
	// It is supplied by the CALLER (ArmAgentDiff, from a CI invocation) rather
	// than read out of the in-tree config, for the reason ArmReceiptGate states:
	// the decision to check a diff, and which diff, is trust-rooted to the
	// workflow, not to the committer-editable file. Empty makes the sub-check a
	// no-op — there is no diff to make a statement about.
	DiffRange string `json:"-"`
}

// IndexSpec is one index_drift pair — a hand-written enumeration of a
// directory's contents, and the directory it must agree with.
type IndexSpec struct {
	// ID names the region: the document fences it with `<!-- index: <id> -->`
	// and `<!-- /index -->`.
	ID string `json:"id"`
	// Doc is the enumerating document, repo-relative.
	Doc string `json:"doc"`
	// Dir is the enumerated directory, repo-relative. In absent mode it is the
	// base each listed path resolves against.
	Dir string `json:"dir"`
	// Entry is the regexp a backticked token in the region must match to count as
	// an entry. It is what keeps the rule off the prose around the list: a token
	// the pattern does not describe (a flag, a tool name) is not an entry.
	Entry string `json:"entry"`
	// DirEntry is the mirror of Entry on the directory side: the regexp a file's
	// stem must match to be an entry, reduced to submatch 1 when the pattern has a
	// group. It is what lets a document enumerate records by IDENTITY rather than
	// by filename — `itd-8` for `itd-8-with-code-bundling.md` — without a second
	// rule: without it the two sides can only agree when the document transcribes
	// whole slugs, which is a listing nobody writes by hand. Empty compares whole
	// stems.
	DirEntry string `json:"dir_entry"`
	// Suffix is the file extension an exact index enumerates (".md" lists file
	// stems); empty enumerates the directory's immediate subdirectories.
	Suffix string `json:"suffix"`
	// Mode is "exact" (default — the listing and the directory must agree) or
	// "absent" (every listed path must NOT exist, the planned-seams shape).
	Mode string `json:"mode"`
}

// ArmReceiptGate returns cfg with the receipt_gate rule armed for a release: it
// is enabled, pointed at the target commit, and its required-gates list is set to
// the caller's list verbatim. This is how a release runs the gate: the CALLER (a
// CI workflow) supplies the arming, so the decision to gate, the target commit,
// and the required-gates list are trust-rooted to the workflow rather than the
// in-tree, committer-editable config (phase-2 review Finding 2). The caller's list
// is authoritative even when empty: an empty list clears the gates so
// checkReceiptGate fails closed, rather than inheriting a config a committer could
// have shrunk (an argless arming must not silently pick up in-tree gates). The
// input cfg is not mutated (the Rules map is copied). Other rules are unchanged;
// the deterministic gates still run alongside.
func ArmReceiptGate(cfg Config, commit string, requiredGates []string) Config {
	rules := make(map[string]RuleConfig, len(cfg.Rules)+1)
	for k, v := range cfg.Rules {
		rules[k] = v
	}
	rc := rules["receipt_gate"]
	rc.Enabled = true
	rc.Commit = commit
	// An armed release gate is blocking by definition — force the severity so the
	// gate's teeth are trust-rooted to the caller (a CI workflow) like Enabled and
	// Commit, never the committer-editable config. A downgraded severity landed in
	// the in-tree file must not defang the gate at release time.
	rc.Severity = severityBlocker
	// Verbatim, including empty: the caller is the trust root, so an empty list
	// clears the gates (fail-closed at check time) rather than inheriting the
	// committer-editable config.
	rc.RequiredGates = requiredGates
	rules["receipt_gate"] = rc
	cfg.Rules = rules
	return cfg
}

// ArmAgentDiff returns cfg with agent_contract's changelog sub-check pointed at a
// git revision range. It is the same arming shape as ArmReceiptGate and for the
// same reason: the sub-check asks whether a CHANGE announced itself, so it needs
// a diff, and the caller (a CI invocation, `record-lint -agent-diff`) is the one
// that knows which. It is deliberately NOT a config key — a range read out of the
// committed file would let a committer point the gate at an empty diff.
//
// Arming only WIDENS the rule: an empty range leaves the changelog sub-check a
// no-op while the frontmatter and canary sub-checks run either way. The rule's
// own Enabled flag is untouched, so this cannot switch a disabled rule on.
// The input cfg is not mutated (the Rules map is copied).
func ArmAgentDiff(cfg Config, diffRange string) Config {
	rules := make(map[string]RuleConfig, len(cfg.Rules)+1)
	for k, v := range cfg.Rules {
		rules[k] = v
	}
	rc := rules[ruleAgentContract]
	rc.DiffRange = diffRange
	rules[ruleAgentContract] = rc
	cfg.Rules = rules
	return cfg
}

// knownRules is every rule name a Lint dispatches on: the keys LintAt (and the
// cross-store and ledger passes it calls) read out of Config.Rules. A config that
// names any other rule is refused at load (validateRuleNames), because a misspelt
// name ("links_reslove") decodes cleanly, reads as armed, is counted by
// ArmedChecks, and runs nothing. A rule added to LintAt and not here is refused
// the first time a config names it, which fails loud rather than green.
var knownRules = map[string]bool{
	"links_resolve":             true,
	ruleLinkAnchors:             true,
	"no_git_metadata":           true,
	"no_brittle_line_refs":      true,
	"persona_registry":          true,
	"directory_coverage":        true,
	"intent_lifecycle":          true,
	"intent_impact_valid":       true,
	"spec_lifecycle":            true,
	"spec_id_unique":            true,
	"forbidden_synonyms":        true,
	"stray_root_docs":           true,
	"context_status_free":       true,
	"surface_coverage":          true,
	"index_drift":               true,
	"receipt_gate":              true,
	"gate_lockstep":             true,
	"issue_id_unique":           true,
	"issue_impact_valid":        true,
	ruleAgentContract:           true,
	ruleCitationFootnotes:       true,
	ruleCitationCrosswalkRows:   true,
	ruleCitationURLSyntax:       true,
	ruleCitationSourcePolicy:    true,
	ruleCitationBaseline:        true,
	ruleContextCitationCurrency: true,
	ruleCrossStoreIDClaim:       true,
	ruleDeliveryState:           true,
	ruleHarnessLeak:             true,
	ruleProseCitationResolves:   true,
	ruleReadingOutstanding:      true,
	ruleRecordProvenance:        true,
	ruleRecordSchema:            true,
}

// validateRuleNames refuses a rule the lint does not run, enabled or not, and
// names the rules it does. It is the name-level twin of strictRuleAndTokenKeys:
// that one refuses a misspelt key inside a rule, this one a misspelt rule. Names
// are checked in sorted order so the refusal is deterministic when several are
// wrong.
func (c Config) validateRuleNames() error {
	names := make([]string, 0, len(c.Rules))
	for name := range c.Rules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if knownRules[name] {
			continue
		}
		known := make([]string, 0, len(knownRules))
		for k := range knownRules {
			known = append(known, k)
		}
		sort.Strings(known)
		return &configError{"rule " + strconv.Quote(name) + " is not a rule this lint runs, so it would be read as armed and check nothing; known rules: " + strings.Join(known, ", ")}
	}
	return nil
}

// ArmedChecks counts the checks a Lint over this configuration runs: every
// banned token and every enabled rule. Zero means a lint runs nothing, and a
// front door must say so rather than report a finding count that implies a check
// happened (loud-staging; iss-2609150805167646). A disabled rule is inert and is
// not counted. Every rule a loaded config names is one the lint runs:
// validateRuleNames refuses any other at load, so a misspelt name cannot be
// counted here as a check that ran.
func (c Config) ArmedChecks() int {
	n := len(c.BannedTokens)
	for _, rc := range c.Rules {
		if rc.Enabled {
			n++
		}
	}
	return n
}

// LoadConfig reads and decodes a record-lint config file. The config is a trust
// boundary: it is a committed, cross-repo-clonable file (a hostile clone can
// commit .abcd/docs-lint.json as a git mode-120000 symlink), and the read is
// reachable automatically through the session hooks (ahoy.Detect), so it is
// guarded exactly like its .abcd/*.json siblings guard.Load and rules.Load —
// a symlinked config directory or leaf is refused, a FIFO/device leaf returns
// immediately instead of hanging the open, and an over-cap file is rejected
// rather than read into an OOM. Error strings stay path-free (iss-29); the
// os.IsNotExist branch is preserved so callers that default an absent config
// still work.
func LoadConfig(path string) (Config, error) {
	// Refuse a symlinked config directory before touching the leaf, so a swapped
	// .abcd cannot redirect the read at a config the repository does not own.
	if di, err := os.Lstat(filepath.Dir(path)); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return Config{}, fmt.Errorf("lint: config directory is a symlink (refusing to follow)")
	}
	data, err := fsutil.ReadGuarded(path, maxLintConfigBytes)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return Config{}, err
		case errors.Is(err, syscall.ELOOP):
			return Config{}, fmt.Errorf("lint: config is a symlink (refusing to follow)")
		case errors.Is(err, fsutil.ErrNotRegular):
			return Config{}, fmt.Errorf("lint: config is not a regular file")
		case errors.Is(err, fsutil.ErrTooBig):
			return Config{}, fmt.Errorf("lint: config exceeds the %d-byte cap", maxLintConfigBytes)
		default:
			return Config{}, err
		}
	}
	return parseConfig(data)
}

// LoadConfigInRoot reads and validates a lint config confined to root: every
// path component is resolved inside root, so a symlinked ancestor cannot walk
// the read out of the tree, and the leaf is size- and regular-file-guarded the
// same way LoadConfig's is. Callers that already hold an os.Root for the repo
// (the site build, for one) use this so their config read matches the
// ancestor-symlink containment their other reads have, rather than re-opening
// by absolute path.
func LoadConfigInRoot(root *os.Root, rel string) (Config, error) {
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxLintConfigBytes)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return Config{}, err
		case errors.Is(err, syscall.ELOOP):
			return Config{}, fmt.Errorf("lint: config is a symlink (refusing to follow)")
		case errors.Is(err, fsutil.ErrNotRegular):
			return Config{}, fmt.Errorf("lint: config is not a regular file")
		case errors.Is(err, fsutil.ErrTooBig):
			return Config{}, fmt.Errorf("lint: config exceeds the %d-byte cap", maxLintConfigBytes)
		default:
			return Config{}, err
		}
	}
	return parseConfig(data)
}

// parseConfig decodes and validates a lint config's bytes. It is the shared
// body of LoadConfig and LoadConfigInRoot, so the two read paths cannot drift
// in what they accept.
func parseConfig(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := strictRuleAndTokenKeys(data); err != nil {
		return Config{}, err
	}
	if err := cfg.validateRuleNames(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateBannedTokens(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateSeverities(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateRecordStores(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateConfiguredPaths(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateArmedInputs(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// armedInputs names, per rule, the input paths the rule cannot check without.
// Each rule returned clean on a blank one BEFORE its own fail-closed guards ran,
// so an enabled rule with a blank input read as armed and checked nothing
// (iss-336; surface_coverage is its sibling).
var armedInputs = map[string][]struct {
	key string
	get func(RuleConfig) string
}{
	"gate_lockstep": {
		{"runbook", func(r RuleConfig) string { return r.Runbook }},
		{"workflow", func(r RuleConfig) string { return r.Workflow }},
	},
	"surface_coverage": {
		{"registry", func(r RuleConfig) string { return r.Registry }},
	},
}

// validateArmedInputs refuses an ENABLED rule whose required input path is
// blank, naming the key. A disabled rule is not checked: it runs nothing on
// purpose, and saying so is what enabled:false is for.
func (c Config) validateArmedInputs() error {
	names := make([]string, 0, len(armedInputs))
	for name := range armedInputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rc, ok := c.Rules[name]
		if !ok || !rc.Enabled {
			continue
		}
		for _, in := range armedInputs[name] {
			if strings.TrimSpace(in.get(rc)) == "" {
				return &configError{"rule " + strconv.Quote(name) + " is enabled but its " + strconv.Quote(in.key) +
					" is blank, so it would read as armed and check nothing; set " + strconv.Quote(in.key) + " or set \"enabled\": false"}
			}
		}
	}
	return nil
}

// configuredPath is one repo-relative location the config names, paired with the
// field that named it so a refusal can say which key to fix.
type configuredPath struct {
	field string
	value string
}

// validateConfiguredPaths refuses every configured location that is not a plain
// repo-relative path, at LOAD time — before any rule joins it onto the repository
// root and reads what it finds.
//
// The config is a trust boundary the way LoadConfig's own read is: it is a
// committed, cross-repo-clonable file, and a pull request that edits
// `.abcd/record-lint.json` edits the gate that judges it. record_schema's store
// walk joined `record_stores` onto the repo root with no rejection at all, so
// `"adr": "../outside/decisions"` made the gate read `.md` frontmatter from
// outside the checkout and echo it into the CI log
// (iss-2608301308367566). Every OTHER repo-relative field the config carries is
// joined the same way by its own rule, so gating one field would gate the
// instance rather than the pattern.
//
// Containment is checked here rather than only at each use site because the
// per-site guards (containedRepoPath + resolvedInsideRoot, which six of these
// fields already carry) are opt-in: a field added later inherits nothing, and the
// omission is silent — the rule reads the outside tree and reports on it at exit
// 0. One gate at the door cannot be forgotten by a new field, and the six
// per-site guards stay where they are: they also resolve symlinks, which a
// lexical gate cannot see.
//
// fsutil.ValidRelPath is the canonical lexical guard for a path that arrives as
// data (the site manifest and the positioning config both route through it), so
// it is called rather than copied — a second containment predicate is the shape
// the one-canonical-primitive principle refuses, and this one is strictly
// stronger than the package's read-time check: it also refuses an unclean path
// and a backslash, which `..\..\x` needs on the Windows binaries abcd
// cross-compiles.
//
// An EMPTY value is not a path — it means the field is unset, and for a roots
// entry it means the root the entry is relative to — so it passes, while "."
// does not: ValidRelPath refuses a "." segment. The two spell the same location,
// so the refusal has to say which one the config should carry, or the author
// reads a message listing four causes and recognises none of them. The
// *_paths/*_prefixes lists are deliberately not here: they are string prefixes
// matched against repo-relative paths, never joined onto a root and never
// opened, and the shipped config spells them with a trailing slash, which is a
// prefix rather than a path.
func (c Config) validateConfiguredPaths() error {
	for _, root := range c.Roots {
		if err := checkConfiguredPath(configuredPath{"roots entry", root}); err != nil {
			return err
		}
	}
	// Rules is a map, so a config with several faults must not report a different
	// one per run: walk the rule ids, and each rule's stores, in sorted order.
	ids := make([]string, 0, len(c.Rules))
	for id := range c.Rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		rc := c.Rules[id]
		fields := []configuredPath{
			{"intents_dir", rc.IntentsDir},
			{"specs_dir", rc.SpecsDir},
			{"issues_dir", rc.IssuesDir},
			{"registry", rc.Registry},
			{"commands_dir", rc.CommandsDir},
			{"skills_dir", rc.SkillsDir},
			{"snapshot", rc.Snapshot},
			{"target", rc.Target},
			{"receipts_dir", rc.ReceiptsDir},
			{"runbook", rc.Runbook},
			{"workflow", rc.Workflow},
			{"glossary_dir", rc.GlossaryDir},
			{"baseline", rc.Baseline},
			{"changelog", rc.Changelog},
			{"intents_root", rc.IntentsRoot},
			{"agents_dir", rc.AgentsDir},
		}
		// Every record_stores KEY the config wrote, sorted, rather than the engine's
		// own store list: sorting keeps a config with several faulty stores reporting
		// the same one, and walking what was written rather than what is known means
		// this check does not depend on validateRecordStores having already refused an
		// unknown prefix — a reordering of parseConfig would otherwise leave an
		// unknown store's path unjudged.
		prefixes := make([]string, 0, len(rc.RecordStores))
		for prefix := range rc.RecordStores {
			prefixes = append(prefixes, prefix)
		}
		sort.Strings(prefixes)
		for _, prefix := range prefixes {
			fields = append(fields, configuredPath{"record_stores " + quote(prefix), rc.RecordStores[prefix]})
		}
		for _, spec := range rc.Indexes {
			fields = append(fields,
				configuredPath{"index " + quote(spec.ID) + " doc", spec.Doc},
				configuredPath{"index " + quote(spec.ID) + " dir", spec.Dir})
		}
		for _, f := range fields {
			f.field = "rule " + id + ": " + f.field
			if err := checkConfiguredPath(f); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkConfiguredPath is validateConfiguredPaths' single verdict, so every field
// is judged by one predicate and the refusal always names the offending value —
// a message that says only "a path escapes the repository" sends the reader
// hunting through a config with two dozen path keys.
func checkConfiguredPath(p configuredPath) error {
	if p.value == "" || fsutil.ValidRelPath(p.value) {
		return nil
	}
	return &configError{p.field + " " + quote(p.value) +
		" is not a plain repo-relative path; it must be relative, already clean, and carry no \".\", \"..\" " +
		"or backslash segment — an empty value, never \".\", is how a field names the root it is relative to; " +
		"the rule joins it onto the repository root and reads what it finds, so a value that escapes " +
		"reads files the repository does not own and echoes them into the lint output"}
}

// validateRecordStores refuses a record_stores key that names no store this
// engine knows. Only the LOCATION of a store is configurable; which stores exist
// is code, because a config that could add one could also hide a directory
// behind it — the nested-store-root exemption is granted to a store's own root,
// and an unknown key would be claiming an exemption for a directory nothing
// scans. A silently-ignored key also fails the way misspellings always fail
// here: the file still looks armed.
func (c Config) validateRecordStores() error {
	known := recordStorePrefixes()
	for id, rc := range c.Rules {
		var unknown []string
		for prefix := range rc.RecordStores {
			if !known[prefix] {
				unknown = append(unknown, prefix)
			}
		}
		if len(unknown) == 0 {
			continue
		}
		sort.Strings(unknown)
		return &configError{"rule " + id + ": record_stores names no such store: " +
			strings.Join(unknown, ", ") + "; which record stores exist is code, not configuration — only their locations are configurable"}
	}
	for id, rc := range c.Rules {
		if err := validateStoreLayout(id, rc.RecordStores); err != nil {
			return err
		}
	}
	return nil
}

// validateStoreLayout refuses a configured store root that would remove another
// store's coverage.
//
// Refusing an UNKNOWN prefix was only half of it: a known one does the same
// damage and looks like ordinary configuration. Aim rdi at the issue store's
// open/ and that bucket becomes a nested store root — the issue store skips it,
// while rdi ignores every file that is not rdi-N.md — so a malformed issue
// record sitting in it is judged by nobody. Two prefixes on one path is the same
// trick without the nesting: whichever store's grammar the files do not match
// stops seeing them.
//
// Two shapes are refused, and both are about coverage rather than tidiness: a
// root that IS or sits INSIDE a bucket another store declares (by list, or by the
// grammar a minted-bucket store declares), and two prefixes resolving to one
// path. A root that merely sits inside another store's root, beside its buckets,
// is the shipped layout and stays legal.
func validateStoreLayout(ruleID string, stores map[string]string) error {
	type entry struct{ prefix, path string }
	// Built in recordStores order so a config with several faults always reports
	// the same one.
	var entries []entry
	for _, s := range recordStores {
		if p := stores[s.prefix]; p != "" {
			entries = append(entries, entry{s.prefix, strings.Trim(filepath.ToSlash(p), "/")})
		}
	}
	for i, a := range entries {
		for j, b := range entries {
			if i == j {
				continue
			}
			if a.path == b.path {
				if a.prefix > b.prefix {
					continue // report the pair once
				}
				return &configError{"rule " + ruleID + ": record_stores points both " + a.prefix +
					" and " + b.prefix + " at " + quote(a.path) +
					"; two stores on one path means whichever store's filename grammar a file does not match is scanned by nobody"}
			}
			if !strings.HasPrefix(b.path, a.path+"/") {
				continue
			}
			segment := b.path[len(a.path)+1:]
			if k := strings.Index(segment, "/"); k >= 0 {
				segment = segment[:k]
			}
			store, ok := storeByPrefix(a.prefix)
			if !ok || !store.declaresBucket(segment) {
				continue
			}
			return &configError{"rule " + ruleID + ": record_stores puts the " + b.prefix +
				" store at or inside " + quote(a.path+"/"+segment) + ", which is a declared " + store.noun +
				" bucket; the " + a.prefix + " store would then skip that bucket as a nested store root while the " +
				b.prefix + " store ignores every file its own filename grammar does not match, so the records in it are scanned by nobody"}
		}
	}
	return nil
}

// strictRuleAndTokenKeys re-decodes each rule and banned-token OBJECT with
// DisallowUnknownFields: a misspelt key silently zero-values the field it
// missed ("enabld" disarms a rule, "severty" strips its exit-code weight), and
// both misreads survive review because the file still looks armed. The TOP
// level stays lenient on purpose — an annotation key beside the declared ones
// is the JSON commentary convention, and the banlist editor pins that a config
// carrying one still loads.
func strictRuleAndTokenKeys(data []byte) error {
	var raw struct {
		BannedTokens []json.RawMessage          `json:"banned_tokens"`
		Rules        map[string]json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for id, body := range raw.Rules {
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()
		var rc RuleConfig
		if err := dec.Decode(&rc); err != nil {
			return &configError{"rule " + id + ": " + err.Error()}
		}
	}
	for i, body := range raw.BannedTokens {
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()
		var t BannedToken
		if err := dec.Decode(&t); err != nil {
			return &configError{"banned_tokens index " + strconv.Itoa(i) + ": " + err.Error()}
		}
	}
	return nil
}

// severityPinnedRules are the rules that set their own finding severity in CODE
// and never read it from the configuration. There is one, and it is a REPORT:
// reading_outstanding says which reading items nobody has answered, and a
// reading must never fail a push that has nothing to do with it. Pinning the
// severity in code is what makes that structural — a config that could raise it
// to blocker is a gate waiting to happen — and this set is why declaring "info"
// for such a rule is a correct declaration rather than the off-enum value the
// check below exists to refuse.
var severityPinnedRules = map[string]string{ruleReadingOutstanding: severityInfo}

// validateSeverities refuses a severity outside the engine's enum on any
// enabled rule or banned token. The exit paths count Severity == "blocker"
// verbatim, so an off-enum value would emit findings that serialize yet count
// toward no exit code — a clean exit beside a non-empty findings list, which
// the sibling engines (repolint.Evaluate, guard.Validate, banlist.AddPublic)
// name a rule bug and fail closed on. A disabled rule is inert and its
// severity is not consulted, so it is not checked.
//
// A severity-pinned rule is checked against its OWN pinned value instead: it
// counts toward no exit code deliberately, and a config that declared it
// blocker or warn would be describing a gate the engine will not run.
func (c Config) validateSeverities() error {
	for id, rc := range c.Rules {
		if !rc.Enabled {
			continue
		}
		if pinned, isPinned := severityPinnedRules[id]; isPinned {
			if rc.Severity != pinned {
				return &configError{"rule " + id + " has severity " + strconv.Quote(rc.Severity) +
					"; it is a report, not a gate, and its severity is pinned in code to " + strconv.Quote(pinned)}
			}
			continue
		}
		if rc.Severity != severityBlocker && rc.Severity != severityWarn {
			return &configError{"rule " + id + " has severity " + strconv.Quote(rc.Severity) + "; an enabled rule must declare \"blocker\" or \"warn\", or its findings count toward no exit code"}
		}
	}
	for i, t := range c.BannedTokens {
		if t.Severity == severityBlocker || t.Severity == severityWarn {
			continue
		}
		who := t.ID
		if who == "" {
			who = "index " + strconv.Itoa(i)
		}
		return &configError{"banned_tokens entry " + who + " has severity " + strconv.Quote(t.Severity) + "; want \"blocker\" or \"warn\""}
	}
	return nil
}

// validateBannedTokens enforces the strict banned_tokens schema (iss-51): every
// entry must declare a non-empty successor (the machine-readable replacement,
// not prose alone) and a non-empty allow_context (where the token is legitimately
// allowed). A violation is a load-time rejection, so a defective ban can never
// reach the linter. Errors identify the offending entry by id (or index when the
// id is itself absent).
func (c Config) validateBannedTokens() error {
	for i, t := range c.BannedTokens {
		who := t.ID
		if who == "" {
			who = "index " + strconv.Itoa(i)
		}
		if strings.TrimSpace(t.Successor) == "" {
			return &configError{"banned_tokens entry " + who + " has no successor; a ban must declare the machine-readable replacement (iss-51)"}
		}
		if len(t.AllowContext) == 0 {
			return &configError{"banned_tokens entry " + who + " has an empty allow_context; a ban must declare where its token is legitimately allowed (iss-51)"}
		}
	}
	return nil
}
