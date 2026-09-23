package release

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// irisQuote is the persona quote itd-73's press release carries, exactly as the
// record has it. A quote on the page is verified against these bytes.
const irisQuote = `"I stopped typing version numbers," said Iris, a product thinker. "The release tells me."`

// niaQuote is the second persona quote itd-73's press release carries: the
// source a truncated quote is cut from.
const niaQuote = `"We never did lose a record," said Nia, a facilitator.`

// pageRepo is a ready cut whose press-release set is known: itd-73 and itd-74
// (additive, shipped since the tag). Around them sit every record the set must
// exclude: a fixed issue (iss-51), an internal intent (itd-97), a superseded
// intent that left shipped/ (itd-40), and two planned intents, one carrying
// `target_release` (itd-90, itd-91).
func pageRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := releasedRepo(t)
	r.Write(shippedDir+"itd-40-superseded.md", "---\nid: itd-40\nimpact: additive\n---\n# The Old Shape\n")
	r.Commit("an intent shipped at the base")
	r.Git("tag", "-d", "v0.4.0")
	r.Git("tag", "v0.4.0")

	r.Write("CHANGELOG.md", baseChangelog)
	r.Remove(shippedDir + "itd-40-superseded.md")
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\n## Press Release\n\n"+
			"> The release version is derived from what shipped,\n> not typed by hand.\n>\n> "+
			strings.Replace(irisQuote, " said", "\n> said", 1)+"\n>\n> "+niaQuote+"\n\n"+
			"## Why This Matters\n\n\"A sentence outside the press release,\" said Iris, a product thinker.\n")
	r.Write(shippedDir+"itd-74-release-page.md",
		"---\nid: itd-74\nimpact: additive\n---\n\n# Every Release Has A Page\n\n## Press Release\n\n> A page per release.\n")
	r.Record(shippedDir+"itd-97-plumbing.md", "itd-97", "internal")
	r.Record(resolvedDir+"iss-51-crash.md", "iss-51", "fix")
	r.Write(plannedDir+"itd-90-later.md", "---\nid: itd-90\nimpact: additive\n---\n# Later\n")
	r.Write(plannedDir+"itd-91-targeted.md",
		"---\nid: itd-91\nimpact: additive\ntarget_release: v0.5.0\n---\n# Targeted\n")
	r.Commit("ship two intents, a fix, plumbing, and a supersession")
	return r
}

// pageEntries satisfies the changelog bijection for pageRepo.
func pageEntries() []ChangelogEntry {
	return []ChangelogEntry{
		{Section: SectionAdded, Records: []string{"itd-73", "itd-40"}, Text: "A version is a fact; it replaces the old shape."},
		{Section: SectionAdded, Records: []string{"itd-74"}, Text: "Every release has a page."},
		{Section: SectionFixed, Records: []string{"iss-51"}, Text: "The crash is gone."},
	}
}

// goodPage satisfies the page bijection for pageRepo.
func goodPage() *PressReleasePayload {
	return &PressReleasePayload{
		Headlines: []Headline{{Records: []string{"itd-73"}, Text: "The version of a release is now a fact read from what shipped."}},
		Listed:    []string{"itd-74"},
		Quotes:    []Quote{{Record: "itd-73", Text: irisQuote, Attribution: "Iris, a product thinker"}},
	}
}

func marshalPage(t *testing.T, nextTag string, entries []ChangelogEntry, page *PressReleasePayload) []byte {
	t.Helper()
	data, err := json.Marshal(ChangelogPayload{
		SchemaVersion: ChangelogSchemaVersion,
		PromptVersion: "0.4.0",
		NextTag:       nextTag,
		Entries:       entries,
		PressRelease:  page,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return data
}

// refusalOf asserts err is a payload refusal and returns it.
func refusalOf(t *testing.T, err error) *PayloadRefusal {
	t.Helper()
	var refusal *PayloadRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want a *PayloadRefusal", err)
	}
	return refusal
}

// codesOf lists a refusal's reason codes.
func codesOf(r *PayloadRefusal) []string {
	var out []string
	for _, reason := range r.Reasons {
		out = append(out, string(reason.Code))
	}
	return out
}

// reasonWith returns the first reason carrying code, and whether one did.
func reasonWith(r *PayloadRefusal, code ReasonCode) (Reason, bool) {
	for _, reason := range r.Reasons {
		if reason.Code == code {
			return reason, true
		}
	}
	return Reason{}, false
}

func readPage(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, PageFile))
	if err != nil {
		t.Fatalf("reading %s: %v", PageFile, err)
	}
	return string(data)
}

