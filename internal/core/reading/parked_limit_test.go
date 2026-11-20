package reading

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// holdLimit sets one of the package's parking limits for the length of a test.
func holdLimit(t *testing.T, limit *int64, n int64) {
	t.Helper()
	old := *limit
	*limit = n
	t.Cleanup(func() { *limit = old })
}

// parkedSize is the size of one artefact a run parked in the default run
// directory.
func parkedSize(t *testing.T, root, runID, name string) int {
	t.Helper()
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(DefaultRunDir), runID, name))
	if err != nil {
		t.Fatalf("stat the parked %s of %s: %v", name, runID, err)
	}
	return int(fi.Size())
}

// parkedRuns lists the run directories parked under the default run directory.
func parkedRuns(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(DefaultRunDir)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// TestAParkedBundlePastTheFileCapIsReadBack is iss-2611040555379758: an
// assembly parks a bundle larger than the per-file cap an admitted file is
// read under, and the dispatch reads it back instead of failing on a bare
// size-cap message before any provider is contacted.
func TestAParkedBundlePastTheFileCapIsReadBack(t *testing.T) {
	root := fixtureRepo(t)
	// Two documents, each under the cap an admitted file is read under, that
	// together put the bundle past it.
	line := "The repository records what it does, and the record says so plainly.\n"
	body := "# A long reference\n\n" + strings.Repeat(line, (5<<19)/len(line))
	writeFile(t, root, "docs/reference/long-one.md", body)
	writeFile(t, root, "docs/reference/long-two.md", body)
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	size := parkedSize(t, root, res.RunID, BundleFileName)
	if size <= MaxFileBytes {
		t.Fatalf("the bundle is %d bytes, not past the %d-byte file cap the test is about", size, MaxFileBytes)
	}
	p, err := ReadParked(root, res.RunID)
	if err != nil {
		t.Fatalf("ReadParked of a %d-byte bundle an assembly parked: %v", size, err)
	}
	if len(p.Bundle) != size {
		t.Fatalf("ReadParked returned %d bytes of a %d-byte bundle", len(p.Bundle), size)
	}
}

// TestAParkedRunAtTheLimitRoundTripsAndOnePastIsRefusedAtAssembly holds the
// rule both ends of a parked run share: an artefact exactly at the limit its
// reader reads it under is written and read back, and one byte past it is
// refused by the assembly, naming the artefact, its size and the limit, before
// anything is written, dry run included.
func TestAParkedRunAtTheLimitRoundTripsAndOnePastIsRefusedAtAssembly(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit *int64
	}{
		{BundleFileName, &parkedLimit},
		{ManifestFileName, &manifestLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixtureRepo(t)
			first, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
			if err != nil {
				t.Fatalf("assemble: %v", err)
			}
			size := parkedSize(t, root, first.RunID, tc.name)

			holdLimit(t, tc.limit, int64(size))
			at, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
			if err != nil {
				t.Fatalf("assemble with the %s exactly at the %d-byte limit: %v", tc.name, size, err)
			}
			if got := parkedSize(t, root, at.RunID, tc.name); got != size {
				t.Fatalf("the %s is %d bytes on a second assembly and %d on the first", tc.name, got, size)
			}
			if _, err := ReadParked(root, at.RunID); err != nil {
				t.Fatalf("ReadParked with the %s exactly at the %d-byte limit: %v", tc.name, size, err)
			}

			holdLimit(t, tc.limit, int64(size-1))
			before := parkedRuns(t, root)
			for _, dry := range []bool{false, true} {
				_, err = Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: dry})
				if err == nil {
					t.Fatalf("assemble (dry run %v) with the %s one byte past the %d-byte limit was not refused",
						dry, tc.name, size-1)
				}
				for _, want := range []string{tc.name, strconv.Itoa(size) + " bytes", strconv.Itoa(size-1) + "-byte limit"} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("the refusal (dry run %v) does not name %q: %v", dry, want, err)
					}
				}
			}
			if after := parkedRuns(t, root); len(after) != len(before) {
				t.Fatalf("a refused assembly parked a run: %v, then %v", before, after)
			}
		})
	}
}
