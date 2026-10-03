package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The tests in this file drive the streamed call (iss-2610030931521214): the
// adapter asks for a stream, assembles the server-sent events into the answer
// the contract judges, and bounds the call with a first-byte limit, an idle
// limit between chunks and a total cap instead of one end-to-end deadline. Every
// duration is a few hundred milliseconds at most, standing in for minutes.

// chunkLine is one server-sent event carrying a chat-completion chunk: content
// as the delta (none when empty) and finish as the finish reason (null when
// empty).
func chunkLine(model, content, finish string) string {
	delta := map[string]any{}
	if content != "" {
		delta["content"] = content
	}
	var fr any
	if finish != "" {
		fr = finish
	}
	b, _ := json.Marshal(map[string]any{
		"id": "gen-1", "object": "chat.completion.chunk", "model": model,
		"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": fr}},
	})
	return "data: " + string(b)
}

// streamHead starts an event stream and returns the writer's flusher.
func streamHead(w http.ResponseWriter) http.Flusher {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fl := w.(http.Flusher)
	fl.Flush()
	return fl
}

// emit writes lines to the stream, each ending in a newline, and flushes.
func emit(w http.ResponseWriter, lines ...string) {
	for _, l := range lines {
		_, _ = io.WriteString(w, l+"\n")
	}
	w.(http.Flusher).Flush()
}

// streams answers every call with lines, as one event stream.
func streams(lines ...string) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, lines...)
	}
}

// completeWithin runs the call and fails the test when it has not returned
// within limit, so a timeout that does not fire fails fast instead of hanging.
func completeWithin(t *testing.T, limit time.Duration, ctx context.Context, c *Client, contract func([]byte) error) (Result, error, time.Duration) {
	t.Helper()
	type out struct {
		res Result
		err error
	}
	done := make(chan out, 1)
	start := time.Now()
	go func() {
		res, err := c.Complete(ctx, request(), contract)
		done <- out{res, err}
	}()
	select {
	case o := <-done:
		return o.res, o.err, time.Since(start)
	case <-time.After(limit):
		t.Fatalf("the call had not returned after %s", limit)
	}
	return Result{}, nil, 0
}

// TestCompleteAsksForAStreamAndAssemblesIt: the request asks for a stream and
// says it reads one, and the events are assembled into the answer the
// contract judges: the content deltas joined, the model and the finish reason
// read, keep-alive comments, blank lines, other fields and a data line with no
// space after its colon all tolerated.
func TestCompleteAsksForAStreamAndAssemblesIt(t *testing.T) {
	var accept string
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		accept = r.Header.Get("Accept")
		streamHead(w)
		emit(w,
			": keep-alive",
			"",
			"event: message",
			chunkLine("typesafe/jev-1.13-20260915", "", ""),
			"",
			chunkLine("typesafe/jev-1.13-20260915", `{"verdict":`, ""),
			"",
			strings.Replace(chunkLine("typesafe/jev-1.13-20260915", `"yes"}`, ""), "data: ", "data:", 1),
			"",
			": still here",
			chunkLine("typesafe/jev-1.13-20260915", "", "stop"),
			"",
			"data: [DONE]",
			"",
		)
	})
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("content = %q", res.Content)
	}
	if res.ModelReported != "typesafe/jev-1.13-20260915" || res.ModelAsked != "typesafe/jev-1.13" {
		t.Fatalf("models: asked %q, reported %q", res.ModelAsked, res.ModelReported)
	}
	if res.FinishReason != "stop" {
		t.Fatalf("finish reason = %q", res.FinishReason)
	}
	if got := string(f.last.Load().body["stream"]); got != "true" {
		t.Fatalf("stream = %s, want true", got)
	}
	if !strings.Contains(accept, "text/event-stream") {
		t.Fatalf("Accept = %q, want it to name text/event-stream", accept)
	}
}

