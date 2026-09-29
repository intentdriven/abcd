package release

import (
	"strings"
	"testing"
)

// releaseLeak carries a marker and a home path; releaseShortLeak is the same
// under the record-id length cap, so the refusal that matches the id, rather
// than the one that counts its bytes, is the one reached.
const (
	releaseLeak      = "zzleak-7f3a /Users/zzotherperson/notes" // abcd-lint:allow — a planted home path the refusal must not echo
	releaseShortLeak = "zzleak-7f3a/Users/zzotherperson"        // abcd-lint:allow — a planted home path the refusal must not echo
)

// TestChangelogRefusalsDoNotEchoThePayload — iss-2609290218032954. The
// changelog payload is host-composed, and its refusals quoted a stale next_tag,
// an out-of-set section, a malformed record id and a headline's attributed name
// through termsafe.Sanitize alone, which strips control sequences and redacts
// nothing; the decoder's undeclared-field message was returned the same way. A
// value is described now, and a key the reader needs is named redacted.
func TestChangelogRefusalsDoNotEchoThePayload(t *testing.T) {
	cases := []struct {
		name  string
		raw   func(t *testing.T) []byte
		leaks []string
		names string
	}{
		{"next_tag", func(t *testing.T) []byte {
			return marshalPage(t, releaseLeak, pageEntries(), goodPage())
		}, []string{"zzleak-7f3a", "zzotherperson"}, "next_tag"},
		{"section", func(t *testing.T) []byte {
			e := pageEntries()
			e[0].Section = Section(releaseLeak)
			return marshalPage(t, "v0.4.1", e, goodPage())
		}, []string{"zzleak-7f3a", "zzotherperson"}, "section"},
		{"changelog record id", func(t *testing.T) []byte {
			e := pageEntries()
			e[0].Records = append(e[0].Records, releaseShortLeak)
			return marshalPage(t, "v0.4.1", e, goodPage())
		}, []string{"zzleak-7f3a", "zzotherperson"}, "record id"},
		{"page record id", func(t *testing.T) []byte {
			p := goodPage()
			p.Listed = append(p.Listed, releaseShortLeak)
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, []string{"zzleak-7f3a", "zzotherperson"}, "record id"},
		{"headline attribution", func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = `Nobody types a version any more, said Zzleakname, a product thinker.`
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}, []string{"Zzleakname"}, "attributes words"},
		{"undeclared key", func(t *testing.T) []byte {
			raw := string(marshalPage(t, "v0.4.1", pageEntries(), goodPage()))
			return []byte(strings.Replace(raw, `{`, `{"reviewer_notes /Users/zzotherperson/notes":1,`, 1)) // abcd-lint:allow — a planted home path in a KEY
		}, []string{"zzotherperson"}, "reviewer_notes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := pageRepo(t)
			_, err := Ingest(r.Root(), liveSurface(), tc.raw(t), cutAt)
			if err == nil {
				t.Fatal("a payload carrying the leak was accepted")
			}
			for _, part := range tc.leaks {
				if strings.Contains(err.Error(), part) {
					t.Errorf("the refusal echoes the payload (%q): %v", part, err)
				}
			}
			if !strings.Contains(err.Error(), tc.names) {
				t.Errorf("the refusal no longer names %s: %v", tc.names, err)
			}
		})
	}
}

// TestPersonaRefusalDoesNotEchoTheName — iss-2609290218032954. With the
// repository's persona_registry rule armed, the rendered page's finding named
// the attributed persona through scanner.RedactRefusal, whose patterns know
// tokens and paths but not a person's name, so the name was echoed verbatim
// while the headline refusal for the same value described it. The persona
// finding describes it too: the line number is what the reader needs.
func TestPersonaRefusalDoesNotEchoTheName(t *testing.T) {
	for _, tc := range []struct {
		name  string
		page  func() *PressReleasePayload
		leaks string
	}{
		{"headline attribution", func() *PressReleasePayload {
			p := goodPage()
			p.Headlines[0].Text = `Nobody types a version any more, said Zzleakperson, a product thinker.`
			return p
		}, "Zzleakperson"},
		{"quote attribution", func() *PressReleasePayload {
			p := goodPage()
			p.Quotes = []Quote{{Record: "itd-73", Text: niaQuote, Attribution: "Nia"}}
			return p
		}, "Nia"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := pageRepo(t)
			r.Write(".abcd/record-lint.json", `{"roots": [".abcd/development"], "rules": {"persona_registry": `+
				`{"enabled": true, "severity": "blocker", "registry": ".abcd/development/personas.json"}}}`+"\n")
			r.Write(".abcd/development/personas.json", `{"personas": [{"name": "Iris"}]}`+"\n")
			r.Commit("the persona registry")
			_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), tc.page()), cutAt)
			reason, ok := reasonWith(refusalOf(t, err), ReasonPersonaRegistry)
			if !ok {
				t.Fatalf("no %s reason: %v", ReasonPersonaRegistry, err)
			}
			if strings.Contains(reason.Detail, tc.leaks) {
				t.Errorf("the persona finding echoes the attributed name %q: %s", tc.leaks, reason.Detail)
			}
			if !strings.Contains(reason.Detail, "line ") || !strings.Contains(reason.Detail, "not in the registry") {
				t.Errorf("the persona finding no longer locates the fault: %s", reason.Detail)
			}
		})
	}
}
