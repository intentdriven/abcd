package loop

import (
	"strings"
	"testing"
)

// specFrontmatterWith is specWithSteps("") with extra frontmatter lines and a
// summary of the caller's.
func specFrontmatterWith(extra, summary string) string {
	return "---\nid: spc-1\nslug: alpha\nintent: itd-10\n" + extra + "---\n# alpha\n\n## Summary\n\n" + summary + "\n"
}

// checkRow returns the named row of a check result, failing the test when the
// result has none.
func checkRow(t *testing.T, res CheckResult, name string) CheckRow {
	t.Helper()
	for _, c := range res.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %s row in %+v", name, res.Checks)
	return CheckRow{}
}

// TestBuildRefusesASpecThatSaysItIsNotReady is iss-2611101318042398's own
// test: a spec whose summary says, in a sentence opening "Not READY", that it
// may not be built yet is refused at the ready check, naming the line, and so
// is a spec asking a question under a bold "Open:" label outside its
// `## Open Questions` section, and one asking an open item in that section.
// A spec saying none of them starts.
func TestBuildRefusesASpecThatSaysItIsNotReady(t *testing.T) {
	cases := []struct {
		name, spec, reason string
	}{
		{"a not-READY sentence in the summary",
			specFrontmatterWith("", "_Filed by a remainder close. Not READY: the first step below needs a ruling before it is built._"),
			"Not READY"},
		{"a bold Open label outside Open Questions",
			specFrontmatterWith("", "- The cache needs a ruling. **Open:** one cache file, or a rollup the verb only reads."),
			"Open:"},
		{"an open item under the spec's Open Questions",
			specFrontmatterWith("", "A written design record.\n\n## Open Questions\n\n- One cache file, or a rollup the verb only reads?"),
			"rollup"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), tc.spec)
			res, err := Check(repo.Root(), "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			row := checkRow(t, res, CheckReady)
			if res.OK || row.OK {
				t.Fatalf("a spec saying it is not ready must be refused: %+v", res.Checks)
			}
			if !strings.Contains(row.Detail, "spc-1") || !strings.Contains(row.Detail, tc.reason) || row.Remedy == "" {
				t.Fatalf("the refusal names the spec, what it says (%q) and a remedy: %+v", tc.reason, row)
			}
		})
	}

	t.Run("a settled spec starts", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions),
			specFrontmatterWith("", "A written design record. Its readiness is NOT READY-shaped prose only in `Not READY` code.\n\n## Open Questions\n\n- Which cache? — resolved: one file.\n\n**Open pull requests a crash left drifting** are listed."))
		res, err := Check(repo.Root(), "itd-10")
		if err != nil {
			t.Fatal(err)
		}
		if row := checkRow(t, res, CheckReady); !row.OK {
			t.Fatalf("a spec stating nothing unsettled is ready: %+v", row)
		}
	})
}

// TestBuildRefusesASpecWhosePrerequisiteHasNotShipped: a spec naming in its
// own `blocked_by` an intent that has not shipped is refused at the blocked
// check, naming that intent and the spec; once the intent ships, the check
// passes.
func TestBuildRefusesASpecWhosePrerequisiteHasNotShipped(t *testing.T) {
	const prereqPlanned = ".abcd/development/intents/planned/itd-77-prereq.md"
	const prereqShipped = ".abcd/development/intents/shipped/itd-77-prereq.md"
	const prereq = "---\nid: itd-77\nslug: prereq\nkind: standalone\n---\n# prereq\n"
	sp := specFrontmatterWith("blocked_by: [itd-77]\n", "A written design record.")

	repo := loopRepo(t, readyIntent("", settledQuestions), sp)
	repo.Write(prereqPlanned, prereq)
	repo.Commit("the prerequisite is planned")
	res, err := Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	row := checkRow(t, res, CheckBlocked)
	if res.OK || row.OK || !strings.Contains(row.Detail, "itd-77") || !strings.Contains(row.Detail, "spc-1") {
		t.Fatalf("a spec whose prerequisite has not shipped is refused, naming both: %+v", row)
	}

	repo.Remove(prereqPlanned)
	repo.Write(prereqShipped, prereq)
	repo.Commit("the prerequisite shipped")
	res, err = Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if row := checkRow(t, res, CheckBlocked); !row.OK {
		t.Fatalf("a shipped prerequisite releases the spec: %+v", row)
	}
}
