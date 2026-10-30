package judge

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The second candidate (logprob.go): a small text model scored by the
// log-probability of each option's answer token, reached by a route whose
// provider block names the model in logprob_models, and held to the same
// allowlist, denylist and window as the first. No test reaches a network: the
// provider is an httptest fake.

// lpFake is an OpenAI-compatible chat endpoint answering reply, counting the
// chat calls and keeping the last request's body and authorization. The model
// listing a call reads first is answered not found and is not counted.
type lpFake struct {
	srv   *httptest.Server
	calls atomic.Int32
	body  atomic.Value
	auth  atomic.Value
}

func newLPFake(t *testing.T, code int, reply string) *lpFake {
	t.Helper()
	f := &lpFake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		f.calls.Add(1)
		raw, _ := io.ReadAll(r.Body)
		f.body.Store(string(raw))
		f.auth.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *lpFake) sent(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	raw, _ := f.body.Load().(string)
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("the request body: %v", err)
	}
	return m
}

// lpAnswer is a chat completion from model whose first choice carries
// logprobs, raw ("" for none).
func lpAnswer(model, content, logprobs string) string {
	ch := map[string]any{"index": 0, "finish_reason": "length", "message": map[string]any{"role": "assistant", "content": content}}
	if logprobs != "" {
		ch["logprobs"] = json.RawMessage(logprobs)
	}
	b, _ := json.Marshal(map[string]any{"model": model, "choices": []any{ch}})
	return string(b)
}

