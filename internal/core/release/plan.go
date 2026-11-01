package release

// plan.go — the practice run of the release cut's ingest step
// (itd-2610301428390406, spc-2610301431131636 step 2): `launch ship
// --changelog-json --dry-run` judges the composed payload against the cut it
// would make and reports every write the cut would lay, writing nothing.

import (
	"time"

	"github.com/intentdriven/abcd/internal/core/dryrun"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/surface"
)

// IngestPlan is what Ingest would do: the result it would report, and its
// writes and the effects that follow from them.
type IngestPlan struct {
	// Result is the result the real ingest would return. For a refused cut it
	// carries the cut alone and Written is false, as Ingest's does.
	Result IngestResult
	// Plan is the practice run's report: empty for a refused cut.
	Plan dryrun.Plan
}

// PlanIngest does everything Ingest does before its first write, with the
// same refusals in the same words: the cut is derived, the payload judged
// against it, every write composed, and the intent records whose missed
// target the cut moves to `next` read and rewritten in memory. It writes
// nothing: it takes neither the CHANGELOG's lock nor the intent store's,
// because each is a file the lock lays, and reads the tree as it stands.
//
// The payload refusal is the *PayloadRefusal Ingest returns, with the result
// carrying the cut, so a front door renders it exactly as it renders the
// real run's.
func PlanIngest(root string, current surface.Snapshot, raw []byte, at time.Time) (IngestPlan, error) {
	res, p, err := planIngest(root, current, raw, at)
	if err != nil || p == nil {
		return IngestPlan{Result: res}, err
	}
	rewrites, err := intent.PlanTargetMoves(root, p.moves)
	if err != nil {
		return IngestPlan{Result: IngestResult{Cut: res.Cut}}, err
	}
	p.cut.records = rewrites
	return IngestPlan{Result: res, Plan: p.cut.dryrun(res.Cut.NextTag)}, nil
}

// dryrun reports the plan's writes in the order execute lays them, then what
// the release workflow does once the dated heading is on the default branch:
// the cut itself tags, pushes and publishes nothing.
func (p cutPlan) dryrun(nextTag string) dryrun.Plan {
	var out dryrun.Plan
	for _, rec := range p.records {
		out.Writes = append(out.Writes, dryrun.Write{Path: rec.Path, Action: dryrun.Replace, Bytes: len(rec.After), Content: rec.After})
	}
	if p.page != nil && p.undo.archived != "" {
		// The outgoing page's bytes, created under the archive and never
		// overwriting one; RELEASE.md itself is then replaced below.
		out.Writes = append(out.Writes, dryrun.Write{Path: p.undo.archived, Action: dryrun.Create, Bytes: len(p.undo.pageBefore)})
	}
	if p.page != nil {
		action := dryrun.Create
		if p.undo.pageExisted {
			action = dryrun.Replace
		}
		out.Writes = append(out.Writes, dryrun.Write{Path: PageFile, Action: action, Bytes: len(p.page), Content: p.page})
	}
	if p.digest != nil {
		action := dryrun.Create
		if p.undo.digestExisted {
			action = dryrun.Replace
		}
		out.Writes = append(out.Writes, dryrun.Write{Path: p.digestRel, Action: action, Bytes: len(p.digest), Content: p.digest})
	}
	out.Writes = append(out.Writes, dryrun.Write{Path: changelogFile, Action: dryrun.Replace, Bytes: len(p.changelog), Content: p.changelog})
	out.Effects = releaseEffects(nextTag)
	return out
}

// releaseEffects is what follows a written cut that this program does not do
// itself: the release workflow tags the version the dated heading names once
// the commit carrying it reaches the default branch, pushes the tag, and
// publishes the release.
func releaseEffects(nextTag string) []dryrun.Effect {
	if nextTag == "" {
		return nil
	}
	return []dryrun.Effect{
		{Effect: "tag", Target: nextTag + ", by the release workflow from the dated heading once this commit reaches the default branch (this command tags nothing)"},
		{Effect: "push", Target: "refs/tags/" + nextTag + " to origin, by the release workflow (this command pushes nothing)"},
		{Effect: "release", Target: nextTag + " on the forge, by the release workflow (this command publishes nothing)"},
	}
}
