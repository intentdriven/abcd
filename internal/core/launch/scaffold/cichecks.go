package scaffold

// Wiring to the managed repo's own CI check names (itd-93 AC1).
//
// A changelog-driven release is decided by merging the release pull request,
// so what gates that merge is what gates the release: the repository's own
// pull-request CI. The scaffold cannot configure branch protection — it writes
// files and holds no token — so it reads the repository's CI workflows, derives
// the check names its pull requests report, and writes them into the machinery
// it lays down: the runbook's merge-gate section, which tells the operator to
// require exactly these contexts on the default branch, and the release
// workflow's verify-job header, which names them as the merge gate the release
// job re-runs a subset of. Nothing is hard-coded to abcd's own check names.
//
// The jobs come from the workflow's parsed YAML. The triggers are read as a subset
// of YAML by a line reader, which refuses any other shape (unreadShape). The
// derivation fails toward omission: a job whose check name GitHub computes at run
// time — a matrix job, a job named by an expression, a reusable-workflow call — is
// left out rather than guessed, and the runbook says a hand-added context may be
// needed. Every name that is kept
// is held to an allowlist before it is written anywhere, because the scaffold
// writes it into YAML and a hostile workflow file must not be able to inject.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// workflowsDir is where GitHub Actions discovers workflows.
const workflowsDir = ".github/workflows"

// maxWorkflowFiles bounds the directory walk; a repository with more workflow
// files than this is not one whose CI the scaffold can summarise faithfully.
const maxWorkflowFiles = 64

