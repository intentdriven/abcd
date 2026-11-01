package reading

// plan.go — the practice run of `reading ingest --reading-json`
// (itd-2610301428390406, spc-2610301431131636 step 2). The ingest's body runs
// with a plan in hand (ingestUnderLock's pl): the envelope, the parked run's
// identity, the definition, the regime, the instrument, every item and the
// prose-citation gate are judged exactly as the real ingest judges them, and
// each write the staged-write protocol would make is reported instead.

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/dryrun"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// ingestPlan is a practice run in the making.
type ingestPlan struct {
	dryrun.Plan
}

// IngestPlan is what Ingest would do: the result it would report and the
// writes the staged-write protocol would make.
type IngestPlan struct {
	Result IngestResult
	Plan   dryrun.Plan
}

// PlanIngest does everything Ingest does before its first write, and reports
// the writes. It takes no stage lock and creates no stage root, because each
// is a write; a stage root that is not a real directory is refused as the
// ingest refuses it. A refusal is the ingest's, in its words, except that a
// refusal the ingest would record says where it would be recorded instead.
func PlanIngest(req IngestRequest) (IngestPlan, error) {
	res := IngestResult{}
	repoRoot := strings.TrimSpace(req.RepoRoot)
	if repoRoot == "" {
		return IngestPlan{}, errors.New("reading: ingesting an output needs a repository root")
	}
	if strings.TrimSpace(req.OutputPath) == "" {
		return IngestPlan{}, errors.New("reading: ingest needs the reading's output JSON")
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return IngestPlan{}, fmt.Errorf("reading: opening the repository root: %w", err)
	}
	defer root.Close()
	pl := &ingestPlan{}
	switch fi, err := root.Lstat(IngestStageDir); {
	case err == nil && (fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir()):
		return IngestPlan{}, fmt.Errorf("reading: %s is not a real directory; the ingest stage is a directory in this "+
			"repository, never a pointer at another one", IngestStageDir)
	case err != nil && !os.IsNotExist(err):
		return IngestPlan{}, fmt.Errorf("reading: preparing the ingest stage: %w", err)
	}
	lock := IngestStageDir + "/" + stageLockName
	if _, err := root.Lstat(lock); os.IsNotExist(err) {
		pl.Writes = append(pl.Writes, dryrun.Write{Path: lock, Action: dryrun.Create,
			Pending: "the stage's lock file, laid empty when the ingest takes the lock"})
	}
	if err := ingestUnderLock(root, repoRoot, req, &res, pl); err != nil {
		return IngestPlan{Result: res}, err
	}
	return IngestPlan{Result: res, Plan: pl.Plan}, nil
}

// plannedItemID stands for a minted reading-item id: the family and a
// sixteen-digit stamp, so a planned run record has the size the real one has.
var plannedItemID = issueschema.ReadingItemFamily + "-" + strings.Repeat("0", 16)

// planWrite is write's practice run: the ledger record of each item, which
// the ledger writer mints and composes; the manifest promoted beside the run;
// and the run metadata, the commit marker. The stage is laid and cleared
// within the run, so it leaves nothing and is not planned.
func planWrite(root *os.Root, repoRoot string, res *IngestResult, out Output, m Manifest, def Definition,
	free payloadField, items []capture.ReadingItem, pl *ingestPlan) error {
	ledgerRun := recordid.IssuesRelDir + "/" + issueschema.ReadingsDir + "/" + out.RunID
	// The ledger writer takes the ledger's allocator lock, laying its file the
	// first time any record is written. The name is the ledger's own; the
	// tree-wide practice-run test holds the two to one another.
	if lock := recordid.IssuesRelDir + "/.iss-alloc.lock"; func() bool { _, err := root.Lstat(lock); return os.IsNotExist(err) }() {
		pl.Writes = append(pl.Writes, dryrun.Write{Path: lock, Action: dryrun.Create,
			Pending: "the ledger's allocator lock, laid empty when the records take it"})
	}
	var refs, sized []capture.ReadingRecordRef
	for range items {
		rel := ledgerRun + "/" + issueschema.ReadingItemFamily + "-" + dryrun.MintedID + ".md"
		pl.Writes = append(pl.Writes, dryrun.Write{Path: rel, Action: dryrun.Create,
			Pending: "the item's id is minted, and its record composed and redacted, by the ledger writer when it is recorded"})
		refs = append(refs, capture.ReadingRecordRef{ID: issueschema.ReadingItemFamily + "-" + dryrun.MintedID, Path: rel})
		sized = append(sized, capture.ReadingRecordRef{ID: plannedItemID, Path: ledgerRun + "/" + plannedItemID + ".md"})
	}
	res.Records = refs

	runRel := ReadingsRecordDir + "/" + out.RunID
	mData, err := encode(m)
	if err != nil {
		return err
	}
	pl.Writes = append(pl.Writes, dryrun.Write{Path: runRel + "/" + ManifestFileName, Action: dryrun.Create,
		Bytes: len(mData), Content: mData})
	run := RunRecord{
		Type: RunType, SchemaVersion: SchemaVersion, RunID: out.RunID,
		Position: def.Position, Regime: def.Regime, TargetCommit: m.TargetCommit,
		ManifestSHA256: out.ManifestSHA256, Instrument: sanitizeInstrument(out.Instrument, free),
		Records: sized, RefusedItems: res.RefusedItems, RefusedCount: res.RefusedCount,
		CandidateRun: m.CandidateRun, Candidates: m.Candidates, Exercised: m.Exercised,
		Bounds: statedBounds(root, m),
	}
	if run.RefusedItems == nil {
		run.RefusedItems = []ItemRefusal{}
	}
	if run.Records == nil {
		run.Records = []capture.ReadingRecordRef{}
	}
	rData, err := encode(run)
	if err != nil {
		return err
	}
	pending := ""
	if len(items) > 0 {
		pending = "it lists the records' ids, minted when they are recorded"
	}
	pl.Writes = append(pl.Writes, dryrun.Write{Path: runRel + "/" + RunFileName, Action: dryrun.Create,
		Bytes: len(rData), Pending: pending})
	res.RunRecordPath = runRel + "/" + RunFileName
	return nil
}
