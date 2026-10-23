package lint_test

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// `make precheck` is the minutes-long pass a lane runs before it queues a
// preflight (iss-2610230016523625): preflight's gates and the audit tests that
// read the committed tree, without the full unit and race stages. On 2026-10-22 two lanes
// each lost a preflight slot, queued for over an hour, to a sentence in the
// build brief that TestSurfaceChapterProseStatesNoShape refuses; the precheck
// is what would have said so in minutes, before the queue. These tests hold it
// to preflight and to the audit tests it claims to run.

// precheckPrereqs reads the `precheck:` recipe's prerequisites, as
// preflightPrereqs reads preflight's.
func precheckPrereqs(t *testing.T, root string) []string {
	t.Helper()
	for _, line := range strings.Split(readRepoFile(t, root, "Makefile"), "\n") {
		rest, ok := strings.CutPrefix(line, "precheck:")
		if !ok || strings.HasPrefix(strings.TrimSpace(rest), "export ") {
			continue
		}
		return strings.Fields(rest)
	}
	t.Fatal("Makefile declares no `precheck:` recipe")
	return nil
}

// TestPrecheckRunsEveryPreflightGateButTheLoadCheck: a gate preflight gains is
// a gate precheck runs, so a lane's precheck never passes a tree a preflight
// will refuse at a gate. The load check alone is left out: it reads the
// machine, not the tree, and warns without failing.
func TestPrecheckRunsEveryPreflightGateButTheLoadCheck(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	var want []string
	for _, g := range preflightPrereqs(t, root) {
		if g != "load-check" {
			want = append(want, g)
		}
	}
	got := precheckPrereqs(t, root)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("precheck's prerequisites = %v; want preflight's without load-check, %v", got, want)
	}
}

// TestPrecheckAuditPatternCatchesTheLateFailures: the audit tests precheck runs
// in the command package are chosen by what their names say they read (a
// surface, a page, a brief, a committed baseline), so a new audit test is
// picked up without a list to keep. The pattern must still match the tests
// that have failed preflights late.
func TestPrecheckAuditPatternCatchesTheLateFailures(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	m := regexp.MustCompile(`(?m)^PRECHECK_AUDIT_TESTS := (.+)$`).FindStringSubmatch(readRepoFile(t, root, "Makefile"))
	if m == nil {
		t.Fatal("Makefile declares no PRECHECK_AUDIT_TESTS")
	}
	re, err := regexp.Compile(m[1])
	if err != nil {
		t.Fatalf("PRECHECK_AUDIT_TESTS is not a pattern go test -run takes: %v", err)
	}
	for _, name := range []string{
		"TestSurfaceChapterProseStatesNoShape",        // failed two lanes' preflights on 2026-10-22
		"TestReferenceMatchesCommittedPage",           // the generated CLI reference
		"TestSurfaceSnapshotMatchesCommittedBaseline", // the release surface baseline
		"TestSelfProvisionParagraphsMatchTheShippedSets",
		"TestGuardApproveDocsQuoteTheBashCall", // quotes sentences in the guard, build and implement pages
		"TestBakedIdentityInSync",              // the identity block against its generated copy
		"TestInstallSurfacesAgree",             // the site package's install pages
	} {
		if !re.MatchString(name) {
			t.Errorf("PRECHECK_AUDIT_TESTS does not match %s", name)
		}
	}
}

// TestPrecheckRunsTheAuditTests: the recipe itself runs the audit packages in
// full and the pattern-chosen audit tests of the command and site packages, so
// deleting a test line fails here rather than leaving a precheck that passes
// on gates alone.
func TestPrecheckRunsTheAuditTests(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	recipe, ok := makeRecipe(readRepoFile(t, root, "Makefile"), "precheck")
	if !ok {
		t.Fatal("Makefile declares no `precheck:` recipe")
	}
	for _, want := range []string{
		"go build ./...",
		"go vet ./...",
		"go test -timeout 20m ./internal/core/lint ./internal/core/launch/scaffold ./internal/core/mdrecord " +
			"./internal/core/issueschema ./internal/core/guard ./internal/core/glossary ./internal/core/reading",
		"go test -timeout 20m -run '$(PRECHECK_AUDIT_TESTS)' ./internal/surface/cli ./internal/core/site",
	} {
		if !strings.Contains(recipe, want) {
			t.Errorf("the precheck recipe does not run %q:\n%s", want, recipe)
		}
	}
}

// TestPrecheckRunsOnTheDeclaredToolchain: precheck judges with CI's toolchain,
// as preflight does (iss-2609261850045839), or it can pass what preflight fails.
// The export is matched as a whole line, so a commented-out copy does not pass.
func TestPrecheckRunsOnTheDeclaredToolchain(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if !regexp.MustCompile(`(?m)^precheck: export GOTOOLCHAIN := go\$\(GO_TOOLCHAIN_VERSION\)$`).MatchString(readRepoFile(t, root, "Makefile")) {
		t.Fatal("precheck does not run on the toolchain go.mod declares")
	}
}

// TestAgentsTellsALaneToPrecheckBeforeAPreflight: the convention lives where
// every agent reads it, not only in one session's memory.
func TestAgentsTellsALaneToPrecheckBeforeAPreflight(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if !strings.Contains(readRepoFile(t, root, "AGENTS.md"), "make precheck") {
		t.Fatal("AGENTS.md does not tell a lane to run make precheck before queuing a preflight")
	}
}