// checkNameRe is the allowlist for a derived check name written into the
// scaffolded files: letters, digits, space and a small punctuation set. It
// excludes every YAML and shell metacharacter (`:`, `#`, quotes, `$`, braces,
// backticks, newlines), so a name can only ever be inert text.
var checkNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()/+-]{0,99}$`)

// jobIDRe is what GitHub accepts as a job id.
var jobIDRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// DeriveCIChecks reads repoRoot's own CI workflows and returns the check names
// its pull requests report, sorted and de-duplicated. Only workflows triggered
// by `pull_request`, `pull_request_target` or `merge_group` gate a merge, and
// the scaffold's own workflows (the release pair and the dependency
// re-authoring) are never counted. An unreadable or
// absent workflows directory yields nil: the runbook then says no merge gate
// was found.
func DeriveCIChecks(repoRoot string) []string {
	seen := map[string]bool{}
	forEachWorkflow(repoRoot, func(_, src string) {
		for _, check := range workflowChecks(src) {
			seen[check] = true
		}
	}, func(_, _ string) {})
	return sortedKeys(seen)
}

// forEachWorkflow calls fn with the name and contents of each of repoRoot's
// own CI workflow files, in directory order, bounded by maxWorkflowFiles, and
// unread with what it could not read and why; an absent directory is neither. The
// scaffold's own workflows (the release pair and the dependency re-authoring)
// are never passed: the re-authoring workflow runs on pull requests but judges
// none, and counting it would change the release rendering the moment a
// repository opted in. A file that cannot be read is skipped.
func forEachWorkflow(repoRoot string, fn func(name, src string), unread func(name, why string)) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(workflowsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			unread(workflowsDir, "cannot be listed")
		}
		return
	}
	files := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		if rel := workflowsDir + "/" + name; rel == ReleaseYMLPath || rel == AutoReleaseYMLPath || rel == ReauthorYMLPath {
			continue
		}
		if files++; files > maxWorkflowFiles {
			unread(workflowsDir, fmt.Sprintf("holds more than the %d workflow files abcd reads", maxWorkflowFiles))
			return
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxWorkflowBytes)
		if err != nil {
			unread(name, "cannot be read safely (a link, or larger than abcd reads)")
			continue
		}
		fn(name, string(data))
	}
}

// sortedKeys returns a set's members sorted, or nil for an empty set.
func sortedKeys(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// RequiredChecks returns the check names a ruleset may require on
// defaultBranch without blocking a merge (itd-2610290746077716): the checks
// EVERY pull request into it reports. Stricter than DeriveCIChecks, which
// lists what a pull request may report, because a required check that never
// reports holds the merge for ever. A workflow is counted only when it runs on
// `pull_request` or `pull_request_target` with no `paths`/`paths-ignore`
// filter, no `branches-ignore` filter, and a `branches` filter (if any) that
// matches defaultBranch; and, when mergeQueue is set, also on `merge_group`,
// since a queue entry waits on the same contexts. A matrix job a run may skip is
// left out as well (skippableJobs): skipped, it reports one check under its
// unexpanded name, so the contexts its values name never arrive. A job with no
// matrix still counts, since skipped it reports its one name, which a ruleset
// reads as passed. Each workflow or job left out is named in notes, so a person can
// see why a check is not required.
// The trigger is read as a subset of YAML (unreadShape): a trigger in any
// other form is left out, never read as unfiltered.
// Fails toward omission, like DeriveCIChecks: an empty answer requires nothing.
//
// A shorter list reads as an answer, so a workflow the read cannot take whole is
// no omission: a file that does not parse or cannot be read, too many files, or a
// counted workflow whose jobs or needs GitHub would refuse return an error naming
// each, and no checks at all.
func RequiredChecks(repoRoot, defaultBranch string, mergeQueue bool) (checks, notes []string, err error) {
	seen := map[string]bool{}
	var refused []string
	refuse := func(name, why string) { refused = append(refused, name+" "+why) }
	forEachWorkflow(repoRoot, func(name, src string) {
		jobs, err := readJobs(src)
		if err != nil {
			refuse(name, "does not read as a workflow: "+boundedError(err))
			return
		}
		lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
		on := topLevelBlock(lines, "on")
		prBody, pr := eventBlock(on, "pull_request")
		if !pr {
			prBody, pr = eventBlock(on, "pull_request_target")
		}
		if !pr {
			return
		}
		why := ""
		switch {
		case flowMapping(on, "pull_request") || flowMapping(on, "pull_request_target") || unreadShape(on, prBody):
			why = "writes its pull-request trigger in a form abcd does not read (a flow mapping, an alias, " +
				"an anchor, a tag, a merge key or a key that is not one of GitHub's filters)"
		case blockHasKey(prBody, "paths") || blockHasKey(prBody, "paths-ignore"):
			why = "runs only when some paths change"
		case blockHasKey(prBody, "branches-ignore"):
			why = "has a branches-ignore filter on its pull-request trigger"
		case blockHasKey(prBody, "branches") && !branchListMatches(blockList(prBody, "branches"), defaultBranch):
			why = "its pull-request trigger leaves out the default branch"
		case blockHasKey(prBody, "types") && !coversPushes(blockList(prBody, "types")):
			why = "runs only on some pull-request activities (its types filter leaves out opened, synchronize or reopened)"
		case mergeQueue && !hasEvent(on, "merge_group"):
			why = "does not run on the merge queue, which is on"
		}
		if why != "" {
			notes = append(notes, name+" "+why+", so its checks are not required")
			return
		}
		if len(jobs) == 0 {
			refuse(name, "names no jobs")
			return
		}
		skippable, err := skippableJobs(jobs)
		if err != nil {
			refuse(name, "does not read as a workflow: "+boundedError(err))
			return
		}
		for _, job := range jobs {
			if job.matrix && skippable[job.id] {
				notes = append(notes, name+": job "+job.id+" is a matrix job abcd cannot tell every run reports: its "+
					"`if:` or a job it needs may skip it, and a skipped matrix job reports only its unexpanded name, so "+
					"it is not required. If every pull request does report it, require it by hand or through "+
					"github.required_checks")
				continue
			}
			if cs, ok := job.checkNames(); ok {
				for _, c := range cs {
					seen[c] = true
				}
				continue
			}
			notes = append(notes, name+": job "+job.id+" reports a name only the run knows "+
				"(a matrix, an expression or a reusable workflow), so it is not required")
		}
	}, refuse)
	if len(refused) > 0 {
		return nil, nil, errors.New("abcd cannot read every workflow, so it proposes no required checks: " +
			strings.Join(refused, "; ") + ". Fix the file, or name the checks in github.required_checks")
	}
	return sortedKeys(seen), notes, nil
}

// boundedError is an error from reading a workflow file, short enough for a note:
// the parser can quote the file.
func boundedError(err error) string {
	const limit = 200
	if s := err.Error(); len(s) <= limit {
		return s
	}
	return strings.ToValidUTF8(err.Error()[:limit], "") + "..."
}

// flowMapping reports whether the `on:` block, or event's own entry in it, is
// written as a flow mapping (`{...}`) or an alias (`*name`). Its filters are
// then inline or elsewhere, where the line reader cannot see them, so such a
// trigger is never read as unfiltered.
func flowMapping(on []string, event string) bool {
	re := eventRe(event)
	level := onLevel(on)
	for _, l := range on {
		if ind := indentOf(l); ind != 0 && ind != level {
			continue
		}
		t := strings.TrimSpace(l)
		if i := strings.Index(t, " #"); i >= 0 {
			t = t[:i]
		}
		if t = stripNodeProps(t); strings.HasPrefix(t, "{") && re.MatchString(t) {
			return true
		}
		if k, v, ok := strings.Cut(t, ":"); ok && strings.Trim(strings.TrimSpace(k), `"'`) == event {
			if v = stripNodeProps(v); strings.HasPrefix(v, "{") || strings.HasPrefix(v, "*") {
				return true
			}
		}
	}
	return false
}

