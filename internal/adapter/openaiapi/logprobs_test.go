package openaiapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// The answer's log-probabilities (logprobs.go): asked for only by a call made
// with WithTopLogprobs, read from one body or a stream, and handed to the
// caller's sink once the answer is admitted.

// withLogprobs is a completion body whose first choice carries a logprobs
// member, raw as given.
func withLogprobs(model, content, logprobs string) string {
	b, _ := json.Marshal(map[string]any{
		"id": "gen-1", "object": "chat.completion", "model": model,
		"choices": []any{map[string]any{"index": 0, "finish_reason": "length",
			"message":  map[string]any{"role": "assistant", "content": content},
			"logprobs": json.RawMessage(logprobs)}},
	})
	return string(b)
}

func answers(body string) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}
}

const twoAlternatives = `{"content":[{"token":"3","logprob":-0.2,"top_logprobs":[{"token":"3","logprob":-0.2},` +
	`{"token":" 2","logprob":-1.8},{"token":"x","logprob":0.5}]}]}`

func TestACallThatAsksSendsTheFieldsAndReadsTheFirstTokensAlternatives(t *testing.T) {
	f := newFake(t, answers(withLogprobs("small/model", "3", twoAlternatives)))
	var got []TokenLogprob
	carried := false
	c := mustClient(t, f.base(), testKey, WithTopLogprobs(50, func(top []TokenLogprob, ok bool) { got, carried = top, ok }))
	if _, err := c.Complete(context.Background(), request(), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	body := f.last.Load().body
	if string(body["logprobs"]) != "true" || string(body["top_logprobs"]) != "20" {
		t.Fatalf("the call asks for log-probabilities, at most %d alternatives: %v", MaxTopLogprobs, body)
	}
	// The positive log-probability does not read and is dropped.
	if !carried || len(got) != 2 || got[0] != (TokenLogprob{Token: "3", Logprob: -0.2}) || got[1].Token != " 2" {
		t.Fatalf("alternatives = %+v (carried %v)", got, carried)
	}
}

func TestACallThatAsksReadsTheAlternativesFromAStream(t *testing.T) {
	ev := func(content, finish, logprobs string) string {
		ch := map[string]any{"index": 0, "delta": map[string]any{"content": content}}
		if finish != "" {
			ch["finish_reason"] = finish
		}
		if logprobs != "" {
			ch["logprobs"] = json.RawMessage(logprobs)
		}
		b, _ := json.Marshal(map[string]any{"model": "small/model", "choices": []any{ch}})
		return "data: " + string(b)
	}
	f := newFake(t, streams(ev("yes", "", `{"content":[{"token":"yes","logprob":-0.1,"top_logprobs":[{"token":"yes","logprob":-0.1},{"token":"no","logprob":-2.4}]}]}`),
		ev("", "length", `{"content":[{"token":"later","logprob":-0.01,"top_logprobs":[]}]}`), "data: [DONE]"))
	var got []TokenLogprob
	c := mustClient(t, f.base(), testKey, WithTopLogprobs(5, func(top []TokenLogprob, _ bool) { got = top }))
	if _, err := c.Complete(context.Background(), request(), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(got) != 2 || got[0].Token != "yes" || got[1].Token != "no" {
		t.Fatalf("the first token's alternatives, not a later token's: %+v", got)
	}
}

func TestAnAnswerWithNoLogprobsSaysSo(t *testing.T) {
	f := newFake(t, ok("small/model", "3"))
	called, carried := false, true
	c := mustClient(t, f.base(), testKey, WithTopLogprobs(5, func(_ []TokenLogprob, ok bool) { called, carried = true, ok }))
	if _, err := c.Complete(context.Background(), request(), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !called || carried {
		t.Fatalf("the sink hears that the answer carried none: called %v carried %v", called, carried)
	}
}

// A call made without the option sends no logprobs field, and a logprobs
// member of any shape in its answer is not read and refuses nothing.
func TestACallThatDoesNotAskIsUnchanged(t *testing.T) {
	f := newFake(t, answers(withLogprobs("small/model", `{"verdict":"yes"}`, `"not the protocol's shape"`)))
	c := mustClient(t, f.base(), testKey)
	res, err := c.Complete(context.Background(), request(), jsonObject)
	if err != nil || string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("Complete = %q, %v", res.Content, err)
	}
	body := f.last.Load().body
	if _, ok := body["logprobs"]; ok {
		t.Fatalf("a call that did not ask sends no logprobs field: %v", body)
	}
	if _, ok := body["top_logprobs"]; ok {
		t.Fatalf("a call that did not ask sends no top_logprobs field: %v", body)
	}
}

// A token the provider echoes the key in is scrubbed before the caller sees
// it.
func TestAnAlternativesTokenIsScrubbedOfTheKey(t *testing.T) {
	full := `{"content":[{"token":"a","logprob":-0.5,"top_logprobs":[{"token":"` + testKey + `","logprob":-0.5}]}]}`
	f := newFake(t, answers(withLogprobs("small/model", "a", full)))
	var got []TokenLogprob
	c := mustClient(t, f.base(), testKey, WithTopLogprobs(5, func(top []TokenLogprob, _ bool) { got = top }))
	if _, err := c.Complete(context.Background(), request(), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	for _, g := range got {
		if strings.Contains(g.Token, testKey) {
			t.Fatalf("the key reached the caller: %+v", got)
		}
	}
}
