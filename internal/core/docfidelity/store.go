package docfidelity

// store.go — what the judgement reads from a repository, the saved review it
// finds there, and the one writer of that review (ruling DR3): the delegated
// docs review saves a verdict receipt labelled with the commit it reviewed;
// `spec close` and `launch ship` find it automatically, and a matching receipt
// proceeds while a missing or stale one refuses with "run the docs review
// first" — the way the pre-push hook finds a preflight receipt for the commit
// it pushes. The receipt is the release gate's VSA receipt, judged by the
// release gate's own reader (lint.CheckGateReceipt), never a second one.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	// GateName is the receipt's detector and its file name.
	GateName = "doc-fidelity"
	// ReceiptsDir holds the saved reviews, <dir>/<commit>/doc-fidelity.json.
	// It is the local tier, as the preflight receipts are: a review is a fact
	// about one commit on one machine, found by the verbs that close and cut.
	ReceiptsDir = ".abcd/.work.local/doc-fidelity"
	// ChaptersDir is the brief's surface chapters.
	ChaptersDir = ".abcd/development/brief/04-surfaces"
	// SnapshotPath is the committed command-tree snapshot. Its presence arms
	// the gate: it marks the repository whose binary the brief describes.
	SnapshotPath = ".abcd/development/release/surface.json"
	// AgentsDir is the plugin's agent prompts, one <name>.md per agent.
	AgentsDir = "agents"

	maxChapterBytes = 4 << 20
	// maxSentenceBytes bounds a reviewed sentence and its drafted replacement:
	// each is quoted from, or written into, one line of a chapter, and a line
	// longer than this (a wide table row) is quoted in part.
	maxSentenceBytes = 2048
	maxPayloadBytes  = 1 << 20
)

// Armed reports whether the repository ships the binary the gate judges: it
// carries the command-tree snapshot and the brief's surface chapters. Another
// repository's brief does not describe abcd's verbs, so it is not judged.
func Armed(root string) bool {
	for _, rel := range []string{SnapshotPath, ChaptersDir} {
		if _, err := os.Lstat(filepath.Join(root, rel)); err != nil {
			return false
		}
	}
	return true
}

// ReadInputs reads the chapters, the agent set and the review flags beside the
// command tree the caller derived from the binary. No file exempts a surface
// from layer 1: every surface the binary ships is named by a chapter or
// refuses.
func ReadInputs(root string, commands []surface.Command, population []string) (Inputs, error) {
	in := Inputs{Commands: commands, Population: population, Chapters: map[string]string{}}
	names, err := regularMarkdown(root, ChaptersDir)
	if err != nil {
		return Inputs{}, err
	}
	for _, name := range names {
		data, err := fsutil.ReadGuarded(filepath.Join(root, ChaptersDir, name), maxChapterBytes)
		if err != nil {
			return Inputs{}, fmt.Errorf("reading the chapter %s/%s: %w", ChaptersDir, name, err)
		}
		in.Chapters[name] = string(data)
	}
	agents, err := regularMarkdown(root, AgentsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Inputs{}, err
	}
	for _, a := range agents {
		in.Agents = append(in.Agents, strings.TrimSuffix(a, ".md"))
	}
	flags, err := readFlags(root)
	if err != nil {
		return Inputs{}, err
	}
	in.Flags = flags
	return in, nil
}

// regularMarkdown lists the regular *.md files directly under rel, sorted.
func regularMarkdown(root, rel string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, rel))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", rel, err)
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// SavedReview is layer 2 as ruled: the review the delegated reviewer saved for
// Commit, found in Root.
type SavedReview struct{ Root, Commit string }

// Review reads the saved review. Every way of not having a usable verdict is a
// status that refuses; none is a pass.
func (s SavedReview) Review() Review {
	r := Review{Commit: s.Commit}
	rel := filepath.Join(ReceiptsDir, s.Commit, GateName+".json")
	if _, err := os.Lstat(filepath.Join(s.Root, rel)); errors.Is(err, os.ErrNotExist) {
		r.Status = ReviewNone
		if named := newestOtherReview(s.Root, s.Commit); named != "" {
			r.Status, r.Named = ReviewStale, named
		}
		return r
	}
	got, err := lint.CheckGateReceipt(s.Root, ReceiptsDir, s.Commit, GateName)
	if err != nil {
		r.Status, r.Problems = ReviewInvalid, []string{err.Error()}
		return r
	}
	for _, f := range got.Failing {
		chapter := f.Chapter
		if docOf(f.Doc) == DocBrief {
			// A review saved before Record parsed the chapter may name it under
			// its directory; it is read through the same parse, and a name the
			// parse refuses is kept as written, for Apply to refuse by name.
			if name, err := chapterFile(chapter); err == nil {
				chapter = name
			}
		}
		r.Findings = append(r.Findings, Sentence{Doc: docOf(f.Doc), Chapter: chapter, Sentence: f.Sentence,
			Evidence: f.Evidence, Replacement: f.Replacement})
	}
	r.Verdict = got.Verdict
	switch {
	case got.Parsed && got.Verdict == "HOLD":
		r.Status = ReviewHold
	case got.Parsed && got.Verdict != "PROMOTE":
		r.Status = ReviewInconclusive
	case len(got.Problems) > 0:
		r.Status = ReviewInvalid
		for _, p := range got.Problems {
			r.Problems = append(r.Problems, p.Message)
		}
	default:
		r.Status = ReviewMatch
	}
	return r
}

