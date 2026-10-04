package lifeboat

// embark.go is the read-and-write half of the M5 record round-trip (itd-88,
// adr-35): it takes a PACKED lifeboat (an UNTRUSTED directory) and reports, or
// performs, the write of its record families back into a target repo. The
// lifeboat is untrusted input — every read is bounded, every path is validated,
// and no symlink is ever followed — so a hostile or corrupt lifeboat cannot
// exhaust memory, smuggle a file past manifest verification, or steer a write
// outside the target family. The core is transport-agnostic: it returns
// structured results and never prints (the surface renders them).
//
// The shape vocabulary (EmbarkPlan, EmbarkResult, Conflict, PlannedEmbark,
// IgnoredFile, MarkerResult, CoverageHandoff, the embarkFamilies inverse table
// and the closure/exclusion sets) lives in embark_types.go, shared with the
// packer so the two sides cannot drift.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxProvenanceBytes caps the _provenance.json read (its own manifest header).
const maxProvenanceBytes = 1 << 20

// EmbarkProbe inspects the lifeboat at lifeboatDir against targetDir WITHOUT
// writing. Both are resolved to absolute dirs (the CLI defaults targetDir to
// cwd). It gates the lifeboat, verifies the manifest, computes the plan (mapped
// writes, conflicts, ignored files), predicts the marker action, and reads the
// coverage handoff. It returns a structural error for a non-lifeboat /
// schema-too-new / failed-verification / bad-target input; a plan WITH conflicts
// is a SUCCESS (probe is a report, never a refusal).
func EmbarkProbe(lifeboatDir, targetDir string) (EmbarkPlan, error) {
	pr, err := runPlanner(lifeboatDir, targetDir)
	if err != nil {
		return EmbarkPlan{}, err
	}
	// Predict the marker action without writing (one code path with the write
	// side, so a probe cannot mispredict what a real embark would do).
	marker := embarkMarker(pr.targetAbs, true)
	return EmbarkPlan{
		SchemaVersion:        EmbarkSchemaVersion,
		LifeboatDir:          fsutil.RedactHome(pr.lifeboatAbs),
		TargetDir:            fsutil.RedactHome(pr.targetAbs),
		SourceName:           pr.prov.SourceName,
		ManifestVerified:     true,
		ManifestSHA256:       pr.prov.ManifestSHA256,
		RecordManifestSHA256: pr.prov.RecordManifestSHA256,
		Coverage:             pr.coverage,
		Planned:              pr.planned,
		Conflicts:            pr.conflicts,
		Ignored:              pr.ignored,
		Marker:               marker,
	}, nil
}

// EmbarkFrom performs the write. It runs the same planner as EmbarkProbe; if the
// plan carries ANY conflict it returns (result-with-Conflicts, ErrEmbarkConflicts)
// having written NOTHING. It judges the plan again under the target's ledger
// and intent locks, and a conflict found then refuses the same way. Otherwise
// it writes each ActionCreate file through
// os.Root containment + independent lexical validation + an exclusive create,
// skips ActionUnchanged files, ensures the marker last, and returns the summary.
func EmbarkFrom(lifeboatDir, targetDir string) (EmbarkResult, error) {
	pr, err := runPlanner(lifeboatDir, targetDir)
	if err != nil {
		return EmbarkResult{}, err
	}
	res := EmbarkResult{
		SchemaVersion: EmbarkSchemaVersion,
		LifeboatDir:   fsutil.RedactHome(pr.lifeboatAbs),
		TargetDir:     fsutil.RedactHome(pr.targetAbs),
		SourceName:    pr.prov.SourceName,
		Coverage:      pr.coverage,
		Ignored:       pr.ignored,
	}
	if len(pr.conflicts) > 0 {
		// Refuse: write nothing, hand the surface the full conflict slice for one
		// bulk report. The marker is only PREDICTED (dry run — no write), so the
		// "nothing was written" promise holds.
		res.Conflicts = pr.conflicts
		res.Marker = embarkMarker(pr.targetAbs, true)
		return res, ErrEmbarkConflicts
	}

	if afterEmbarkPlan != nil {
		afterEmbarkPlan()
	}
	// The write runs under the target's ledger lock, then its intent store's
	// lock, then its spec store's — the one order every path holding more than
	// one takes — and every planned write is judged again under them: a record
	// created at a planned target between the classification above and this
	// write — a capture, an intent or spec mint — refuses the whole write as a
	// conflict instead of being replaced (iss-2609262143265180,
	// iss-2609262218306589, iss-2609262218309668).
	var (
		written, unchanged, bytesW int
		families                   map[string]int
		late                       []Conflict
	)
	err = intent.WithLedgerThenMintLock(pr.targetAbs, embarkLedgerLock(pr.targetAbs, pr.planned), func() error {
		if duringEmbarkWrite != nil {
			duringEmbarkWrite()
		}
		var planned []PlannedEmbark
		planned, late = rejudgeEmbark(pr.targetAbs, pr.planned)
		if len(late) > 0 {
			return nil
		}
		var werr error
		written, unchanged, bytesW, families, werr = writeEmbark(pr.targetAbs, planned)
		return werr
	})
	if err != nil {
		return EmbarkResult{}, fmt.Errorf("embark: %w", err)
	}
	if len(late) > 0 {
		res.Conflicts = late
		res.Marker = embarkMarker(pr.targetAbs, true)
		return res, ErrEmbarkConflicts
	}
	res.Written = written
	res.Unchanged = unchanged
	res.BytesWritten = bytesW
	res.Families = families
	// Marker last: the records land first, then the current block is re-injected
	// (never foreign prose copied) into the conventions file the target chose.
	res.Marker = embarkMarker(pr.targetAbs, false)
	return res, nil
}

