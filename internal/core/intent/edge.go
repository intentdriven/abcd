package intent

// edge.go is `abcd intent edge` (iss-2610040758579292): the verb that adds and
// removes an intent's dependency edges, `blocked_by` and `builds_on`, after the
// record exists. The build's pre-start blocked check (startBlockedRow) and the
// edge rules of record-lint read those lists, and before this nothing wrote
// them: `abcd capture link` edits an ISSUE's blocked_by, and `abcd intent link`
// writes a spec_id, so a project hand-typed every edge between its intents.
//
// It mirrors capture link: the removals are applied before the additions, so
// the same id on both sides is a net no-op; an id already on the list
// collapses, so a repeated edge is idempotent; every target must name an
// intent this checkout's store holds, in any bucket, because record-lint's
// record_schema rule refuses a cross-reference whose target is not in the
// corpus; a record cannot name itself; and a removal of an edge the list does
// not hold is refused naming the current list. Every refusal comes before
// anything is written. The subject may sit in any bucket and never moves.
//
// Two differences from capture link, both from the intent record's own shape:
// an emptied list is written `[]` rather than dropped, because the create path
// writes `builds_on: []` on every draft; and a list spelled as a block sequence
// stays one, since the items are the author's spelling.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// BuildsOnKey is the frontmatter key of the softer dependency edge: cheaper or
// better if the other intent exists first. BlockedByKey is the hard one.
const BuildsOnKey = "builds_on"

// EdgeRequest edits one intent's dependency edges. At least one of the four
// lists must be non-empty.
type EdgeRequest struct {
	ID           string   // the subject itd-N, in any bucket
	BlockedBy    []string // itd-N ids to append to blocked_by
	Unblock      []string // itd-N ids to remove from blocked_by
	BuildsOn     []string // itd-N ids to append to builds_on
	DropBuildsOn []string // itd-N ids to remove from builds_on
}

// EdgeResult is the outcome of a successful Edge: the subject, its
// repo-relative path and bucket, and both lists AS WRITTEN (an empty list comes
// back empty, never null).
type EdgeResult struct {
	IntentID  string   `json:"intent_id"`
	Path      string   `json:"path"`
	Bucket    string   `json:"bucket"`
	BlockedBy []string `json:"blocked_by"`
	BuildsOn  []string `json:"builds_on"`
}

// edgeOp is one list's requested change, already validated.
type edgeOp struct {
	key        string
	addFlag    string
	removeFlag string
	add        []string
	remove     []string
}

// Edge adds to, or removes from, one intent's blocked_by and builds_on lists.
// The read, the re-check of every removal against the bytes on disk and the
// write are one critical section under the store's lock, as every other intent
// write is.
func Edge(repoRoot string, req EdgeRequest) (EdgeResult, error) {
	const verb = "intent: edge"
	if len(req.BlockedBy)+len(req.Unblock)+len(req.BuildsOn)+len(req.DropBuildsOn) == 0 {
		return EdgeResult{}, fmt.Errorf("%s: nothing to do — give --blocked-by, --unblock, --builds-on and/or --drop-builds-on, each a comma-separated list of itd-N ids (nothing written)", verb)
	}
	if !recordid.ValidIntentID(req.ID) {
		return EdgeResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", req.ID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return EdgeResult{}, err
	}
	// The subject is located before the targets are judged, so an unknown
	// subject is reported as the fault rather than a target.
	it, ok := corpus.Lookup(req.ID)
	if !ok {
		return EdgeResult{}, fmt.Errorf("intent: %s not found in any bucket (nothing written)", req.ID)
	}
	ops := []*edgeOp{
		{key: BlockedByKey, addFlag: "--blocked-by", removeFlag: "--unblock"},
		{key: BuildsOnKey, addFlag: "--builds-on", removeFlag: "--drop-builds-on"},
	}
	if ops[0].add, err = validateEdgeTargets(corpus, verb, ops[0].addFlag, it.ID, req.BlockedBy); err != nil {
		return EdgeResult{}, err
	}
	if ops[1].add, err = validateEdgeTargets(corpus, verb, ops[1].addFlag, it.ID, req.BuildsOn); err != nil {
		return EdgeResult{}, err
	}
	if ops[0].remove, err = validateEdgeRemovals(verb, ops[0].removeFlag, req.Unblock); err != nil {
		return EdgeResult{}, err
	}
	if ops[1].remove, err = validateEdgeRemovals(verb, ops[1].removeFlag, req.DropBuildsOn); err != nil {
		return EdgeResult{}, err
	}

	res := EdgeResult{IntentID: it.ID, Path: it.Path, Bucket: it.Bucket, BlockedBy: []string{}, BuildsOn: []string{}}
	abs := filepath.Join(repoRoot, it.Path)
	err = withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, it.Path)
		if err != nil {
			return err
		}
		content := string(data)
		fields := frontmatter.Fields(strings.Split(content, "\n"))
		for _, op := range ops {
			if err := refuseHandShapedList(it.ID, op.key, fields); err != nil {
				return err
			}
			current := frontmatterList(content, op.key)
			// Removals first, each judged against the bytes under the lock.
			next := make([]string, 0, len(current)+len(op.add))
			for _, dep := range op.remove {
				if !containsSameID(current, dep) {
					return fmt.Errorf("%s: %s %s is not in %s's %s, which is currently %s (nothing written)",
						verb, op.removeFlag, dep, it.ID, op.key, renderEdgeList(current))
				}
			}
			for _, dep := range current {
				if !containsSameID(op.remove, dep) {
					next = append(next, dep)
				}
			}
			for _, dep := range op.add {
				if !containsSameID(next, dep) {
					next = append(next, dep)
				}
			}
			if len(op.add) > 0 || len(op.remove) > 0 {
				if content, err = setFrontmatterList(content, op.key, next); err != nil {
					return err
				}
			}
			if op.key == BlockedByKey {
				res.BlockedBy = next
			} else {
				res.BuildsOn = next
			}
		}
		return writeIntentFile(abs, it.Path, content)
	})
	if err != nil {
		return EdgeResult{}, err
	}
	return res, nil
}