func docOf(d string) string {
	if d == DocPublic {
		return DocPublic
	}
	return DocBrief
}

// newestOtherReview names the most recently saved review for a commit other
// than head, "" when there is none.
func newestOtherReview(root, head string) string {
	entries, err := os.ReadDir(filepath.Join(root, ReceiptsDir))
	if err != nil {
		return ""
	}
	var best string
	var bestAt time.Time
	for _, e := range entries {
		if !e.IsDir() || e.Name() == head || !gitutil.IsFullSHA(e.Name()) {
			continue
		}
		fi, err := os.Lstat(filepath.Join(root, ReceiptsDir, e.Name(), GateName+".json"))
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		if best == "" || fi.ModTime().After(bestAt) {
			best, bestAt = e.Name(), fi.ModTime()
		}
	}
	return best
}

// Gate is both layers: what the per-task report calls, and what Enforce runs
// for a population of one intent at `spec close` or every intent shipped since
// the last tag at `launch ship`. armed is false, with a zero
// verdict, where the repository does not ship the binary the gate judges.
func Gate(root string, commands []surface.Command, population []string, report bool) (v Verdict, armed bool, err error) {
	if !Armed(root) {
		return Verdict{}, false, nil
	}
	in, err := ReadInputs(root, commands, population)
	if err != nil {
		return Verdict{}, true, err
	}
	head, err := gitutil.ResolveCommit(root, "HEAD")
	if err != nil {
		return Verdict{}, true, fmt.Errorf("resolving the commit under review: %w", err)
	}
	return judge(in, SavedReview{Root: root, Commit: head}, report), true, nil
}

// Enforce is what both enforcement points run, `spec close` and `launch ship`,
// with their population. Layer 1 runs whatever the population: a close or a
// cut that ships no intent can still carry a surface an issue fix added, and
// the coverage floor is cheap enough to run at every enforcement point
// (spc-2609020903498198). Layer 2's saved review is required only where an
// intent ships (iss-2610020728118137), so with no population this is layer 1
// alone and no review is read.
func Enforce(root string, commands []surface.Command, population []string) (v Verdict, armed bool, err error) {
	if len(population) > 0 {
		return Gate(root, commands, population, false)
	}
	if !Armed(root) {
		return Verdict{}, false, nil
	}
	in, err := ReadInputs(root, commands, nil)
	if err != nil {
		return Verdict{}, true, err
	}
	return coverage(in, false), true, nil
}

// payload is what the delegated reviewer hands back. It carries no subject,
// detector or manifest hash: those label the receipt, and the reviewer never
// labels its own verdict — Record stamps them from the checkout it reviewed.
type payload struct {
	VerificationResult string `json:"verificationResult"`
	JudgeModel         string `json:"judgeModel"`
	Tier               string `json:"tier"`
	Verifier           *struct {
		ID string `json:"id"`
	} `json:"verifier,omitempty"`
	Failing []lint.ReceiptFinding `json:"failing"`
}