// afterEmbarkPlan and duringEmbarkWrite are test seams, nil outside tests:
// the first is called between the plan and the locks, so a test can land a
// record in that window; the second with the locks held, before the
// re-judgement, so a test can prove a concurrent writer waits.
var afterEmbarkPlan, duringEmbarkWrite func()

// embarkLedgerLock is the target ledger's lock when the plan creates an issue
// record there, and no lock otherwise: taking it creates the ledger, which an
// embark carrying no issue must not plant.
func embarkLedgerLock(targetAbs string, planned []PlannedEmbark) func(func() error) error {
	for _, p := range planned {
		if p.Family == "issues" && p.Action == ActionCreate {
			return func(fn func() error) error { return capture.WithLedgerLock(targetAbs, fn) }
		}
	}
	return func(fn func() error) error { return fn() }
}

// rejudgeEmbark classifies every planned write again against the target as it
// is now, returning the writes to perform or the conflicts that refuse them.
func rejudgeEmbark(targetAbs string, planned []PlannedEmbark) ([]PlannedEmbark, []Conflict) {
	var (
		out       []PlannedEmbark
		conflicts []Conflict
	)
	for _, p := range planned {
		pe, cf := classifyEmbark(targetAbs, p.LifeboatPath, p.TargetPath, p.Family, p.Content)
		if cf != nil {
			conflicts = append(conflicts, *cf)
			continue
		}
		out = append(out, pe)
	}
	return out, conflicts
}

// verifyManifest re-hashes every non-excluded file in the lifeboat and compares
// the result to _provenance.json's manifest_sha256. It enforces the trust
// boundary during the walk: it refuses a symlink anywhere in the tree, a path
// that fails validRelPath, a file over maxEmbarkFileBytes, a tree over
// maxEmbarkFiles / maxEmbarkTotalBytes. A flipped, missing, or extra
// manifest-relevant file changes the reproduced hash and is fatal. It returns
// nil iff the lifeboat is intact. The excluded set (_provenance.json, the
// post-pack layer-3 graveyard/lessons.json and graveyard/low-confidence/**) is
// the same set the packer left out of manifest_sha256.
func verifyManifest(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := proveOperand("lifeboat", abs); err != nil {
		return err
	}
	if !fsutil.IsRealDir(abs) {
		return fmt.Errorf("lifeboat %s is not a directory", filepath.Base(abs))
	}
	prov, err := readProvenance(abs)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return err
	}
	defer root.Close()

	rels, err := walkLifeboatFiles(root)
	if err != nil {
		return err
	}
	files := make([]PlannedFile, 0, len(rels))
	total := 0
	for _, rel := range rels {
		if isManifestExcluded(rel) {
			continue // the layer-3 interpretation and the header are not sealed
		}
		data, err := readLifeboatFile(root, rel)
		if err != nil {
			return err
		}
		total += len(data)
		if total > maxEmbarkTotalBytes {
			return fmt.Errorf("lifeboat exceeds the %d-byte total cap", maxEmbarkTotalBytes)
		}
		files = append(files, PlannedFile{Path: rel, Content: data})
	}
	got := ManifestSHA256(files)
	if got != prov.ManifestSHA256 {
		return errors.New("lifeboat manifest verification failed: the on-disk tree does not match _provenance.json")
	}
	return nil
}

