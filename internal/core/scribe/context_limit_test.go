package scribe

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/reading"
)

// holdLimit sets one of the scribe's parking limits for the length of a test.
func holdLimit(t *testing.T, limit *int64, n int64) {
	t.Helper()
	old := *limit
	*limit = n
	t.Cleanup(func() { *limit = old })
}

// sessionOver assembles a session over an already built fixture, with one
// disposition for its first item.
func sessionOver(t *testing.T, f fixture) session {
	t.Helper()
	supplied := f.items[0] + ": accepted — " + groundA + ".\n"
	dispPath := supply(t, supplied)
	res, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: dispPath})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return session{fixture: f, supplied: supplied, dispPath: dispPath, res: res}
}

// ingestOne ingests the session's one disposition.
func (s session) ingestOne(t *testing.T) (IngestResult, error) {
	t.Helper()
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	return s.ingest(t, s.write(t, o))
}

// TestScribeIngestReadsAContextPastTheFileCap is iss-2611040555430199: over a
// ledger whose records add up past the per-file cap, assemble parks the
// context and ingest reads it back and lands the disposition, instead of
// refusing every ingest after the session has transcribed against it.
func TestScribeIngestReadsAContextPastTheFileCap(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	// Six open issues, each under the ledger's record limit, that together put
	// the context past the file cap.
	line := "The issue is recorded here at length, as a long ledger records it.\n"
	body := strings.Repeat(line, (900<<10)/len(line))
	for i := 1; i <= 6; i++ {
		id := "iss-" + strconv.Itoa(100+i)
		writeFile(t, f.repo, ".abcd/work/issues/open/"+id+"-a-long-issue.md", "---\nid: "+id+"\n---\n"+body)
	}
	s := sessionOver(t, f)
	ctxPath := filepath.Join(s.repo, filepath.FromSlash(DefaultRunDir), fixtureRun, ContextFileName)
	fi, err := os.Stat(ctxPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() <= reading.MaxFileBytes {
		t.Fatalf("the context is %d bytes, not past the %d-byte file cap the test is about", fi.Size(), reading.MaxFileBytes)
	}
	res, err := s.ingestOne(t)
	if err != nil {
		t.Fatalf("Ingest against a %d-byte context assemble parked: %v", fi.Size(), err)
	}
	if len(res.Dispositions) != 1 {
		t.Fatalf("Ingest landed %d dispositions, want 1", len(res.Dispositions))
	}
}

// TestScribeSessionAtTheLimitRoundTripsAndOnePastIsRefusedAtAssemble holds the
// rule both ends of a scribe session share: an artefact exactly at the limit
// the ingest reads it under is parked and ingested, and one byte past it is
// refused by assemble, naming the artefact, its size and the limit, before
// anything is written, dry run included.
func TestScribeSessionAtTheLimitRoundTripsAndOnePastIsRefusedAtAssemble(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit *int64
		of    func(AssembleResult) any
	}{
		{ContextFileName, &contextLimit, func(r AssembleResult) any { return r.Context }},
		{ManifestFileName, &manifestLimit, func(r AssembleResult) any { return r.Manifest }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, positionDetection, 1)
			supplied := f.items[0] + ": accepted — " + groundA + ".\n"
			dry, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: supply(t, supplied), DryRun: true})
			if err != nil {
				t.Fatalf("dry-run Assemble: %v", err)
			}
			raw, err := encode(tc.of(dry))
			if err != nil {
				t.Fatal(err)
			}
			size := len(raw)

			holdLimit(t, tc.limit, int64(size-1))
			for _, dryRun := range []bool{false, true} {
				_, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun,
					DispositionsPath: supply(t, f.items[0]+": accepted — "+groundA+".\n"), DryRun: dryRun})
				if err == nil {
					t.Fatalf("Assemble (dry run %v) with the %s one byte past the %d-byte limit was not refused",
						dryRun, tc.name, size-1)
				}
				for _, want := range []string{tc.name, strconv.Itoa(size) + " bytes", strconv.Itoa(size-1) + "-byte limit"} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("the refusal (dry run %v) does not name %q: %v", dryRun, want, err)
					}
				}
			}
			if _, err := os.Stat(filepath.Join(f.repo, filepath.FromSlash(DefaultRunDir), fixtureRun)); !os.IsNotExist(err) {
				t.Fatalf("a refused assemble left the run directory behind: %v", err)
			}

			holdLimit(t, tc.limit, int64(size))
			s := sessionOver(t, f)
			parked := filepath.Join(s.repo, filepath.FromSlash(DefaultRunDir), fixtureRun, tc.name)
			fi, err := os.Stat(parked)
			if err != nil {
				t.Fatal(err)
			}
			if fi.Size() != int64(size) {
				t.Fatalf("the parked %s is %d bytes and the dry run's %d", tc.name, fi.Size(), size)
			}
			if _, err := s.ingestOne(t); err != nil {
				t.Fatalf("Ingest with the %s exactly at the %d-byte limit: %v", tc.name, size, err)
			}
		})
	}
}
