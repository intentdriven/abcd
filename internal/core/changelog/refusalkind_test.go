package changelog

import "testing"

// TestDeriveRefusalKind pins the machine-readable classification beside the
// prose reason. A caller that has to ACT on a refusal — the ship verb decides
// whether to name a record, a version, or a missing tag — must not have to
// pattern-match English to tell the three apart.
func TestDeriveRefusalKind(t *testing.T) {
	tests := []struct {
		name  string
		build func(t *testing.T) *fixtureRepo
		want  RefusalKind
	}{
		{
			name: "no anchor tag",
			build: func(t *testing.T) *fixtureRepo {
				r := newFixtureRepo(t)
				r.record(shippedDir+"itd-1-first.md", "itd-1", "additive")
				r.commit("no release has ever been tagged")
				return r
			},
			want: RefusalNoReleaseTag,
		},
		{
			name: "changelog heading ahead of the tag",
			build: func(t *testing.T) *fixtureRepo {
				r := releasedRepo(t)
				r.write("CHANGELOG.md", changelogWith("0.2.0"))
				r.commit("the ship PR merged; the tag has not landed yet")
				return r
			},
			want: RefusalReleaseInFlight,
		},
		{
			name: "an added record carries no impact",
			build: func(t *testing.T) *fixtureRepo {
				r := releasedRepo(t)
				r.write(shippedDir+"itd-2-second.md", "---\nid: itd-2\n---\n# itd-2\n")
				r.commit("ship an unlabelled record")
				return r
			},
			want: RefusalUnlabelledRecord,
		},
		{
			name: "a record move left uncommitted",
			build: func(t *testing.T) *fixtureRepo {
				r := releasedRepo(t)
				r.record(shippedDir+"itd-2-second.md", "itd-2", "fix")
				return r
			},
			want: RefusalUncommittedRecords,
		},
		{
			name: "a clean cut carries no refusal kind",
			build: func(t *testing.T) *fixtureRepo {
				r := releasedRepo(t)
				r.record(shippedDir+"itd-2-second.md", "itd-2", "fix")
				r.commit("ship a fix")
				return r
			},
			want: RefusalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := Derive(tt.build(t).root)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if d.RefusalKind != tt.want {
				t.Errorf("RefusalKind = %q (reason %q), want %q", d.RefusalKind, d.RefusalReason, tt.want)
			}
			if (d.RefusalKind != RefusalNone) != d.Refused {
				t.Errorf("RefusalKind %q and Refused %v disagree", d.RefusalKind, d.Refused)
			}
		})
	}
}