// plannerResult is the shared output of runPlanner, consumed by both EmbarkProbe
// and EmbarkFrom so the read path and the write path plan identically.
type plannerResult struct {
	lifeboatAbs string
	targetAbs   string
	prov        Provenance
	planned     []PlannedEmbark
	conflicts   []Conflict
	ignored     []IgnoredFile
	coverage    *CoverageHandoff
}

// runPlanner gates the lifeboat and the target, verifies the manifest, then
// walks the lifeboat mapping only the embarked families into planned writes (or
// conflicts), reporting every other file. It writes nothing. Ordering is
// deterministic: the walk is sorted, so planned/conflicts/ignored are emitted in
// lifeboat-path order.
func runPlanner(lifeboatDir, targetDir string) (plannerResult, error) {
	lifeboatAbs, err := filepath.Abs(lifeboatDir)
	if err != nil {
		return plannerResult{}, err
	}
	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return plannerResult{}, err
	}

	// Both operands are proved against a symlinked ancestor before either is
	// read, so a refused target is refused before the lifeboat is verified.
	if err := proveOperand("lifeboat", lifeboatAbs); err != nil {
		return plannerResult{}, err
	}
	if err := proveOperand("target", targetAbs); err != nil {
		return plannerResult{}, err
	}

	// Gate the lifeboat: a real directory carrying a parseable _provenance.json.
	if !fsutil.IsRealDir(lifeboatAbs) {
		return plannerResult{}, fmt.Errorf("lifeboat %s is not a directory", filepath.Base(lifeboatAbs))
	}
	if !isAbcdLifeboat(lifeboatAbs) {
		return plannerResult{}, fmt.Errorf("%s is not an abcd lifeboat (no parseable %s)", filepath.Base(lifeboatAbs), ProvenanceName)
	}
	prov, err := readProvenance(lifeboatAbs)
	if err != nil {
		return plannerResult{}, err
	}
	// Schema gate: refuse a lifeboat newer than this abcd understands, message
	// mirroring the graveyard lessons ingest.
	if prov.SchemaVersion > SchemaVersion {
		return plannerResult{}, update.TooNew("lifeboat",
			prov.SchemaVersion, SchemaVersion)
	}
	if err := verifyManifest(lifeboatAbs); err != nil {
		return plannerResult{}, err
	}
	// Gate the target: a real directory (a symlinked or absent target is
	// structural). Non-git is allowed — embark has no git requirement.
	if !fsutil.IsRealDir(targetAbs) {
		return plannerResult{}, fmt.Errorf("target %s is not a directory", filepath.Base(targetAbs))
	}

	root, err := os.OpenRoot(lifeboatAbs)
	if err != nil {
		return plannerResult{}, err
	}
	defer root.Close()

	rels, err := walkLifeboatFiles(root)
	if err != nil {
		return plannerResult{}, err
	}

	var (
		planned   []PlannedEmbark
		conflicts []Conflict
		ignored   []IgnoredFile
	)
	// One lifeboat file per embark target: a second claimant (a bucket-less
	// intent alongside its bucketed twin) would be a silent last-writer-wins
	// overwrite, so it surfaces as a conflict instead.
	//
	// The key is case-folded on a filesystem that folds case: writeEmbark commits
	// through an overwriting rename, so two targets that differ only in case are
	// ONE file there, and a byte-sensitive map would plan both as Create, write
	// the second over the first, and still report two written. The lifeboat is
	// untrusted content that names both files, so the loser is the attacker's
	// choice. Folding makes the collision a conflict, and one conflict refuses the
	// whole write.
	claimed := map[string]claimedTarget{}
	for _, rel := range rels {
		family, targetRel, disp, detail := resolveTarget(rel)
		switch disp {
		case dispReportOnly:
			ignored = append(ignored, IgnoredFile{LifeboatPath: rel, Reason: IgnoredReportOnly})
		case dispUnmapped:
			ignored = append(ignored, IgnoredFile{LifeboatPath: rel, Reason: IgnoredUnmapped, Detail: detail})
		case dispUnknown:
			ignored = append(ignored, IgnoredFile{LifeboatPath: rel, Reason: IgnoredUnknown})
		case dispPlanned:
			data, err := readLifeboatFile(root, rel)
			if err != nil {
				return plannerResult{}, err
			}
			// Defence in depth: the table already guarantees a safe targetRel, but
			// re-validate before it becomes a write path.
			if !validRelPath(targetRel) {
				return plannerResult{}, fmt.Errorf("embark: refusing unsafe target path %q", targetRel)
			}
			key := fsutil.FoldPath(targetRel, caseFoldingFS())
			if first, dup := claimed[key]; dup {
				detail := "also mapped from " + first.lifeboatRel
				if first.targetRel != targetRel {
					detail += " as " + first.targetRel + ", which differs only in case — one file on this filesystem"
				}
				conflicts = append(conflicts, Conflict{
					Path:         targetRel,
					LifeboatPath: rel,
					Kind:         ConflictDuplicateTarget,
					Detail:       detail,
				})
				continue
			}
			claimed[key] = claimedTarget{lifeboatRel: rel, targetRel: targetRel}
			pe, cf := classifyEmbark(targetAbs, rel, targetRel, family, data)
			if cf != nil {
				conflicts = append(conflicts, *cf)
			} else {
				planned = append(planned, pe)
			}
		}
	}

	return plannerResult{
		lifeboatAbs: lifeboatAbs,
		targetAbs:   targetAbs,
		prov:        prov,
		planned:     planned,
		conflicts:   conflicts,
		ignored:     ignored,
		coverage:    readCoverageHandoff(lifeboatAbs, prov.PassBExemption),
	}, nil
}

