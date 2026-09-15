package lint

// The agent trust-contract family (agent_contract): the itd-5 prompt-quality
// contract, enforced.
//
// `agents/` holds host-delegated PROMPTS — the one part of the shipped surface a
// model reads as instruction — and it sits in neither lint root (record-lint
// walks .abcd/development, docs-lint walks docs/ and README.md). agents/README.md
// has documented the contract since M6 and named the linter that would enforce it
// as not yet built, so the five prompts that read the most attacker-influenceable
// input in the repository acquired the contract by hand and nothing checked that
// the sixth would (iss-278).
//
// This is that detector. It is a dedicated rule rather than an extra entry in
// cfg.Roots: the record stores' schema rules judge a RECORD's shape, and an agent
// prompt is not a record — it has a different frontmatter, a different lifecycle,
// and its own changelog. So the rule enumerates the tree itself, in the
// once-outside-the-loop, repo-root-scoped style of checkStrayRootDocs and
// checkDeliveryState.
//
// Three sub-checks, per agents/README.md § The itd-5 contract:
//
//  1. the trust-contract frontmatter (prompt_version, reads_untrusted_input,
//     capability_scope.task_classes, capability_scope.designed_for);
//  2. the injection-canary fixture every untrusted-input agent must ship;
//  3. a per-agent CHANGELOG entry for an agent added or changed in a diff.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const ruleAgentContract = "agent_contract"

// defaultAgentsDir is the prompt tree, repo-relative. It is a directory of
// PROMPTS, not of records, which is why it is named here rather than in
// cfg.Roots.
const defaultAgentsDir = "agents"

// agentCanaryFixture is the per-agent injection-canary contract: an agent that
// reads attacker-influenceable input carries at least one, under its own
// fixtures directory (agents/README.md § Injection canaries).
const agentCanaryFixture = "injection-canary.json"

