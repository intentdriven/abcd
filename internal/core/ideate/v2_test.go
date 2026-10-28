package ideate

// v2_test.go pins spc-2610142138047122 (itd-2610141646454347): schema version
// 2 of the verdict, in which every leg proves what it records. Each test below
// is one of the intent's acceptance criteria, in order; the criterion number is
// in the test's comment.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/source"
)

// leg returns the i-th leg of a test payload for mutation.
func leg(p map[string]any, i int) map[string]any { return p["legs"].([]any)[i].(map[string]any) }

// reviewer returns the i-th reviewer of a test payload's adversarial leg.
func reviewer(p map[string]any, i int) map[string]any {
	return leg(p, 2)["reviewers"].([]any)[i].(map[string]any)
}

// claim returns the i-th claim of a test payload's research leg.
func claim(p map[string]any, i int) map[string]any {
	return leg(p, 0)["claims"].([]any)[i].(map[string]any)
}

// TestV2RefusesAnUnresolvedIDInAnyLeg is criterion 1: a record id that does
// not resolve, written anywhere in the verdict's legs or its alternatives, is
// refused, and the refusal names where it was and the id.
func TestV2RefusesAnUnresolvedIDInAnyLeg(t *testing.T) {
	root := seedRepo(t)
	cases := map[string]struct {
		mutate func(p map[string]any)
		where  string
		id     string
	}{
		"primary source": {func(p map[string]any) { claim(p, 0)["primary_source"] = "itd-9999" }, "research", "itd-9999"},
		"claim text":     {func(p map[string]any) { claim(p, 1)["claim"] = "spc-9999 already says so" }, "research", "spc-9999"},
		"grill record": {func(p map[string]any) {
			leg(p, 1)["hits"] = []any{map[string]any{"record": "iss-9999", "relation": "covered"}}
		}, "record-grill", "iss-9999"},
		"grill note": {func(p map[string]any) {
			leg(p, 1)["hits"].([]any)[0].(map[string]any)["note"] = "as adr-9999 put it"
		}, "record-grill", "adr-9999"},
		"kill attempt": {func(p map[string]any) {
			reviewer(p, 1)["kill_attempts"].([]any)[0].(map[string]any)["attempt"] = "itd-9998 already killed this"
		}, "adversarial-review", "itd-9998"},
		"reviewer source": {func(p map[string]any) { reviewer(p, 0)["sources"] = []any{"spc-9998"} }, "adversarial-review", "spc-9998"},
		"rejected alternative": {func(p map[string]any) {
			p["rejected_alternatives"] = []any{map[string]any{"alternative": "fold it into iss-9998", "why_rejected": "a reason"}}
		}, "rejected alternatives", "iss-9998"},
		"why rejected": {func(p map[string]any) {
			p["rejected_alternatives"] = []any{map[string]any{"alternative": "something", "why_rejected": "ADR-9998 rules it out"}}
		}, "rejected alternatives", "adr-9998"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := validPayload()
			tc.mutate(p)
			_, err := Record(root, "an-uncited-idea", publicStore, encode(t, p), at)
			var cite *CitationError
			if !errors.As(err, &cite) {
				t.Fatalf("Record error = %v, want a *CitationError", err)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.where) || !strings.Contains(strings.ToLower(msg), tc.id) {
				t.Errorf("the refusal does not name the leg %q and the id %q: %s", tc.where, tc.id, msg)
			}
		})
	}
	assertNothingWritten(t, root)

	// Every spelling the record writes a resolvable id in is resolved, not
	// refused: a padded, upper-case ADR handle in a kill attempt names adr-35.
	p := validPayload()
	reviewer(p, 0)["kill_attempts"].([]any)[0].(map[string]any)["attempt"] = "ADR-0035 and itd-104-ideate both bear on this"
	claim(p, 0)["primary_source"] = "itd-104"
	if _, err := Record(root, "a-cited-idea", publicStore, encode(t, p), at); err != nil {
		t.Errorf("resolvable ids in prose were refused: %v", err)
	}
}