// claimedTarget is the lifeboat file that already claimed one embark target,
// kept with the target's own spelling so a case-variant collision can say which
// two names collapsed rather than reporting an identical-looking duplicate.
type claimedTarget struct {
	lifeboatRel string
	targetRel   string
}

// disposition is where resolveTarget routes one lifeboat file.
type disposition int

const (
	dispPlanned disposition = iota
	dispReportOnly
	dispUnmapped
	dispUnknown
)

// resolveTarget maps a lifeboat-relative path to a target write via the
// embarkFamilies inverse table (the single source of truth). A file under a
// known family root that cannot resolve (unknown bucket, unsafe leaf, or
// bucket-less where no default exists) is Unmapped; a report-only path is
// ReportOnly; anything else is Unknown. A family is checked before the
// report-only prefixes so an unresolvable in-family file is Unmapped, not
// Unknown. detail is a short human note for the Unmapped case.
func resolveTarget(rel string) (family, targetRel string, disp disposition, detail string) {
	for _, f := range embarkFamilies {
		if !strings.HasPrefix(rel, f.LifeboatPrefix) {
			continue
		}
		rest := rel[len(f.LifeboatPrefix):]
		if f.Buckets == nil && f.BucketValid == nil {
			// Flat family (adrs): <prefix><leaf>.
			leaf := rest
			if strings.Contains(leaf, "/") || safeLeaf(leaf) == "" {
				return f.Name, "", dispUnmapped, "unresolvable path under " + f.Name
			}
			return f.Name, f.TargetPrefix + leaf, dispPlanned, ""
		}
		// Bucketed family.
		seg := strings.SplitN(rest, "/", 2)
		if len(seg) == 1 {
			// Bucket-less file directly under the family root.
			if f.DefaultBucket == "" {
				return f.Name, "", dispUnmapped, "bucket-less " + f.Name + " has no default bucket"
			}
			leaf := rest
			if safeLeaf(leaf) == "" {
				return f.Name, "", dispUnmapped, "unsafe leaf under " + f.Name
			}
			return f.Name, f.TargetPrefix + f.DefaultBucket + "/" + leaf, dispPlanned, ""
		}
		bucket, leaf := seg[0], seg[1]
		known := containsStr(f.Buckets, bucket)
		if f.BucketValid != nil {
			known = f.BucketValid(bucket)
		}
		if !known {
			return f.Name, "", dispUnmapped, "unknown " + f.Name + " bucket " + sanitize(bucket)
		}
		if strings.Contains(leaf, "/") || safeLeaf(leaf) == "" {
			return f.Name, "", dispUnmapped, "unsafe leaf under " + f.Name
		}
		if f.Leaf != "" && leaf != f.Leaf {
			return f.Name, "", dispUnmapped, "only " + f.Leaf + " is embarked under " + f.Name
		}
		return f.Name, f.TargetPrefix + bucket + "/" + leaf, dispPlanned, ""
	}
	for _, p := range reportOnlyPrefixes {
		if strings.HasPrefix(rel, p) {
			return "", "", dispReportOnly, ""
		}
	}
	return "", "", dispUnknown, ""
}

