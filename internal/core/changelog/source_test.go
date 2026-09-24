package changelog

import "testing"

// TestRecordSourceMaterial pins the title/summary a record contributes to the
// cut. The composer writes prose FROM these two fields, so what they extract is
// a contract, not a convenience: an empty title would leave a changelog line
// with nothing to name the record by.
//
// The two record families are deliberately shaped differently in this repo —
// intents open with an `# ` heading, issues carry no heading at all and lead with
// a body paragraph — so the table walks both, plus the degenerate cases.
func TestRecordSourceMaterial(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		body        string
		wantTitle   string
		wantSummary string
	}{
		{
			name: "intent: h1 heading and the first prose paragraph",
			file: "itd-80-intent-lifecycle-automation.md",
			body: "---\nid: itd-80\nslug: intent-lifecycle-automation\nimpact: additive\n---\n\n" +
				"# An Intent Ships Itself\n\n## Press Release\n\n" +
				"> **abcd ships intent-lifecycle automation:** an intent moves\n> from planned to shipped.\n\nMore prose.\n",
			wantTitle:   "An Intent Ships Itself",
			wantSummary: "**abcd ships intent-lifecycle automation:** an intent moves from planned to shipped.",
		},
		{
			name: "issue: no heading, so the slug titles it and the body summarises",
			file: "iss-1-launch-phase-ownership.md",
			body: "---\nid: \"iss-1\"\nslug: \"launch-phase-ownership\"\nimpact: internal\n---\n\n" +
				"launch phase ownership contradicts across the record.\n",
			wantTitle:   "launch-phase-ownership",
			wantSummary: "launch phase ownership contradicts across the record.",
		},
		{
			name:        "no heading and no slug falls back to the id",
			file:        "itd-7-x.md",
			body:        "---\nid: itd-7\nimpact: fix\n---\n\nsomething happened.\n",
			wantTitle:   "itd-7",
			wantSummary: "something happened.",
		},
		{
			name:        "frontmatter only leaves the summary empty, never the title",
			file:        "itd-8-x.md",
			body:        "---\nid: itd-8\nimpact: fix\n---\n",
			wantTitle:   "itd-8",
			wantSummary: "",
		},
		{
			name:        "a heading deeper than h1 is not a title",
			file:        "itd-9-x.md",
			body:        "---\nid: itd-9\nimpact: fix\n---\n\n## Section\n\nthe prose.\n",
			wantTitle:   "itd-9",
			wantSummary: "the prose.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFixtureRepo(t)
			r.commit("empty base")
			r.git("tag", "v0.1.0")
			rel := shippedDir + tt.file
			r.write(rel, tt.body)
			r.commit("ship it")

			set, err := ShippedSince(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("ShippedSince: %v", err)
			}
			if len(set.Added) != 1 {
				t.Fatalf("added = %d records, want 1", len(set.Added))
			}
			got := set.Added[0]
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.Summary != tt.wantSummary {
				t.Errorf("Summary = %q, want %q", got.Summary, tt.wantSummary)
			}
		})
	}
}

// TestRecordSummaryIsCapped pins the bound on the source material. The cut is
// rendered to a terminal and handed to a host as JSON, so one pathological record
// must not be able to blow either up.
func TestRecordSummaryIsCapped(t *testing.T) {
	long := make([]byte, maxSummaryRunes*3)
	for i := range long {
		long[i] = 'a'
	}
	r := newFixtureRepo(t)
	r.commit("empty base")
	r.git("tag", "v0.1.0")
	r.write(shippedDir+"itd-1-long.md", "---\nid: itd-1\nimpact: fix\n---\n\n"+string(long)+"\n")
	r.commit("ship it")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := len([]rune(set.Added[0].Summary)); got > maxSummaryRunes {
		t.Errorf("summary is %d runes, want at most %d", got, maxSummaryRunes)
	}
}

// TestPressReleaseSectionExtractsTheWholeSection pins the source a release page's
// quotes are verified against: the body of the intent's `## Press Release`
// section, every paragraph of it, and nothing outside it. A quote lifted from
// the record's other sections must fail verification, so the section's edges
// are the contract.
func TestPressReleaseSectionExtractsTheWholeSection(t *testing.T) {
	body := "---\nid: itd-80\nimpact: additive\n---\n\n# An Intent Ships Itself\n\n" +
		"## Press Release\n\n> The first paragraph,\n> wrapped.\n>\n> \"A quote,\" said Nia, a facilitator.\n\n" +
		"## Why This Matters\n\nNot part of the press release.\n"
	r := newFixtureRepo(t)
	r.commit("empty base")
	r.git("tag", "v0.1.0")
	r.write(shippedDir+"itd-80-x.md", body)
	r.write(shippedDir+"itd-81-y.md", "---\nid: itd-81\nimpact: additive\n---\n\n# No section\n\nprose.\n")
	r.write(shippedDir+"itd-82-z.md", "---\nid: itd-82\nimpact: additive\n---\n\n# Lower case\n\n## Press release\n\nthe last section.\n")
	r.commit("ship them")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	got := map[string]string{}
	for _, rec := range set.Added {
		got[rec.ID] = rec.PressRelease
	}
	want := "> The first paragraph,\n> wrapped.\n>\n> \"A quote,\" said Nia, a facilitator."
	if got["itd-80"] != want {
		t.Errorf("itd-80 PressRelease = %q, want %q", got["itd-80"], want)
	}
	if got["itd-81"] != "" {
		t.Errorf("itd-81 has no press release section, got %q", got["itd-81"])
	}
	if got["itd-82"] != "the last section." {
		t.Errorf("itd-82 PressRelease = %q, want the section running to the end of the file", got["itd-82"])
	}
}