// TestPageHeadingNamesItsVersion is the happy path and the page's shape: the
// heading names the release it describes, the headline carries its citation, the
// quote sits under the headline that tells its intent, the rest is listed by
// title, and the last line points at the changelog.
func TestPageHeadingNamesItsVersion(t *testing.T) {
	r := pageRepo(t)
	res, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if !res.Written || !res.Page.Written {
		t.Fatalf("written=%v page=%+v, want both written; refusals=%v", res.Written, res.Page, refusalKinds(res.Cut))
	}
	want := "# Release 0.4.1 (2026-07-21)\n\n" +
		"The version of a release is now a fact read from what shipped. (itd-73)\n\n" +
		"> " + irisQuote + " (itd-73)\n\n" +
		"Also in this release:\n\n" +
		"- Every Release Has A Page (itd-74)\n\n" +
		"The line-by-line record of this release is its section in CHANGELOG.md.\n"
	if got := readPage(t, r.Root()); got != want {
		t.Errorf("RELEASE.md =\n%s\nwant\n%s", got, want)
	}
	if res.Page.Heading != "# Release 0.4.1 (2026-07-21)" || res.Page.Path != PageFile {
		t.Errorf("page result = %+v", res.Page)
	}
	if res.Page.Headlines != 1 || res.Page.Listed != 1 || res.Page.Quotes != 1 || res.Page.Archived != "" {
		t.Errorf("page counts = %+v, want 1 headline, 1 listed, 1 quote, nothing archived on a first cut", res.Page)
	}
	if v, _, ok := parsePageHeading(want); !ok || v != "0.4.1" {
		t.Errorf("the rendered heading does not parse back to its version: %q ok=%v", v, ok)
	}
}

// TestArchiveIsNamedFromTheOutgoingHeading: the page a cut replaces moves to the
// archive under the version ITS heading names, byte for byte.
func TestArchiveIsNamedFromTheOutgoingHeading(t *testing.T) {
	r := pageRepo(t)
	outgoing := "# Release 0.3.9 (2026-06-01)\n\nAn earlier release. (itd-12)\n"
	r.Write(PageFile, outgoing)
	r.Commit("the previous release page")

	res, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	wantArchive := ArchiveDir + "/0.3.9.md"
	if res.Page.Archived != wantArchive {
		t.Errorf("Archived = %q, want %q", res.Page.Archived, wantArchive)
	}
	data, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(wantArchive)))
	if err != nil {
		t.Fatalf("reading the archive: %v", err)
	}
	if string(data) != outgoing {
		t.Errorf("archive = %q, want the outgoing page byte for byte", data)
	}
	if !strings.HasPrefix(readPage(t, r.Root()), "# Release 0.4.1 (2026-07-21)\n") {
		t.Errorf("RELEASE.md was not replaced:\n%s", readPage(t, r.Root()))
	}
}

// TestPageBijection: every intent in the set is cited once, nothing outside it,
// and each fault is named with its cause.
func TestPageBijection(t *testing.T) {
	tests := []struct {
		name       string
		page       func(p *PressReleasePayload)
		wantCode   ReasonCode
		wantDetail string
	}{
		{"an intent in the set is omitted", func(p *PressReleasePayload) { p.Listed = nil }, ReasonMissing, "itd-74"},
		{"an id not in the cut", func(p *PressReleasePayload) { p.Listed = append(p.Listed, "itd-999") }, ReasonOutsideSet, "not in this cut"},
		{"an issue", func(p *PressReleasePayload) { p.Listed = append(p.Listed, "iss-51") }, ReasonOutsideSet, "issue"},
		{"an internal intent", func(p *PressReleasePayload) { p.Listed = append(p.Listed, "itd-97") }, ReasonOutsideSet, "internal"},
		{"a removed intent", func(p *PressReleasePayload) { p.Listed = append(p.Listed, "itd-40") }, ReasonOutsideSet, "removed"},
		{"an intent cited twice", func(p *PressReleasePayload) { p.Listed = append(p.Listed, "itd-73") }, ReasonDuplicateCitation, "itd-73"},
		{"no headline", func(p *PressReleasePayload) {
			p.Headlines = nil
			p.Quotes = nil
			p.Listed = []string{"itd-73", "itd-74"}
		}, ReasonNoHeadline, "headline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := pageRepo(t)
			before := treeDigest(t, r.Root())
			page := goodPage()
			tt.page(page)
			res, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), page), cutAt)
			refusal := refusalOf(t, err)
			reason, ok := reasonWith(refusal, tt.wantCode)
			if !ok {
				t.Fatalf("codes = %v, want %s", codesOf(refusal), tt.wantCode)
			}
			if !strings.Contains(reason.Detail, tt.wantDetail) {
				t.Errorf("detail %q does not say %q", reason.Detail, tt.wantDetail)
			}
			if res.Written || res.Page.Written {
				t.Error("a refused page reported a write")
			}
			if after := treeDigest(t, r.Root()); after != before {
				t.Error("a refused page changed the working tree")
			}
		})
	}
}