var (
	// A prompt_version is a semver, because the changelog entry is keyed on it:
	// a version that cannot be spelled cannot be reconciled with an entry, and
	// the bump grammar (MAJOR schema break / MINOR behaviour / PATCH edit) is
	// what the entry is FOR.
	promptVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)
	// A nested frontmatter key: the `capability_scope` block's members sit one
	// level in, and the shared top-level scanner (internal/core/frontmatter)
	// deliberately ignores them. This is the ONLY nesting the contract has, so it
	// is read here rather than by promoting the whole package to a YAML parser.
	agentNestedKeyRe = regexp.MustCompile(`^[ \t]+([A-Za-z0-9_]+)[ \t]*:(.*)$`)
	// A caller-supplied diff range reaches a git subprocess, so it is validated
	// as a revision (or revision range) before use: a leading '-' would be read
	// as an OPTION by git, which is argument injection into a gate that a CI
	// workflow arms. The class is deliberately narrow — the spellings a range
	// actually takes (sha, ref, HEAD~2, a..b, a...b) and nothing else.
	agentDiffRangeRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/@^~-]*(?:\.{2,3}[A-Za-z0-9][A-Za-z0-9._/@^~-]*)?$`)
)

// maxAgentPromptBytes caps EVERY read this rule makes into the agent tree — the
// prompts and the per-agent changelog alike. A prompt is a few kilobytes of
// markdown and a changelog is a few more; the cap bounds a hostile tree without
// ever constraining a real one, exactly as maxLintConfigBytes does for the
// config. One cap rather than one per file kind: two constants is how the reads
// come to disagree about what "too big" means.
const maxAgentPromptBytes = 1 << 20

// agentPrompt is one agent prompt as this rule sees it.
type agentPrompt struct {
	// name is the file stem, which is the agent's identity: the fixtures
	// directory and the changelog entry are both keyed on it.
	name string
	// rel is the prompt file, repo-relative.
	rel string
	// fields are the top-level frontmatter keys.
	fields map[string]fmField
	// scope are the members of the nested capability_scope block.
	scope map[string]string
}

// checkAgentContract enforces the itd-5 trust contract over the agent-prompt
// tree. A tree that is not there contributes nothing and is not an error — a
// repository with no agents is a state, not a fault.
func checkAgentContract(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	dir := cfg.AgentsDir
	if dir == "" {
		dir = defaultAgentsDir
	}
	if err := containedRepoPath(dir); err != nil {
		return nil, &configError{ruleAgentContract + ": agents_dir " + quote(dir) + " " + err.Error() +
			"; the lint reads only inside the repository"}
	}
	dirAbs := filepath.Join(repoRoot, filepath.FromSlash(dir))
	entries, err := os.ReadDir(dirAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var prompts []agentPrompt
	var out []Finding
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !hasMarkdownExt(name) {
			continue
		}
		// README.md and CHANGELOG.md are the tree's own prose, not prompts.
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if strings.EqualFold(stem, "README") || strings.EqualFold(stem, "CHANGELOG") {
			continue
		}
		rel := filepath.Join(dir, name)
		realPath, err := containedRealPath(repoRoot, filepath.Join(dirAbs, name))
		if err != nil {
			return nil, &configError{ruleAgentContract + ": prompt " + quote(rel) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		content, err := fsutil.ReadGuarded(realPath, maxAgentPromptBytes)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(content), "\n")
		prompts = append(prompts, agentPrompt{
			name:   stem,
			rel:    rel,
			fields: frontmatterFields(lines),
			scope:  agentCapabilityScope(lines),
		})
	}

	for _, p := range prompts {
		out = append(out, checkAgentTrustContract(repoRoot, dir, p, cfg.Severity)...)
	}

	changelogFindings, err := checkAgentChangelog(repoRoot, dir, prompts, cfg)
	if err != nil {
		return nil, err
	}
	return append(out, changelogFindings...), nil
}

// checkAgentTrustContract is sub-checks 1 and 2 for one prompt: the itd-5
// frontmatter, and the injection canary an untrusted-input agent must ship.
//
// The DECLARATION is required of every prompt, not only of the ones that admit
// they read untrusted input. A rule that fires only on `reads_untrusted_input:
// true` is a rule any new agent opts out of by deleting one line — and the class
// this exists to catch is precisely a prompt that reads attacker-influenceable
// input while saying nothing about it. Silence is not `false`; it is undeclared,
// and it is reported.
func checkAgentTrustContract(repoRoot, dir string, p agentPrompt, severity string) []Finding {
	add := func(msg string) Finding {
		return Finding{File: p.rel, Line: 1, RuleID: ruleAgentContract, Severity: severity, Message: msg}
	}
	var out []Finding

	declared, ok := p.fields["reads_untrusted_input"]
	if !ok {
		// Reported, not returned: the prompt_version check below is owed to
		// every prompt, declared or not, and checkAgentChangelog relies on it
		// having run when it treats an empty version as already reported.
		out = append(out, add("agent prompt declares no 'reads_untrusted_input': the itd-5 trust contract "+
			"(agents/README.md) is declared, never inferred — an undeclared prompt reads as safe to every reader "+
			"and to this gate, which is how a prompt that reads attacker-influenceable input ships without a canary. "+
			"Declare 'reads_untrusted_input: true' (and carry the contract fields) or 'false'"))
	}

	// prompt_version is the itd-5 contract for EVERY prompt: it is what the
	// per-agent changelog entry is keyed on, whatever the prompt reads.
	if v, ok := p.fields["prompt_version"]; !ok || strings.TrimSpace(v.value) == "" {
		out = append(out, add("agent prompt is missing 'prompt_version': the itd-5 contract versions every prompt, "+
			"and the per-agent CHANGELOG entry is keyed on it"))
	} else if !promptVersionRe.MatchString(strings.Trim(strings.TrimSpace(v.value), `"'`)) {
		out = append(out, add("agent prompt's 'prompt_version' is not a semver ("+quote(v.value)+
			"): the bump grammar (MAJOR schema break, MINOR behaviour, PATCH edit) and the CHANGELOG entry both key on it"))
	}

	if !isTrueValue(declared.value) {
		return out
	}

	// The untrusted-input contract: the capability scope, then the canary.
	if len(p.scope) == 0 {
		out = append(out, add("agent prompt reads untrusted input but declares no 'capability_scope': the itd-5 "+
			"contract requires 'capability_scope.task_classes' and 'capability_scope.designed_for'"))
	} else {
		if strings.TrimSpace(p.scope["task_classes"]) == "" {
			out = append(out, add("agent prompt reads untrusted input but is missing 'capability_scope.task_classes': "+
				"an agent whose scope is undeclared has no bound on what it may be dispatched for"))
		}
		if strings.TrimSpace(p.scope["designed_for"]) == "" {
			out = append(out, add("agent prompt reads untrusted input but is missing 'capability_scope.designed_for': "+
				"the one-line statement of what the prompt was built to do"))
		}
	}

	// The canary must be a REGULAR, non-empty file. A bare existence test is
	// satisfied by an empty file or by a symlink to anything at all, and a canary
	// that asserts nothing is worse than an absent one: it reports the contract as
	// met. Lstat, so a symlink is judged as the link it is rather than as its
	// target.
	canaryRel := filepath.Join(dir, p.name, "fixtures", agentCanaryFixture)
	info, err := os.Lstat(filepath.Join(repoRoot, canaryRel))
	switch {
	case err != nil:
		out = append(out, add("agent prompt reads untrusted input but ships no "+agentCanaryFixture+
			" fixture (expected at "+filepath.ToSlash(canaryRel)+"): the canary is the only thing that asserts the "+
			"prompt quotes hostile input as data rather than obeying it"))
	case !info.Mode().IsRegular():
		out = append(out, add("agent prompt's "+filepath.ToSlash(canaryRel)+
			" is not a regular file; a canary that is a link or a directory asserts nothing"))
	case info.Size() == 0:
		out = append(out, add("agent prompt's "+filepath.ToSlash(canaryRel)+
			" is empty; a canary that carries no hostile payload and no expectation reports the contract met without testing it"))
	}
	return out
}