// validateEdgeTargets checks a list of ids to add, before anything is written:
// the shape, that none names the subject, and that every one is an intent this
// checkout's store holds, in any bucket. It returns each as the store spells
// it, repeats collapsed, order preserved. Naming a shipped, superseded or
// discipline intent is legal: existence is what the edge claims, and whether it
// still holds anything up is the build gate's question (it follows a
// supersession to the record that replaced it).
func validateEdgeTargets(corpus Corpus, verb, flag, subject string, ids []string) ([]string, error) {
	var out []string
	for _, dep := range ids {
		if !recordid.ValidIntentID(dep) {
			return nil, fmt.Errorf("%s: %s token %q must match itd-N (nothing written)", verb, flag, dep)
		}
		if recordid.SameID(dep, subject) {
			return nil, fmt.Errorf("%s: %s %s names the record itself, and an intent cannot depend on itself (nothing written)", verb, flag, dep)
		}
		target, ok := corpus.Lookup(dep)
		if !ok {
			return nil, fmt.Errorf("%s: %s %s not found in the intent store, in any bucket (nothing written)", verb, flag, dep)
		}
		if !containsSameID(out, target.ID) {
			out = append(out, target.ID)
		}
	}
	return out, nil
}

// validateEdgeRemovals checks the shape of each id to remove and collapses
// repeats. Whether the list holds it is judged under the lock.
func validateEdgeRemovals(verb, flag string, ids []string) ([]string, error) {
	var out []string
	for _, dep := range ids {
		if !recordid.ValidIntentID(dep) {
			return nil, fmt.Errorf("%s: %s token %q must match itd-N (nothing written)", verb, flag, dep)
		}
		if !containsSameID(out, dep) {
			out = append(out, dep)
		}
	}
	return out, nil
}

// refuseHandShapedList refuses a list key in a shape the writer cannot rewrite
// without guessing: a key spelled with whitespace before its colon (which the
// reader honours and the writer's key pattern does not match, so a write would
// add a second key), or a value that is neither an inline list, null, nor a
// block sequence.
func refuseHandShapedList(id, key string, fields map[string]frontmatter.Field) error {
	f, ok := fields[key]
	if !ok {
		return nil
	}
	v := strings.TrimSpace(frontmatter.StripComment(f.Value))
	if !f.SpacedKey && (v == "" || frontmatter.IsNull(v) || (strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]"))) {
		return nil
	}
	return fmt.Errorf("intent: edge: %s carries a `%s` line in a shape no verb writes (line %d); repair it by hand to `%s: [itd-N, ...]` and re-run (nothing written)",
		id, key, f.Line, key)
}

// setFrontmatterList writes key's list as items, in the spelling the record
// already uses: a block sequence keeps its items one per line at their
// indentation (an emptied one becomes `key: []`); an inline list, a null and an
// absent key are written inline.
func setFrontmatterList(content, key string, items []string) (string, error) {
	lines := strings.Split(content, "\n")
	closing, err := frontmatterClose(lines)
	if err != nil {
		return "", err
	}
	if k := frontmatterKeyLine(lines, closing, key); k >= 0 &&
		strings.TrimSpace(frontmatter.StripComment(strings.TrimRight(lines[k], "\r")[len(key)+1:])) == "" {
		end, indent := blockEnd(lines, k, closing)
		repl := []string{key + ": []"}
		if len(items) > 0 {
			repl = []string{key + ":"}
			for _, item := range items {
				repl = append(repl, indent+"- "+item)
			}
		}
		out := make([]string, 0, len(lines)-(end-k)+len(repl))
		out = append(out, lines[:k]...)
		out = append(out, repl...)
		out = append(out, lines[end:]...)
		return strings.Join(out, "\n"), nil
	}
	return setFrontmatterFields(content, map[string]string{key: "[" + strings.Join(items, ", ") + "]"})
}

// containsSameID reports whether list names id, canonically.
func containsSameID(list []string, id string) bool {
	for _, v := range list {
		if recordid.SameID(v, id) {
			return true
		}
	}
	return false
}

// renderEdgeList spells a list the way the record does, or `(none)`.
func renderEdgeList(ids []string) string {
	if len(ids) == 0 {
		return "(none)"
	}
	return "[" + strings.Join(ids, ", ") + "]"
}
