package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// The guided path in the session (spc-2610031241482088, step 3): one turn
// per question, a keyless look-up only on a yes, suggestions from the
// person's own connections, narrowing with no second request, and a done turn
// that prints one command and every path it writes.

// guideFake is a keyless stand-in: it answers its model list with code and
// ids, and remembers every request and the Authorization header it carried.
type guideFake struct {
	srv  *httptest.Server
	code int
	ids  []string
	// redirect, when set, answers the list with a redirect to it.
	redirect string
	// body, when set, is the list's body as it is.
	body string

	mu    sync.Mutex
	paths []string
	auth  []string
}

func newGuideFake(t *testing.T, code int, ids []string) *guideFake {
	t.Helper()
	f := &guideFake{code: code, ids: ids}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.paths = append(f.paths, r.Method+" "+r.URL.Path)
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		f.mu.Unlock()
		if f.redirect != "" {
			http.Redirect(w, r, f.redirect, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(f.code)
		if f.body != "" {
			_, _ = w.Write([]byte(f.body))
			return
		}
		data := make([]map[string]string, len(f.ids))
		for i, id := range f.ids {
			data[i] = map[string]string{"id": id}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *guideFake) base() string { return f.srv.URL + "/v1" }

func (f *guideFake) requests() (paths, auth []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...), append([]string(nil), f.auth...)
}

// turnOf runs one turn the way the page does: the resume object goes out as
// JSON and comes back as JSON.
func turnOf(t *testing.T, f *fx, prev *GuideTurn, answer *string, first GuideRequest) (GuideTurn, error) {
	t.Helper()
	req := first
	req.Roots = f.roots
	if prev != nil {
		raw, err := json.Marshal(prev.Resume)
		if err != nil {
			t.Fatal(err)
		}
		var back GuideState
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatal(err)
		}
		req = GuideRequest{Roots: f.roots, Resume: &back, Answer: answer, EnvNames: first.EnvNames}
	}
	return Guide(context.Background(), req)
}

// drive runs the guide from its first turn through answers, failing on a
// refusal, and returns every turn.
func drive(t *testing.T, f *fx, first GuideRequest, answers ...string) []GuideTurn {
	t.Helper()
	turn, err := turnOf(t, f, nil, nil, first)
	if err != nil {
		t.Fatalf("the first turn: %v", err)
	}
	turns := []GuideTurn{turn}
	for i, a := range answers {
		a := a
		next, err := turnOf(t, f, &turns[len(turns)-1], &a, first)
		if err != nil {
			t.Fatalf("answer %d (%q): %v", i+1, a, err)
		}
		turns = append(turns, next)
	}
	return turns
}

func last(turns []GuideTurn) GuideTurn { return turns[len(turns)-1] }

// asked is the one question a turn asks.
func asked(t *testing.T, turn GuideTurn) question.Question {
	t.Helper()
	if turn.Ask == nil || len(turn.Ask.Questions) != 1 {
		t.Fatalf("the turn asks no single question: %+v", turn)
	}
	return turn.Ask.Questions[0]
}

func material(q question.Question) string {
	var b strings.Builder
	for _, m := range q.Material {
		b.WriteString(m.Text + "\n")
	}
	return b.String()
}

func optionValues(q question.Question) []string {
	out := []string{}
	for _, o := range q.Options {
		out = append(out, o.Value)
	}
	return out
}

// TestGuideAsksBeforeLookingUp (G1): the look-up question shows the scheme
// and host and sends nothing; "Type the name myself" sends nothing; a yes
// sends exactly one GET of the model list, carrying no key; a stand-in that
// answers with a redirect is refused, naming it, and its target sees nothing.
func TestGuideAsksBeforeLookingUp(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, []string{"vendor/coder-large", "vendor/coder-small"})
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base()}

	turns := drive(t, f, first)
	q := asked(t, last(turns))
	if q.ID != GuideQLookup || !strings.Contains(material(q), "http://"+svc.srv.Listener.Addr().String()) {
		t.Fatalf("the first question is %q, its material %q; want the look-up showing the scheme and host", q.ID, material(q))
	}
	if paths, _ := svc.requests(); len(paths) != 0 {
		t.Fatalf("a request went out before the person answered: %q", paths)
	}

	turns = drive(t, f, first, "type")
	if q := asked(t, last(turns)); q.ID != GuideQTyped || !strings.Contains(material(q), "You chose to type the name") {
		t.Fatalf("after declining, the guide asks %q: %q", q.ID, material(q))
	}
	if paths, _ := svc.requests(); len(paths) != 0 {
		t.Fatalf("declining the look-up sent %q", paths)
	}

	turns = drive(t, f, first, "Look up the list")
	if q := asked(t, last(turns)); q.ID != GuideQModel {
		t.Fatalf("after a yes the guide asks %q", q.ID)
	}
	paths, auth := svc.requests()
	if !reflect.DeepEqual(paths, []string{"GET /v1/models"}) || !reflect.DeepEqual(auth, []string{""}) {
		t.Fatalf("a yes sent %q carrying %q; want one keyless GET of the list", paths, auth)
	}

	target := newGuideFake(t, http.StatusOK, []string{"vendor/elsewhere"})
	moved := newGuideFake(t, http.StatusOK, nil)
	moved.redirect = target.base() + "/models"
	turns = drive(t, newFx(t), GuideRequest{BaseURL: moved.base()}, "lookup")
	q = asked(t, last(turns))
	if q.ID != GuideQTyped || !strings.Contains(material(q), "redirect") {
		t.Fatalf("a redirecting service reached %q: %q", q.ID, material(q))
	}
	if paths, _ := target.requests(); len(paths) != 0 {
		t.Fatalf("the redirect's target received %q", paths)
	}
	if _, auth := moved.requests(); !reflect.DeepEqual(auth, []string{""}) {
		t.Fatalf("the redirecting service saw %q", auth)
	}
}