// checkAgentChangelog is sub-check 3, in two parts.
//
// The TREE-shaped part runs always: every prompt's current prompt_version must
// have its own entry in the per-agent CHANGELOG. That is the itd-5 contract
// stated as a property of the tree ("one entry per agent per version bump"), and
// it needs no git, so it holds on every invocation — a new prompt with no entry
// and a bumped version with no entry are both caught by `make record-lint`.
//
// The DIFF-shaped part runs only when a range is armed, and it does the one thing
// the tree cannot say: a prompt whose body changed in the range without its
// prompt_version changing. The version is what the entry is keyed on, so an
// unbumped edit is a change that can never acquire an entry — it is invisible to
// the tree-shaped part by construction.
//
// The range is supplied by the CALLER (ArmAgentDiff, from a CI invocation), never
// read out of the in-tree config, for the reason ArmReceiptGate states: a gate a
// committer can point at an empty range is a gate a committer can disarm.
func checkAgentChangelog(repoRoot, dir string, prompts []agentPrompt, cfg RuleConfig) ([]Finding, error) {
	changelogRel := cfg.Changelog
	if changelogRel == "" {
		changelogRel = filepath.ToSlash(filepath.Join(dir, "CHANGELOG.md"))
	}
	// The changelog read is guarded exactly as the prompt read above is, and for
	// the same reason: BOTH the path and the file it names are repo-controlled —
	// the path out of the in-tree lint config, the file out of the tree — so a
	// fork pull request supplies them and CI's `go run ./cmd/record-lint` is what
	// reads them. Unguarded, `agents/CHANGELOG.md` as a symlink to /dev/zero was
	// read until the runner ran out of memory, and a configured
	// `"changelog": "../../../../etc/hosts"` read outside the checkout and echoed
	// the traversed path into the finding's File field.
	//
	// containedRepoPath refuses the configured path, containedRealPath refuses a
	// symlink out of the tree, and fsutil.ReadGuarded (O_NOFOLLOW, non-regular
	// refused, size capped) refuses the device. No third mechanism: these are the
	// primitives the siblings in this function already use.
	if err := containedRepoPath(changelogRel); err != nil {
		return nil, &configError{ruleAgentContract + ": changelog " + quote(changelogRel) + " " + err.Error() +
			"; the lint reads only inside the repository"}
	}
	realPath, err := containedRealPath(repoRoot, filepath.Join(repoRoot, filepath.FromSlash(changelogRel)))
	if err != nil {
		return nil, &configError{ruleAgentContract + ": changelog " + quote(changelogRel) + " " + err.Error() +
			"; the lint reads only inside the repository"}
	}
	// A changelog that is simply absent is a state, not a fault: every prompt
	// then lacks an entry, which is what the loop below reports.
	data, err := fsutil.ReadGuarded(realPath, maxAgentPromptBytes)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	entries := agentChangelogEntries(string(data))

	var out []Finding
	for _, p := range prompts {
		version := promptVersionOf(p)
		if version == "" {
			continue // already reported by the frontmatter sub-check
		}
		if entries[p.name+" "+version] {
			continue
		}
		out = append(out, Finding{
			File: changelogRel, Line: 0, RuleID: ruleAgentContract, Severity: cfg.Severity,
			Message: changelogRel + " carries no entry for " + p.name + " " + version +
				"; the itd-5 contract announces every prompt version in the changelog (add a '### " +
				p.name + " " + version + "' entry)",
		})
	}

	unbumped, err := checkAgentVersionBump(repoRoot, changelogRel, prompts, cfg)
	if err != nil {
		return nil, err
	}
	return append(out, unbumped...), nil
}

