package intent

// target.go is the intent record's `target_release` (itd-2609212103572513):
// the release a planned intent must land by, written by `abcd intent target`
// or by `abcd intent plan --target`, reported by the preview and the cut, and
// never refused on (adr-2609212115255771, decision 3).
//
// The field lives on a PLANNED record only. A draft is not committed to, so a
// target on it would be a promise nothing reads — the draft takes its target
// as it is planned. A shipped or superseded record has nothing left to land,
// so every move out of planned/ drops the line (the spec close, the bundle
// close and the supersession), and record-lint's record_schema rule refuses
// one found there. The value's shape is launch.ValidTargetRelease, the one
// predicate the verbs, the lint and the cut share.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// TargetResult reports a completed Target. Previous is the target the record
// carried before, empty when it carried none; Written is false when the record
// already carried this target, so nothing was written.
type TargetResult struct {
	IntentID string `json:"intent_id"`
	Path     string `json:"path"`
	Target   string `json:"target_release"`
	Previous string `json:"previous,omitempty"`
	Written  bool   `json:"written"`
}

// Target writes `target_release: <version>` onto a planned intent: `next`, or
// a release tag vX.Y.Z. A second target replaces the first, and the result
// names the one it replaced; the same target again writes nothing. A draft is
// refused naming `intent plan --target`, and a shipped, superseded or
// discipline record is refused, each with nothing written.
//
// The read, the re-check and the write are one critical section under the
// store's advisory lock, as every other intent write is, so a concurrent
// write to the same record is never overwritten from stale bytes.
func Target(repoRoot, intentID, version string) (TargetResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return TargetResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	if err := launch.ValidTargetRelease(version); err != nil {
		return TargetResult{}, fmt.Errorf("intent: target: %w (nothing written)", err)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return TargetResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return TargetResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	if err := refuseTargetBucket(it); err != nil {
		return TargetResult{}, err
	}

	res := TargetResult{IntentID: it.ID, Path: it.Path, Target: version}
	abs := filepath.Join(repoRoot, it.Path)
	err = withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, it.Path)
		if err != nil {
			return err
		}
		content := string(data)
		current, malformed := targetField(frontmatter.Fields(strings.Split(content, "\n")))
		if malformed {
			return fmt.Errorf("intent: %s carries a `%s:` value in a shape no verb writes (a list, a map or a block scalar); repair or remove the line by hand, which record-lint's record_schema rule reports (nothing written)",
				it.ID, launch.TargetReleaseKey)
		}
		res.Previous = current
		if current == version {
			return nil
		}
		updated, err := setFrontmatterFields(content, map[string]string{launch.TargetReleaseKey: version})
		if err != nil {
			return err
		}
		if err := writeIntentFile(abs, it.Path, updated); err != nil {
			return err
		}
		res.Written = true
		return nil
	})
	if err != nil {
		return TargetResult{}, err
	}
	return res, nil
}

// refuseTargetBucket admits a planned record and refuses every other shelf,
// naming the verb that does apply where there is one.
func refuseTargetBucket(it Intent) error {
	switch it.Bucket {
	case BucketPlanned:
		return nil
	case BucketDrafts:
		return fmt.Errorf("intent: %s is a draft; a target is a promise about planned work, so it is written as the draft is planned: `abcd intent plan %s --target <version>` (nothing written)",
			it.ID, it.ID)
	}
	return fmt.Errorf("intent: %s is in %s; a target names the release unshipped work must land by, so it is written only onto a planned intent (nothing written)",
		it.ID, it.Bucket)
}

// targetField reads the target out of a scanned frontmatter map: the value,
// empty when the key is absent, blank or null, and malformed when the key
// carries a shape no verb writes (a flow collection or a block-scalar header,
// frontmatter.ScalarString's refusals). The value is returned as written, legal or not; the
// shape of a legal one is launch.ValidTargetRelease's to judge.
func targetField(fields map[string]frontmatter.Field) (value string, malformed bool) {
	f, ok := fields[launch.TargetReleaseKey]
	if !ok || frontmatter.IsNull(strings.TrimSpace(f.Value)) {
		return "", false
	}
	v, ok := frontmatter.ScalarString(f.Value)
	if !ok {
		return "", true
	}
	return v, false
}

// dropTarget returns content without its `target_release:` line, for the moves
// that take a record out of planned/. A record carrying none is returned
// unchanged.
func dropTarget(content string) (string, error) {
	return removeFrontmatterField(content, launch.TargetReleaseKey)
}

// Targets lists every planned intent carrying a target, sorted by id number:
// the list `launch --dry-run` and the cut report (criterion 2). A value that
// is not a legal target is listed with the reason, never dropped — the report
// says what the record says, and the record lint is what refuses it.
func Targets(repoRoot string) ([]launch.TargetedIntent, error) {
	corpus, err := Load(repoRoot)
	if err != nil {
		return nil, err
	}
	var out []launch.TargetedIntent
	for _, it := range corpus.Intents {
		if it.Bucket != BucketPlanned || it.TargetRelease == "" {
			continue
		}
		row := launch.TargetedIntent{ID: it.ID, Path: filepath.ToSlash(it.Path), Target: it.TargetRelease}
		if err := launch.ValidTargetRelease(it.TargetRelease); err != nil {
			row.Invalid = err.Error()
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return intentNum(out[i].ID) < intentNum(out[j].ID) })
	return out, nil
}

// intentNum is an intent id's number, for ordering; a malformed id sorts last.
func intentNum(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "itd-"))
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return n
}