// TestV2RefusesAPrimarySourceOfNoKnownShape is criterion 2: a primary source
// is an https:// address, a filed source key, or a record id, and nothing else.
func TestV2RefusesAPrimarySourceOfNoKnownShape(t *testing.T) {
	root := seedRepo(t)
	for _, bad := range []string{
		"a remark heard in passing",
		"http://example.invalid/plain",
		"file:///tmp/notes.md",
		"https://",
		"https://example.invalid/a remark",
		"source:",
		"source:Not A Key",
		"itd-104 and more",
	} {
		p := validPayload()
		claim(p, 0)["primary_source"] = bad
		_, err := Record(root, "a-sourced-idea", publicStore, encode(t, p), at)
		if err == nil {
			t.Errorf("primary source %q was accepted", bad)
			continue
		}
		if !strings.Contains(err.Error(), "claim 1") || !strings.Contains(err.Error(), "https://") {
			t.Errorf("primary source %q: the refusal does not name the claim and the accepted shapes: %v", bad, err)
		}
	}
	assertNothingWritten(t, root)

	// With no corpus on the machine a well-formed key is accepted: there is
	// nothing to resolve it against, and its shape is still checked.
	for i, good := range []string{"https://example.invalid/paper.pdf", "itd-104", "adr-35", "source:survey2026"} {
		p := validPayload()
		claim(p, 0)["primary_source"] = good
		if _, err := Record(root, "a-sourced-idea-"+string(rune('a'+i)), publicStore, encode(t, p), at); err != nil {
			t.Errorf("primary source %q was refused: %v", good, err)
		}
	}
}

// TestV2ResolvesASourceKeyAgainstThePresentCorpus is criterion 2's corpus
// half: where the machine has a sources corpus, a source:<key> must name an
// entry in it.
func TestV2ResolvesASourceKeyAgainstThePresentCorpus(t *testing.T) {
	root := seedRepo(t)
	corpus := filedCorpus(t, "survey2026")
	store := publicStore
	store.SourcesDir = corpus

	p := validPayload()
	claim(p, 0)["primary_source"] = "source:survey2026"
	if _, err := Record(root, "a-filed-source", store, encode(t, p), at); err != nil {
		t.Fatalf("a filed source key was refused: %v", err)
	}

	p = validPayload()
	claim(p, 0)["primary_source"] = "source:never-filed"
	_, err := Record(root, "an-unfiled-source", store, encode(t, p), at)
	if err == nil {
		t.Fatal("a key the corpus does not carry was accepted")
	}
	if !strings.Contains(err.Error(), "source:never-filed") || !strings.Contains(err.Error(), "claim 1") {
		t.Errorf("the refusal does not name the claim and the key: %v", err)
	}
}

