package repolint_test

import (
	"sort"
	"strings"
	"testing"
)

// layoutFindings returns the files three-tier-layout names, sorted.
func layoutFindings(t *testing.T, b *repoBuilder) []string {
	t.Helper()
	var got []string
	for _, f := range b.run().Findings {
		if f.RuleID == "three-tier-layout" {
			got = append(got, f.File)
		}
	}
	sort.Strings(got)
	return got
}

// TestRule_LocalArtifactResidualEvasions — iss-173. The placement check matched
// the three names exactly, and only directly under a committed tier, so three
// shapes one step off the modelled incident passed clean: a handover file one
// directory below a tier root, a local-tier artefact at the .abcd/ root itself,
// and a name spelled in another case on a case-sensitive filesystem. NEXT.md is
// a name the local tier owns outright, so it is flagged at any depth in a
// committed tier; scratch/ and logs/ are ordinary words a durable record may
// legitimately nest (a study's logs/), so they are flagged where the local
// tier would put them — directly under a tier and at the .abcd/ root — and in
// any case.
func TestRule_LocalArtifactResidualEvasions(t *testing.T) {
	cases := map[string]struct {
		build func(b *repoBuilder) *repoBuilder
		want  []string
	}{
		"nested handover": {
			func(b *repoBuilder) *repoBuilder { return b.file(".abcd/work/notes/NEXT.md", "x\n") },
			[]string{".abcd/work/notes/NEXT.md"},
		},
		"deeply nested handover in the durable record": {
			func(b *repoBuilder) *repoBuilder {
				return b.file(".abcd/development/research/study/NEXT.md", "x\n")
			},
			[]string{".abcd/development/research/study/NEXT.md"},
		},
		"handover at the .abcd root": {
			func(b *repoBuilder) *repoBuilder { return b.file(".abcd/NEXT.md", "x\n") },
			[]string{".abcd/NEXT.md"},
		},
		"scratch and logs at the .abcd root": {
			func(b *repoBuilder) *repoBuilder { return b.dir(".abcd/scratch").dir(".abcd/logs") },
			[]string{".abcd/logs", ".abcd/scratch"},
		},
		"lower-case handover": {
			func(b *repoBuilder) *repoBuilder { return b.file(".abcd/work/next.md", "x\n") },
			[]string{".abcd/work/next.md"},
		},
		"capitalised scratch": {
			func(b *repoBuilder) *repoBuilder { return b.dir(".abcd/development/Scratch") },
			[]string{".abcd/development/Scratch"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := tc.build(newFixtureRepo(t).conforming()).commit()
			got := layoutFindings(t, b)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("three-tier-layout names %v, want exactly %v", got, tc.want)
			}
		})
	}
}

// TestRule_LocalArtifactWideningKeepsLegitimateTierContent: the widening must
// not start flagging what a committed tier legitimately holds — a study's own
// logs/ or scratch/ nested in the durable record, a file whose name merely
// begins like a handover, and the local tier's own contents.
func TestRule_LocalArtifactWideningKeepsLegitimateTierContent(t *testing.T) {
	b := newFixtureRepo(t).conforming().
		dir(".abcd/development/research/study/logs").
		file(".abcd/development/research/study/logs/run-1.txt", "x\n").
		dir(".abcd/work/reviews/scratch").
		file(".abcd/work/NEXT-steps.md", "x\n").
		file(".abcd/work/reviews/next.md.bak", "x\n").
		dir(".abcd/.work.local/scratch").
		dir(".abcd/.work.local/logs").
		commit()
	if got := layoutFindings(t, b); len(got) != 0 {
		t.Errorf("legitimate tier content was flagged: %v", got)
	}
}