// threeHundred is a stand-in's long list: 300 names, a few containing coder.
func threeHundred() []string {
	ids := make([]string, 0, 300)
	for i := range 300 {
		switch i {
		case 40, 140, 240, 290:
			ids = append(ids, fmt.Sprintf("vendor%d/coder-%d", i%3, i))
		default:
			ids = append(ids, fmt.Sprintf("vendor%d/model-%d", i%3, i))
		}
	}
	return ids
}

// TestGuideSuggestsTheModelsAlreadyUsed (G2): a person whose two connections
// name two of the 300 listed models is offered exactly those two and the
// typed part; a first-time user is offered the typed part alone.
func TestGuideSuggestsTheModelsAlreadyUsed(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, threeHundred())
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{
		"alpha":{"base_url":"https://alpha.example.com/v1","models":["vendor1/model-7","not/listed"]},
		"beta":{"base_url":"https://beta.example.com/v1","models":["vendor2/model-200"]}}}}`)
	q := asked(t, last(drive(t, f, GuideRequest{BaseURL: svc.base()}, "lookup")))
	if got := optionValues(q); !reflect.DeepEqual(got, []string{"vendor1/model-7", "vendor2/model-200"}) || q.Typed == "" {
		t.Fatalf("the model question offers %q (typed %q); want the two models already used and the typed part", got, q.Typed)
	}
	if !strings.Contains(material(q), "lists 300 models") {
		t.Errorf("the material does not say how many are listed: %q", material(q))
	}

	q = asked(t, last(drive(t, newFx(t), GuideRequest{BaseURL: svc.base()}, "lookup")))
	if len(q.Options) != 0 || q.Typed == "" {
		t.Fatalf("a first-time user is offered %q (typed %q); want the typed part alone", optionValues(q), q.Typed)
	}
}

// TestGuideNarrowsWithoutASecondRequest (G2): typing "coder" offers the
// first three of the names containing it and says how many match; more
// typing narrows again; a match chosen ends the model question; the stand-in
// receives one request across every turn.
func TestGuideNarrowsWithoutASecondRequest(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, threeHundred())
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base()}
	turns := drive(t, f, first, "lookup", "CODER")
	q := asked(t, last(turns))
	want := []string{"vendor1/coder-40", "vendor2/coder-140", "vendor0/coder-240"}
	if q.ID != GuideQNarrow || !reflect.DeepEqual(optionValues(q), want) {
		t.Fatalf("narrowing by coder asks %q offering %q; want %q", q.ID, optionValues(q), want)
	}
	if m := material(q); !strings.Contains(m, "4 of 300 match") || !strings.Contains(m, "first 3 are shown") {
		t.Errorf("the narrowing material: %q", m)
	}
	// question.Matches is the rule: the same names the Terminal's list keeps.
	for _, id := range want {
		if !question.Matches("CODER", question.Option{Value: id, Label: id}) {
			t.Fatalf("%s is offered but the shared rule does not keep it", id)
		}
	}

	// One guided run: narrowing again, no match, then a match chosen; the
	// look-up is made once across every turn.
	before, _ := svc.requests()
	turns = drive(t, f, first, "lookup", "coder", "coder-29", "zzz", "coder", "vendor2/coder-140", "none")
	if q := asked(t, turns[3]); !reflect.DeepEqual(optionValues(q), []string{"vendor2/coder-290"}) {
		t.Fatalf("narrowing again offers %q", optionValues(q))
	}
	if q := asked(t, turns[4]); len(q.Options) != 0 || !strings.Contains(material(q), "None of the 300") {
		t.Fatalf("no match offers %q: %q", optionValues(q), material(q))
	}
	done := last(turns).Done
	if done == nil || !strings.Contains(done.Command, "--model vendor2/coder-140") {
		t.Fatalf("choosing a match ends with %+v", last(turns))
	}
	if after, _ := svc.requests(); len(after)-len(before) != 1 {
		t.Fatalf("one guided run sent %d requests across its turns, want one", len(after)-len(before))
	}
	// A fragment that is exactly one listed id chooses it at once.
	turns = drive(t, f, first, "lookup", "vendor0/model-3")
	if q := asked(t, last(turns)); q.ID != GuideQTakes {
		t.Fatalf("an exact id typed goes to %q", q.ID)
	}
}

// TestGuideCapsTheCarriedList (G2, open question 2): a look-up listing more
// ids than MaxCarriedBytes holds carries the first that fit, in the service's
// order, and the count of the rest; the model question says how many the
// service lists and how many are carried. A part of a name matching none
// carried asks for the full name, which is taken as any typed name is
// (validModel and the key check ask again; an exact name ends the guide). A
// resume object's count is checked on the way back in, and a carried list over
// the budget is cut to it again.
func TestGuideCapsTheCarriedList(t *testing.T) {
	ids := make([]string, 2000)
	for i := range ids {
		ids[i] = fmt.Sprintf("vendor%d/a-long-enough-model-name-%04d", i%3, i)
	}
	svc := newGuideFake(t, http.StatusOK, ids)
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base()}
	turn := last(drive(t, f, first, "lookup"))
	l := turn.Resume.Listed
	if l == nil || l.More == 0 || len(l.Models)+l.More != len(ids) || !reflect.DeepEqual(l.Models, ids[:len(l.Models)]) {
		t.Fatalf("2,000 ids listed carry %+v", l)
	}
	if b, _ := json.Marshal(l.Models); len(b) > MaxCarriedBytes {
		t.Fatalf("the carried ids are %d bytes of JSON; the budget is %d", len(b), MaxCarriedBytes)
	}
	host := strings.TrimPrefix(strings.TrimSuffix(svc.base(), "/v1"), "http://")
	if m := material(asked(t, turn)); !strings.Contains(m, fmt.Sprintf("%s lists 2000 models; the guide carries the first %d.", host, len(l.Models))) {
		t.Fatalf("the model question over a capped list says %q", m)
	}
	uncarried := ids[len(ids)-1]
	turns := drive(t, f, first, "lookup", "name-1999", "bad;name", keyShapedAnswer(7), uncarried, "none")
	full := fmt.Sprintf("%s listed %d more models than the guide carries; type the model's full name.", host, l.More)
	for i, want := range []string{full, "is not a model name abcd accepts", "looks like a key"} {
		if q := asked(t, turns[2+i]); q.ID != GuideQTyped || !strings.Contains(material(q), want) {
			t.Fatalf("turn %d asks %q: %q; want the full name asked for, saying %q", 2+i, q.ID, material(q), want)
		}
	}
	if done := last(turns).Done; done == nil || !strings.Contains(done.Command, "--model "+uncarried) {
		t.Fatalf("the full name typed ends with %+v", last(turns))
	}
	// A part matching a carried id still narrows over the carried list.
	if q := asked(t, last(drive(t, f, first, "lookup", "name-0001"))); q.ID != GuideQNarrow {
		t.Fatalf("a part matching carried ids asks %q", q.ID)
	}

	var refusal *GuideRefusal
	frag := "name-1999"
	for _, tc := range []struct {
		name string
		edit func(*GuideListed)
		want string
	}{
		{"a negative count", func(l *GuideListed) { l.More = -1 }, "counts -1 more names"},
		{"a count on a list it did not get", func(l *GuideListed) { l.Status, l.Models, l.More = ListedNeedsKey, nil, 3 }, "counts 3 more names"},
		{"a count past what a look-up keeps", func(l *GuideListed) { l.More = 5000 }, "a look-up keeps at most 5000"},
	} {
		edited := turn
		cp := *turn.Resume.Listed
		edited.Resume.Listed = &cp
		tc.edit(edited.Resume.Listed)
		if _, err := turnOf(t, f, &edited, &frag, first); !errors.As(err, &refusal) || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s = %v; want a refusal saying %q", tc.name, err, tc.want)
		}
	}
	// A list carried whole, over the budget, is cut to it again.
	before, _ := svc.requests()
	edited := turn
	edited.Resume.Listed = &GuideListed{Status: ListedListed, Host: host, Models: ids}
	next, err := turnOf(t, f, &edited, &frag, first)
	if err != nil {
		t.Fatal(err)
	}
	got := next.Resume.Listed
	if got == nil {
		t.Fatalf("a carried list over the budget comes back as none: %+v", next)
	}
	if !reflect.DeepEqual(got.Models, l.Models) || got.More != l.More {
		t.Fatalf("a carried list over the budget comes back as %d ids, more %d; want %d, more %d", len(got.Models), got.More, len(l.Models), l.More)
	}
	if after, _ := svc.requests(); len(after) != len(before) {
		t.Fatalf("a carried list sent %d requests", len(after)-len(before))
	}
}

// TestGuideFallsBackToTyping (G7): a list answered not found, an empty list
// and a body that is not a list each reach the typed model question, its
// material giving the reason; a name validModel refuses asks again.
func TestGuideFallsBackToTyping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   int
		ids    []string
		body   string
		reason string
	}{
		{"not found", http.StatusNotFound, nil, "", "publishes no model list: it answered not found."},
		{"an empty list", http.StatusOK, []string{}, "", "publishes no model list: it listed no models."},
		{"not a list", http.StatusOK, nil, `{"models":"none"}`, "publishes no model list: it answered with something that is not a model list."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newGuideFake(t, tc.code, tc.ids)
			svc.body = tc.body
			f := newFx(t)
			q := asked(t, last(drive(t, f, GuideRequest{BaseURL: svc.base()}, "lookup")))
			if q.ID != GuideQTyped || q.Typed == "" || !strings.Contains(material(q), tc.reason) {
				t.Fatalf("the fallback asks %q (typed %q): %q; want the reason %q", q.ID, q.Typed, material(q), tc.reason)
			}
			q = asked(t, last(drive(t, f, GuideRequest{BaseURL: svc.base()}, "lookup", "m;rm -rf")))
			if q.ID != GuideQTyped || !strings.Contains(material(q), "is not a model name abcd accepts") {
				t.Fatalf("a name validModel refuses is not asked again: %q %q", q.ID, material(q))
			}
			turns := drive(t, f, GuideRequest{BaseURL: svc.base()}, "lookup", "vendor/typed-model", "none")
			if d := last(turns).Done; d == nil || !strings.Contains(d.Command, "--model vendor/typed-model") {
				t.Fatalf("a typed name ends with %+v", last(turns))
			}
		})
	}
}

// TestGuideNeverOffersAnIdValidModelRefuses: an id the adapter keeps but
// validModel refuses (a shell character inside the bound) is never offered,
// and a resume object carrying one is checked again on the way in, so it
// never reaches the printed command.
func TestGuideNeverOffersAnIdValidModelRefuses(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, []string{"vendor/m;touch-x", "vendor/m$(id)", "vendor/clean"})
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base()}
	turns := drive(t, f, first, "lookup")
	if l := last(turns).Resume.Listed; l == nil || !reflect.DeepEqual(l.Models, []string{"vendor/clean"}) {
		t.Fatalf("the carried list is %+v; want only the clean id", l)
	}
	q := asked(t, last(drive(t, f, first, "lookup", "c")))
	if !reflect.DeepEqual(optionValues(q), []string{"vendor/clean"}) {
		t.Fatalf("narrowing by c offers %q", optionValues(q))
	}

	// A tampered resume object carries the shell id back in: it is checked
	// again on the way in, so it is neither offered nor chosen by name.
	tampered := last(turns)
	tampered.Resume.Listed.Models = []string{"vendor/m;touch-x", "vendor/clean"}
	for _, ans := range []string{"c", "vendor/m;touch-x"} {
		ans := ans
		next, err := turnOf(t, f, &tampered, &ans, first)
		if err != nil {
			t.Fatal(err)
		}
		q := asked(t, next)
		if q.ID != GuideQNarrow || strings.Contains(strings.Join(optionValues(q), " "), ";") {
			t.Fatalf("after the tampered list, %q asks %q offering %q", ans, q.ID, optionValues(q))
		}
		if !reflect.DeepEqual(next.Resume.Listed.Models, []string{"vendor/clean"}) {
			t.Fatalf("the tampered list is carried on as %q", next.Resume.Listed.Models)
		}
	}
	// An id carrying a key, carried back in, is dropped the same way.
	keyed := "vendor/" + keyShapedAnswer(41)
	tampered.Resume.Listed.Models = []string{keyed, "vendor/clean"}
	frag := "vendor"
	next, err := turnOf(t, f, &tampered, &frag, first)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(next); strings.Contains(string(b), keyed) || !reflect.DeepEqual(next.Resume.Listed.Models, []string{"vendor/clean"}) {
		t.Fatalf("a carried id holding a key reaches the turn: %s", b)
	}
}

// TestGuideRefusesAnAnswerItsQuestionDoesNotAdmit: an answer that is none of
// an options question's answers is refused naming the question, and so is a
// resume object whose recorded answers do not fit the questions the replay
// re-derives.
func TestGuideRefusesAnAnswerItsQuestionDoesNotAdmit(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, []string{"vendor/coder"})
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base()}
	turns := drive(t, f, first)
	bad := "maybe"
	if _, err := turnOf(t, f, &turns[0], &bad, first); err == nil || !strings.Contains(err.Error(), `question "lookup"`) {
		t.Fatalf("an answer the look-up does not take = %v", err)
	}
	var refusal *GuideRefusal
	turns = drive(t, f, first, "type", "vendor/coder")
	edited := last(turns)
	edited.Resume.Answers = []GuideAnswer{{ID: GuideQLookup, Value: "type"}, {ID: GuideQHome, Value: "abcd"}}
	ans := "none"
	if _, err := turnOf(t, f, &edited, &ans, first); !errors.As(err, &refusal) || !strings.Contains(err.Error(), `answers "home" where the guide asks "model-name"`) {
		t.Fatalf("an edited resume skipping a question = %v", err)
	}
	edited = last(turns)
	edited.Resume.Answers[1].Value = "bad;name"
	if _, err := turnOf(t, f, &edited, &ans, first); !errors.As(err, &refusal) {
		t.Fatalf("a recorded answer its question refuses = %v", err)
	}
	edited = last(turns)
	edited.Resume.SchemaVersion = 2
	if _, err := turnOf(t, f, &edited, &ans, first); !errors.As(err, &refusal) {
		t.Fatalf("a resume object of another schema = %v", err)
	}
	// The answer is for the question the resume object leaves open, and that
	// must be the question the replay reaches. (The cases above edited the
	// answers turns shares, so the guide is driven afresh.)
	turns = drive(t, f, first, "type", "vendor/coder")
	edited = last(turns)
	edited.Resume.Open = GuideQHome
	if _, err := turnOf(t, f, &edited, &ans, first); !errors.As(err, &refusal) || !strings.Contains(err.Error(), `the answer is for "home", and the guide asks "takes-key"`) {
		t.Fatalf("an answer for another open question = %v", err)
	}
	// More answers than a guided setup takes are refused before any replay.
	edited = last(turns)
	edited.Resume.Answers = nil
	for range maxGuideAnswers + 1 {
		edited.Resume.Answers = append(edited.Resume.Answers, GuideAnswer{ID: GuideQLookup, Value: "type"})
	}
	if _, err := turnOf(t, f, &edited, &ans, first); !errors.As(err, &refusal) || !strings.Contains(err.Error(), "carries 65 answers; a guided setup takes at most 64") {
		t.Fatalf("a resume object carrying 65 answers = %v", err)
	}
	// An answer over the bound is asked again, saying the bound.
	q := asked(t, last(drive(t, f, first, "type", strings.Repeat("a", maxGuideAnswerBytes+1))))
	if q.ID != GuideQTyped || !strings.Contains(material(q), "The answer is 2049 bytes; an answer here is at most 2048.") {
		t.Fatalf("an answer over the bound asks %q: %q", q.ID, material(q))
	}
	// The carried model list is for the address the replay reaches.
	listed := last(drive(t, f, first, "lookup"))
	listed.Resume.Listed.Host = "elsewhere.example"
	frag := "coder"
	if _, err := turnOf(t, f, &listed, &frag, first); !errors.As(err, &refusal) || !strings.Contains(err.Error(), `model list is for "elsewhere.example"`) {
		t.Fatalf("a carried list for another host = %v", err)
	}
}

// TestGuideKeyHomesAndTheCommand: the home question offers exactly the
// store's three homes and decide later; each path ends in one command with
// the paths credential.WritesFor names and then the settings file; the
// external home offers at most three _API_KEY names, the provider's first,
// never a value; a service that lists only for a key prints no --model and
// skips the key question.
func TestGuideKeyHomesAndTheCommand(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, []string{"vendor/coder"})
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base(), EnvNames: []string{"PATH", "ZED_API_KEY", "LOCAL_API_KEY", "ALPHA_API_KEY", "BETA_API_KEY", "bad name_API_KEY"}}
	q := asked(t, last(drive(t, f, first, "type", "vendor/coder", "key")))
	if got := optionValues(q); !reflect.DeepEqual(got, []string{KeyHomeExternal, KeyHomeABCD, KeyHomeKeychain}) || q.Later.Label != "Decide later" {
		t.Fatalf("the home question offers %q and %q", got, q.Later.Label)
	}
	q = asked(t, last(drive(t, f, first, "type", "vendor/coder", "key", "external")))
	if got := optionValues(q); !reflect.DeepEqual(got, []string{"LOCAL_API_KEY", "ALPHA_API_KEY", "BETA_API_KEY"}) {
		t.Fatalf("the variable question offers %q", got)
	}
	// A variable whose name holds a key is never offered.
	keyedName := "ghp_" + testsecret.Synthetic(42, 36) + "_API_KEY"
	keyedFirst := GuideRequest{BaseURL: svc.base(), EnvNames: []string{keyedName, "LOCAL_API_KEY"}}
	q = asked(t, last(drive(t, f, keyedFirst, "type", "vendor/coder", "key", "external")))
	if got := optionValues(q); !reflect.DeepEqual(got, []string{"LOCAL_API_KEY"}) {
		t.Fatalf("with a variable whose name holds a key, the variable question offers %q", got)
	}
	for home, want := range map[string]struct {
		answers []string
		command string
		writes  []string
	}{
		"none":     {[]string{"none"}, "--model vendor/coder --home none", []string{"~/.abcd/config.json"}},
		"abcd":     {[]string{"key", "abcd"}, "--model vendor/coder --home abcd", []string{credential.StorePath, "~/.abcd/config.json"}},
		"keychain": {[]string{"key", "keychain"}, "--model vendor/coder --home keychain", []string{credential.KeychainItem("local"), credential.IndexPath, "~/.abcd/config.json"}},
		"external": {[]string{"key", "external", "MY_API_KEY"}, "--model vendor/coder --home external --env MY_API_KEY", []string{credential.IndexPath, "~/.abcd/config.json"}},
	} {
		d := last(drive(t, f, first, append([]string{"type", "vendor/coder"}, want.answers...)...)).Done
		if d == nil {
			t.Fatalf("%s: no done turn", home)
		}
		if wantCmd := "abcd ahoy connect local --base-url " + svc.base() + " " + want.command; d.Command != wantCmd {
			t.Errorf("%s: command %q, want %q", home, d.Command, wantCmd)
		}
		if !reflect.DeepEqual(d.Writes, want.writes) || d.PicksInTerminal {
			t.Errorf("%s: writes %q (picks %v), want %q", home, d.Writes, d.PicksInTerminal, want.writes)
		}
	}

	keyed := newGuideFake(t, http.StatusUnauthorized, nil)
	turns := drive(t, newFx(t), GuideRequest{BaseURL: keyed.base()}, "lookup")
	q = asked(t, last(turns))
	if q.ID != GuideQHome || !strings.Contains(material(q), "lists its models only for a key") {
		t.Fatalf("a service needing the key to list asks %q: %q", q.ID, material(q))
	}
	d := last(drive(t, newFx(t), GuideRequest{BaseURL: keyed.base()}, "lookup", "keychain")).Done
	if d == nil || !d.PicksInTerminal || strings.Contains(d.Command, "--model") {
		t.Fatalf("a service needing the key to list ends with %+v", d)
	}
}

// TestGuideDecideLaterStops: decide later at any question ends the guide
// with nothing printed but the stop, and the resume object kept, so the next
// turn asks the same question again.
func TestGuideDecideLaterStops(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, []string{"vendor/coder"})
	f := newFx(t)
	first := GuideRequest{BaseURL: svc.base()}
	turns := drive(t, f, first, "type", "Decide later")
	stopped := last(turns)
	if stopped.Stopped != GuideStopped || stopped.Ask != nil || stopped.Done != nil || stopped.Resume.Open != GuideQTyped {
		t.Fatalf("decide later = %+v", stopped)
	}
	again, err := turnOf(t, f, &stopped, nil, first)
	if err != nil || asked(t, again).ID != GuideQTyped {
		t.Fatalf("resuming after a stop = %+v, %v", again, err)
	}
}

// TestGuideNamesTheProvider: the provider is named after the host, local for
// this machine, api and www passed over, with -2 appended while the name is
// taken; a positional name already configured is refused.
func TestGuideNamesTheProvider(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"example":{"base_url":"https://api.example.com/v1","models":["m"]}}}}`)
	g := &guide{cfg: mustLoad(t, f)}
	for host, want := range map[string]string{
		"localhost": "local", "127.0.0.1": "local", "api.example.com": "example-2", "www.openrouter.ai": "openrouter",
		"api.example.co.uk": "co", "192.0.2.7": "provider", "harness.example": "harness-2",
	} {
		if got := g.deriveProvider(host); got != want {
			t.Errorf("deriveProvider(%q) = %q, want %q", host, got, want)
		}
	}
	if _, err := Guide(context.Background(), GuideRequest{Roots: f.roots, Provider: "example", BaseURL: "https://api.example.com/v1"}); err == nil ||
		!strings.Contains(err.Error(), "already configured") {
		t.Fatalf("a positional name already configured = %v", err)
	}
}

