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
// The derivation is a line reader over the GitHub Actions layout, not a YAML
// parser (no dependency is added for it), and it fails toward omission: a job
// whose check name GitHub computes at run time — a matrix job, a job named by an
// expression, a reusable-workflow call — is left out rather than guessed, and
// the runbook says a hand-added context may be needed. Every name that is kept
// is held to an allowlist before it is written anywhere, because the scaffold
// writes it into YAML and a hostile workflow file must not be able to inject.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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

// jobKeyRe matches a job id line under `jobs:`; the capture is the id.
var jobKeyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*):\s*(#.*)?$`)

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
	})
	return sortedKeys(seen)
}

// forEachWorkflow calls fn with the name and contents of each of repoRoot's
// own CI workflow files, in directory order, bounded by maxWorkflowFiles. The
// scaffold's own workflows (the release pair and the dependency re-authoring)
// are never passed: the re-authoring workflow runs on pull requests but judges
// none, and counting it would change the release rendering the moment a
// repository opted in. A file that cannot be read is skipped.
func forEachWorkflow(repoRoot string, fn func(name, src string)) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(workflowsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
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
			return
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxWorkflowBytes)
		if err != nil {
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
// since a queue entry waits on the same contexts. Each workflow or job left
// out is named in notes, so a person can see why a check is not required.
// Fails toward omission, like DeriveCIChecks: an empty answer requires nothing.
func RequiredChecks(repoRoot, defaultBranch string, mergeQueue bool) (checks, notes []string) {
	seen := map[string]bool{}
	forEachWorkflow(repoRoot, func(name, src string) {
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
		case flowMapping(on, "pull_request") || flowMapping(on, "pull_request_target") || flowBody(prBody):
			why = "writes its pull-request trigger as a flow mapping or an alias, whose filters abcd does not read"
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
		for _, job := range splitJobs(topLevelBlock(lines, "jobs")) {
			if cs, ok := job.checkNames(); ok {
				for _, c := range cs {
					seen[c] = true
				}
				continue
			}
			notes = append(notes, name+": job "+job.id+" reports a name only the run knows "+
				"(a matrix, an expression or a reusable workflow), so it is not required")
		}
	})
	return sortedKeys(seen), notes
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

// flowBody reports whether an event's body hides its filters from the line
// reader: a flow mapping on the line after its key, an alias, or a merge key
// (`<<: *name`) that brings filters in from an anchor.
func flowBody(body []string) bool {
	first := true
	for _, l := range body {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if first {
			// An anchor or tag alone on its line leaves the node's value to come.
			if t = stripNodeProps(t); t == "" {
				continue
			}
		}
		if first && (strings.HasPrefix(t, "{") || strings.HasPrefix(t, "*")) {
			return true
		}
		first = false
		if strings.HasPrefix(t, "<<") {
			return true
		}
	}
	return false
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
	jobs := topLevelBlock(lines, "jobs")
	var out []string
	for _, job := range splitJobs(jobs) {
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

// ciJob is one job's id and its own direct keys.
type ciJob struct {
	id     string
	name   string
	named  bool
	matrix bool
	uses   bool
	// matrixLines are the lines indented beneath the job's `matrix:` key.
	matrixLines []string
}

// splitJobs reads the jobs block into its jobs, recording each job's own
// `name:`, whether it declares a matrix, and whether it calls a reusable
// workflow. Job ids sit at the block's first indentation; a job's own keys at
// the next one.
func splitJobs(block []string) []ciJob {
	jobIndent := -1
	keyIndent := -1
	matrixIndent := -1
	var jobs []ciJob
	for _, l := range block {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " "))
		if matrixIndent >= 0 {
			if ind > matrixIndent {
				jobs[len(jobs)-1].matrixLines = append(jobs[len(jobs)-1].matrixLines, l)
				continue
			}
			matrixIndent = -1
		}
		if jobIndent < 0 {
			jobIndent = ind
		}
		switch {
		case ind == jobIndent:
			m := jobKeyRe.FindStringSubmatch(t)
			if m == nil {
				// Not a job id (a flow mapping, an anchor): stop trusting the block.
				return jobs
			}
			jobs = append(jobs, ciJob{id: m[1]})
			keyIndent = -1
		case ind > jobIndent && len(jobs) > 0:
			if keyIndent < 0 {
				keyIndent = ind
			}
			job := &jobs[len(jobs)-1]
			if ind == keyIndent {
				k, v, _ := strings.Cut(t, ":")
				switch strings.TrimSpace(k) {
				case "name":
					job.name, job.named = scalar(v), true
				case "uses":
					job.uses = true
				case "strategy":
					// An inline strategy may hold a matrix this reader cannot
					// expand, so the job's names are the run's to decide.
					if strings.TrimSpace(v) != "" {
						job.matrix = true
					}
				}
			} else if strings.HasPrefix(t, "matrix:") {
				job.matrix = true
				matrixIndent = ind
			}
		}
	}
	return jobs
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

// checkNames is checkName with one matrix shape expanded: a single key whose
// values are listed inline (`os: [a, b]`), with no include or exclude, which
// GitHub reports as one check per value, `<name> (<value>)`. Any other matrix
// still reports names only the run knows.
func (j ciJob) checkNames() ([]string, bool) {
	if !j.matrix {
		if n, ok := j.checkName(); ok {
			return []string{n}, true
		}
		return nil, false
	}
	if j.uses || len(j.matrixLines) != 1 {
		return nil, false
	}
	k, v, ok := strings.Cut(strings.TrimSpace(j.matrixLines[0]), ":")
	k = strings.TrimSpace(k)
	v = strings.TrimSpace(v)
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	if !ok || k == "include" || k == "exclude" || !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil, false
	}
	base := j.id
	if j.named {
		base = j.name
	}
	var out []string
	for _, item := range strings.Split(v[1:len(v)-1], ",") {
		item = scalar(item)
		name := base + " (" + item + ")"
		if item == "" || strings.Contains(item, "${{") || !checkNameRe.MatchString(name) {
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