// TestPageRefusesAPlannedIntent and TestPageRefusesATargetReleaseIntent are the
// structural half of the no-forecast rule: a planned intent is not in the cut,
// with or without a target release, so citing it is outside the set.
func TestPageRefusesAPlannedIntent(t *testing.T) {
	assertOutsideSet(t, "itd-90")
}

func TestPageRefusesATargetReleaseIntent(t *testing.T) {
	assertOutsideSet(t, "itd-91")
}

func assertOutsideSet(t *testing.T, id string) {
	t.Helper()
	r := pageRepo(t)
	page := goodPage()
	page.Headlines = append(page.Headlines, Headline{Records: []string{id}, Text: "Coming in the next release."})
	_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), page), cutAt)
	reason, ok := reasonWith(refusalOf(t, err), ReasonOutsideSet)
	if !ok || !strings.Contains(reason.Detail, id) || !strings.Contains(reason.Detail, "not in this cut") {
		t.Errorf("reason = %+v, want outside-set naming %s as not in this cut", reason, id)
	}
}

// TestPageForAnEmptySetIsRefused: a fixes-only cut has no page, so a payload
// carrying one is refused rather than written.
func TestPageForAnEmptySetIsRefused(t *testing.T) {
	r := shippableRepo(t)
	r.Remove(shippedDir + "itd-73-derived-versioning.md")
	r.Commit("the intent did not ship after all")
	entries := []ChangelogEntry{{Section: SectionFixed, Records: []string{"iss-51"}, Text: "the crash is gone."}}
	page := &PressReleasePayload{Headlines: []Headline{{Records: []string{"iss-51"}, Text: "A fix."}}}
	_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", entries, page), cutAt)
	if _, ok := reasonWith(refusalOf(t, err), ReasonPageForEmptySet); !ok {
		t.Errorf("codes = %v, want page-for-empty-set", codesOf(refusalOf(t, err)))
	}
}