// prFilterKeys are the filters GitHub reads under a pull-request trigger.
var prFilterKeys = map[string]bool{
	"branches": true, "branches-ignore": true, "paths": true, "paths-ignore": true, "types": true,
}

// plainKeyRe matches a key written plainly: a bare word, or one quoted whole.
var plainKeyRe = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_-]*|"[A-Za-z_][A-Za-z0-9_-]*"|'[A-Za-z_][A-Za-z0-9_-]*')$`)

// unreadShape reports whether the `on:` block's events or the pull-request
// trigger's body leave the YAML subset this line reader parses. That subset is
// the safety net under RequiredChecks: at the events' level only plain keys
// and list items; under the trigger only GitHub's five filter keys, written
// plainly, whose values are an inline list or `- item` lines, with no anchor,
// alias or tag in front of a value or an item. Any other shape (a key behind
// an anchor or a tag, a complex `?` key, a merge key, a flow mapping, an alias,
// an unknown key) is unread, and the workflow is left out rather than read as
// unfiltered, so a shape not foreseen here falls to omission, never to a
// required check.
func unreadShape(on, prBody []string) bool {
	if level := onLevel(on); level > 0 {
		for _, l := range on {
			t := plainLine(l)
			if t == "" || indentOf(l) != level || t == "-" || strings.HasPrefix(t, "- ") {
				continue
			}
			if k, _, ok := strings.Cut(t, ":"); !ok || !plainKeyRe.MatchString(strings.TrimSpace(k)) {
				return true
			}
		}
	}
	keyIndent := -1
	for _, l := range prBody {
		t := plainLine(l)
		if t == "" {
			continue
		}
		ind := indentOf(l)
		if keyIndent < 0 {
			keyIndent = ind
		}
		switch {
		case ind < keyIndent:
			return true
		case ind > keyIndent:
			// A filter's value: only a `- item` line.
			if !strings.HasPrefix(t, "- ") || !plainValue(strings.TrimSpace(t[2:])) {
				return true
			}
		default:
			k, v, ok := strings.Cut(t, ":")
			k = strings.TrimSpace(k)
			if !ok || !plainKeyRe.MatchString(k) || !prFilterKeys[strings.Trim(k, `"'`)] {
				return true
			}
			if v = strings.TrimSpace(v); v != "" && !plainValue(v) {
				return true
			}
		}
	}
	return false
}

// plainLine is a line trimmed of its indentation and trailing comment, or ""
// for a blank or comment line.
func plainLine(l string) string {
	t := strings.TrimSpace(l)
	if strings.HasPrefix(t, "#") {
		return ""
	}
	if i := strings.Index(t, " #"); i >= 0 {
		t = strings.TrimSpace(t[:i])
	}
	return t
}

// plainValue reports whether a filter's value, or one list item, is a plain
// or quoted scalar or a one-line flow sequence of them: nothing a YAML node
// property (`&` anchor, `*` alias, `!` tag), a block scalar or a nested
// collection could change the meaning of.
func plainValue(v string) bool {
	if strings.HasPrefix(v, "[") {
		if !strings.HasSuffix(v, "]") {
			return false
		}
		for _, item := range strings.Split(v[1:len(v)-1], ",") {
			if item = strings.TrimSpace(item); item == "" || !plainValue(item) {
				return false
			}
		}
		return true
	}
	return v != "" && !strings.ContainsAny(v[:1], "&*!|>{[?@`%")
}

