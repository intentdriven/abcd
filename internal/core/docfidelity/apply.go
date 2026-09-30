package docfidelity

// apply.go — draft and apply, review after (itd-60 ruling). The judge returns
// a verdict and the edits the reviewer drafted; this is the distinct writer
// that applies them to the brief in the shipping change's working tree and
// records a review flag the product thinker reads. A judge that writes is a
// judge whose output depends on who ran it, so nothing here is reached by
// Judge, and nothing in Judge writes.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// FlagsPath is the review-flag ledger: every drafted edit the gate applied,
// kept until the product thinker has read it and removes its entry. It is in
// the shared working tier, so the flag travels in the change's own diff.
const FlagsPath = ".abcd/work/brief-review-flags.json"

// chapterNameRe is a chapter's file name: one path component, no traversal.
var chapterNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`)

type flagFile struct {
	SchemaVersion int    `json:"schema_version"`
	Flags         []Flag `json:"flags"`
}

func readFlags(root string) ([]Flag, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(root, FlagsPath), maxPayloadBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", FlagsPath, err)
	}
	var f flagFile
	if err := jsonstrict.Decode(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", FlagsPath, err)
	}
	if f.SchemaVersion != 1 {
		return nil, fmt.Errorf("%s: schema_version %d, want 1", FlagsPath, f.SchemaVersion)
	}
	return f.Flags, nil
}

// Apply writes each edit into its chapter and records one flag per edit,
// naming the reviewed commit. Every edit is checked before anything is
// written: a chapter outside the surfaces directory, a sentence or replacement
// that is not one bounded line, or a sentence the chapter does not carry
// exactly once, refuses the whole apply and writes nothing.
func Apply(root string, edits []Edit, commit string, at time.Time) ([]Flag, error) {
	next := map[string]string{}
	for _, e := range edits {
		if !chapterNameRe.MatchString(e.Chapter) {
			return nil, fmt.Errorf("the edit's chapter %q is not a chapter file under %s", e.Chapter, ChaptersDir)
		}
		if e.Sentence == "" || e.Replacement == "" {
			return nil, fmt.Errorf("the edit to %s carries no sentence or no replacement", e.Chapter)
		}
		for what, text := range map[string]string{"sentence": e.Sentence, "replacement": e.Replacement} {
			if err := checkLine(what, text); err != nil {
				return nil, fmt.Errorf("the edit to %s: %w", e.Chapter, err)
			}
		}
		text, ok := next[e.Chapter]
		if !ok {
			data, err := fsutil.ReadGuarded(filepath.Join(root, ChaptersDir, e.Chapter), maxChapterBytes)
			if err != nil {
				return nil, fmt.Errorf("reading %s/%s: %w", ChaptersDir, e.Chapter, err)
			}
			text = string(data)
		}
		switch n := strings.Count(text, e.Sentence); n {
		case 1:
		case 0:
			return nil, fmt.Errorf("%s does not carry the sentence %q, so the drafted edit has nothing to replace", e.Chapter, e.Sentence)
		default:
			return nil, fmt.Errorf("%s carries the sentence %q %d times, so which one is false is not known", e.Chapter, e.Sentence, n)
		}
		next[e.Chapter] = strings.Replace(text, e.Sentence, e.Replacement, 1)
	}
	existing, err := readFlags(root)
	if err != nil {
		return nil, err
	}
	var added []Flag
	for _, e := range edits {
		added = append(added, Flag{Chapter: e.Chapter, Sentence: e.Sentence, Replacement: e.Replacement,
			Evidence: e.Evidence, Commit: commit, Applied: at.UTC().Format(time.RFC3339)})
	}
	data, err := json.MarshalIndent(flagFile{SchemaVersion: 1, Flags: append(existing, added...)}, "", "  ")
	if err != nil {
		return nil, err
	}
	for chapter, text := range next {
		if err := fsutil.WriteFileAtomicPreserveMode(filepath.Join(root, ChaptersDir, chapter), []byte(text)); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(FlagsPath)), 0o755); err != nil {
		return nil, err
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(root, FlagsPath), append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	return added, nil
}

// Request is what an unattended routine hands the delegated reviewer, the way
// `launch ship` hands the changelog composer its request block: the commit to
// review, the population whose delivery is judged, the chapters to read, and
// the verb that saves the verdict.
type Request struct {
	Commit     string   `json:"commit"`
	Population []string `json:"population"`
	Chapters   []string `json:"chapters"`
	RecordWith string   `json:"record_with"`
	Shape      string   `json:"verdict_shape"`
}

// VerdictShape is the payload `abcd docs fidelity record` accepts.
const VerdictShape = `{"verificationResult": "PROMOTE|HOLD|INCONCLUSIVE", "judgeModel": "<pinned model id>", ` +
	`"tier": "full|shallow", "failing": [{"doc": "brief|public", "chapter": "<04-surfaces file or public doc>", ` +
	`"sentence": "<the false sentence, verbatim, from one line>", "replacement": "<drafted correction, optional>", ` +
	`"evidence": "<file:line showing the divergence>", "disposition": "confirmed"}]}`

// NewRequest composes the reviewer's request for commit.
func NewRequest(commit string, in Inputs) Request {
	chapters := make([]string, 0, len(in.Chapters))
	for name := range in.Chapters {
		chapters = append(chapters, ChaptersDir+"/"+name)
	}
	sort.Strings(chapters)
	return Request{Commit: commit, Population: append([]string{}, in.Population...), Chapters: chapters,
		RecordWith: "abcd docs fidelity record --verdict-json <file|->", Shape: VerdictShape}
}