// filedCorpus initialises a sources corpus under a temporary home with one
// public entry, key, and returns its directory.
func filedCorpus(t *testing.T, key string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GIT_AUTHOR_NAME", "Fixture")
	t.Setenv("GIT_AUTHOR_EMAIL", "fixture@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Fixture")
	t.Setenv("GIT_COMMITTER_EMAIL", "fixture@example.invalid")
	dir := abcdhome.Path(home, "sources")
	if _, err := source.Init(dir); err != nil {
		t.Fatalf("source.Init: %v", err)
	}
	doc := filepath.Join(t.TempDir(), "survey.md")
	if err := os.WriteFile(doc, []byte("# survey\nbody\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Add(source.AddRequest{Corpus: dir, Key: key, Title: "A survey", Class: source.ClassPublic, Original: doc}); err != nil {
		t.Fatalf("source.Add: %v", err)
	}
	return dir
}

// TestV2RefusesAGrillWithNoHitsAndNoSearches is criterion 3: an empty grill
// must say what it searched, so it can be told from a broken one.
func TestV2RefusesAGrillWithNoHitsAndNoSearches(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	leg(p, 1)["hits"] = []any{}
	delete(leg(p, 1), "searches")
	_, err := Record(root, "an-unsearched-idea", publicStore, encode(t, p), at)
	if err == nil {
		t.Fatal("a grill with no hits and no searches was accepted")
	}
	if !strings.Contains(err.Error(), "searches") {
		t.Errorf("the refusal does not name the missing searches: %v", err)
	}

	for name, search := range map[string]map[string]any{
		"no-count":       {"terms": []any{"gate"}, "places": []any{"intents"}},
		"negative-count": {"terms": []any{"gate"}, "places": []any{"intents"}, "count": -1},
		"no-terms":       {"terms": []any{}, "places": []any{"intents"}, "count": 0},
		"blank-term":     {"terms": []any{"  "}, "places": []any{"intents"}, "count": 0},
		"no-places":      {"terms": []any{"gate"}, "count": 0},
	} {
		p := validPayload()
		leg(p, 1)["searches"] = []any{search}
		if _, err := Record(root, "a-bad-search", publicStore, encode(t, p), at); err == nil {
			t.Errorf("%s: the search was accepted", name)
		}
	}
	assertNothingWritten(t, root)

	// An empty grill that says what it searched is a finding, and the record
	// shows the searches it rests on.
	p = validPayload()
	leg(p, 1)["hits"] = []any{}
	leg(p, 1)["searches"] = []any{map[string]any{
		"terms": []any{"admission gate"}, "places": []any{".abcd/development/intents/"}, "count": 0,
	}}
	res, err := Record(root, "a-searched-idea", publicStore, encode(t, p), at)
	if err != nil {
		t.Fatalf("an empty grill with searches was refused: %v", err)
	}
	if res.Searches != 1 {
		t.Errorf("Searches = %d, want 1", res.Searches)
	}
	body := readFile(t, root, res.Path)
	for _, want := range []string{"| Terms | Places | Count |", "| admission gate | .abcd/development/intents/ | 0 |"} {
		if !strings.Contains(body, want) {
			t.Errorf("the record is missing %q:\n%s", want, body)
		}
	}
}

// TestV2RefusesAReframedVerdictWithoutAReframing is criterion 4.
func TestV2RefusesAReframedVerdictWithoutAReframing(t *testing.T) {
	root := seedRepo(t)
	for name, reframing := range map[string]any{"absent": nil, "blank": "   "} {
		p := validPayload()
		p["verdict"] = "reframed"
		if reframing != nil {
			p["reframing"] = reframing
		}
		_, err := Record(root, "a-reframed-idea", publicStore, encode(t, p), at)
		if err == nil {
			t.Errorf("%s: a reframed verdict with no reframing was accepted", name)
		} else if !strings.Contains(err.Error(), "reframing") {
			t.Errorf("%s: the refusal does not name the reframing: %v", name, err)
		}
	}
	// A reframing beside a verdict that is not reframed contradicts it.
	p := validPayload()
	p["reframing"] = "gate only ideas that would mint an intent"
	if _, err := Record(root, "a-contradicted-idea", publicStore, encode(t, p), at); err == nil {
		t.Error("a reframing beside a surviving verdict was accepted")
	}
	assertNothingWritten(t, root)

	p = validPayload()
	p["verdict"] = "reframed"
	p["reframing"] = "gate only ideas that would mint an intent"
	res, err := Record(root, "a-reframed-idea", publicStore, encode(t, p), at)
	if err != nil {
		t.Fatalf("a reframed verdict with its reframing was refused: %v", err)
	}
	body := readFile(t, root, res.Path)
	if !strings.Contains(body, "## The reframing\n\ngate only ideas that would mint an intent\n") {
		t.Errorf("the record does not carry the reframing in its own section:\n%s", body)
	}
}

// TestV2KeepsEachReviewerUnderItsLabel is criterion 5: two reviewers' kill
// attempts, verdicts and sources are each kept under that reviewer.
func TestV2KeepsEachReviewerUnderItsLabel(t *testing.T) {
	root := seedRepo(t)
	res, err := Record(root, "a-reviewed-idea", publicStore, encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if res.Reviewers != 2 || res.KillAttempts != 2 {
		t.Errorf("Reviewers = %d, KillAttempts = %d; want 2 and 2", res.Reviewers, res.KillAttempts)
	}
	body := readFile(t, root, res.Path)
	a := strings.Index(body, "### Reviewer reviewer-a — survives")
	b := strings.Index(body, "### Reviewer reviewer-b — survives")
	if a < 0 || b < 0 || b < a {
		t.Fatalf("the record does not hold a section per reviewer, in order:\n%s", body)
	}
	first, second := body[a:b], body[b:]
	for _, want := range []string{"the gate adds friction to capture", "https://example.invalid/paper.pdf"} {
		if !strings.Contains(first, want) || strings.Contains(second, want) {
			t.Errorf("%q is not kept under reviewer-a alone:\n%s", want, body)
		}
	}
	if !strings.Contains(second, "no evidence the legs improve outcomes") || strings.Contains(first, "no evidence the legs improve outcomes") {
		t.Errorf("reviewer-b's attempt is not kept under reviewer-b alone:\n%s", body)
	}
	if !strings.Contains(second, "Sources consulted: none named.") {
		t.Errorf("reviewer-b's empty sources are not stated:\n%s", body)
	}

	for name, mutate := range map[string]func(p map[string]any){
		"duplicate-label": func(p map[string]any) { reviewer(p, 1)["label"] = "reviewer-a" },
		"blank-label":     func(p map[string]any) { reviewer(p, 0)["label"] = " " },
		"no-verdict":      func(p map[string]any) { delete(reviewer(p, 0), "verdict") },
		"bad-source":      func(p map[string]any) { reviewer(p, 0)["sources"] = []any{"what I remembered"} },
	} {
		p := validPayload()
		mutate(p)
		if _, err := Record(root, "a-misreviewed-idea", publicStore, encode(t, p), at); err == nil {
			t.Errorf("%s: the reviewer was accepted", name)
		}
	}
}

// TestV2RefusesASplitWithoutARuling is criterion 6: reviewers whose verdicts
// differ need the ruling that settled them, naming who ruled.
func TestV2RefusesASplitWithoutARuling(t *testing.T) {
	root := seedRepo(t)
	split := func() map[string]any {
		p := validPayload()
		reviewer(p, 1)["verdict"] = "killed"
		return p
	}
	_, err := Record(root, "a-split-idea", publicStore, encode(t, split()), at)
	if err == nil {
		t.Fatal("a split between reviewers with no ruling was accepted")
	}
	if !strings.Contains(err.Error(), "ruling") {
		t.Errorf("the refusal does not name the ruling: %v", err)
	}
	for name, ruling := range map[string]map[string]any{
		"no-by":     {"verdict": "survives", "reason": "the fatal attempt rested on a misread claim"},
		"no-reason": {"by": "the product thinker", "verdict": "survives"},
		"bad-verdict": {"by": "the product thinker", "verdict": "maybe",
			"reason": "the fatal attempt rested on a misread claim"},
	} {
		p := split()
		leg(p, 2)["ruling"] = ruling
		if _, err := Record(root, "a-split-idea", publicStore, encode(t, p), at); err == nil {
			t.Errorf("%s: the ruling was accepted", name)
		}
	}
	assertNothingWritten(t, root)

	p := split()
	leg(p, 2)["ruling"] = map[string]any{
		"by": "the product thinker", "verdict": "survives", "reason": "the fatal attempt rested on a misread claim",
	}
	res, err := Record(root, "a-split-idea", publicStore, encode(t, p), at)
	if err != nil {
		t.Fatalf("a split with its ruling was refused: %v", err)
	}
	body := readFile(t, root, res.Path)
	for _, want := range []string{"### Ruling", "**survives**, ruled by the product thinker", "the fatal attempt rested on a misread claim"} {
		if !strings.Contains(body, want) {
			t.Errorf("the record is missing %q:\n%s", want, body)
		}
	}
}

// TestV2RefusesAVerifiedSelfQuotation is criterion 7: a claim checked only
// against the idea's own source is quoted, never verified.
func TestV2RefusesAVerifiedSelfQuotation(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	claim(p, 0)["primary_source"] = "https://example.invalid/origin-post"
	_, err := Record(root, "a-self-quoted-idea", publicStore, encode(t, p), at)
	if err == nil {
		t.Fatal("a verified claim resting on the idea's own source was accepted")
	}
	if !strings.Contains(err.Error(), "claim 1") || !strings.Contains(err.Error(), "quoted") {
		t.Errorf("the refusal does not name the claim and the quoted status: %v", err)
	}
	// The origin sources are stated explicitly, so the check cannot be skipped
	// by leaving them out.
	p = validPayload()
	delete(p, "origin_sources")
	if _, err := Record(root, "a-sourceless-idea", publicStore, encode(t, p), at); err == nil {
		t.Error("a verdict that does not say where the idea came from was accepted")
	}
	assertNothingWritten(t, root)

	p = validPayload()
	claim(p, 0)["primary_source"] = "https://example.invalid/origin-post"
	claim(p, 0)["status"] = "quoted"
	res, err := Record(root, "a-quoted-idea", publicStore, encode(t, p), at)
	if err != nil {
		t.Fatalf("a quoted claim was refused: %v", err)
	}
	if res.Claims.Quoted != 1 || res.Claims.Verified != 0 || res.Claims.Total() != 2 {
		t.Errorf("Claims = %+v; want one quoted, none verified, two in all", res.Claims)
	}
	body := readFile(t, root, res.Path)
	for _, want := range []string{"| quoted |", "- https://example.invalid/origin-post"} {
		if !strings.Contains(body, want) {
			t.Errorf("the record is missing %q:\n%s", want, body)
		}
	}

	// An idea with no source behind it says so with an empty list.
	p = validPayload()
	p["origin_sources"] = []any{}
	if _, err := Record(root, "an-unsourced-idea", publicStore, encode(t, p), at); err != nil {
		t.Errorf("an explicitly empty origin_sources was refused: %v", err)
	}
}

// TestV2RefusesAVersionOnePayload is criterion 8.
func TestV2RefusesAVersionOnePayload(t *testing.T) {
	if SchemaVersion != 2 {
		t.Fatalf("SchemaVersion = %d, want 2", SchemaVersion)
	}
	root := seedRepo(t)
	p := validPayload()
	p["schema_version"] = 1
	_, err := Record(root, "an-old-idea", publicStore, encode(t, p), at)
	if err == nil {
		t.Fatal("a version-1 payload was accepted")
	}
	if !strings.Contains(err.Error(), "version 2") {
		t.Errorf("the refusal does not name version 2: %v", err)
	}
	assertNothingWritten(t, root)
}

// TestCommandPageGivesEachReviewerOneIdeaPerContext is criterion 9, read off
// the command page itself: leg 3 tells the runner to give each reviewer one
// idea per context, and the payload example is the shape this binary decodes.
func TestCommandPageGivesEachReviewerOneIdeaPerContext(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "commands", "ideate.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	start := strings.Index(page, "### Leg 3")
	end := strings.Index(page, "## Record the verdict")
	if start < 0 || end < start {
		t.Fatal("the command page has no leg 3 section before its record section")
	}
	if !strings.Contains(strings.ToLower(page[start:end]), "one idea per context") {
		t.Errorf("leg 3 does not tell the runner to give each reviewer one idea per context:\n%s", page[start:end])
	}

	m := regexp.MustCompile("(?s)```json\n(.*?)\n```").FindStringSubmatch(page)
	if m == nil {
		t.Fatal("the command page carries no JSON payload example")
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(m[1])))
	dec.DisallowUnknownFields()
	var p Payload
	if err := dec.Decode(&p); err != nil {
		t.Fatalf("the page's payload example does not decode as this binary's payload: %v", err)
	}
	if p.SchemaVersion != SchemaVersion {
		t.Errorf("the page's example is schema_version %d, want %d", p.SchemaVersion, SchemaVersion)
	}
	if p.OriginSources == nil || len(p.Legs) != 3 || len(p.Legs[1].Searches) == 0 ||
		len(p.Legs[2].Reviewers) < 2 || p.Legs[2].Ruling == nil {
		t.Errorf("the page's example does not show origin_sources, searches, two reviewers and a ruling: %+v", p)
	}
}