// stripNodeProps drops the anchor (`&name`) and tag (`!tag`) properties a
// YAML node may carry before its value, so a flow mapping or alias behind
// them is seen for what it is; an anchor names a value, it never hides one.
func stripNodeProps(t string) string {
	t = strings.TrimSpace(t)
	for strings.HasPrefix(t, "&") || strings.HasPrefix(t, "!") {
		i := strings.IndexAny(t, " \t")
		if i < 0 {
			return ""
		}
		t = strings.TrimSpace(t[i:])
	}
	return t
}

// coversPushes reports whether a pull-request `types:` filter keeps the three
// activities GitHub runs on by default, so the workflow still reports on every
// pull request and on every push to one.
func coversPushes(types []string) bool {
	have := map[string]bool{}
	for _, t := range types {
		have[t] = true
	}
	return have["opened"] && have["synchronize"] && have["reopened"]
}

// eventRe matches one event name standing alone in an `on:` value.
func eventRe(event string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^A-Za-z_])` + regexp.QuoteMeta(event) + `([^A-Za-z_]|$)`)
}

// hasEvent reports whether an `on:` block names event, in any of its forms
// (`on: event`, `on: [a, event]`, a list item, or a mapping key).
func hasEvent(on []string, event string) bool {
	re := eventRe(event)
	level := onLevel(on)
	for _, l := range on {
		if ind := indentOf(l); ind != 0 && ind != level {
			// Deeper than the event names: a filter or an input, never a trigger.
			continue
		}
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if i := strings.Index(t, " #"); i >= 0 {
			t = t[:i]
		}
		if k, _, ok := strings.Cut(t, ":"); ok && !strings.HasPrefix(t, "[") && !strings.HasPrefix(t, "-") {
			// A mapping line names only its key; its value is a filter.
			if re.MatchString(strings.Trim(strings.TrimSpace(k), `"'`)) {
				return true
			}
			continue
		}
		if re.MatchString(t) {
			return true
		}
	}
	return false
}

// eventBlock returns whether an `on:` block names event and, when event is a
// mapping key there, the lines indented beneath it (its filters).
func eventBlock(on []string, event string) (body []string, present bool) {
	if !hasEvent(on, event) {
		return nil, false
	}
	keyIndent := -1
	level := onLevel(on)
	for _, l := range on {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " "))
		if keyIndent >= 0 {
			if ind <= keyIndent {
				break
			}
			body = append(body, l)
			continue
		}
		if ind != 0 && ind != level {
			continue
		}
		if k, _, ok := strings.Cut(t, ":"); ok && strings.Trim(strings.TrimSpace(k), `"'`) == event {
			keyIndent = ind
		}
	}
	return body, true
}

// onLevel is the indentation of the event names in an `on:` block: that of
// its first indented line. The value written on the `on:` line itself is
// held at indentation 0.
func onLevel(on []string) int {
	for _, l := range on {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if ind := indentOf(l); ind > 0 {
			return ind
		}
	}
	return 0
}

// indentOf is a line's indentation in spaces.
func indentOf(l string) int {
	return len(l) - len(strings.TrimLeft(l, " "))
}

// blockHasKey reports whether one of body's lines is the key key.
func blockHasKey(body []string, key string) bool {
	for _, l := range body {
		if k, _, ok := strings.Cut(strings.TrimSpace(l), ":"); ok && strings.Trim(strings.TrimSpace(k), `"'`) == key {
			return true
		}
	}
	return false
}

// blockList returns the items of key's list in body: an inline flow sequence
// (`key: [a, 'b']`) or the `- item` lines indented beneath it.
func blockList(body []string, key string) []string {
	var out []string
	keyIndent := -1
	for _, l := range body {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " "))
		if keyIndent >= 0 {
			if ind <= keyIndent && !strings.HasPrefix(t, "-") || ind < keyIndent {
				break
			}
			if strings.HasPrefix(t, "-") {
				out = append(out, scalar(strings.TrimPrefix(t, "-")))
				continue
			}
			break
		}
		k, v, ok := strings.Cut(t, ":")
		if !ok || strings.Trim(strings.TrimSpace(k), `"'`) != key {
			continue
		}
		v = scalar(v)
		if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
			for _, item := range strings.Split(v[1:len(v)-1], ",") {
				if item = scalar(item); item != "" {
					out = append(out, item)
				}
			}
			return out
		}
		if v != "" {
			return append(out, v)
		}
		keyIndent = ind
	}
	return out
}

