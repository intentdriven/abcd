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

// neverEnds answers the chat call with an event stream of 1 MiB keep-alive
// comments, more in all than the stream's byte bound, until the client goes
// away. pause is the wait between two comments: none stands for a runner that
// carries the stream to the bound at once, a pause for one too loaded to
// carry it there before the call's own deadline (iss-2610080243367274).
func neverEnds(pause time.Duration) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return chatOnly(func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		line := ": " + strings.Repeat("z", 1<<20)
		for i := 0; i < 80 && r.Context().Err() == nil; i++ {
			emit(w, line)
			if pause > 0 {
				select {
				case <-time.After(pause):
				case <-r.Context().Done():
					return
				}
			}
		}
	})
}

// totalLimit is what every refusal at the call's total cap names.
const totalLimit = "the call's total limit"

// capWait is how long a test waits for a call bounded by the total cap
// limit: the cap, and a margin of twice the cap for the refusal to come back
// on a loaded runner, so the wait grows with the deadline it waits for.
func capWait(limit time.Duration) time.Duration { return 3 * limit }

// TestEveryStreamFailureIsRefusedWithoutTheKey is the streamed counterpart of
// TestEveryFailureIsRefusedWithoutTheKey.
func TestEveryStreamFailureIsRefusedWithoutTheKey(t *testing.T) {
	big := strings.Repeat("x", 64<<10)
	var flood []string
	for i := 0; i <= MaxResponseBytes/len(big); i++ {
		flood = append(flood, chunkLine("m", big, ""), "")
	}
	// The streams that never end are bounded by the call's own total cap, set
	// short here, and not by how fast the runner carries them: a loaded runner
	// that cannot read the flood to its byte bound sees the cap refuse it
	// instead (iss-2610080243367274).
	const (
		floodCap   = 10 * time.Second
		trickleCap = 2 * time.Second
	)
	cases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
		want    string
		// total, when set, is the client's total cap; the call is then
		// waited for a margin derived from it (capWait), and the cap's own
		// refusal is accepted beside want.
		total time.Duration
	}{
		{"an error event echoing the key", streams(chunkLine("m", `{"verdict":`, ""), "",
			`data: {"error":{"message":"provider overloaded `+testKey+`","code":502}}`, ""), "reported an error", 0},
		{"an event that is not JSON", streams(`data: {"choices": [`, ""), "not a chat completion", 0},
		{"no choice in any chunk", streams(`data: {"model":"m","choices":[]}`, "", "data: [DONE]", ""), "no choice", 0},
		{"no model reported", streams(chunkLine("", `{"verdict":"yes"}`, "stop"), "", "data: [DONE]", ""), "reported no model", 0},
		{"the model changes mid-answer", streams(chunkLine("m", `{"verdict":`, ""), "", chunkLine("other", `"yes"}`, "stop"), "",
			"data: [DONE]", ""), "more than one model", 0},
		{"an assembled answer past the bound", streams(append(flood, chunkLine("m", "", "stop"), "", "data: [DONE]", "")...), "larger than", 0},
		{"one event past the bound", streams("data: "+strings.Repeat("y", MaxResponseBytes+10), ""), "larger than", 0},
		{"a stream that never ends", neverEnds(0), "larger than", floodCap},
		{"a stream that never ends, carried too slowly to reach the bound", neverEnds(500 * time.Millisecond), totalLimit, trickleCap},
		{"an answer that fails the output contract", streams(chunkLine("m", `{"verdict":""}`, "stop"), "", "data: [DONE]", ""), "output contract", 0},
		{"an empty stream", streams(), "ended before the answer was complete", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, tc.handler)
			c, wait := mustClient(t, f.base(), testKey), 30*time.Second
			if tc.total > 0 {
				c, wait = mustClient(t, f.base(), testKey, WithTimeout(tc.total)), capWait(tc.total)
			}
			_, err, _ := completeWithin(t, wait, context.Background(), c, jsonObject)
			if err == nil {
				t.Fatal("Complete succeeded; want a refusal")
			}
			assertNoKey(t, err)
			if !strings.Contains(err.Error(), tc.want) && (tc.total == 0 || !strings.Contains(err.Error(), totalLimit)) {
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
	f := newFake(t, chatOnly(func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) { <-r.Context().Done() }))
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
	f := newFake(t, chatOnly(func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, chunkLine("m", `{"verdict":`, ""), "")
		<-r.Context().Done()
		close(gone)
	}))
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
	f := newFake(t, chatOnly(func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
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
	}))
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
// for a local server reading a large prompt. The event limit is the gateway
// window the record observed (about 600 seconds), well inside the total cap,
// and a failed call's body is waited for briefly.
func TestTheDefaultBoundsOutlastTheOldDeadline(t *testing.T) {
	c := mustClient(t, "https://example.com/v1", testKey)
	got := fmt.Sprintf("%s %s %s %s %s", c.firstByte, c.idle, c.events, c.timeout, c.errorWait)
	if got != "5m0s 2m0s 10m0s 30m0s 30s" {
		t.Fatalf("first-byte, idle, event, total and error-body = %s", got)
	}
}

