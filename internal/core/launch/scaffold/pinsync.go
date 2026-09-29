package scaffold

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The committed workflows are a RENDERED artifact of these templates, and
// TestSelfScaffoldParity holds the two byte-identical. Dependabot only ever sees
// the rendered side — its github-actions ecosystem discovers `.github/workflows/*`
// and composite action manifests, and no ecosystem scans a .tmpl under internal/ —
// so every action bump it opens breaks parity and can never go green on its own
// (iss-209). This file closes that loop from the only direction that works:
// propagate the pin the bump landed on the workflow BACK into the template.
//
// The reverse (re-rendering the template over the workflow) would revert the bump
// and turn a red PR into a silently useless green one, which is why the flow is
// one-way and stated here rather than left to be rediscovered.
const (
	releaseTemplateRel     = "internal/core/launch/scaffold/templates/release.yml.tmpl"
	autoReleaseTemplateRel = "internal/core/launch/scaffold/templates/auto-release.yml.tmpl"
	reauthorTemplateRel    = "internal/core/launch/scaffold/templates/dependency-reauthor.yml.tmpl"
)

// maxTemplateBytes caps the guarded reads below, matching maxWorkflowBytes: these
// are short text files, and anything larger is not one this repo wrote.
const maxTemplateBytes = maxWorkflowBytes

// pinSyncPairs maps each committed workflow to the template it was rendered from.
// Only these three carry action pins; runbook.md.tmpl has none.
var pinSyncPairs = []struct{ workflow, template string }{
	{ReleaseYMLPath, releaseTemplateRel},
	{AutoReleaseYMLPath, autoReleaseTemplateRel},
	{ReauthorYMLPath, reauthorTemplateRel},
}

// usesLineRe splits a workflow `uses:` line into the parts the propagation must
// preserve and the one part it replaces. Group 1 is everything up to and including
// `uses: ` (indent and any `- ` list marker), group 2 the action name, group 3 the
// ref and any trailing ` # vX.Y.Z` comment.
//
// The action name stops at `@`, which is what keeps actions/attest from matching
// actions/attest-build-provenance: keys are exact names, never prefixes.
var usesLineRe = regexp.MustCompile(`^(\s*(?:-\s+)?uses:\s+)([^@\s]+)@(\S+.*)$`)

// SyncRepoPins propagates the action pins in the committed workflows into the
// templates they were rendered from. It returns the repo-relative paths of the
// templates that differ, newest state applied only when write is true.
//
// With write=false it is a pure report and touches nothing — the mode CI uses to
// decide whether a dependabot bump needs a follow-up commit, and the mode a
// contributor's test run uses to fail loudly rather than silently rewriting a
// tracked file underneath them.
func SyncRepoPins(repoRoot string, write bool) ([]string, error) {
	var changed []string
	for _, pair := range pinSyncPairs {
		workflow, err := fsutil.ReadGuarded(filepath.Join(repoRoot, filepath.FromSlash(pair.workflow)), maxWorkflowBytes)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", pair.workflow, err)
		}
		tmplPath := filepath.Join(repoRoot, filepath.FromSlash(pair.template))
		tmpl, err := fsutil.ReadGuarded(tmplPath, maxTemplateBytes)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", pair.template, err)
		}
		synced, differs, err := syncTemplatePins(workflow, tmpl)
		if err != nil {
			return nil, fmt.Errorf("%s -> %s: %w", pair.workflow, pair.template, err)
		}
		if !differs {
			continue
		}
		changed = append(changed, pair.template)
		if !write {
			continue
		}
		// Preserve the file's existing mode rather than imposing one: the template is
		// a tracked source file and this tool is not the authority on its permissions.
		if err := fsutil.WriteFileAtomicPreserveMode(tmplPath, synced); err != nil {
			return nil, fmt.Errorf("write %s: %w", pair.template, err)
		}
	}
	return changed, nil
}

// syncTemplatePins rewrites every `uses:` line in tmpl whose action the workflow
// also pins, so the two carry the same ref. It reports whether anything changed.
//
// Only the ref is touched. Indentation, the `- ` list marker, the action name and
// the trailing version comment's position all survive byte-for-byte, because
// TestSelfScaffoldParity compares BYTES — a stray whitespace normalisation here
// would break the very parity this exists to restore.
//
// An action the workflow does not pin is left exactly as it stands. The bare
// rendering guards regions abcd's own workflow never emits, so "absent from the
// workflow" is a normal state, not an instruction to delete or blank a line.
func syncTemplatePins(workflow, tmpl []byte) ([]byte, bool, error) {
	pins, err := collectPins(workflow)
	if err != nil {
		return nil, false, err
	}
	lines := strings.Split(string(tmpl), "\n")
	changed := false
	for i, line := range lines {
		m := usesLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		want, ok := pins[m[2]]
		if !ok || m[3] == want {
			continue
		}
		lines[i] = m[1] + m[2] + "@" + want
		changed = true
	}
	if !changed {
		return tmpl, false, nil
	}
	return []byte(strings.Join(lines, "\n")), true, nil
}

// collectPins indexes a workflow's `uses:` lines by exact action name.
//
// An action pinned to two different refs in one workflow is refused rather than
// resolved: there is no single answer to propagate, and picking one silently would
// write an unreviewed ref into release machinery — precisely the outcome SHA
// pinning exists to prevent.
func collectPins(workflow []byte) (map[string]string, error) {
	pins := map[string]string{}
	for _, line := range strings.Split(string(workflow), "\n") {
		m := usesLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		action, ref := m[2], m[3]
		if prev, seen := pins[action]; seen && prev != ref {
			return nil, fmt.Errorf("%s is pinned to two different refs (%q and %q); resolve the workflow before syncing", action, prev, ref)
		}
		pins[action] = ref
	}
	return pins, nil
}