// checkAgentVersionBump is the diff-armed half: a prompt changed in the range
// whose diff adds no prompt_version line changed without a bump, so the entry the
// tree-shaped half demands can never be written for it.
//
// The bump is read out of the DIFF rather than by resolving the range's base
// revision and reading the file there: `git diff --unified=0 <range> -- <path>`
// already answers "did this line change" in one call, and parsing a base out of
// the three range spellings would be a second, avoidable grammar.
func checkAgentVersionBump(repoRoot, changelogRel string, prompts []agentPrompt, cfg RuleConfig) ([]Finding, error) {
	if cfg.DiffRange == "" {
		return nil, nil
	}
	if !agentDiffRangeRe.MatchString(cfg.DiffRange) {
		return nil, &configError{ruleAgentContract + ": diff range " + quote(cfg.DiffRange) +
			" is not a git revision or revision range; a value git would read as an option is refused"}
	}
	if !gitutil.InRepo(repoRoot) {
		return nil, nil
	}

	changed, err := changedPaths(repoRoot, cfg.DiffRange)
	if err != nil {
		return nil, &configError{ruleAgentContract + ": reading the diff for " + quote(cfg.DiffRange) +
			": " + err.Error()}
	}

	var out []Finding
	for _, p := range prompts {
		rel := filepath.ToSlash(p.rel)
		if !changed[rel] {
			continue
		}
		bumped, err := promptVersionChanged(repoRoot, cfg.DiffRange, rel)
		if err != nil {
			return nil, &configError{ruleAgentContract + ": reading the diff for " + quote(rel) + ": " + err.Error()}
		}
		if bumped {
			continue
		}
		out = append(out, Finding{
			File: rel, Line: 1, RuleID: ruleAgentContract, Severity: cfg.Severity,
			Message: "agent '" + p.name + "' changed in this diff without a 'prompt_version' bump, so no new " +
				changelogRel + " entry can be written for it; bump the version (MAJOR schema break, MINOR " +
				"behaviour, PATCH edit) and add the matching entry",
		})
	}
	return out, nil
}

