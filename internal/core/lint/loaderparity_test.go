package lint

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// intentParityCfg is the record-lint config the parity tests run: the intent
// tree at its real repo-relative home, so ONE fixture is read by both the lint
// and the loader.
func intentParityCfg() Config {
	return Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: severityBlocker, IntentsDir: "intents"},
		},
	}
}

// TestIntentLintRefusesEveryRecordTheLoaderRefuses pins the lint↔loader contract
// for the intent store: any intent file intent.Load rejects must be a record-lint
// BLOCKER, so it can never merge green. Both halves are driven from the SAME
// fixture, so the contract cannot be asserted against a fixture the loader does
// not actually refuse.
//
// The contract has two axes, and a record has to survive both to load:
//
//   - the id VALUE (iss-2608270500198764): intent.Load fail-closes the whole
//     CORPUS on one malformed id (Validate → ^itd-[0-9]+$), while
//     intent_lifecycle validated status/kind/spec_id/superseded_by and never the
//     id itself.
//   - the field EXTRACTION (iss-2608270926031827): the loader calls
//     frontmatter.Fields on the RAW lines, which requires `---` on line 0, while
//     the lint's frontmatterFields skips a leading preamble (blank lines, an
//     attribution comment) first. A preamble-led record therefore carried a
//     perfectly good id to the lint and NO fields at all to the loader.
//
// Either way the record merged green and then bricked every intent verb,
// `abcd spec close` included, for everyone who pulled it.
func TestIntentLintRefusesEveryRecordTheLoaderRefuses(t *testing.T) {
	const good = "id: itd-999\nslug: thing\nkind: standalone\nspec_id: spc-10-thing\n"
	cases := []struct {
		name    string
		content string
		want    string // substring the blocker's message must carry
	}{
		// The VALUE axis: the delimiter is where the loader wants it, the id is not.
		{"absent-id", "---\nkind: standalone\nspec_id: spc-10-thing\n---\n# x\n", "id"},
		{"empty-id", "---\nid:\nkind: standalone\nspec_id: spc-10-thing\n---\n# x\n", "id"},
		{"null-id", "---\nid: null\nkind: standalone\nspec_id: spc-10-thing\n---\n# x\n", "id"},
		{"tilde-id", "---\nid: ~\nkind: standalone\nspec_id: spc-10-thing\n---\n# x\n", "id"},
		{"not-an-id", "---\nid: TBD\nkind: standalone\nspec_id: spc-10-thing\n---\n# x\n", "id"},
		// The EXTRACTION axis: the id is perfect, but the loader never sees it.
		{"blank-line-led", "\n---\n" + good + "---\n# x\n", "frontmatter"},
		{"comment-led", "<!-- generated -->\n---\n" + good + "---\n# x\n", "frontmatter"},
		{"multiline-comment-led", "<!--\nnote\n-->\n---\n" + good + "---\n# x\n", "frontmatter"},
	}
	const rel = ".abcd/development/intents/shipped/itd-999-thing.md"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, rel, tc.content)

			// Half one: the loader refuses the file (and with it the whole corpus).
			if _, err := intent.Load(root); err == nil {
				t.Fatalf("intent.Load accepted %s: the fixture no longer pins the loader's rule", tc.name)
			}

			// Half two: the lint must therefore refuse it too.
			fs, err := Lint(intentParityCfg(), root)
			if err != nil {
				t.Fatal(err)
			}
			var blocked bool
			for _, f := range fs {
				if f.File == rel && f.RuleID == "intent_lifecycle" &&
					f.Severity == severityBlocker && strings.Contains(f.Message, tc.want) {
					blocked = true
				}
			}
			if !blocked {
				t.Fatalf("lint passed an intent record the loader refuses (%s); findings: %+v", tc.name, fs)
			}
		})
	}
}

// TestIntentLintAcceptsWhatTheLoaderAccepts is the other direction of the same
// contract: a well-formed record the loader reads must be lint-clean, so arming
// the id check cannot turn a healthy corpus red.
func TestIntentLintAcceptsWhatTheLoaderAccepts(t *testing.T) {
	root := t.TempDir()
	const rel = ".abcd/development/intents/shipped/itd-999-thing.md"
	writeFile(t, root, rel, "---\nid: itd-999\nslug: thing\nkind: standalone\nspec_id: spc-10-thing\n---\n# x\n")

	corpus, err := intent.Load(root)
	if err != nil {
		t.Fatalf("intent.Load refused a well-formed record: %v", err)
	}
	if len(corpus.Intents) != 1 {
		t.Fatalf("expected 1 intent, got %d", len(corpus.Intents))
	}
	fs, err := Lint(intentParityCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "intent_lifecycle"); n != 0 {
		t.Fatalf("well-formed intent must be lint-clean; got %+v", fs)
	}
}

