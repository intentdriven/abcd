package scribe

// plan.go — the practice run of `scribe ingest --scribe-json`
// (itd-2610301428390406, spc-2610301431131636 step 2). Every check the scribe
// makes of its own — the closed shapes, the run's identity against the parked
// context and the researcher's text, the run committed and not yet
// transcribed, nothing authored — runs as the real ingest runs it
// (checkIngest), and the records the capture verbs would write are reported
// instead of written.
//
// The capture verbs mint each record's id and apply their own redaction and
// refusals at the write; those are the ledger's, not the scribe's, and a
// practice run cannot make them without writing. The plan says so on each
// record it names.

import (
	"os"
	"path"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/dryrun"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// IngestPlan is what Ingest would do: the result it would report, each record
// with the id the ledger mints shown as dryrun.MintedID, and the writes.
type IngestPlan struct {
	Result IngestResult
	Plan   dryrun.Plan
}

// ledgerPending is what each planned ledger record carries: the part of it
// only the write settles.
const ledgerPending = "the capture verb mints its id and applies its own redaction and refusals when it writes"

// PlanIngest makes every check Ingest makes before its first write and
// reports the records it would land, in the order it lands them, then the
// promoted manifest. It writes nothing.
func PlanIngest(req IngestRequest) (IngestPlan, error) {
	out, m, err := checkIngest(req)
	if err != nil {
		return IngestPlan{}, err
	}
	res := newIngestResult(out, m)
	var plan dryrun.Plan
	ledger := recordid.IssuesRelDir
	// The capture verbs take the ledger's allocator lock, laying its file the
	// first time any record is written. The name is the ledger's own; the
	// tree-wide practice-run test holds the two to one another.
	if lock := path.Join(ledger, ".iss-alloc.lock"); !exists(req.RepoRoot, lock) && len(out.Dispositions)+len(out.Admissions)+len(out.Surprises) > 0 {
		plan.Writes = append(plan.Writes, dryrun.Write{Path: lock, Action: dryrun.Create,
			Pending: "the ledger's allocator lock, laid empty when the first record takes it"})
	}
	minted := issueschema.DispositionFamily + "-" + dryrun.MintedID
	// standing counts, per item, the dispositions that would stand when an
	// admission is written: those on the ledger and those this ingest lands
	// first. An admission over an item with none writes its `accepted`
	// disposition beside it.
	standing := map[string]int{}
	for _, d := range out.Dispositions {
		rel := path.Join(ledger, issueschema.DispositionsDir, d.Item, minted+".md")
		plan.Writes = append(plan.Writes, dryrun.Write{Path: rel, Action: dryrun.Create, Pending: ledgerPending})
		res.Dispositions = append(res.Dispositions, capture.DispositionResult{ID: minted, Item: d.Item, State: d.State, Path: rel})
		standing[d.Item]++
	}
	for _, a := range out.Admissions {
		fate, err := capture.ItemFate(req.RepoRoot, out.Run, a.Item)
		if err != nil {
			return IngestPlan{}, err
		}
		r := capture.AdmitResult{Item: a.Item, Run: out.Run}
		if len(fate.Dispositions)+standing[a.Item] == 0 {
			rel := path.Join(ledger, issueschema.DispositionsDir, a.Item, minted+".md")
			plan.Writes = append(plan.Writes, dryrun.Write{Path: rel, Action: dryrun.Create, Pending: ledgerPending})
			r.Disposition, r.DispositionPath, r.DispositionWritten = minted, rel, true
			standing[a.Item]++
		}
		adm := issueschema.AdmissionFamily + "-" + dryrun.MintedID
		rel := path.Join(ledger, issueschema.AdmissionsDir, out.Run, adm+".md")
		plan.Writes = append(plan.Writes, dryrun.Write{Path: rel, Action: dryrun.Create, Pending: ledgerPending})
		r.Admission, r.Path = adm, rel
		res.Admissions = append(res.Admissions, r)
	}
	for _, sp := range out.Surprises {
		id := issueschema.SurpriseFamily + "-" + dryrun.MintedID
		rel := path.Join(ledger, issueschema.SurprisesDir, id+".md")
		plan.Writes = append(plan.Writes, dryrun.Write{Path: rel, Action: dryrun.Create, Pending: ledgerPending})
		res.Surprises = append(res.Surprises, capture.SurpriseResult{ID: id, Path: rel, OccasionedBy: sp.OccasionedBy})
	}
	// The manifest is promoted only behind a record that landed, as Ingest
	// promotes it.
	if len(res.Landed()) > 0 {
		rel := issueschema.ReadingsRecordDir + "/" + out.Run + "/" + ManifestFileName
		plan.Writes = append(plan.Writes, dryrun.Write{Path: rel, Action: dryrun.Create,
			Pending: "the parked manifest, promoted write-once beside the run once every record has landed"})
		res.Manifest = rel
	}
	return IngestPlan{Result: res, Plan: plan}, nil
}

// exists reports whether rel names anything under root.
func exists(root, rel string) bool {
	r, err := os.OpenRoot(root)
	if err != nil {
		return false
	}
	defer r.Close()
	_, err = r.Lstat(rel)
	return err == nil
}
