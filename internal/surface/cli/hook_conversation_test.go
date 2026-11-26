package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/question"
)

// hook_conversation_test.go — iss-2611260602137306: a prompt the person types
// while away has that session read them as present for a while, at the stop
// and at a question, and a wake's prompt or the host's own opens nothing.

// promptAt runs the prompt hook for session on prompt and returns the context
// it hands the agent.
func promptAt(t *testing.T, session, root, prompt string) string {
	t.Helper()
	so, se, err := runSplit(t, awayPromptPayload(session, root, prompt), "hook", "prompt-router")
	if err != nil {
		t.Fatalf("prompt hook: %v\n%s", err, se)
	}
	return decodePromptHook(t, so).Context
}

const conversationMark = "this session reads them as present"

// TestPromptOpensConversationWindow: the person's prompt opens the window, so
// the next stop with work owed is allowed with the present line; a wake's
// prompt and a background task's report open none, so those sessions' stops
// stay blocked.
func TestPromptOpensConversationWindow(t *testing.T) {
	root := awayCheckout(t)
	setAway(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)

	// The person: the window opens, and the stop after it is the present one.
	if ctx := promptAt(t, "talk", root, "status?"); !strings.Contains(ctx, conversationMark) {
		t.Fatalf("the person's prompt did not open the window:\n%s", ctx)
	}
	pinStopClock(t, time.Now().UTC().Add(time.Minute), time.UTC)
	out, so, _ := runStop(t, stopHookPayload(t, "talk", root, false, "Here is the status."))
	if out.Decision != "" || !strings.Contains(out.SystemMessage, "feat/behind") {
		t.Errorf("a stop inside the window: %s; want it allowed, naming the owed item", so)
	}
	// Past the window the checks hold again.
	pinStopClock(t, time.Now().UTC().Add(autonomy.ConversationWindow+2*time.Minute), time.UTC)
	if out, so, _ := runStop(t, stopHookPayload(t, "talk", root, false, "Done.")); out.Decision != "block" {
		t.Errorf("a stop past the window: %s; want it blocked", so)
	}

	// A wake the stop check has seen: its prompt opens nothing.
	pinStopClock(t, time.Now().UTC(), time.UTC)
	if out, so, _ := runStop(t, stopHookPayload(t, "wake", root, false, "Done.", "*/10 * * * *")); out.Decision != "block" {
		t.Fatalf("first stop while away: %s; want it blocked", so)
	}
	if ctx := promptAt(t, "wake", root, "look"); strings.Contains(ctx, conversationMark) {
		t.Errorf("a wake's prompt opened the window:\n%s", ctx)
	}
	// A background task's report opens nothing either.
	if ctx := promptAt(t, "wake", root, "<task-notification>\n<task-id>a1</task-id>\n</task-notification>"); strings.Contains(ctx, conversationMark) {
		t.Errorf("a background report opened the window:\n%s", ctx)
	}
	// With no wake pending, an unchanged set blocks, as it would with no
	// prompt at all: neither prompt made the session present.
	pinStopClock(t, time.Now().UTC().Add(time.Minute), time.UTC)
	if out, so, _ := runStop(t, stopHookPayload(t, "wake", root, true, "Still running.")); out.Decision != "block" {
		t.Errorf("a stop after only a wake and a report: %s; want it blocked", so)
	}
}

// TestQuestionAdmittedInsideConversationWindow: while away a question is
// refused, but inside the window of the session asking it, it is checked as
// if the person were present, by the hook and by the verb the question
// drafter runs.
func TestQuestionAdmittedInsideConversationWindow(t *testing.T) {
	root := awayCheckout(t)
	setAwayHere(t)
	if reason := hookDenyOf(t, chipCall(t, root, "Product Q1")); !strings.Contains(reason, "the person is away") {
		t.Fatalf("before the person spoke: %q; want the away refusal", reason)
	}
	if res, _, _ := runQuestionVerb(t, "clean-product-chip"); res.Hook != question.OutcomeRefuseAway {
		t.Fatalf("the verb before the person spoke: %q; want refuse-away", res.Hook)
	}
	// askToolPayload asks as session s1.
	if ctx := promptAt(t, "s1", root, "let's go through the open questions"); !strings.Contains(ctx, conversationMark) {
		t.Fatalf("the person's prompt did not open the window:\n%s", ctx)
	}
	stdout, _, _ := runGuardTriple(chipCall(t, root, "Product Q1"))
	if strings.Contains(stdout, "the person is away") {
		t.Errorf("inside the window the question was refused as away:\n%s", stdout)
	}
	if res, _, _ := runQuestionVerb(t, "clean-product-chip"); res.Hook == question.OutcomeRefuseAway {
		t.Errorf("inside the window the verb still says refuse-away")
	}
	// Another session of the checkout, not in its window, is still away.
	other := strings.Replace(chipCall(t, root, "Product Q1"), `"session_id":"s1"`, `"session_id":"s2"`, 1)
	if reason := hookDenyOf(t, other); !strings.Contains(reason, "the person is away") {
		t.Errorf("another session's question: %q; want the away refusal", reason)
	}
}