// promptVersionChanged reports whether the range's diff for one path adds a
// prompt_version line.
func promptVersionChanged(repoRoot, rangeSpec, rel string) (bool, error) {
	diff, err := gitutil.Run(repoRoot, "diff", "--unified=0", rangeSpec, "--", rel)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+prompt_version:") {
			return true, nil
		}
	}
	return false, nil
}

// promptVersionOf reads a prompt's declared version, unquoted.
func promptVersionOf(p agentPrompt) string {
	return strings.Trim(strings.TrimSpace(p.fields["prompt_version"].value), `"'`)
}

// agentChangelogHeadingRe matches a per-agent entry heading — "### <name>
// <semver>", with anything after (the changelog annotates a rename in the same
// heading). The heading level is not pinned: an entry is an entry at any depth.
var agentChangelogHeadingRe = regexp.MustCompile(`^#{1,6}[ \t]+([a-z0-9][a-z0-9-]*)[ \t]+v?(\d+\.\d+\.\d+)\b`)

// agentChangelogEntries indexes a changelog by "<agent> <version>".
func agentChangelogEntries(text string) map[string]bool {
	entries := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if m := agentChangelogHeadingRe.FindStringSubmatch(line); m != nil {
			entries[m[1]+" "+m[2]] = true
		}
	}
	return entries
}

// changedPaths returns the slash-separated repo-relative paths the range touched.
// NUL-delimited (-z) so a path git would otherwise quote is read verbatim, and
// `--` terminates the revision list so a range that somehow survived validation
// still cannot be read as a pathspec.
func changedPaths(repoRoot, rangeSpec string) (map[string]bool, error) {
	out, err := gitutil.Run(repoRoot, "diff", "--name-only", "-z", rangeSpec, "--")
	if err != nil {
		return nil, err
	}
	changed := map[string]bool{}
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			changed[p] = true
		}
	}
	return changed, nil
}

// agentCapabilityScope reads the members of the nested `capability_scope` block
// out of a prompt's leading frontmatter. The block runs from `capability_scope:`
// to the next TOP-LEVEL key (a line at column 0) or the closing delimiter.
//
// Membership is decided by column, not by "did this line parse as a nested key".
// The first cut flipped out of the block on any line the nested-key pattern did
// not match — which a YAML BLOCK SEQUENCE item (`  - oracle_review`) does not —
// so a prompt whose task_classes is written as a block list lost every member
// after it, and the gate told its author to add a `designed_for` that was
// plainly there. A member written as a block sequence takes its items as its
// value, so it reads as present either way; the inline-list convention
// (agents/README.md) is a style rule this parser does not adjudicate.
func agentCapabilityScope(lines []string) map[string]string {
	if start := frontmatterOpen(lines); start > 0 {
		lines = lines[start:]
	}
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	scope := map[string]string{}
	inScope, member := false, ""
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(line) == "---" {
			break
		}
		indented := line != "" && (line[0] == ' ' || line[0] == '\t')
		if !indented {
			inScope, member = strings.HasPrefix(line, "capability_scope:"), ""
			continue
		}
		if !inScope {
			continue
		}
		if m := agentNestedKeyRe.FindStringSubmatch(line); m != nil {
			member = m[1]
			if _, seen := scope[member]; !seen {
				scope[member] = strings.TrimSpace(m[2])
			}
			continue
		}
		// A block-sequence item belongs to the member it sits under.
		if item := strings.TrimSpace(line); member != "" && strings.HasPrefix(item, "- ") {
			scope[member] = strings.TrimSpace(scope[member] + " " + strings.TrimSpace(item[2:]))
		}
	}
	if len(scope) == 0 {
		return nil
	}
	return scope
}

// isTrueValue reads a frontmatter boolean the way the prompts spell it. Only the
// affirmative spellings count: anything else (including an empty value) is not a
// declaration that the prompt reads untrusted input.
func isTrueValue(v string) bool {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(v), `"'`)) {
	case "true", "yes":
		return true
	}
	return false
}