// classifyEmbark decides one planned write against the current target, lexical +
// lstat with NO symlink follow. A parent component that is a file or symlink is a
// parent-not-dir conflict; a target that exists and is not a regular file is a
// target-not-regular conflict; a regular file with equal bytes is an idempotent
// unchanged skip; differing bytes is an exists-differs conflict; an absent target
// is a create.
func classifyEmbark(targetAbs, lifeboatRel, targetRel, family string, content []byte) (PlannedEmbark, *Conflict) {
	if cf := checkParents(targetAbs, targetRel, lifeboatRel); cf != nil {
		return PlannedEmbark{}, cf
	}
	full := filepath.Join(targetAbs, filepath.FromSlash(targetRel))
	// Guarded read closes the lstat→read TOCTOU: a symlink swapped in after a
	// separate lstat would be followed by a plain os.ReadFile. ReadGuarded checks
	// regular-file on the open fd (O_NOFOLLOW) and caps the size in one call. An
	// absent target is a create; a symlink/non-regular/oversize/unreadable target is
	// a not-regular conflict.
	existing, err := fsutil.ReadGuarded(full, maxEmbarkFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PlannedEmbark{
				LifeboatPath: lifeboatRel, TargetPath: targetRel, Family: family,
				Bytes: len(content), Action: ActionCreate, Content: content,
			}, nil
		}
		detail := "cannot read the target file"
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, fsutil.ErrTooBig) {
			detail = "target exists and is not a readable regular file"
		}
		return PlannedEmbark{}, &Conflict{
			Path: targetRel, LifeboatPath: lifeboatRel, Kind: ConflictTargetNotRegular,
			Detail: detail,
		}
	}
	if bytes.Equal(existing, content) {
		return PlannedEmbark{
			LifeboatPath: lifeboatRel, TargetPath: targetRel, Family: family,
			Bytes: len(content), Action: ActionUnchanged, Content: content,
		}, nil
	}
	return PlannedEmbark{}, &Conflict{
		Path: targetRel, LifeboatPath: lifeboatRel, Kind: ConflictExistsDiffers,
		Detail: "target file differs from the lifeboat copy",
	}
}

// checkParents walks the existing parent components of targetRel under targetAbs
// and returns a parent-not-dir conflict for the first that is a symlink or a
// non-directory. A component that does not yet exist ends the walk — it (and
// every deeper one) is created by the contained MkdirAll on the write path.
func checkParents(targetAbs, targetRel, lifeboatRel string) *Conflict {
	dir := path.Dir(targetRel)
	if dir == "." {
		return nil
	}
	cur := targetAbs
	for _, seg := range strings.Split(dir, "/") {
		cur = filepath.Join(cur, seg)
		fi, err := os.Lstat(cur)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return &Conflict{Path: targetRel, LifeboatPath: lifeboatRel, Kind: ConflictParentNotDir, Detail: "cannot inspect a parent directory"}
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return &Conflict{Path: targetRel, LifeboatPath: lifeboatRel, Kind: ConflictParentNotDir, Detail: "a parent path is a symlink"}
		}
		if !fi.IsDir() {
			return &Conflict{Path: targetRel, LifeboatPath: lifeboatRel, Kind: ConflictParentNotDir, Detail: "a parent path is not a directory"}
		}
	}
	return nil
}

// writeEmbark performs the no-conflict write set through the two-layer idiom
// (os.Root containment + independent lexical validation + an exclusive create
// through createIntoLifeboat). A file whose write faults is removed; the SET is
// not transactional, which is acceptable — the conflict gate ran first, unchanged
// files are skipped, and a re-run is idempotent, so a partial set from an I/O
// fault re-completes on the next embark. A file a crash leaves half-written is
// reported by that embark as a conflict, never replaced.
func writeEmbark(targetAbs string, planned []PlannedEmbark) (written, unchanged, bytesWritten int, families map[string]int, err error) {
	families = map[string]int{}
	root, err := os.OpenRoot(targetAbs)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	defer root.Close()
	for _, p := range planned {
		if p.Action == ActionUnchanged {
			unchanged++
			continue
		}
		if !validRelPath(p.TargetPath) {
			return 0, 0, 0, nil, fmt.Errorf("refusing unsafe target path %q", p.TargetPath)
		}
		// Every planned write is a create, so it is an exclusive one: a file that
		// landed at the target after the rejudge, from a writer that took none of
		// the locks, fails the write loudly instead of being replaced.
		if err := createIntoLifeboat(root, p.TargetPath, p.Content); err != nil {
			if errors.Is(err, os.ErrExist) {
				return 0, 0, 0, nil, fmt.Errorf("refusing to replace %s: it appeared after the plan judged it absent: %w", p.TargetPath, err)
			}
			return 0, 0, 0, nil, err
		}
		written++
		bytesWritten += len(p.Content)
		families[p.Family]++
	}
	return written, unchanged, bytesWritten, families, nil
}

