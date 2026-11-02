package release

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/dryrun"
)

// plannedPaths is each planned write as "action path", in plan order.
func plannedPaths(p dryrun.Plan) []string {
	var out []string
	for _, w := range p.Writes {
		out = append(out, string(w.Action)+" "+w.Path)
	}
	return out
}

// TestPlanIngestWritesNothingAndPlansTheRealWrites (spc-2610301431131636 step
// 2): the practice run of a valid payload names every write the cut lays, in
// the order it lays them, with the bytes it would lay, and the tag, push and
// release the workflow makes after it; the tree is untouched; and the real
// ingest then writes exactly those paths, with the result the plan reported.
func TestPlanIngestWritesNothingAndPlansTheRealWrites(t *testing.T) {
	r := targetedShippable(t)
	root := r.Root()
	raw := marshalPayload(t, "v0.4.1", goodEntries())
	before := treeDigest(t, root)

	plan, err := PlanIngest(root, liveSurface(), raw, cutAt)
	if err != nil {
		t.Fatalf("PlanIngest: %v", err)
	}
	if after := treeDigest(t, root); after != before {
		t.Fatal("the practice run changed the working tree")
	}
	want := []string{"replace " + plannedDir + "itd-91-due.md", "create " + PageFile, "replace " + changelogFile}
	if got := plannedPaths(plan.Plan); !reflect.DeepEqual(got, want) {
		t.Errorf("planned writes = %v, want %v", got, want)
	}
	var effects []string
	for _, e := range plan.Plan.Effects {
		effects = append(effects, e.Effect)
		if !strings.Contains(e.Target, "v0.4.1") {
			t.Errorf("the %s effect does not name the version: %q", e.Effect, e.Target)
		}
	}
	if strings.Join(effects, ",") != "tag,push,release" {
		t.Errorf("effects = %v, want tag, push and release", effects)
	}
	if !plan.Result.Written || plan.Result.Heading != "## [0.4.1] - 2026-07-21" {
		t.Errorf("the planned result = written %v, heading %q", plan.Result.Written, plan.Result.Heading)
	}

	ops := newRecordingOps(root)
	res, err := ingest(root, liveSurface(), raw, cutAt, ops)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	// The recording seam logs a create of a missing page as its replace, the
	// one way osOps lays RELEASE.md.
	wantOps := []string{"replace " + plannedDir + "itd-91-due.md", "replace " + PageFile, "replace " + changelogFile}
	if strings.Join(ops.log, "|") != strings.Join(wantOps, "|") {
		t.Errorf("the real writes = %v, want %v", ops.log, wantOps)
	}
	for _, w := range plan.Plan.Writes {
		if got := readRel(t, root, w.Path); got != string(w.Content) || len(got) != w.Bytes {
			t.Errorf("%s: the real write laid %d bytes, the plan %d, and they differ", w.Path, len(got), w.Bytes)
		}
	}
	if res.Heading != plan.Result.Heading || !reflect.DeepEqual(res.Cited, plan.Result.Cited) ||
		!reflect.DeepEqual(res.Page, plan.Result.Page) {
		t.Errorf("the real result differs from the planned one:\nreal %+v\nplan %+v", res, plan.Result)
	}
}

// TestPlanIngestRefusesAsIngestDoes: a payload the ingest refuses is refused by
// the practice run with the same reasons, writing nothing; a refused cut plans
// nothing and is no error, as the ingest reports it.
func TestPlanIngestRefusesAsIngestDoes(t *testing.T) {
	r := shippableRepo(t)
	root := r.Root()
	raw := marshalPayload(t, "v0.4.1", goodEntries()[:1]) // the bijection misses itd-73
	before := treeDigest(t, root)

	_, perr := PlanIngest(root, liveSurface(), raw, cutAt)
	if after := treeDigest(t, root); after != before {
		t.Fatal("the refused practice run changed the working tree")
	}
	_, rerr := Ingest(root, liveSurface(), raw, cutAt)
	var pr, rr *PayloadRefusal
	if !errors.As(perr, &pr) || !errors.As(rerr, &rr) {
		t.Fatalf("practice %v, real %v: want a payload refusal from both", perr, rerr)
	}
	if perr.Error() != rerr.Error() || !reflect.DeepEqual(pr.Reasons, rr.Reasons) {
		t.Errorf("the refusals differ:\npractice %v\nreal     %v", pr.Reasons, rr.Reasons)
	}

	refused := releasedRepo(t)
	refused.Write("CHANGELOG.md", baseChangelog)
	refused.Record(resolvedDir+"iss-97-toctou.md", "iss-97", "internal")
	refused.Commit("resolve an internal issue only")
	plan, err := PlanIngest(refused.Root(), liveSurface(), raw, cutAt)
	if err != nil || plan.Result.Cut.Ready || plan.Result.Written || len(plan.Plan.Writes)+len(plan.Plan.Effects) != 0 {
		t.Errorf("a refused cut's practice run = ready %v, written %v, plan %+v, err %v; want a refused cut planning nothing",
			plan.Result.Cut.Ready, plan.Result.Written, plan.Plan, err)
	}
}