// specParityCfg is the spec-side mirror: the spec store and the intent tree at
// their real repo-relative homes, with the committed repo's own content
// exemption armed (a `status: superseded` record skips the content-AUTHORING
// checks).
func specParityCfg() Config {
	return Config{
		Roots:          []string{".abcd/development"},
		ExemptIfStatus: []string{"superseded"},
		Rules: map[string]RuleConfig{
			"spec_lifecycle": {Enabled: true, Severity: severityBlocker, SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
}

// TestSpecLintRefusesEveryRecordTheLoaderRefuses pins the same two-axis contract
// on the spec store, and mostly through the content exemption, because that is
// where the spec side failed open:
//
//   - the VALUE axis (iss-2608270500207987): a `status: superseded` line makes a
//     spec content-exempt, and spec_lifecycle skipped such a file entirely — so
//     the ONE rule that checks a spec's id and intent back-link never ran.
//     spec.Load has no exemption concept and validates both unconditionally.
//   - the EXTRACTION axis (iss-2608270926031827): a preamble before `---` hides
//     the whole block from spec.Load while the lint reads it happily. Exempt or
//     not, the record is lint-green and the store aborts.
//
// A lint-green spec of either shape aborted the WHOLE store: `abcd <spc-N>`
// dispatch, the spec mint, and every intent verb that loads specs. Being
// historical excuses a record from how it is WRITTEN, never from being
// well-formed (iss-39).
func TestSpecLintRefusesEveryRecordTheLoaderRefuses(t *testing.T) {
	cases := []struct {
		name string
		spec string
		want string // substring the blocker's message must carry
	}{
		{"bad-intent", "---\nid: spc-99\nslug: foo\nintent: itd-bogus\nstatus: superseded\n---\n# foo\n", "intent"},
		{"absent-intent", "---\nid: spc-99\nslug: foo\nstatus: superseded\n---\n# foo\n", "intent"},
		{"null-intent", "---\nid: spc-99\nslug: foo\nintent: null\nstatus: superseded\n---\n# foo\n", "intent"},
		{"bad-id", "---\nid: spc-\nslug: foo\nintent: itd-10\nstatus: superseded\n---\n# foo\n", "id"},
		{"absent-id", "---\nslug: foo\nintent: itd-10\nstatus: superseded\n---\n# foo\n", "id"},
		// The EXTRACTION axis, on both sides of the exemption: everything the lint
		// reads is well-formed, and the loader sees no fields at all.
		{"blank-line-led-exempt", "\n---\nid: spc-99\nslug: foo\nintent: itd-10\nstatus: superseded\n---\n# foo\n", "frontmatter"},
		{"comment-led-exempt", "<!-- generated -->\n---\nid: spc-99\nslug: foo\nintent: itd-10\nstatus: superseded\n---\n# foo\n", "frontmatter"},
		{"blank-line-led", "\n---\nid: spc-99\nslug: foo\nintent: itd-10\n---\n# foo\n", "frontmatter"},
		{"comment-led", "<!-- generated -->\n---\nid: spc-99\nslug: foo\nintent: itd-10\n---\n# foo\n", "frontmatter"},
	}
	const rel = ".abcd/development/specs/open/spc-99-foo.md"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md",
				"---\nid: itd-10\nslug: alpha\nkind: standalone\nspec_id: spc-99\n---\n# ok\n")
			writeFile(t, root, rel, tc.spec)

			// Half one: the loader refuses the file (and with it the whole store).
			if _, err := spec.Load(root); err == nil {
				t.Fatalf("spec.Load accepted %s: the fixture no longer pins the loader's rule", tc.name)
			}

			// Half two: the lint must therefore refuse it too, exemption or not.
			fs, err := Lint(specParityCfg(), root)
			if err != nil {
				t.Fatal(err)
			}
			var blocked bool
			for _, f := range fs {
				if f.File == rel && f.RuleID == "spec_lifecycle" &&
					f.Severity == severityBlocker && strings.Contains(f.Message, tc.want) {
					blocked = true
				}
			}
			if !blocked {
				t.Fatalf("lint passed a spec the loader refuses (%s); findings: %+v", tc.name, fs)
			}
		})
	}
}

// TestSpecLintAcceptsWhatTheLoaderAccepts is the spec side of the other
// direction: a well-formed record whose frontmatter opens on line 1 loads and
// must stay lint-clean, so arming the preamble refusal cannot turn a healthy
// store red.
func TestSpecLintAcceptsWhatTheLoaderAccepts(t *testing.T) {
	root := t.TempDir()
	const rel = ".abcd/development/specs/open/spc-99-foo.md"
	writeFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nkind: standalone\nspec_id: spc-99\n---\n# ok\n")
	writeFile(t, root, rel, "---\nid: spc-99\nslug: foo\nintent: itd-10\n---\n# foo\n")

	store, err := spec.Load(root)
	if err != nil {
		t.Fatalf("spec.Load refused a well-formed record: %v", err)
	}
	if len(store.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(store.Specs))
	}
	fs, err := Lint(specParityCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "spec_lifecycle"); n != 0 {
		t.Fatalf("well-formed spec must be lint-clean; got %+v", fs)
	}
}

// TestSpecLintKeepsTheContentExemptionForAuthoring is the guard on the other
// side of the same change: the exemption still excuses a historical spec from
// the AUTHORING checks — the forbidden `status:` field that made it exempt in
// the first place, and the bidirectional-agreement and intent-existence checks
// — so arming well-formedness cannot turn the historical record red.
func TestSpecLintKeepsTheContentExemptionForAuthoring(t *testing.T) {
	root := t.TempDir()
	const rel = ".abcd/development/specs/open/spc-99-foo.md"
	// Well-formed id and intent link, but: a forbidden status: field, an intent
	// that exists in no bucket, and hence no back-link agreement.
	writeFile(t, root, rel, "---\nid: spc-99\nslug: foo\nintent: itd-404\nstatus: superseded\n---\n# foo\n")

	fs, err := Lint(specParityCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "spec_lifecycle"); n != 0 {
		t.Fatalf("content-exempt spec must keep its authoring exemption; got %+v", fs)
	}
}