// TestAPlainJSONAnswerToAStreamRequestIsAccepted: a server that answers a
// streaming request with one chat-completion body is read as before.
func TestAPlainJSONAnswerToAStreamRequestIsAccepted(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"model":"m","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"{\"verdict\":\"yes\"}"}}],`+
			`"usage":{"prompt_tokens":11,"completion_tokens":5,"total_tokens":16}}`)
	})
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(f.last.Load().body["stream"]) != "true" {
		t.Fatal("the request did not ask for a stream")
	}
	if string(res.Content) != `{"verdict":"yes"}` || res.FinishReason != "stop" {
		t.Fatalf("content %q, finish %q", res.Content, res.FinishReason)
	}
	if res.Usage == nil || res.Usage.TotalTokens != 16 || res.Usage.PromptTokens != 11 || res.Usage.CompletionTokens != 5 {
		t.Fatalf("usage = %+v", res.Usage)
	}
}

// TestUsageInTheLastChunkIsRead: a server reporting usage sends it in a final
// chunk with no choice; that chunk is read for the usage and is not mistaken
// for an answer with no choice.
func TestUsageInTheLastChunkIsRead(t *testing.T) {
	f := newFake(t, streams(
		chunkLine("m", `{"verdict":"no"}`, ""), "",
		chunkLine("m", "", "stop"), "",
		`data: {"id":"gen-1","object":"chat.completion.chunk","model":"m","choices":[],"usage":{"prompt_tokens":40,"completion_tokens":7,"total_tokens":47}}`, "",
		"data: [DONE]", "",
	))
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(res.Content) != `{"verdict":"no"}` {
		t.Fatalf("content = %q", res.Content)
	}
	if res.Usage == nil || res.Usage.PromptTokens != 40 || res.Usage.CompletionTokens != 7 || res.Usage.TotalTokens != 47 {
		t.Fatalf("usage = %+v", res.Usage)
	}
}

// TestAStreamWithoutDone: a stream that closes after a finish reason with no
// [DONE] is a complete answer; one that closes before any finish reason was cut
// off mid-answer, and is refused rather than used.
func TestAStreamWithoutDone(t *testing.T) {
	finished := newFake(t, streams(chunkLine("m", `{"verdict":"yes"}`, ""), "", chunkLine("m", "", "length"), ""))
	res, err := mustClient(t, finished.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("a finished stream with no [DONE]: %v", err)
	}
	if string(res.Content) != `{"verdict":"yes"}` || res.FinishReason != "length" {
		t.Fatalf("content %q, finish %q", res.Content, res.FinishReason)
	}

	cut := newFake(t, streams(chunkLine("m", `{"verdict":"yes"}`, ""), ""))
	_, err = mustClient(t, cut.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err == nil {
		t.Fatal("a stream cut off before any finish reason was accepted")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "ended before the answer was complete") {
		t.Fatalf("error = %v", err)
	}
}

// TestEveryStreamFailureIsRefusedWithoutTheKey is the streamed counterpart of
// TestEveryFailureIsRefusedWithoutTheKey.
func TestEveryStreamFailureIsRefusedWithoutTheKey(t *testing.T) {
	big := strings.Repeat("x", 64<<10)
	var flood []string
	for i := 0; i <= MaxResponseBytes/len(big); i++ {
		flood = append(flood, chunkLine("m", big, ""), "")
	}
	cases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
		want    string
	}{
		{"an error event echoing the key", streams(chunkLine("m", `{"verdict":`, ""), "",
			`data: {"error":{"message":"provider overloaded `+testKey+`","code":502}}`, ""), "reported an error"},
		{"an event that is not JSON", streams(`data: {"choices": [`, ""), "not a chat completion"},
		{"no choice in any chunk", streams(`data: {"model":"m","choices":[]}`, "", "data: [DONE]", ""), "no choice"},
		{"no model reported", streams(chunkLine("", `{"verdict":"yes"}`, "stop"), "", "data: [DONE]", ""), "reported no model"},
		{"the model changes mid-answer", streams(chunkLine("m", `{"verdict":`, ""), "", chunkLine("other", `"yes"}`, "stop"), "",
			"data: [DONE]", ""), "more than one model"},
		{"an assembled answer past the bound", streams(append(flood, chunkLine("m", "", "stop"), "", "data: [DONE]", "")...), "larger than"},
		{"one event past the bound", streams("data: "+strings.Repeat("y", MaxResponseBytes+10), ""), "larger than"},
		{"a stream that never ends", func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
			streamHead(w)
			line := ": " + strings.Repeat("z", 1<<20)
			for i := 0; i < 80 && r.Context().Err() == nil; i++ {
				emit(w, line)
			}
		}, "larger than"},
		{"an answer that fails the output contract", streams(chunkLine("m", `{"verdict":""}`, "stop"), "", "data: [DONE]", ""), "output contract"},
		{"an empty stream", streams(), "ended before the answer was complete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, tc.handler)
			_, err, _ := completeWithin(t, 30*time.Second, context.Background(), mustClient(t, f.base(), testKey), jsonObject)
			if err == nil {
				t.Fatal("Complete succeeded; want a refusal")
			}
			assertNoKey(t, err)
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to name %q", err, tc.want)
			}
			if errors.Is(err, ErrUnreachable) {
				t.Fatalf("a server that answered is not unreachable: %v", err)
			}
			if len(err.Error()) > 1024 {
				t.Fatalf("error is %d bytes; a provider's stream must not flood it", len(err.Error()))
			}
		})
	}
}

// TestAKeySplitAcrossChunksIsScrubbed: the scrub judges the assembled answer,
// so a key the provider echoes across two deltas does not survive.
func TestAKeySplitAcrossChunksIsScrubbed(t *testing.T) {
	half := len(testKey) / 2
	f := newFake(t, streams(
		chunkLine("m", "echo "+testKey[:half], ""), "",
		chunkLine("m", testKey[half:]+" done", "stop"), "",
		"data: [DONE]", "",
	))
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if strings.Contains(string(res.Content), testKey) || !strings.Contains(string(res.Content), "[credential]") {
		t.Fatalf("content = %q", res.Content)
	}
}

// TestASlowFirstTokenWithinTheFirstByteLimitIsWaitedFor: a server that takes
// longer than the idle limit to start answering (reading a long prompt) is
// waited for up to the first-byte limit.
func TestASlowFirstTokenWithinTheFirstByteLimitIsWaitedFor(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		select {
		case <-time.After(500 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		streams(chunkLine("m", `{"verdict":"yes"}`, "stop"), "", "data: [DONE]", "")(w, r, nil)
	})
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(5*time.Second), WithIdleTimeout(150*time.Millisecond), WithTimeout(10*time.Second))
	res, err, _ := completeWithin(t, 15*time.Second, context.Background(), c, jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("content = %q", res.Content)
	}
}

// TestNoFirstByteWithinTheLimitIsRefused: a server that takes the brief and
// sends nothing is abandoned at the first-byte limit, long before the total
// cap, and is not unreachable (the brief was sent).
func TestNoFirstByteWithinTheLimitIsRefused(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) { <-r.Context().Done() })
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(200*time.Millisecond), WithIdleTimeout(time.Minute), WithTimeout(time.Minute))
	_, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded against a server that never answered")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "no answer within 200ms") || errors.Is(err, ErrUnreachable) {
		t.Fatalf("error = %v", err)
	}
	if d > 5*time.Second {
		t.Fatalf("the first-byte limit took %s", d)
	}
}

// TestSteadyChunksPastTheIdleLimitAreReadToTheEnd: an answer that streams for
// many times the idle limit, a chunk arriving within it each time, is read to
// the end; only silence counts against the idle limit, never the call's age.
func TestSteadyChunksPastTheIdleLimitAreReadToTheEnd(t *testing.T) {
	const n = 25
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, chunkLine("m", `{"verdict":"`, ""), "")
		for i := 0; i < n; i++ {
			select {
			case <-time.After(40 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
			emit(w, chunkLine("m", "a", ""), "")
		}
		emit(w, chunkLine("m", `"}`, "stop"), "", "data: [DONE]", "")
	})
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(150*time.Millisecond), WithIdleTimeout(150*time.Millisecond), WithTimeout(20*time.Second))
	res, err, d := completeWithin(t, 25*time.Second, context.Background(), c, jsonObject)
	if err != nil {
		t.Fatalf("Complete after %s: %v", d, err)
	}
	if want := `{"verdict":"` + strings.Repeat("a", n) + `"}`; string(res.Content) != want {
		t.Fatalf("content = %q", res.Content)
	}
	if d < 4*150*time.Millisecond {
		t.Fatalf("the stream took %s, not long enough to prove the idle limit is per chunk", d)
	}
}

// TestAMidStreamStallHitsTheIdleLimit: a server that stops sending mid-answer
// is abandoned at the idle limit, and sees the client go away.
func TestAMidStreamStallHitsTheIdleLimit(t *testing.T) {
	gone := make(chan struct{})
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, chunkLine("m", `{"verdict":`, ""), "")
		<-r.Context().Done()
		close(gone)
	})
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(200*time.Millisecond), WithTimeout(time.Minute))
	_, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded against a stream that stalled")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "sent nothing for 200ms") || errors.Is(err, ErrUnreachable) {
		t.Fatalf("error = %v", err)
	}
	if d > 5*time.Second {
		t.Fatalf("the idle limit took %s", d)
	}
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the server never saw the client go away after the idle limit")
	}
}

