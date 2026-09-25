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
// the scaffold's own release workflows are never counted. An unreadable or
// absent workflows directory yields nil: the runbook then says no merge gate
// was found.
func DeriveCIChecks(repoRoot string) []string {
	dir := filepath.Join(repoRoot, filepath.FromSlash(workflowsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	files := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		if rel := workflowsDir + "/" + name; rel == ReleaseYMLPath || rel == AutoReleaseYMLPath {
			continue
		}
		if files++; files > maxWorkflowFiles {
			break
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxWorkflowBytes)
		if err != nil {
			continue
		}
		for _, check := range workflowChecks(string(data)) {
			seen[check] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
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
}

// splitJobs reads the jobs block into its jobs, recording each job's own
// `name:`, whether it declares a matrix, and whether it calls a reusable
// workflow. Job ids sit at the block's first indentation; a job's own keys at
// the next one.
func splitJobs(block []string) []ciJob {
	jobIndent := -1
	keyIndent := -1
	var jobs []ciJob
	for _, l := range block {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " "))
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
				}
			} else if strings.HasPrefix(t, "matrix:") {
				job.matrix = true
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