// TestPagePayloadRefusals walks the structural refusals, one row per code. Each
// refuses the payload whole and leaves the tree as it was.
func TestPagePayloadRefusals(t *testing.T) {
	sessionURL := "https://agent-host.dev/code/session_" + testsecret.Synthetic(62, 22)
	tests := []struct {
		name     string
		raw      func(t *testing.T) []byte
		wantCode ReasonCode
	}{
		{"an unknown field inside the page", func(t *testing.T) []byte {
			raw := string(marshalPage(t, "v0.4.1", pageEntries(), goodPage()))
			return []byte(strings.Replace(raw, `"listed":`, `"surprise":true,"listed":`, 1))
		}, ReasonUnknownField},
		{"an oversize headline", func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = strings.Repeat("a", maxEntryProseBytes+1)
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, ReasonTextOversize},
		{"a malformed id", func(t *testing.T) []byte {
			p := goodPage()
			p.Listed = append(p.Listed, "../../etc/passwd")
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, ReasonMalformedID},
		{"an id outside the set", func(t *testing.T) []byte {
			p := goodPage()
			p.Listed = append(p.Listed, "itd-999")
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, ReasonOutsideSet},
		{"a heading", func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = "  # A forged heading"
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, ReasonHeading},
		{"a fence", func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = "Text then ~~~ a fence."
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, ReasonFence},
		{"an outbound-policy leak", func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = "Composed at " + sessionURL
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, ReasonOutboundPolicy},
		{"an outbound-policy leak in the changelog", func(t *testing.T) []byte {
			e := pageEntries()
			e[2].Text = "Fixed, see " + sessionURL
			return marshalPage(t, "v0.4.1", e, goodPage())
		}, ReasonOutboundPolicy},
		{"a stale cut", func(t *testing.T) []byte {
			return marshalPage(t, "v9.9.9", pageEntries(), goodPage())
		}, ReasonStaleCut},
		{"a schema 1 payload", func(t *testing.T) []byte {
			raw := string(marshalPage(t, "v0.4.1", pageEntries(), nil))
			return []byte(strings.Replace(raw, `"schema_version":2`, `"schema_version":1`, 1))
		}, ReasonSchemaVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := pageRepo(t)
			before := treeDigest(t, r.Root())
			res, err := Ingest(r.Root(), liveSurface(), tt.raw(t), cutAt)
			refusal := refusalOf(t, err)
			if _, ok := reasonWith(refusal, tt.wantCode); !ok {
				t.Errorf("codes = %v, want %s", codesOf(refusal), tt.wantCode)
			}
			if res.Written {
				t.Error("a refused payload reported a write")
			}
			if after := treeDigest(t, r.Root()); after != before {
				t.Error("a refused payload changed the working tree")
			}
			if strings.Contains(refusal.Error(), sessionURL) {
				t.Error("the refusal echoes the leaked session URL")
			}
		})
	}
}

// TestPageRefusalCollectsEveryReason: after decoding, the validator reports the
// whole list at once, so the composer sees every fault in one attempt.
func TestPageRefusalCollectsEveryReason(t *testing.T) {
	r := pageRepo(t)
	p := goodPage()
	p.Headlines[0].Text = "# heading"
	p.Listed = []string{"itd-999"}
	p.Quotes[0].Text = `"I never said this," said Iris, a product thinker.`
	e := pageEntries()
	e[2].Section = "Security"
	_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", e, p), cutAt)
	refusal := refusalOf(t, err)
	for _, want := range []ReasonCode{ReasonHeading, ReasonOutsideSet, ReasonMissing, ReasonQuoteNotVerbatim, ReasonSectionNotWritable} {
		if _, ok := reasonWith(refusal, want); !ok {
			t.Errorf("codes = %v, want %s among them", codesOf(refusal), want)
		}
	}
	for _, reason := range refusal.Reasons {
		if reason.At == "" {
			t.Errorf("reason %+v names no payload path", reason)
		}
	}
}

// TestQuoteMustBeVerbatim: a quote matches, word for word and with its
// attribution, a quote inside the `## Press Release` of an intent the page
// tells.
func TestQuoteMustBeVerbatim(t *testing.T) {
	tests := []struct {
		name     string
		quote    Quote
		wantCode ReasonCode
	}{
		{"a changed word", Quote{Record: "itd-73", Text: strings.Replace(irisQuote, "typing", "writing", 1), Attribution: "Iris, a product thinker"}, ReasonQuoteNotVerbatim},
		{"a changed attribution", Quote{Record: "itd-73", Text: irisQuote, Attribution: "Iris, the product thinker"}, ReasonQuoteNotVerbatim},
		{"an attribution outside the text", Quote{Record: "itd-73", Text: `"I stopped typing version numbers,"`, Attribution: "Iris"}, ReasonQuoteNotVerbatim},
		{"no attribution", Quote{Record: "itd-73", Text: irisQuote}, ReasonQuoteNotVerbatim},
		{"a sentence from outside the press release", Quote{Record: "itd-73", Text: `"A sentence outside the press release," said Iris, a product thinker.`, Attribution: "Iris, a product thinker"}, ReasonQuoteNotVerbatim},
		{"a quote from an intent the page only lists", Quote{Record: "itd-74", Text: "A page per release.", Attribution: "A page"}, ReasonQuoteSource},
		{"a quote that cleaning would alter", Quote{Record: "itd-73", Text: irisQuote + " <!-- x -->", Attribution: "Iris, a product thinker"}, ReasonQuoteNotVerbatim},
		{"a quote truncated at its opening", Quote{Record: "itd-73", Text: `did lose a record," said Nia, a facilitator.`, Attribution: "Nia, a facilitator"}, ReasonQuoteNotVerbatim},
		{"a quote cut mid-word", Quote{Record: "itd-73", Text: `"We never did lose a record," said Nia, a facilit`, Attribution: "Nia"}, ReasonQuoteNotVerbatim},
		{"a quote cut mid-sentence", Quote{Record: "itd-73", Text: `"We never did lose a record," said Nia, a`, Attribution: "Nia"}, ReasonQuoteNotVerbatim},
		{"a sentence that is not a quote", Quote{Record: "itd-73", Text: "The release version is derived from what shipped, not typed by hand.", Attribution: "The release"}, ReasonQuoteNotVerbatim},
		{"a one-letter attribution", Quote{Record: "itd-73", Text: irisQuote, Attribution: "a"}, ReasonQuoteNotVerbatim},
		{"an attribution cut mid-word", Quote{Record: "itd-73", Text: irisQuote, Attribution: "Ir"}, ReasonQuoteNotVerbatim},
		{"an attribution cut mid-phrase", Quote{Record: "itd-73", Text: irisQuote, Attribution: "Iris, a"}, ReasonQuoteNotVerbatim},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := pageRepo(t)
			p := goodPage()
			p.Quotes = []Quote{tt.quote}
			_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), p), cutAt)
			reason, ok := reasonWith(refusalOf(t, err), tt.wantCode)
			if !ok {
				t.Fatalf("codes = %v, want %s", codesOf(refusalOf(t, err)), tt.wantCode)
			}
			if !strings.Contains(reason.Detail, tt.quote.Record) {
				t.Errorf("detail %q does not name the record", reason.Detail)
			}
		})
	}

	for _, ok := range []struct {
		name  string
		quote Quote
	}{
		{"a quote wrapped across lines in the source passes", Quote{Record: "itd-73", Text: irisQuote, Attribution: "Iris, a product thinker"}},
		{"a whole quoted sentence passes", Quote{Record: "itd-73", Text: niaQuote, Attribution: "Nia, a facilitator"}},
		{"the first sentence of a quote passes", Quote{Record: "itd-73", Text: `"I stopped typing version numbers," said Iris, a product thinker.`, Attribution: "Iris, a product thinker"}},
		{"an attribution naming the speaker alone passes", Quote{Record: "itd-73", Text: niaQuote, Attribution: "Nia"}},
	} {
		t.Run(ok.name, func(t *testing.T) {
			r := pageRepo(t)
			p := goodPage()
			p.Quotes = []Quote{ok.quote}
			if _, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), p), cutAt); err != nil {
				t.Errorf("Ingest refused a verbatim quote: %v", err)
			}
		})
	}
}