func mustLoad(t *testing.T, f *fx) *APIConfig {
	t.Helper()
	c, err := LoadAPI(f.roots)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestGuideQuestionsPassTheAskingLimits: every question the guide asks, on
// every path, passes the structural check and the asking limits.
func TestGuideQuestionsPassTheAskingLimits(t *testing.T) {
	listed := newGuideFake(t, http.StatusOK, threeHundred())
	keyed := newGuideFake(t, http.StatusUnauthorized, nil)
	missing := newGuideFake(t, http.StatusNotFound, nil)
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"alpha":{"base_url":"https://alpha.example.com/v1","models":["vendor1/model-7"]}}}}`)
	env := []string{"ALPHA_API_KEY", "LOCAL_API_KEY"}
	var all []GuideTurn
	for _, path := range []struct {
		first   GuideRequest
		answers []string
	}{
		{GuideRequest{EnvNames: env}, []string{"ftp://nowhere", "http://192.0.2.1/v1", listed.base(), "lookup", "coder", "zzz", "vendor1/model-7", "key", "external", "ALPHA_API_KEY"}},
		{GuideRequest{BaseURL: keyed.base()}, []string{"lookup", "abcd"}},
		{GuideRequest{BaseURL: missing.base()}, []string{"lookup", "bad;name", "vendor/m", "none"}},
		{GuideRequest{BaseURL: missing.base()}, []string{"type", "vendor/m", "key", "keychain"}},
	} {
		all = append(all, drive(t, f, path.first, path.answers...)...)
	}
	n := 0
	for _, turn := range all {
		if turn.Ask == nil {
			continue
		}
		n++
		if fs := question.Check(*turn.Ask); len(fs) != 0 {
			t.Errorf("%s fails the structural check: %v", turn.Ask.Questions[0].ID, fs)
		}
		for _, who := range []question.Person{question.Unnamed, question.Facilitator} {
			for _, fd := range question.CheckLimits(turn.Ask.Fields(), question.Default, question.Addressee{Person: who}) {
				t.Errorf("%s: %s", turn.Ask.Questions[0].ID, fd.String())
			}
		}
	}
	if n < 12 {
		t.Fatalf("only %d questions were checked", n)
	}
}

// keyShapedAnswer is a value of the shape the secret scanner knows, built at
// runtime so no key-shaped literal is committed.
func keyShapedAnswer(seed uint64) string { return "sk-proj-" + testsecret.Synthetic(seed, 48) }

// plainKeyAnswer is a plain sk- key, the shape older OpenAI keys and many
// OpenAI-compatible services issue (iss-2610040202190813), built at runtime.
func plainKeyAnswer(seed uint64) string { return "sk-" + testsecret.Synthetic(seed, 40) }

// openRouterKeyAnswer is an OpenRouter key's shape, built at runtime.
func openRouterKeyAnswer(seed uint64) string { return "sk-or-v1-" + testsecret.SyntheticHex(seed, 64) }

// TestGuideNeverEchoesATypedValueThatIsNotAName: a key pasted where the guide
// asks for a name (the model typed, part of a listed model's name, the
// variable's name, or any other question) is refused with the one message
// that says why, and asked again; no turn after it carries the value, and a
// typed text the question cannot take is never quoted back in the retry.
func TestGuideNeverEchoesATypedValueThatIsNotAName(t *testing.T) {
	const keyMsg = "looks like a key; the guide takes a name, never a key"
	pasted := keyShapedAnswer(31)
	asName := "ghp_" + testsecret.Synthetic(32, 36) // a valid variable name in form
	listed := newGuideFake(t, http.StatusOK, []string{"vendor/coder"})
	f := newFx(t)
	first := GuideRequest{BaseURL: listed.base(), EnvNames: []string{"LOCAL_API_KEY"}}
	for _, tc := range []struct {
		name    string
		before  []string
		bad     string
		id      string
		after   []string
		message string
	}{
		{"the typed model", []string{"type"}, pasted, GuideQTyped, []string{"vendor/coder", "none"}, keyMsg},
		{"part of a listed name", []string{"lookup"}, pasted, GuideQModel, []string{"vendor/coder", "none"}, keyMsg},
		{"an options question", nil, pasted, GuideQLookup, []string{"type", "vendor/coder", "none"}, keyMsg},
		{"the variable's name", []string{"type", "vendor/coder", "key", "external"}, pasted, GuideQEnv, []string{"MY_API_KEY"}, keyMsg},
		{"a key shaped as a name", []string{"type", "vendor/coder", "key", "external"}, asName, GuideQEnv, []string{"MY_API_KEY"}, keyMsg},
		{"a plain sk- key as the typed model", []string{"type"}, plainKeyAnswer(33), GuideQTyped, []string{"vendor/coder", "none"}, keyMsg},
		{"a plain sk- key as part of a listed name", []string{"lookup"}, plainKeyAnswer(34), GuideQModel, []string{"vendor/coder", "none"}, keyMsg},
		{"an OpenRouter key as the typed model", []string{"type"}, openRouterKeyAnswer(35), GuideQTyped, []string{"vendor/coder", "none"}, keyMsg},
		{"a variable name it cannot take", []string{"type", "vendor/coder", "key", "external"}, "not a name!", GuideQEnv, []string{"MY_API_KEY"}, "is not a variable's name"},
		{"a model name it cannot take", []string{"type"}, "bad;name", GuideQTyped, []string{"vendor/coder", "none"}, "is not a model name abcd accepts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			answers := append(append(append([]string{}, tc.before...), tc.bad), tc.after...)
			turns := drive(t, f, first, answers...)
			retry := turns[len(tc.before)+1]
			q := asked(t, retry)
			if q.ID != tc.id || !strings.Contains(material(q), tc.message) {
				t.Fatalf("%q at %q asks %q: %q; want it asked again saying %q", tc.bad, tc.id, q.ID, material(q), tc.message)
			}
			if last(turns).Done == nil {
				t.Fatalf("the guide did not end after the retry: %+v", last(turns))
			}
			for i, turn := range turns[len(tc.before)+1:] {
				b, err := json.Marshal(turn)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), tc.bad) {
					t.Fatalf("turn %d after the answer quotes it back: %s", i, b)
				}
			}
		})
	}
	// A resume object carrying a key as an answer is refused without it.
	turns := drive(t, f, first, "type", "vendor/coder")
	edited := last(turns)
	edited.Resume.Answers[1].Value = pasted
	ans := "none"
	_, err := turnOf(t, f, &edited, &ans, first)
	if err == nil || !strings.Contains(err.Error(), keyMsg) || strings.Contains(err.Error(), pasted) {
		t.Fatalf("a resume object carrying a key = %v", err)
	}
}

// TestShellQuote: a word the shell takes as it is stays bare; a word opening
// with ~ (which the shell expands to a home directory), one holding a quote,
// and the empty word are single-quoted.
func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"vendor/coder": "vendor/coder",
		"a~b":          "a~b",
		"~v/x":         "'~v/x'",
		"it's":         `'it'\''s'`,
		"":             "''",
		"a b":          "'a b'",
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestGuideQuotesAListedIdOpeningWithATilde: a listed id OpenRouter-style,
// opening with ~, reaches the printed command quoted, so a pasted command
// passes it as it is and never as a home directory.
func TestGuideQuotesAListedIdOpeningWithATilde(t *testing.T) {
	svc := newGuideFake(t, http.StatusOK, []string{"~vendor/coder"})
	d := last(drive(t, newFx(t), GuideRequest{BaseURL: svc.base()}, "lookup", "~vendor/coder", "none")).Done
	if d == nil || !strings.Contains(d.Command, "--model '~vendor/coder' ") {
		t.Fatalf("a listed id opening with ~ ends with %+v", d)
	}
}