// TestTheTotalCapBoundsASteadyStream: a stream that never stops sending, so
// the idle limit never fires, is abandoned at the total cap.
func TestTheTotalCapBoundsASteadyStream(t *testing.T) {
	gone := make(chan struct{})
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		for {
			select {
			case <-time.After(20 * time.Millisecond):
				emit(w, chunkLine("m", "a", ""), "")
			case <-r.Context().Done():
				close(gone)
				return
			}
		}
	})
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(200*time.Millisecond), WithTimeout(500*time.Millisecond))
	_, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded against a stream that never ended")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "total limit of 500ms") || errors.Is(err, ErrUnreachable) {
		t.Fatalf("error = %v", err)
	}
	if d > 5*time.Second {
		t.Fatalf("the total cap took %s", d)
	}
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the server never saw the client go away after the total cap")
	}
}

// TestACancelledCallReachesTheServer: cancelling the caller's context mid-
// stream closes the connection, so the server sees the client go away and can
// stop generating.
func TestACancelledCallReachesTheServer(t *testing.T) {
	sent := make(chan struct{})
	gone := make(chan struct{})
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, chunkLine("m", `{"verdict":`, ""), "")
		close(sent)
		<-r.Context().Done()
		close(gone)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-sent
		cancel()
	}()
	_, err, _ := completeWithin(t, 10*time.Second, ctx, mustClient(t, f.base(), testKey), jsonObject)
	if err == nil {
		t.Fatal("a cancelled call succeeded")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "was cancelled") || errors.Is(err, ErrUnreachable) {
		t.Fatalf("error = %v", err)
	}
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the server never saw the cancelled client go away")
	}
}

// TestTheDefaultBoundsOutlastTheOldDeadline pins the defaults the record
// grounds: a reasoning model's answer runs past ten minutes, so the total cap
// is well beyond it, and the first-byte and idle limits are each long enough
// for a local server reading a large prompt.
func TestTheDefaultBoundsOutlastTheOldDeadline(t *testing.T) {
	c := mustClient(t, "https://example.com/v1", testKey)
	got := fmt.Sprintf("%s %s %s", c.firstByte, c.idle, c.timeout)
	if got != "5m0s 2m0s 30m0s" {
		t.Fatalf("first-byte, idle and total = %s", got)
	}
}