// TestPageRefusesARepeatedQuote: a quote carried twice would render twice under
// its headline, so the second is refused as a duplicate citation naming the
// first, rather than collapsed silently behind the composer's back.
func TestPageRefusesARepeatedQuote(t *testing.T) {
	r := pageRepo(t)
	p := goodPage()
	p.Quotes = []Quote{p.Quotes[0], {Record: "itd-73", Text: niaQuote, Attribution: "Nia"}, p.Quotes[0]}
	_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), p), cutAt)
	reason, ok := reasonWith(refusalOf(t, err), ReasonDuplicateCitation)
	if !ok {
		t.Fatalf("codes = %v, want %s", codesOf(refusalOf(t, err)), ReasonDuplicateCitation)
	}
	if reason.At != "press_release.quotes[2]" || !strings.Contains(reason.Detail, "press_release.quotes[0]") ||
		!strings.Contains(reason.Detail, "itd-73") {
		t.Errorf("reason = %+v, want quotes[2] refused naming itd-73 and quotes[0]", reason)
	}
	if n := len(refusalOf(t, err).Reasons); n != 1 {
		t.Errorf("%d reasons, want 1 (the second, distinct quote is not a duplicate): %v", n, codesOf(refusalOf(t, err)))
	}
}

// TestPageRefusesAnUnregisteredPersona: the release page sits at the repository
// root, outside record-lint's roots, so the repository's persona_registry rule
// is run over the rendered page at the cut. A headline attributing words to a
// persona the registry does not hold is refused; a registered one passes.
func TestPageRefusesAnUnregisteredPersona(t *testing.T) {
	for _, tc := range []struct {
		name    string
		speaker string
		refused bool
	}{
		{"an unregistered persona", "Zed", true},
		{"a registered persona", "Iris", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := pageRepo(t)
			r.Write(".abcd/record-lint.json", `{"roots": [".abcd/development"], "rules": {"persona_registry": `+
				`{"enabled": true, "severity": "blocker", "registry": ".abcd/development/personas.json"}}}`+"\n")
			r.Write(".abcd/development/personas.json", `{"personas": [{"name": "Iris"}]}`+"\n")
			r.Commit("the persona registry")
			p := goodPage()
			p.Headlines[0].Text = `"It works," said ` + tc.speaker + `, who cut the release.`
			_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), p), cutAt)
			if !tc.refused {
				if err != nil {
					t.Fatalf("Ingest refused a registered persona: %v", err)
				}
				return
			}
			reason, ok := reasonWith(refusalOf(t, err), ReasonPersonaRegistry)
			if !ok || !strings.Contains(reason.Detail, tc.speaker) {
				t.Errorf("reason = %+v, want persona-registry naming %s", reason, tc.speaker)
			}
		})
	}
}