// lpRoots writes a machine configuration naming provider lp at base, with the
// block's extra fields, and judgements routing embark-lesson-ranking as route.
func lpRoots(t *testing.T, base, block, route string) layered.Roots {
	t.Helper()
	roots := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	p := abcdhome.Path(roots.Home, "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"oracle":{"api":{"lp":{"base_url":"` + base + `","models":["small/model"]` + block + `}},` +
		`"judgements":{"embark-lesson-ranking":"` + route + `"}}}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return roots
}

func lessonQ() Question {
	return Question{Type: EmbarkLessonRanking, Kind: KindScore, Ask: "How far does this lesson apply?", Min: 0, Max: 4}
}

const marked = `,"logprob_models":["small/model"]`

// The option is the most probable answer token's, its probability normalised
// over the options; an alternative naming no option counts for none of them.
func TestTheLogprobCandidatePicksTheMostProbableOptionNormalised(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model-0901", "3", `{"content":[{"token":"3","logprob":-0.2,"top_logprobs":[`+
		`{"token":"3","logprob":-0.2},{"token":" 2","logprob":-1.8},{"token":"Three","logprob":-2.5}]}]}`))
	rt, err := Load(lpRoots(t, f.srv.URL+"/v1", marked, "lp/small/model"), "")
	if err != nil {
		t.Fatal(err)
	}
	a, rc, err := rt.Decide(context.Background(), State{Text: "the lesson"}, lessonQ(), Pending())
	if err != nil {
		t.Fatalf("Decide: %v (receipt %+v)", err, rc)
	}
	want := math.Exp(-0.2) / (math.Exp(-0.2) + math.Exp(-1.8))
	if rc.Route != RouteAdapter || a.Value != "3" || math.Abs(a.Probability-want) > 1e-9 || a.Model != "small/model-0901" {
		t.Fatalf("answer %+v receipt %+v, want 3 at %v", a, rc, want)
	}
	sent := f.sent(t)
	if string(sent["logprobs"]) != "true" || string(sent["top_logprobs"]) != strconv.Itoa(openaiapi.MaxTopLogprobs) ||
		string(sent["max_tokens"]) != "1" || !strings.Contains(string(sent["messages"]), "one digit from 0 to 4") {
		t.Fatalf("the request asks one token and its log-probabilities: %v", sent)
	}
}

// A choose question's options are labelled by letter, and the letter's
// probability is the option's.
func TestTheLogprobCandidateReadsAChooseQuestionByItsLetters(t *testing.T) {
	q := dupQuestion()
	ls, err := labels(q)
	if err != nil {
		t.Fatal(err)
	}
	a, err := fromLogprobs(q, ls, "B", []openaiapi.TokenLogprob{{Token: "B", Logprob: -0.1}, {Token: " a", Logprob: -2.0}, {Token: "b", Logprob: -3.0}})
	if err != nil || a.Value != q.Options[1] || a.Probability <= 0.5 || a.Probability >= 1 {
		t.Fatalf("answer %+v, err %v", a, err)
	}
	if !strings.Contains(renderLabelled(State{Text: "s"}, q, ls), "B. "+q.Options[1]) {
		t.Fatal("the options are labelled by letter")
	}
	many := Question{Type: DuplicateMatch, Kind: KindChoose, Ask: "which", Options: make([]string, 27)}
	for i := range many.Options {
		many.Options[i] = "o" + strconv.Itoa(i)
	}
	if _, err := labels(many); err == nil {
		t.Fatal("27 options cannot each be one letter")
	}
	if _, err := labels(Question{Type: EmbarkLessonRanking, Kind: KindScore, Ask: "how far", Min: 0, Max: 10}); err == nil {
		t.Fatal("a score of 10 is not one digit")
	}
}

// No alternative names an option: no answer, so a type turned on falls back
// to the host with a receipt, never a guess.
func TestTheLogprobCandidateWithNoOptionTokenIsAFailure(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model", "The", `{"content":[{"token":"The","logprob":-0.1,"top_logprobs":[`+
		`{"token":"The","logprob":-0.1},{"token":"I","logprob":-2.3}]}]}`))
	rt, err := Load(lpRoots(t, f.srv.URL+"/v1", marked, "lp/small/model"), "")
	if err != nil {
		t.Fatal(err)
	}
	a, rc, err := rt.Decide(context.Background(), State{Text: "the lesson"}, lessonQ(), Answered("1"))
	if err != nil || rc.Route != RouteHost || a.Value != "1" || !strings.Contains(rc.Fallback, "none of the 2 alternatives") {
		t.Fatalf("the host decides with a receipt: answer %+v receipt %+v err %v", a, rc, err)
	}
	if f.calls.Load() != 1 {
		t.Fatalf("asked once, got %d calls", f.calls.Load())
	}
}

// An answer with no logprobs member is no answer either.
func TestTheLogprobCandidateWithNoLogprobsIsAFailure(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model", "3", ""))
	rt, err := Load(lpRoots(t, f.srv.URL+"/v1", marked, "lp/small/model"), "")
	if err != nil {
		t.Fatal(err)
	}
	a, rc, err := rt.Decide(context.Background(), State{Text: "the lesson"}, lessonQ(), Answered("2"))
	if err != nil || rc.Route != RouteHost || a.Value != "2" || !strings.Contains(rc.Fallback, "no log-probabilities") {
		t.Fatalf("the host decides with a receipt: answer %+v receipt %+v err %v", a, rc, err)
	}
}

// Every call passes the provider's models list and the denylist first: a
// marked model the block does not list, and a route to an unlisted model, are
// refused when the configuration is read; a target built anywhere is refused
// by the call before it is sent; a reported model the denylist names is
// discarded. None reaches the fake but the last.
func TestTheLogprobCandidateIsRefusedByTheAllowlistBeforeAnyCall(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model", "3", `{"content":[{"token":"3","logprob":-0.1}]}`))
	if _, err := Load(lpRoots(t, f.srv.URL+"/v1", `,"logprob_models":["other/model"]`, "host"), ""); err == nil ||
		!strings.Contains(err.Error(), "not on the block's models list") {
		t.Fatalf("a marked model the block does not list: %v", err)
	}
	if _, err := Load(lpRoots(t, f.srv.URL+"/v1", marked, "lp/other/model"), ""); err == nil ||
		!strings.Contains(err.Error(), "refused before any call") {
		t.Fatalf("a route to an unlisted model: %v", err)
	}
	roots := lpRoots(t, f.srv.URL+"/v1", marked, "host")
	cfg, err := oracle.LoadAPI(roots)
	if err != nil {
		t.Fatal(err)
	}
	unlisted := oracle.Target{Provider: "lp", Model: "other/model", Mode: oracle.JudgementOn}
	if cfg.ScoresByLogprob(unlisted) {
		t.Fatal("an unlisted target is never scored by log-probability")
	}
	if _, err := (LogprobAdapter{Calls: cfg, Target: unlisted}).Judge(context.Background(), State{Text: "s"}, lessonQ()); err == nil ||
		!strings.Contains(err.Error(), "refused before any call") {
		t.Fatalf("a target built anywhere is refused by the call: %v", err)
	}
	if f.calls.Load() != 0 {
		t.Fatalf("a refused route reached the provider %d time(s)", f.calls.Load())
	}
	evil := newLPFake(t, 200, lpAnswer("small/model-evil", "3", `{"content":[{"token":"3","logprob":-0.1}]}`))
	denied := lpRoots(t, evil.srv.URL+"/v1", marked, "host")
	p := abcdhome.Path(denied.Home, "config.json")
	raw, _ := os.ReadFile(p)
	if err := os.WriteFile(p, []byte(strings.Replace(string(raw), `{"oracle":{`, `{"oracle":{"denylist":["small/model-evil"],`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	dcfg, err := oracle.LoadAPI(denied)
	if err != nil {
		t.Fatal(err)
	}
	tgt := oracle.Target{Provider: "lp", Model: "small/model", Mode: oracle.JudgementOn}
	if _, err := (LogprobAdapter{Calls: dcfg, Target: tgt}).Judge(context.Background(), State{Text: "s"}, lessonQ()); err == nil ||
		!strings.Contains(err.Error(), "discarded") {
		t.Fatalf("a reported model the denylist names is discarded: %v", err)
	}
}

// A state larger than the window is not sent to the second candidate either.
func TestTheLogprobCandidateIsNotSentAStateOverTheWindow(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model", "3", `{"content":[{"token":"3","logprob":-0.1}]}`))
	rt, err := Load(lpRoots(t, f.srv.URL+"/v1", marked, "lp/small/model"), "")
	if err != nil {
		t.Fatal(err)
	}
	rt.Window = 10
	a, rc, err := rt.Decide(context.Background(), State{Text: strings.Repeat("word ", 100)}, lessonQ(), Answered("0"))
	if err != nil || rc.Route != RouteHost || a.Value != "0" || !strings.Contains(rc.Fallback, "window") || f.calls.Load() != 0 {
		t.Fatalf("over the window: answer %+v receipt %+v err %v calls %d", a, rc, err, f.calls.Load())
	}
}

// The key resolves by name through the credential store, as the first
// candidate's does, and no failure carries it.
func TestTheLogprobCandidateSendsTheKeyByNameAndNeverEchoesIt(t *testing.T) {
	const key = "sk-or-v1-0123456789abcdef-not-a-real-key"
	f := newLPFake(t, 401, `{"error":{"message":"bad key `+key+`"}}`)
	roots := lpRoots(t, f.srv.URL+"/v1", `,"key":"lp"`+marked, "host")
	if _, err := credential.SetMachine(roots.Home, "lp", key); err != nil {
		t.Fatal(err)
	}
	cfg, err := oracle.LoadAPI(roots)
	if err != nil {
		t.Fatal(err)
	}
	tgt := oracle.Target{Provider: "lp", Model: "small/model", Mode: oracle.JudgementOn}
	_, err = LogprobAdapter{Calls: cfg, Creds: credential.Machine(roots.Home), Target: tgt}.Judge(context.Background(), State{Text: "s"}, lessonQ())
	if err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("a refused call fails without the key: %v", err)
	}
	if auth, _ := f.auth.Load().(string); auth != "Bearer "+key {
		t.Fatal("the key named by the block was not sent")
	}
}

// A route whose block does not mark its model is the first candidate, as
// before.
func TestAnUnmarkedRouteIsTheFirstCandidate(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model", `{"value":"3","probability":0.9}`, ""))
	rt, err := Load(lpRoots(t, f.srv.URL+"/v1", "", "lp/small/model"), "")
	if err != nil {
		t.Fatal(err)
	}
	a, rc, err := rt.Decide(context.Background(), State{Text: "the lesson"}, lessonQ(), Pending())
	if err != nil || rc.Route != RouteAdapter || a.Value != "3" || a.Probability != 0.9 {
		t.Fatalf("answer %+v receipt %+v err %v", a, rc, err)
	}
	if _, ok := f.sent(t)["logprobs"]; ok {
		t.Fatal("the first candidate asks for no log-probabilities")
	}
}

// The model's own answer is its first token. One that is none of the labels
// is no answer, even when a label sits among the alternatives below it: the
// candidate never promotes an alternative over what the model said.
func TestTheLogprobCandidateRefusesAnOffLabelAnswerWhateverTheAlternatives(t *testing.T) {
	f := newLPFake(t, 200, lpAnswer("small/model", "no", `{"content":[{"token":"no","logprob":-0.01,"top_logprobs":[`+
		`{"token":"no","logprob":-0.01},{"token":"C","logprob":-7.0}]}]}`))
	cfg, err := oracle.LoadAPI(lpRoots(t, f.srv.URL+"/v1", marked, "host"))
	if err != nil {
		t.Fatal(err)
	}
	q := Question{Type: EligibilityResidual, Kind: KindChoose, Ask: "Which hand-back, if any?",
		Options: []string{"no", "user-visible", "trust-rule"}}
	tgt := oracle.Target{Provider: "lp", Model: "small/model", Mode: oracle.JudgementOn}
	a, err := LogprobAdapter{Calls: cfg, Target: tgt}.Judge(context.Background(), State{Text: "s"}, q)
	if err == nil || !strings.Contains(err.Error(), `answered "no"`) || !strings.Contains(err.Error(), "A, B, C") {
		t.Fatalf("an off-label answer is no answer: answer %+v, err %v", a, err)
	}
	if f.calls.Load() != 1 {
		t.Fatalf("asked once, got %d calls", f.calls.Load())
	}
}