// Record validates the reviewer's verdict and saves it as the receipt for the
// commit the checkout stands at, returning the receipt's repo-relative path
// and the review as the gate now reads it. A refused payload writes nothing.
func Record(root string, raw []byte, at time.Time) (string, Review, error) {
	if len(raw) > maxPayloadBytes {
		return "", Review{}, fmt.Errorf("the verdict is larger than %d bytes", maxPayloadBytes)
	}
	var p payload
	if err := jsonstrict.Decode(raw, &p); err != nil {
		return "", Review{}, fmt.Errorf("the verdict is refused: %w", err)
	}
	switch p.VerificationResult {
	case "PROMOTE", "HOLD", "INCONCLUSIVE":
	default:
		return "", Review{}, fmt.Errorf("verificationResult %q is not PROMOTE, HOLD or INCONCLUSIVE", p.VerificationResult)
	}
	if strings.TrimSpace(p.JudgeModel) == "" {
		return "", Review{}, errors.New("judgeModel is empty: a verdict names the judge that produced it")
	}
	if why := lint.FloatingJudgeModel(p.JudgeModel); why != "" {
		return "", Review{}, fmt.Errorf("judgeModel %q is %s: a verdict names the pinned judge that produced it "+
			"(a version or date, never latest), so the review can be re-run against the same judge", p.JudgeModel, why)
	}
	if p.Tier != "full" && p.Tier != "shallow" {
		return "", Review{}, fmt.Errorf("tier %q is not full or shallow", p.Tier)
	}
	brief := 0
	for i, f := range p.Failing {
		if f.Doc != DocBrief && f.Doc != DocPublic {
			return "", Review{}, fmt.Errorf("failing[%d].doc %q is not brief or public", i, f.Doc)
		}
		if strings.TrimSpace(f.Chapter) == "" || strings.TrimSpace(f.Sentence) == "" || strings.TrimSpace(f.Evidence) == "" {
			return "", Review{}, fmt.Errorf("failing[%d] needs its chapter, the sentence and the evidence", i)
		}
		if f.Doc == DocBrief {
			// The same parse Apply reads the chapter through, so a verdict saved
			// here is one --apply can write; the receipt keeps the file name.
			name, err := chapterFile(f.Chapter)
			if err != nil {
				return "", Review{}, fmt.Errorf("failing[%d]: %w", i, err)
			}
			p.Failing[i].Chapter = name
		}
		if strings.TrimSpace(f.Disposition) == "" {
			return "", Review{}, fmt.Errorf("failing[%d] carries no disposition", i)
		}
		if err := checkLine("sentence", f.Sentence); err != nil {
			return "", Review{}, fmt.Errorf("failing[%d]: %w", i, err)
		}
		if err := checkLine("replacement", f.Replacement); err != nil {
			return "", Review{}, fmt.Errorf("failing[%d]: %w", i, err)
		}
		if f.Replacement != "" && f.Replacement == f.Sentence {
			return "", Review{}, fmt.Errorf("failing[%d].replacement must differ from the sentence", i)
		}
		if f.Doc == DocBrief {
			brief++
		}
	}
	if p.VerificationResult == "HOLD" && brief == 0 {
		return "", Review{}, errors.New("a HOLD names no false brief sentence, so there is nothing to correct: a verdict with no confirmed brief sentence is PROMOTE")
	}
	if p.VerificationResult == "PROMOTE" && brief > 0 {
		return "", Review{}, fmt.Errorf("a PROMOTE names %d false brief sentence(s), so the brief is not current: a verdict with a confirmed brief sentence is HOLD", brief)
	}
	head, err := gitutil.ResolveCommit(root, "HEAD")
	if err != nil {
		return "", Review{}, fmt.Errorf("resolving the commit under review: %w", err)
	}
	manifest, err := lint.ReleaseGateManifestHash(root)
	if err != nil {
		return "", Review{}, fmt.Errorf("reading the release-gate manifest: %w", err)
	}
	verifier := "host-delegated docs review"
	if p.Verifier != nil && strings.TrimSpace(p.Verifier.ID) != "" {
		verifier = p.Verifier.ID
	}
	if p.Failing == nil {
		p.Failing = []lint.ReceiptFinding{}
	}
	receipt := map[string]any{
		"subject":            map[string]any{"digest": map[string]string{"gitCommit": head}},
		"verifier":           map[string]string{"id": verifier},
		"timeVerified":       at.UTC().Format(time.RFC3339),
		"verificationResult": p.VerificationResult,
		"judgeModel":         p.JudgeModel,
		"tier":               p.Tier,
		"policy":             map[string]string{"detector": GateName, "version": "1"},
		"failing":            p.Failing,
	}
	if manifest != "" {
		receipt["manifestHash"] = manifest
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", Review{}, err
	}
	rel := filepath.Join(ReceiptsDir, head, GateName+".json")
	if err := os.MkdirAll(filepath.Join(root, ReceiptsDir, head), 0o755); err != nil {
		return "", Review{}, err
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(root, rel), append(data, '\n'), 0o644); err != nil {
		return "", Review{}, err
	}
	return filepath.ToSlash(rel), SavedReview{Root: root, Commit: head}.Review(), nil
}

// checkLine refuses a sentence or replacement the gate would match in, or
// write into, a chapter unless it is one line of at most maxSentenceBytes: a
// string spanning lines replaces headings and paragraphs, not a sentence.
func checkLine(what, text string) error {
	if strings.ContainsAny(text, "\n\r") {
		return fmt.Errorf("the %s spans lines: quote it from one line of its chapter", what)
	}
	if len(text) > maxSentenceBytes {
		return fmt.Errorf("the %s is %d bytes, over the %d-byte bound", what, len(text), maxSentenceBytes)
	}
	return nil
}