// markerSkipNoFile is the note a marker carries when the target's setup chose
// no conventions file for abcd's block (docs.target skip).
const markerSkipNoFile = "this project's setup chose no conventions file for abcd's block"

// markerSkipUnreadable is the note a marker carries when the target's
// .abcd/config.json cannot be read, so its choice of file is unknown and
// nothing is planted; the records still land, and a re-run plants the block
// once the file reads.
const markerSkipUnreadable = "this project's .abcd/config.json could not be read, so abcd's block was not planted; " +
	"fix that file and run embark again"

// markerFile chooses the conventions file embark plants abcd's block into,
// from the docs.target the target's setup saved, read with setup's own reader
// (ahoy.SavedDocsTarget): agents_md and no choice at all give AGENTS.md, the
// one conventions file abcd writes (adr-2610030814023326); skip, a retired
// value and an unreadable settings file give no file, with the note that says
// why. A retired value's note is the one ahoy.RetiredDocsTarget gives every
// front door.
func markerFile(targetAbs string) (file, note string) {
	v, err := ahoy.SavedDocsTarget(targetAbs)
	if err != nil {
		return "", markerSkipUnreadable
	}
	if why, retired := ahoy.RetiredDocsTarget(v); retired {
		return "", why
	}
	if v == "skip" {
		return "", markerSkipNoFile
	}
	return "AGENTS.md", ""
}

// embarkMarker predicts (dryRun) or performs the CURRENT abcd marker block in the
// conventions file the target's setup chose (markerFile) via ahoy.EnsureMarker,
// then maps the outcome to a MarkerAction. Probe and from both come through
// here, so a probe cannot mispredict the file or the action: the dry run asks
// the write's own questions, of the file and of the folder that must take its
// lock and temporary file, without writing (iss-2610032202263648).
// The install/refresh distinction is derived from whether the file already
// carried a block (via the exported ahoy.StripMarkerBlock), since EnsureMarker's
// signature reports only whether it changed. No chosen file, a
// symlinked/unreadable/unwritable one, and a root that cannot take a new file,
// is MarkerActionSkip (non-fatal — the records still land).
func embarkMarker(targetAbs string, dryRun bool) MarkerResult {
	file, note := markerFile(targetAbs)
	if file == "" {
		return MarkerResult{Action: MarkerActionSkip, Note: note}
	}
	p := filepath.Join(targetAbs, file)
	hadBlock := false
	// Guarded read: the target repo is arbitrary, so a symlinked or oversize/device
	// conventions file must not be followed or read unbounded. Any error leaves
	// hadBlock false (best-effort), as before.
	if data, err := fsutil.ReadGuarded(p, maxEmbarkFileBytes); err == nil {
		if _, had := ahoy.StripMarkerBlock(data); had {
			hadBlock = true
		}
	}
	changed, err := ahoy.EnsureMarker(p, dryRun)
	res := MarkerResult{Target: file, Changed: changed}
	switch {
	case err != nil:
		res.Action = MarkerActionSkip
		res.Changed = false
		res.Note = sanitize(err.Error())
	case !changed:
		res.Action = MarkerActionCurrent
	case hadBlock:
		res.Action = MarkerActionRefresh
	default:
		res.Action = MarkerActionInstall
	}
	return res
}

// readCoverageHandoff reads the lifeboat's coverage.json (bounded) and pulls the
// blanks-first payload. Absent → Present:false; present-but-unparseable →
// Degraded (a note, never fatal). Only blank sections become BlankPrompts, each
// sanitised before it can reach a terminal.
func readCoverageHandoff(abs string, exempt *PassBExemption) *CoverageHandoff {
	h := readPackedCoverage(abs)
	// The exemption is a fact about the PACKAGE, not about coverage.json, so it
	// travels even when the coverage is absent or unreadable: a reader still has
	// to be told which pass did not run.
	//
	// Its reason comes verbatim from _provenance.json, which manifest
	// verification deliberately excludes, so it is free text in any lifeboat
	// anyone can hand a user. It is sanitised HERE, where the handoff is built,
	// like every other string in it — the handoff is also what --json emits, and
	// a consumer piping that anywhere never sees the human render's own masking.
	if exempt != nil {
		h.PassBExemption = &PassBExemption{Reason: sanitize(exempt.Reason)}
	}
	return h
}