// keepAlives answers with an event stream that sends the given events and then
// only keep-alive comments, one every 20ms, until the client goes away, which
// it reports on gone.
func keepAlives(gone chan<- struct{}, events ...string) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return chatOnly(func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, events...)
		for {
			select {
			case <-time.After(20 * time.Millisecond):
				emit(w, ": keep-alive")
			case <-r.Context().Done():
				close(gone)
				return
			}
		}
	})
}

// TestAStreamOfOnlyKeepAlivesHitsTheEventLimit: a server that keeps the
// connection alive with comments and never sends an event, before its answer
// begins or in the middle of it, is abandoned at the event limit, long before
// the total cap, and the refusal says it sent only keep-alives.
func TestAStreamOfOnlyKeepAlivesHitsTheEventLimit(t *testing.T) {
	for name, events := range map[string][]string{
		"before any event": nil,
		"after an event":   {chunkLine("m", `{"verdict":`, ""), ""},
	} {
		t.Run(name, func(t *testing.T) {
			gone := make(chan struct{})
			f := newFake(t, keepAlives(gone, events...))
			c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(200*time.Millisecond), WithTimeout(time.Minute))
			c.events = 300 * time.Millisecond
			_, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
			if err == nil {
				t.Fatal("Complete succeeded against a stream of keep-alives")
			}
			assertNoKey(t, err)
			if !strings.Contains(err.Error(), "sent only keep-alives for 300ms") || errors.Is(err, ErrUnreachable) {
				t.Fatalf("error = %v", err)
			}
			if d > 5*time.Second {
				t.Fatalf("the event limit took %s", d)
			}
			select {
			case <-gone:
			case <-time.After(5 * time.Second):
				t.Fatal("the server never saw the client go away after the event limit")
			}
		})
	}
}

// TestKeepAlivesUntilTheTotalCapAreNamedAsSuch: when the total cap ends a
// stream that sent only keep-alives, the refusal says so, rather than saying
// the server was still answering.
func TestKeepAlivesUntilTheTotalCapAreNamedAsSuch(t *testing.T) {
	gone := make(chan struct{})
	f := newFake(t, keepAlives(gone))
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(200*time.Millisecond), WithTimeout(400*time.Millisecond))
	c.events = time.Minute
	_, err, _ := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded against a stream of keep-alives")
	}
	if !strings.Contains(err.Error(), "sent only keep-alives for 400ms") || strings.Contains(err.Error(), "still answering") {
		t.Fatalf("error = %v", err)
	}
}