// branchListMatches reports whether a `branches` filter covers branch, by
// GitHub's filter globs: `**` matches any run of characters, `*` any run but
// `/`, and `?` one character. A negated pattern (`!x`) is not interpreted, so a
// list holding one is treated as not matching, the direction that requires
// nothing.
func branchListMatches(patterns []string, branch string) bool {
	matched := false
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") {
			return false
		}
		if branchGlob(p).MatchString(branch) {
			matched = true
		}
	}
	return matched
}

// branchGlob compiles one filter glob into an anchored regular expression.
func branchGlob(p string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString("[^/]*")
		case p[i] == '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// workflowChecks returns the check names one workflow file reports, or nil
// when the workflow does not gate a pull request.
func workflowChecks(src string) []string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if !gatesAMerge(topLevelBlock(lines, "on")) {
		return nil
	}
	jobs, err := readJobs(src)
	if err != nil {
		return nil
	}
	var out []string
	for _, job := range jobs {
		if name, ok := job.checkName(); ok {
			out = append(out, name)
		}
	}
	return out
}

// topLevelBlock returns the value of a top-level key: its inline remainder
// followed by every indented line up to the next top-level key.
func topLevelBlock(lines []string, key string) []string {
	var out []string
	in := false
	for _, l := range lines {
		if l == "" || strings.HasPrefix(strings.TrimSpace(l), "#") {
			if in {
				out = append(out, l)
			}
			continue
		}
		top := !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t")
		if top {
			if in {
				break
			}
			k, rest, ok := strings.Cut(l, ":")
			k = strings.Trim(strings.TrimSpace(k), `"'`)
			if ok && k == key {
				in = true
				if r := strings.TrimSpace(rest); r != "" {
					out = append(out, r)
				}
			}
			continue
		}
		if in {
			out = append(out, l)
		}
	}
	return out
}

// mergeTriggerRe matches an event that runs a workflow against a pull request
// or its merge-queue entry.
var mergeTriggerRe = regexp.MustCompile(`(^|[^A-Za-z_])(pull_request|pull_request_target|merge_group)([^A-Za-z_]|$)`)

// gatesAMerge reports whether an `on:` block names a pull-request or
// merge-queue event.
func gatesAMerge(on []string) bool {
	for _, l := range on {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if i := strings.Index(t, " #"); i >= 0 {
			t = t[:i]
		}
		if mergeTriggerRe.MatchString(t) {
			return true
		}
	}
	return false
}

// jobCondition is what a job's `if:` says about whether a run going ahead runs it.
type jobCondition int

const (
	// condFollowsNeeds is no `if:`, or one that adds nothing to the implicit
	// success(): the job runs unless an ancestor was skipped or failed.
	condFollowsNeeds jobCondition = iota
	// condAlwaysRuns reads only the run's own state, so the job runs whatever
	// happened to its ancestors.
	condAlwaysRuns
	// condCannotTell is any other condition, which this reader does not evaluate.
	condCannotTell
)

// ciJob is one job as its workflow's parsed YAML gives it.
type ciJob struct {
	id    string
	name  string
	named bool
	uses  bool
	// matrix says the job has a matrix, or a strategy too computed to look into.
	matrix bool
	// matrixValues are the values of a matrix with one key, listed, with no
	// include or exclude, each of which GitHub reports as it is written. It is nil
	// for any other matrix.
	matrixValues []string
	cond         jobCondition
	needs        []string
}