// readPackedCoverage reads coverage.json alone. It is what a caller wants when
// it needs the coverage summary and not the package-level declarations around it.
func readPackedCoverage(abs string) *CoverageHandoff {
	p := filepath.Join(abs, "coverage.json")
	// Guarded read closes the lstat→read TOCTOU and caps the size on the open fd.
	data, err := fsutil.ReadGuarded(p, maxEmbarkFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &CoverageHandoff{Present: false}
		}
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, fsutil.ErrTooBig) {
			return &CoverageHandoff{Present: true, Degraded: true, Note: "coverage.json is not a readable regular file"}
		}
		return &CoverageHandoff{Present: true, Degraded: true, Note: "coverage.json could not be read"}
	}
	var cov Coverage
	if err := json.Unmarshal(data, &cov); err != nil {
		return &CoverageHandoff{Present: true, Degraded: true, Note: "coverage.json is present but could not be parsed"}
	}
	out := &CoverageHandoff{Present: true, Summary: cov.Summary}
	for _, s := range cov.Sections {
		if s.Status != StatusBlank {
			continue
		}
		out.Blanks = append(out.Blanks, BlankPrompt{
			Section:  s.Name,
			Kind:     s.Kind,
			Question: sanitize(s.Question),
			Searched: sanitizeAll(s.Searched),
		})
	}
	return out
}

// walkLifeboatFiles returns every regular file's lifeboat-relative POSIX path,
// sorted, after enforcing the trust boundary through the containment root: no
// symlink anywhere (a symlinked entry is fatal, not followed), every path
// validRelPath, and the tree bounded against a hostile shape. It reads no content.
func walkLifeboatFiles(root *os.Root) ([]string, error) {
	return walkLifeboatFilesBounded(root, maxEmbarkFiles, maxDirEntries, maxEmbarkDepth)
}

// walkLifeboatFilesBounded is walkLifeboatFiles with its three bounds injected so
// a test can exercise each at an affordable scale: the whole-walk entry cap
// (limit, counting directories AND regular files), the per-directory read bound
// (perDir), and the depth cap (maxDepth). The shipped caps stay consts.
//
// This mirrors probe.go's walkFilesBounded — the hardened sibling this walk had
// diverged from (iss-112/114/116, GH #337/#343). Two bounds are load-bearing that
// the old fs.WalkDir(root.FS(), …) walk lacked:
//
//   - Directories count against limit. The old walk incremented only on regular
//     files, so a lifeboat made only of directories — a deep chain or a wide
//     fan-out — never reached the file cap and walked unbounded.
//   - Each directory is read with a bounded readDirBounded(perDir) — the same
//     per-directory guard ListDir and the probe walk use — so one very wide
//     directory cannot materialise its whole entry list (the old fs.WalkDir did
//     ReadDir(-1) + sort before the callback ever saw an entry) before the cap
//     applies. Exceeding perDir in a single directory refuses the lifeboat.
//
// It descends through a sub-root per directory (openWalkDir, which refuses a
// directory swapped for a FIFO without blocking — iss-337), so a chain
// costs O(entries) rather than O(entries × depth) while the os.Root containment
// guarantee the FS() walk had survives unchanged. Unlike the probe, every bound
// is FATAL rather than a silent truncation: manifest verification needs the
// complete file list, so a bounded-away subtree is a refusal, not a smaller answer.
func walkLifeboatFilesBounded(root *os.Root, limit, perDir, maxDepth int) ([]string, error) {
	var rels []string
	count := 0 // regular files AND directories, against limit
	var walk func(dirRoot *os.Root, prefix string, depth int) error
	walk = func(dirRoot *os.Root, prefix string, depth int) error {
		entries, more := readDirBounded(dirRoot, perDir)
		if more {
			return fmt.Errorf("lifeboat directory %q exceeds the %d-entry cap", prefix, perDir)
		}
		for _, e := range entries {
			name := e.Name()
			rel := name
			if prefix != "." {
				rel = prefix + "/" + name
			}
			if e.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("lifeboat contains a symlink at %q (refusing to follow)", rel)
			}
			if !validRelPath(rel) {
				return fmt.Errorf("lifeboat contains an unsafe path %q", rel)
			}
			count++
			if count > limit {
				return fmt.Errorf("lifeboat exceeds the %d-file cap", limit)
			}
			if e.IsDir() {
				if depth+1 >= maxDepth {
					return fmt.Errorf("lifeboat nesting at %q exceeds the %d-level cap", rel, maxDepth)
				}
				sub, err := openWalkDir(dirRoot, name)
				if err != nil {
					return walkOpenRefusal(rel, err)
				}
				err = walk(sub, rel, depth+1)
				sub.Close()
				if err != nil {
					return err
				}
				continue
			}
			if !e.Type().IsRegular() {
				return fmt.Errorf("lifeboat contains a non-regular file at %q", rel)
			}
			rels = append(rels, rel)
		}
		return nil
	}
	if err := walk(root, ".", 0); err != nil {
		return nil, err
	}
	sort.Strings(rels)
	return rels, nil
}