// TestEventsWithKeepAlivesBetweenThemAreReadToTheEnd: keep-alives between
// events do not count as events, but each event re-arms the event limit, so an
// answer whose events arrive within it is read to the end however long it runs.
func TestEventsWithKeepAlivesBetweenThemAreReadToTheEnd(t *testing.T) {
	const n = 10
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		streamHead(w)
		emit(w, chunkLine("m", `{"verdict":"`, ""), "")
		for i := 0; i < n; i++ {
			for j := 0; j < 3; j++ {
				select {
				case <-time.After(30 * time.Millisecond):
				case <-r.Context().Done():
					return
				}
				emit(w, ": keep-alive")
			}
			emit(w, chunkLine("m", "a", ""), "")
		}
		emit(w, chunkLine("m", `"}`, "stop"), "", "data: [DONE]", "")
	})
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(time.Minute), WithTimeout(time.Minute))
	c.events = 300 * time.Millisecond
	res, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
	if err != nil {
		t.Fatalf("Complete after %s: %v", d, err)
	}
	if want := `{"verdict":"` + strings.Repeat("a", n) + `"}`; string(res.Content) != want {
		t.Fatalf("content = %q", res.Content)
	}
}

// TestTheCallersOwnDeadlineIsNamedAsTheCallers: a caller's context whose
// deadline passes before any of the adapter's limits is named as the caller's
// deadline, never as the adapter's total cap.
func TestTheCallersOwnDeadlineIsNamedAsTheCallers(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) { <-r.Context().Done() })
	c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(time.Minute), WithTimeout(time.Minute))
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err, _ := completeWithin(t, 10*time.Second, ctx, c, jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded past the caller's deadline")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "caller's deadline") || strings.Contains(err.Error(), "1m0s") || errors.Is(err, ErrUnreachable) {
		t.Fatalf("error = %v", err)
	}
}

// TestAFailedStatusIsReportedWithoutWaitingOutItsBody: a non-200 answer is
// reported on its status once its body has had a short while to arrive, never
// after the first-byte limit; a body sent with it is still quoted.
func TestAFailedStatusIsReportedWithoutWaitingOutItsBody(t *testing.T) {
	t.Run("a body that never arrives", func(t *testing.T) {
		f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
			w.WriteHeader(http.StatusInternalServerError)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		})
		c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(time.Minute), WithTimeout(time.Minute))
		c.errorWait = 200 * time.Millisecond
		_, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
		if err == nil || !strings.Contains(err.Error(), "answered HTTP 500") {
			t.Fatalf("error = %v", err)
		}
		if d > 5*time.Second {
			t.Fatalf("the failed status took %s to report", d)
		}
	})
	t.Run("a body that arrives", func(t *testing.T) {
		f := newFake(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"message":"slow down"}}`)
		})
		c := mustClient(t, f.base(), testKey)
		c.errorWait = 200 * time.Millisecond
		_, err, _ := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
		if err == nil || !strings.Contains(err.Error(), "answered HTTP 429: slow down") {
			t.Fatalf("error = %v", err)
		}
	})
}

// TestAnErrorBodyCutOffIsNotQuoted: a failed call's body that is cut off, by
// the wait for it or by its size bound, can end inside an echoed key, where
// the scrub (which matches whole key forms) cannot see it. So a cut-off body
// is never quoted: the status is reported, and the refusal says the body was
// cut off.
func TestAnErrorBodyCutOffIsNotQuoted(t *testing.T) {
	for name, tc := range map[string]struct {
		body  string
		stall bool
	}{
		"cut by the wait": {body: `{"error":{"message":"invalid key ` + testKey[:len(testKey)-6], stall: true},
		"cut by the size": {body: strings.Repeat(" ", maxErrorBodyBytes-20) + testKey},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, tc.body)
				w.(http.Flusher).Flush()
				if tc.stall {
					<-r.Context().Done()
				}
			})
			c := mustClient(t, f.base(), testKey, WithFirstByteTimeout(time.Minute), WithIdleTimeout(time.Minute), WithTimeout(time.Minute))
			c.errorWait = 200 * time.Millisecond
			_, err, _ := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
			if err == nil {
				t.Fatal("Complete succeeded against a 401")
			}
			if strings.Contains(err.Error(), testKey[:12]) {
				t.Fatalf("a prefix of the key reached the error: %v", err)
			}
			if !strings.Contains(err.Error(), "answered HTTP 401") || !strings.Contains(err.Error(), "cut off") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