// readJobs reads a workflow's jobs in file order from its parsed YAML; a workflow
// with no `jobs:` key has none. It refuses a file that does not parse, and a jobs
// block GitHub would refuse.
func readJobs(src string) ([]ciJob, error) {
	var doc struct {
		Jobs yaml.Node `yaml:"jobs"`
	}
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		return nil, err
	}
	if doc.Jobs.IsZero() {
		return nil, nil
	}
	block := resolved(&doc.Jobs)
	if block.Kind != yaml.MappingNode {
		return nil, errors.New("its jobs key is not a mapping")
	}
	var jobs []ciJob
	seen := map[string]bool{}
	for i := 0; i+1 < len(block.Content); i += 2 {
		key := block.Content[i]
		if key.Kind != yaml.ScalarNode || !jobIDRe.MatchString(key.Value) {
			return nil, fmt.Errorf("line %d holds a job id GitHub would refuse", key.Line)
		}
		if seen[key.Value] {
			return nil, fmt.Errorf("it defines job %s twice", key.Value)
		}
		seen[key.Value] = true
		job, err := readJob(key.Value, resolved(block.Content[i+1]))
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// readJob reads one job's own keys; steps and everything under them are not its
// keys, so nothing written there is read as one.
func readJob(id string, node *yaml.Node) (ciJob, error) {
	if node.Kind != yaml.MappingNode {
		return ciJob{}, fmt.Errorf("job %s is not a mapping", id)
	}
	var raw struct {
		Name     yaml.Node `yaml:"name"`
		If       yaml.Node `yaml:"if"`
		Needs    yaml.Node `yaml:"needs"`
		Uses     yaml.Node `yaml:"uses"`
		Strategy yaml.Node `yaml:"strategy"`
	}
	if err := node.Decode(&raw); err != nil {
		return ciJob{}, fmt.Errorf("job %s: %w", id, err)
	}
	needs, err := readNeeds(&raw.Needs)
	if err != nil {
		return ciJob{}, fmt.Errorf("job %s: %w", id, err)
	}
	job := ciJob{id: id, needs: needs, uses: !raw.Uses.IsZero(), cond: readCondition(&raw.If)}
	if !raw.Name.IsZero() {
		job.named, job.name = true, resolved(&raw.Name).Value
	}
	job.matrix, job.matrixValues = readMatrix(&raw.Strategy)
	return job, nil
}

// resolved is a node, or what it aliases.
func resolved(n *yaml.Node) *yaml.Node {
	for n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

// isNull reports a key given no value.
func isNull(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode && n.Tag == "!!null" }

// readNeeds reads `needs:`: absent or null, one id, or a list of ids.
func readNeeds(n *yaml.Node) ([]string, error) {
	if n.IsZero() {
		return nil, nil
	}
	n = resolved(n)
	switch {
	case isNull(n):
		return nil, nil
	case n.Kind == yaml.ScalarNode:
		return []string{n.Value}, nil
	case n.Kind != yaml.SequenceNode:
		return nil, errors.New("its needs key is neither a job id nor a list of them")
	}
	var needs []string
	for _, item := range n.Content {
		item = resolved(item)
		if isNull(item) {
			continue
		}
		if item.Kind != yaml.ScalarNode {
			return nil, errors.New("its needs list holds something other than a job id")
		}
		needs = append(needs, item.Value)
	}
	return needs, nil
}

// readCondition judges a job's `if:`, written bare or as one expression. It
// evaluates nothing: only the conditions whose answer needs no context are known.
func readCondition(n *yaml.Node) jobCondition {
	if n.IsZero() {
		return condFollowsNeeds
	}
	n = resolved(n)
	if n.Kind != yaml.ScalarNode || (n.Tag != "!!str" && n.Tag != "!!bool") {
		return condCannotTell
	}
	cond := strings.TrimSpace(n.Value)
	if inner, ok := strings.CutPrefix(cond, "${{"); ok {
		inner, closed := strings.CutSuffix(inner, "}}")
		if !closed || strings.Contains(inner, "${{") {
			return condCannotTell
		}
		cond = strings.TrimSpace(inner)
	}
	switch cond {
	case "always()", "!cancelled()":
		return condAlwaysRuns
	case "success()", "true":
		return condFollowsNeeds
	}
	return condCannotTell
}

// readMatrix reads a job's `strategy:`: whether it has a matrix, and the values of
// the one shape whose check names the file settles.
func readMatrix(n *yaml.Node) (matrix bool, values []string) {
	if n.IsZero() {
		return false, nil
	}
	n = resolved(n)
	if isNull(n) {
		return false, nil
	}
	if n.Kind != yaml.MappingNode {
		return true, nil
	}
	var strategy struct {
		Matrix yaml.Node `yaml:"matrix"`
	}
	if err := n.Decode(&strategy); err != nil {
		return true, nil
	}
	if strategy.Matrix.IsZero() {
		return false, nil
	}
	m := resolved(&strategy.Matrix)
	if m.Kind != yaml.MappingNode || len(m.Content) != 2 {
		return true, nil
	}
	key, list := m.Content[0].Value, resolved(m.Content[1])
	if key == "include" || key == "exclude" || key == "<<" || list.Kind != yaml.SequenceNode {
		return true, nil
	}
	for _, item := range list.Content {
		item = resolved(item)
		if item.Kind != yaml.ScalarNode || !reportsAsWritten(item) {
			return true, nil
		}
		values = append(values, item.Value)
	}
	return true, values
}

// plainIntRe is an integer written the one way GitHub reports it.
var plainIntRe = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// reportsAsWritten reports whether GitHub names a matrix value as its text: a
// string with no expression in it, a plain integer or a boolean. It reads a float
// as a number, so an unquoted 1.10 reports as 1.1.
func reportsAsWritten(item *yaml.Node) bool {
	switch item.Tag {
	case "!!str":
		return item.Value != "" && !strings.Contains(item.Value, "${{")
	case "!!int":
		return plainIntRe.MatchString(item.Value)
	case "!!bool":
		return item.Value == "true" || item.Value == "false"
	}
	return false
}

// skippableJobs reports, by job id, the jobs a run going ahead may skip. A job with
// a condition this reader cannot judge may be skipped. So may one following the
// implicit success() with such a job anywhere among its ancestors: success() reads
// every ancestor, including through a job that ran under always()
// (actions/runner#491). A job whose own condition is always() or !cancelled() runs.
// A need that names no job, and a cycle, are refused, as GitHub refuses the file.
func skippableJobs(jobs []ciJob) (map[string]bool, error) {
	byID := make(map[string]ciJob, len(jobs))
	for _, j := range jobs {
		byID[j.id] = j
	}
	const walking, walked = 1, 2
	state := make(map[string]int, len(jobs))
	behind := make(map[string]bool, len(jobs))
	var walk func(j ciJob) error
	walk = func(j ciJob) error {
		state[j.id] = walking
		for _, id := range j.needs {
			need, ok := byID[id]
			if !ok {
				return fmt.Errorf("job %s needs %q, which is no job in the file", j.id, id)
			}
			switch state[id] {
			case walking:
				return fmt.Errorf("job %s needs itself through %s", j.id, id)
			case 0:
				if err := walk(need); err != nil {
					return err
				}
			}
			if need.cond == condCannotTell || behind[id] {
				behind[j.id] = true
			}
		}
		state[j.id] = walked
		return nil
	}
	skippable := make(map[string]bool, len(jobs))
	for _, j := range jobs {
		if state[j.id] == 0 {
			if err := walk(j); err != nil {
				return nil, err
			}
		}
		skippable[j.id] = j.cond == condCannotTell || (j.cond == condFollowsNeeds && behind[j.id])
	}
	return skippable, nil
}

// checkName is the check context GitHub reports for the job, when it is
// knowable from the file: the job's `name:`, else its id. A matrix job, a
// reusable-workflow call and an expression-named job report names only the run
// knows, so they are omitted, never guessed.
func (j ciJob) checkName() (string, bool) {
	if j.matrix || j.uses {
		return "", false
	}
	name := j.id
	if j.named {
		name = j.name
	}
	if !checkNameRe.MatchString(name) {
		return "", false
	}
	return name, true
}

// checkNames is checkName with one matrix shape expanded (ciJob.matrixValues),
// which GitHub reports as one check per value, `<name> (<value>)`. Any other matrix
// still reports names only the run knows.
func (j ciJob) checkNames() ([]string, bool) {
	if !j.matrix {
		if n, ok := j.checkName(); ok {
			return []string{n}, true
		}
		return nil, false
	}
	if j.uses {
		return nil, false
	}
	base := j.id
	if j.named {
		base = j.name
	}
	var out []string
	for _, value := range j.matrixValues {
		name := base + " (" + value + ")"
		if !checkNameRe.MatchString(name) {
			return nil, false
		}
		out = append(out, name)
	}
	return out, len(out) > 0
}

// scalar reads a plain or quoted YAML scalar from the text after a key's colon,
// dropping a trailing comment on an unquoted value.
func scalar(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') {
		if end := strings.IndexByte(v[1:], v[0]); end >= 0 {
			return v[1 : 1+end]
		}
		return v
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