// walkOpenRefusal names a directory the walk could not descend into by its
// path in the lifeboat. The raw open error names the path openWalkDir opened —
// the last component with its "/." suffix — which is not a path the lifeboat
// holds (iss-2609252004013212), so only the cause is kept from it.
func walkOpenRefusal(rel string, err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	return fmt.Errorf("lifeboat directory %q cannot be opened: %w", rel, err)
}

// readLifeboatFile reads one lifeboat file through the containment root behind
// the trust guards: no symlink, regular file, size under maxEmbarkFileBytes. The
// lifeboat is untrusted, so the read goes through fsutil.ReadGuardedInRoot, which
// closes the lstat→open TOCTOU (O_NONBLOCK so a FIFO swapped in cannot block the
// open, an fstat-on-fd regular-file check, and os.SameFile to refuse a descriptor
// that is no longer the vetted file). The leading Lstat is kept only to preserve
// the distinct "is a symlink" message.
func readLifeboatFile(root *os.Root, rel string) ([]byte, error) {
	if fi, err := root.Lstat(rel); err != nil {
		return nil, err
	} else if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("lifeboat file %q is a symlink (refusing to follow)", rel)
	}
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxEmbarkFileBytes)
	if err != nil {
		if errors.Is(err, fsutil.ErrNotRegular) {
			return nil, fmt.Errorf("lifeboat file %q is not a regular file", rel)
		}
		if errors.Is(err, fsutil.ErrTooBig) {
			return nil, fmt.Errorf("lifeboat file %q exceeds the %d-byte cap", rel, maxEmbarkFileBytes)
		}
		return nil, err
	}
	return data, nil
}

// readProvenance reads and parses the lifeboat's _provenance.json behind the
// trust guards (no symlink, regular file, bounded). It is the manifest header and
// the schema/name/hash source.
func readProvenance(abs string) (Provenance, error) {
	var prov Provenance
	p := filepath.Join(abs, ProvenanceName)
	// Guarded read closes the lstat→read TOCTOU (a symlink swapped in after the
	// lstat would be followed) and refuses a non-regular or oversize file on the
	// open fd (O_NOFOLLOW), in one call.
	data, err := fsutil.ReadGuarded(p, maxProvenanceBytes)
	if err != nil {
		if errors.Is(err, fsutil.ErrNotRegular) {
			return prov, fmt.Errorf("lifeboat: %s is not a regular file (or a symlink; refusing to follow)", ProvenanceName)
		}
		if errors.Is(err, fsutil.ErrTooBig) {
			return prov, fmt.Errorf("lifeboat: %s exceeds the %d-byte cap", ProvenanceName, maxProvenanceBytes)
		}
		return prov, fmt.Errorf("lifeboat: reading %s: %w", ProvenanceName, err)
	}
	if err := json.Unmarshal(data, &prov); err != nil {
		return prov, fmt.Errorf("lifeboat: %s is not valid JSON: %w", ProvenanceName, err)
	}
	return prov, nil
}

// isManifestExcluded reports whether a lifeboat path was left out of
// manifest_sha256 (the header and the post-pack layer-3 interpretation), so
// verifyManifest reproduces the pinned hash exactly.
func isManifestExcluded(rel string) bool {
	for _, e := range manifestExcludedExact {
		if rel == e {
			return true
		}
	}
	for _, p := range manifestExcludedPrefixes {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}
	return false
}

// containsStr reports set membership for a small string slice.
func containsStr(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}
