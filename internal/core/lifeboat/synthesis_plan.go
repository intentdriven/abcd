package lifeboat

// synthesis_plan.go — the practice run of the four post-pack verbs that read a
// filled-in input (itd-2610301428390406, spc-2610301431131636 step 2):
// `disembark graveyard --lessons-json`, `principles --principles-json`,
// `press-release --press-release-json` and `review --review-json`.
//
// Each verb is split into a Plan stage, which does everything up to the first
// write — the lifeboat gate, the payload's decode and per-entry validation, the
// composition of every file, and the secret scan over exactly those bytes — and
// an Apply that lays what the plan composed. The real verb is Plan then Apply,
// so a practice run (`--dry-run`) and the real write share every check by
// construction.

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/intentdriven/abcd/internal/core/dryrun"
)

// SynthPlan is what a post-pack verb would do to a packed lifeboat: the files
// it would write, each path relative to the lifeboat, and the files it would
// remove, reported as effects (`remove`, path). Apply lays it.
type SynthPlan struct {
	// Plan is the practice run's report.
	Plan dryrun.Plan

	abs   string
	files []PlannedFile
	// clearLayer3 is the lessons ingest's full replacement of layer 3.
	clearLayer3 bool
	// legacyManifest12, when set, is the manifest prefix whose pre-spc-30
	// review pair the review removes once it has written its own.
	legacyManifest12 string
}

// removeEffect is the effect a planned removal is reported as.
const removeEffect = "remove"

// newSynthPlan plans files into the lifeboat at abs. A path that holds a file
// already is replaced; any other is created.
func newSynthPlan(abs string, files []PlannedFile) SynthPlan {
	p := SynthPlan{abs: abs, files: files}
	for _, f := range files {
		action := dryrun.Create
		if fi, err := os.Lstat(filepath.Join(abs, filepath.FromSlash(f.Path))); err == nil && !fi.IsDir() {
			action = dryrun.Replace
		}
		p.Plan.Writes = append(p.Plan.Writes, dryrun.Write{Path: f.Path, Action: action, Bytes: f.Bytes, Content: f.Content})
	}
	return p
}

// planRemove reports each existing file among rels that the plan does not
// write again as a removal.
func (p *SynthPlan) planRemove(rels []string) {
	written := map[string]bool{}
	for _, f := range p.files {
		written[f.Path] = true
	}
	for _, rel := range rels {
		if written[rel] {
			continue
		}
		if fi, err := os.Lstat(filepath.Join(p.abs, filepath.FromSlash(rel))); err == nil && !fi.IsDir() {
			p.Plan.Effects = append(p.Plan.Effects, dryrun.Effect{Effect: removeEffect, Target: rel})
		}
	}
}

// layer3Files lists the files a lessons ingest's full replacement clears: the
// main lessons file and every file under low-confidence/, read without
// following a link.
func layer3Files(abs string) []string {
	out := []string{path.Join("graveyard", "lessons.json")}
	dir := filepath.Join(abs, "graveyard", "low-confidence")
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
		return out
	}
	var nested []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if rel, rerr := filepath.Rel(abs, p); rerr == nil {
			nested = append(nested, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(nested)
	return append(out, nested...)
}

// apply lays the plan through the lifeboat's containment root: the lessons
// ingest's clear first, then every planned file, then the review's legacy
// pair. The bytes were secret-scanned when the plan was made.
func (p SynthPlan) apply() error {
	if p.abs == "" {
		return errors.New("lifeboat: applying a plan that was never made")
	}
	root, err := os.OpenRoot(p.abs)
	if err != nil {
		return err
	}
	defer root.Close()
	if p.clearLayer3 {
		if err := clearLayer3(root); err != nil {
			return err
		}
	}
	if err := writePlanned(root, p.abs, p.files); err != nil {
		return err
	}
	if p.legacyManifest12 != "" {
		removeLegacyReviewArtefact(root, p.legacyManifest12)
	}
	return nil
}
