package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/core/mode"
)

// embargo_status_test.go: the front door of `abcd embargo fixed`, of bare
// `abcd embargo`, which lists each advisory whose recorded fix is in a
// published release as ready to publish and writes nothing, and of `abcd
// embargo queue`, which while the person is away queues their publish step
// once. The core's judgement is held in
// internal/core/embargo; these tests hold what the surface adds.

// embargoStatusDoc is bare `abcd embargo --json`, as far as these tests read it.
type embargoStatusDoc struct {
	NoStore bool                    `json:"no_store"`
	Ready   []embargo.ReadyAdvisory `json:"ready"`
	Queued  []string                `json:"queued"`
}

func bareEmbargo(t *testing.T) embargoStatusDoc {
	t.Helper()
	code, out, stderr := runEmbargo(t, "", "embargo", "--json")
	if code != 0 {
		t.Fatalf("bare embargo: exit %d, %s%s", code, out, stderr)
	}
	var doc embargoStatusDoc
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("bare embargo --json: %v\n%s", err, out)
	}
	return doc
}

// TestBareEmbargoWithoutAStore: bare `abcd embargo` in a repository with no
// embargo store says so, exits 0, and makes none.
func TestBareEmbargoWithoutAStore(t *testing.T) {
	embargoSurfaceRepo(t)
	if doc := bareEmbargo(t); !doc.NoStore || len(doc.Ready) != 0 || len(doc.Queued) != 0 {
		t.Fatalf("bare embargo without a store: %+v", doc)
	}
	code, out, _ := runEmbargo(t, "", "embargo")
	if code != 0 || !strings.Contains(out, "no embargo store") || !strings.Contains(out, "abcd embargo intake") {
		t.Fatalf("bare embargo text: exit %d, %s", code, out)
	}
}

// TestBareEmbargoListsReadyAndQueueQueuesWhileAway: once `abcd embargo fixed`
// records a fix commit a published release holds, bare `abcd embargo` lists
// the advisory as ready to publish, naming the release, and writes nothing,
// while the person is away as while they are present. `abcd embargo queue`
// refuses while they are present, and while they are away queues their
// publish step once, naming the record and the release and never the
// advisory's id, and publishes nothing.
func TestBareEmbargoListsReadyAndQueueQueuesWhileAway(t *testing.T) {
	r := embargoSurfaceRepo(t)
	seedEmbargo(t)
	code, out, stderr := runEmbargo(t, "", "embargo", "triage", surfaceGHSA, "vulnerability", "--by", "Alice", "--json")
	if code != 0 {
		t.Fatalf("triage: exit %d, %s%s", code, out, stderr)
	}
	var tr struct {
		IssueID string `json:"issue_id"`
	}
	if err := json.Unmarshal([]byte(out), &tr); err != nil || tr.IssueID == "" {
		t.Fatalf("triage document %v %s", err, out)
	}

	r.Write("parser.go", "package parser\n")
	r.Commit("fix: tighten the parser")
	fix := r.Git("rev-parse", "HEAD")
	code, out, stderr = runEmbargo(t, "", "embargo", "fixed", tr.IssueID, fix[:12])
	if code != 0 || !strings.Contains(out, "1 fix commit") || !strings.Contains(out, fix) {
		t.Fatalf("fixed: exit %d, %s%s", code, out, stderr)
	}
	if got := r.Git("status", "--porcelain"); got != "" {
		t.Fatalf("fixed changed the checkout:\n%s", got)
	}

	bare := filepath.Join(t.TempDir(), "origin.git")
	initBare(t, bare)
	r.Git("config", "remote.origin.url", bare)
	t.Cleanup(embargo.SwapOriginRepoForTest(func(string) (string, error) { return "example/example", nil }))
	r.Git("tag", "v0.1.0")
	r.Git("push", "-q", "origin", "refs/tags/v0.1.0")

	doc := bareEmbargo(t)
	if len(doc.Ready) != 1 || doc.Ready[0].IssueID != tr.IssueID || doc.Ready[0].Release != "v0.1.0" || len(doc.Queued) != 0 {
		t.Fatalf("bare embargo while present: %+v", doc)
	}
	code, out, _ = runEmbargo(t, "", "embargo")
	if code != 0 || !strings.Contains(out, "1 advisory ready to publish") || !strings.Contains(out, "abcd embargo publish --release v0.1.0") {
		t.Fatalf("bare embargo text: exit %d, %s", code, out)
	}

	if err := os.MkdirAll(filepath.Join(r.Root(), filepath.FromSlash(mode.TierRelPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	code, out, stderr = runEmbargo(t, "", "embargo", "queue")
	if code != 2 || !strings.Contains(stderr, "not away") || !strings.Contains(stderr, "nothing queued") {
		t.Fatalf("embargo queue while present: exit %d, %s%s", code, out, stderr)
	}
	if code, out, stderr := runEmbargo(t, "", "mode", "autonomous", "--until", "09:00"); code != 0 {
		t.Fatalf("mode autonomous: exit %d, %s%s", code, out, stderr)
	}
	code, out, _ = runEmbargo(t, "", "embargo")
	if doc := bareEmbargo(t); code != 0 || strings.Contains(out, "queued") || len(doc.Queued) != 0 {
		t.Fatalf("bare embargo while away wrote: exit %d, %s %+v", code, out, doc)
	}
	if rec, _, err := autonomy.AwayAt(r.Root(), autonomy.Options{}); err != nil || rec == nil || len(rec.Queue) != 0 {
		t.Fatalf("bare embargo while away queued a step: %+v, %v", rec, err)
	}
	code, out, stderr = runEmbargo(t, "", "embargo", "queue")
	if code != 0 || !strings.Contains(out, "queued the publish step for "+tr.IssueID) || !strings.Contains(out, "nothing was published") {
		t.Fatalf("embargo queue while away: exit %d, %s%s", code, out, stderr)
	}
	code, out, stderr = runEmbargo(t, "", "embargo", "queue", "--json")
	var again struct {
		Queued  []string `json:"queued"`
		Already []string `json:"already_queued"`
	}
	if err := json.Unmarshal([]byte(out), &again); code != 0 || err != nil || len(again.Queued) != 0 || len(again.Already) != 1 {
		t.Fatalf("a second embargo queue: exit %d, %v, %s%s", code, err, out, stderr)
	}
	rec, away, err := autonomy.AwayAt(r.Root(), autonomy.Options{})
	if err != nil || !away {
		t.Fatalf("away record: %v, %v", away, err)
	}
	if len(rec.Queue) != 1 {
		t.Fatalf("queue %+v, want one publish step", rec.Queue)
	}
	step := rec.Queue[0].Step
	if !strings.Contains(step, tr.IssueID) || !strings.Contains(step, "v0.1.0") || strings.Contains(step, surfaceGHSA) {
		t.Fatalf("queued step %q", step)
	}
}
