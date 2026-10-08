package lifeboat

import (
	"fmt"
	"strings"
	"testing"
)

// authoredBriefProse is body prose past nativeGroundedBodyBytes, so a brief file
// carrying it reads as authored rather than as a stub.
const authoredBriefProse = "This section is authored by the project, not scaffolded. It states " +
	"what the section asks in the project's own words, with the reasons behind " +
	"each statement, so a rescuer can rely on it without reconstructing it from " +
	"history or from the shape of the code.\n"

// adviceClause is the part of a partial's reason that says what would ground the
// section: everything after its last "; ".
func adviceClause(reason string) string {
	if i := strings.LastIndex(reason, "; "); i >= 0 {
		return reason[i+2:]
	}
	return reason
}

// TestPartialAdviceNamesWhatGrounds holds a partial's reason to its own advice
// (iss-2610040758387938): what it says would ground the section must genuinely
// ground it, never an input the same adapter rates partial. For each case the
// advice must name the grounding input and none of the partial-ceiling inputs,
// and — where the grounding input is a file — writing that file and re-probing
// must ground the section.
func TestPartialAdviceNamesWhatGrounds(t *testing.T) {
	cases := []struct {
		name    string
		section Section
		source  func(t *testing.T) Source
		fixture func(t *testing.T) string
		// grounds is what the advice must name; ceiling are inputs the adapter
		// rates partial, which the advice must not offer as grounding.
		grounds string
		ceiling []string
		// follow is the file that, written as authored prose, grounds the
		// section; "" when the grounding input is not a file.
		follow string
	}{
		{
			name:    "build-sequence without tags",
			section: "delivery/build-sequence",
			source:  func(t *testing.T) Source { return gitSourceFor(t, "delivery/build-sequence") },
			fixture: func(t *testing.T) string {
				return gitFixture(t, []fixtureCommit{
					{"a.txt", "a\n", "one"}, {"b.txt", "b\n", "two"}, {"c.txt", "c\n", "three"},
				})
			},
			grounds: nativeSectionBriefFile("delivery/build-sequence"),
			ceiling: []string{"tag"},
			follow:  nativeSectionBriefFile("delivery/build-sequence"),
		},
		{
			name:    "scope without a features section",
			section: "product/scope",
			source:  func(t *testing.T) Source { return convSourceForSection(t, "product/scope") },
			fixture: func(t *testing.T) string {
				dir := t.TempDir()
				writeTree(t, dir, map[string]string{"README.md": "# Thing\n\nA thing that does a thing.\n"})
				return dir
			},
			grounds: nativeSectionBriefFile("product/scope"),
			ceiling: []string{"features section", "scope statement"},
			follow:  nativeSectionBriefFile("product/scope"),
		},
		{
			name:    "surfaces without a usage section",
			section: "surfaces",
			source:  func(t *testing.T) Source { return convSourceForSection(t, "surfaces") },
			fixture: func(t *testing.T) string {
				dir := t.TempDir()
				writeTree(t, dir, map[string]string{"README.md": "# Thing\n\nA thing that does a thing.\n"})
				return dir
			},
			grounds: nativeSectionBriefFile("surfaces"),
			ceiling: []string{"usage section", "command/API reference"},
			follow:  nativeSectionBriefFile("surfaces"),
		},
		{
			name:    "naming from a glossary fallback",
			section: "constraints/naming",
			source:  func(t *testing.T) Source { return convSourceForSection(t, "constraints/naming") },
			fixture: func(t *testing.T) string {
				dir := t.TempDir()
				writeTree(t, dir, map[string]string{"GLOSSARY.md": "# Glossary\n\n**thing** — a thing.\n"})
				return dir
			},
			grounds: nativeGlossaryDir,
			ceiling: []string{"naming document"},
			follow:  nativeGlossaryDir + "/terms.md",
		},
		{
			name:    "graveyard from deletions alone",
			section: "graveyard",
			source:  func(t *testing.T) Source { return gitSourceFor(t, "graveyard") },
			fixture: gitFixtureDeletionOnly,
			grounds: "revert",
			// No adapter reads a written record of what was abandoned.
			ceiling: []string{"written record"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.fixture(t)
			ctx, err := newSourceContext(dir)
			if err != nil {
				t.Fatal(err)
			}
			ev := tc.source(t).Probe(ctx)
			ctx.Close()
			if ev.Status != StatusPartial {
				t.Fatalf("%s = %s, want partial (the branch under test)", tc.section, ev.Status)
			}
			advice := adviceClause(ev.Reason)
			if !strings.Contains(advice, tc.grounds) {
				t.Errorf("advice %q does not name %q, the input that grounds %s", advice, tc.grounds, tc.section)
			}
			for _, c := range tc.ceiling {
				if strings.Contains(advice, c) {
					t.Errorf("advice %q offers %q as grounding, but the adapter rates it partial", advice, c)
				}
			}
			if tc.follow == "" {
				return
			}
			writeTree(t, dir, map[string]string{tc.follow: "# Section\n\n" + authoredBriefProse})
			cov, err := Probe(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range cov.Sections {
				if s.Name == tc.section && s.Status != StatusGrounded {
					t.Errorf("after following the advice (%s), %s = %s, want grounded", tc.follow, tc.section, s.Status)
				}
			}
		})
	}
}

// gitSourceFor returns the Tier-0 git source for section.
func gitSourceFor(t *testing.T, section Section) Source {
	t.Helper()
	for _, s := range allSources() {
		if s.Section() == section && s.Tier() == TierGit {
			return s
		}
	}
	t.Fatalf("no git source for section %s", section)
	return nil
}

// TestInternalsReasonCountsEveryPackage: the internals reason reports the true
// package count, not the count left after the citation cap trims what is cited,
// so the row's reason and its "counted but not cited" note agree.
func TestInternalsReasonCountsEveryPackage(t *testing.T) {
	const n = maxLayoutCitations + 10
	files := map[string]string{}
	for i := 0; i < n; i++ {
		files[fmt.Sprintf("internal/pkg%02d/x.go", i)] = "package x\n"
	}
	dir := t.TempDir()
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	ev := convSourceForSection(t, "internals").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("internals = %s, want partial", ev.Status)
	}
	if want := fmt.Sprintf("(%d package(s))", n); !strings.Contains(ev.Reason, want) {
		t.Errorf("reason %q does not report all %d packages", ev.Reason, n)
	}
	if !containsSource(ev.Sources, fmt.Sprintf("%d further package(s) counted but not cited (citation cap %d)", n-maxLayoutCitations, maxLayoutCitations)) {
		t.Errorf("sources %v do not carry the counted-but-not-cited note", ev.Sources)
	}
}
