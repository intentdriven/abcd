package scribe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// echoLeak is a payload value carrying a marker and a home path. A refusal that
// echoes a payload-chosen value carries both to the terminal and the transcript.
const echoLeak = "zzleak-7f3a /Users/zzotherperson/notes long enough to quote" // abcd-lint:allow — a planted home path the refusal must not echo

// assertNoEcho fails when err is nil, carries either half of the leak, or no
// longer names where the fault is.
func assertNoEcho(t *testing.T, err error, names string) {
	t.Helper()
	if err == nil {
		t.Fatal("a payload carrying the leak was accepted")
	}
	for _, part := range []string{"zzleak-7f3a", "zzotherperson"} {
		if strings.Contains(err.Error(), part) {
			t.Errorf("the refusal echoes the payload: %v", err)
		}
	}
	if !strings.Contains(err.Error(), names) {
		t.Errorf("the refusal no longer names %s: %v", names, err)
	}
}

// TestScribeRefusalsDoNotEchoThePayload — iss-2609290218032954. Every scribe
// ingest refusal returns with nothing written, and each of these carried the
// payload's own value back through echo(), which cleans and caps but does not
// redact. A value of a closed shape (a run or item id, a state, a digest) is
// quoted only when it has that shape, and described otherwise; free text the
// researcher did not write is described; an undeclared key, the one value the
// reader needs, is redacted through the canonical scanner and still named.
func TestScribeRefusalsDoNotEchoThePayload(t *testing.T) {
	const supplied = "{0}: accepted — " + groundA + ".\n{1}: I have not decided yet.\n"
	cases := []struct {
		name   string
		mutate func(s session, o *Output)
		names  string
	}{
		{"_type", func(_ session, o *Output) { o.Type = echoLeak }, "_type"},
		{"run", func(_ session, o *Output) { o.Run = echoLeak }, "run"},
		{"context_sha256", func(_ session, o *Output) { o.ContextSHA256 = echoLeak }, "context"},
		{"disposition item", func(_ session, o *Output) { o.Dispositions[0].Item = echoLeak }, "dispositions[0]"},
		{"disposition state", func(_ session, o *Output) { o.Dispositions[0].State = echoLeak }, "state"},
		{"disposition grounds", func(_ session, o *Output) { o.Dispositions[0].Grounds = echoLeak }, "grounds"},
		{"disposition exit_condition", func(_ session, o *Output) { o.Dispositions[0].ExitCondition = echoLeak }, "exit_condition"},
		{"disposition supersedes", func(_ session, o *Output) { o.Dispositions[0].Supersedes = echoLeak }, "cites"},
		{"disposition recurs", func(_ session, o *Output) { o.Dispositions[0].Recurs = []string{echoLeak} }, "cites"},
		{"admission item", func(_ session, o *Output) {
			o.Admissions = []OutAdmission{{Item: echoLeak, Grounds: groundA}}
		}, "admissions[0]"},
		{"surprise occasioned_by", func(_ session, o *Output) {
			o.Surprises = []OutSurprise{{OccasionedBy: echoLeak, Text: groundA}}
		}, "surprises[0]"},
		{"surprise text", func(s session, o *Output) {
			o.Surprises = []OutSurprise{{OccasionedBy: s.items[0], Text: echoLeak}}
		}, "text"},
		{"outstanding", func(_ session, o *Output) { o.Outstanding = []string{echoLeak} }, "outstanding[0]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := assembleSession(t, positionDetection, 2, supplied)
			o := s.out()
			o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
			o.Outstanding = []string{s.items[1]}
			tc.mutate(s, &o)
			_, err := s.ingest(t, s.write(t, o))
			assertNoEcho(t, err, tc.names)
		})
	}
}

// TestScribeKeyRefusalsRedactTheKey: an undeclared key, the subject an entry is
// labelled by, and a repeated key are payload text in a refusal. The key is
// named with the planted home path redacted, and a subject that is not an item
// handle is described.
func TestScribeKeyRefusalsRedactTheKey(t *testing.T) {
	const supplied = "{0}: accepted — " + groundA + ".\n"
	cases := []struct {
		name  string
		raw   func(s session) string
		names string
	}{
		{"undeclared top-level key", func(s session) string {
			return `{"_type":"` + OutputType + `","run":"` + fixtureRun + `","context_sha256":"` + s.res.ContextSHA256 +
				`","reviewer_notes /Users/zzotherperson/notes":"x"}` // abcd-lint:allow — a planted home path in a KEY
		}, "reviewer_notes"},
		{"entry labelled by a leaking subject", func(s session) string {
			item, _ := json.Marshal(echoLeak)
			return `{"_type":"` + OutputType + `","run":"` + fixtureRun + `","context_sha256":"` + s.res.ContextSHA256 +
				`","dispositions":[{"item":` + string(item) + `,"verdict":"x"}]}`
		}, "verdict"},
		{"repeated nested key", func(s session) string {
			return `{"_type":"` + OutputType + `","run":"` + fixtureRun + `","context_sha256":"` + s.res.ContextSHA256 +
				`","dispositions":[{"item":"` + s.items[0] + `","grounds":{"k /Users/zzotherperson/a":1,"k /Users/zzotherperson/a":2}}]}` // abcd-lint:allow — a planted home path in a KEY
		}, "duplicate key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := assembleSession(t, positionDetection, 1, supplied)
			_, err := s.ingest(t, s.writeRaw(t, tc.raw(s)))
			assertNoEcho(t, err, tc.names)
		})
	}
}

// TestScribeParkedPairRefusalsDoNotEcho: the parked manifest sits where a
// scribe session with tools can rewrite it, so its _type, an undeclared key in
// it and its supplied hash are payload-chosen too.
func TestScribeParkedPairRefusalsDoNotEcho(t *testing.T) {
	const supplied = "{0}: accepted — " + groundA + ".\n"
	cases := []struct {
		name   string
		mutate func(m map[string]any)
		names  string
	}{
		{"manifest _type", func(m map[string]any) { m["_type"] = echoLeak }, "manifest"},
		{"manifest undeclared key", func(m map[string]any) { m["notes /Users/zzotherperson/notes"] = "x" }, "notes"}, // abcd-lint:allow — a planted home path in a KEY
		{"manifest supplied hash", func(m map[string]any) {
			m["supplied"] = map[string]any{"dispositions_sha256": echoLeak}
		}, "supplied dispositions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := assembleSession(t, positionDetection, 1, supplied)
			parked := filepath.Join(s.repo, filepath.FromSlash(DefaultRunDir), fixtureRun, ManifestFileName)
			raw, err := os.ReadFile(parked)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			tc.mutate(m)
			raw, _ = json.Marshal(m)
			if err := os.WriteFile(parked, raw, 0o644); err != nil {
				t.Fatal(err)
			}
			o := s.out()
			o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
			_, err = s.ingest(t, s.write(t, o))
			assertNoEcho(t, err, tc.names)
		})
	}
}
