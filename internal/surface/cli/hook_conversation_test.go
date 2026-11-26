package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/question"
)

// hook_conversation_test.go — iss-2611260602137306: a prompt the person types
// while away has that session read them as present for a while, at the stop
// and at a question, and a wake's prompt or the host's own opens nothing.
// Each test owns the hooks' budgets (ownHookBudgets), so a loaded machine
// cannot spend the prompt hook's second before the window is read.

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
// the next stop with work owed is allowed with the present line while the
// session holds a wake, and held, naming the window, while it holds none; a
// wake's prompt and a background task's report open none, so those sessions'
// stops stay blocked.
func TestPromptOpensConversationWindow(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	setAwayAt(t, root, time.Now().Add(-time.Minute), false)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)

	// The person: the window opens. A stop with work owed and no wake is
	// held, since nothing would read this session again; with a wake it is
	// the present one.
	if ctx := promptAt(t, "talk", root, "status?"); !strings.Contains(ctx, conversationMark) {
		t.Fatalf("the person's prompt did not open the window:\n%s", ctx)
	}
	pinStopClock(t, time.Now().UTC().Add(time.Minute), time.UTC)
	out, so, _ := runStop(t, stopHookPayload(t, "talk", root, false, "Here is the status."))
	if out.Decision != "block" || !strings.Contains(out.Reason, "typed a prompt in this session") {
		t.Errorf("a stop inside the window with no wake: %s; want it held, naming the window", so)
	}
	out, so, _ = runStop(t, stopHookPayload(t, "talk", root, true, "Here is the status.", "*/10 * * * *"))
	if out.Decision != "" || !strings.Contains(out.SystemMessage, "feat/behind") {
		t.Errorf("a stop inside the window with a wake: %s; want it allowed, naming the owed item", so)
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

// TestWakeSetWhilePresentOpensNoWindow: a wake the session held at a stop
// while the person was present, or at a stop the check allowed without
// judging it, is known once the mode is set, so its prompt opens nothing.
func TestWakeSetWhilePresentOpensNoWindow(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)

	pinStopClock(t, time.Now().UTC(), time.UTC)
	if out, so, _ := runStop(t, stopHookPayload(t, "early", root, false, "Done.", "*/10 * * * *")); out.Decision != "" {
		t.Fatalf("a present stop: %s; want it allowed", so)
	}
	setAwayAt(t, root, time.Now().Add(-time.Minute), false)
	if ctx := promptAt(t, "early", root, "look"); strings.Contains(ctx, conversationMark) {
		t.Errorf("a wake seen only while present opened the window:\n%s", ctx)
	}

	// A stop allowed because the owed work could not be read keeps the
	// wakes too.
	prev := stopOwedReader
	stopOwedReader = func(context.Context, autonomy.ReadOptions) (autonomy.Owed, error) {
		return autonomy.Owed{}, errors.New("unreadable")
	}
	t.Cleanup(func() { stopOwedReader = prev })
	payload := strings.Replace(stopHookPayload(t, "unread", root, false, "Done.", "*/10 * * * *"), `"prompt":"look"`, `"prompt":"unread look"`, 1)
	if out, so, _ := runStop(t, payload); out.Decision != "" {
		t.Fatalf("a stop whose owed work could not be read: %s; want it allowed", so)
	}
	if ctx := promptAt(t, "unread", root, "unread look"); strings.Contains(ctx, conversationMark) {
		t.Errorf("a wake seen at a stop allowed unread opened the window:\n%s", ctx)
	}
	stopOwedReader = prev

	// So does a stop allowed because the away record could not be read.
	st := awayStore(t, root)
	if err := os.WriteFile(st.RecordPath(), []byte("{garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	payload = strings.Replace(stopHookPayload(t, "norec", root, false, "Done.", "*/10 * * * *"), `"prompt":"look"`, `"prompt":"record look"`, 1)
	if out, so, se := runStop(t, payload); out.Decision != "" || !strings.Contains(se, "not away") {
		t.Fatalf("a stop under an unreadable record: %s %s; want it allowed, naming the record", so, se)
	}
	if err := os.Remove(st.RecordPath()); err != nil {
		t.Fatal(err)
	}
	setAwayAt(t, root, time.Now().Add(-time.Minute), false)
	if ctx := promptAt(t, "norec", root, "record look"); strings.Contains(ctx, conversationMark) {
		t.Errorf("a wake seen at a stop allowed under an unreadable record opened the window:\n%s", ctx)
	}
}

// sessionStateFile is the session's state file in the copy at root.
func sessionStateFile(root, session string) string {
	return filepath.Join(root, filepath.FromSlash(autonomy.SessionsDir), session+".json")
}

// TestPresentStopLeavesAnUnreadableStateAsItIs: a present stop listing a wake
// the session's state does not hold, whose state cannot be read, writes
// nothing over it, and names it.
func TestPresentStopLeavesAnUnreadableStateAsItIs(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)
	path := sessionStateFile(root, "broken")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	garbage := []byte("{not a state")
	if err := os.WriteFile(path, garbage, 0o600); err != nil {
		t.Fatal(err)
	}
	pinStopClock(t, time.Now().UTC(), time.UTC)
	out, so, se := runStop(t, stopHookPayload(t, "broken", root, false, "Done.", "*/10 * * * *"))
	if out.Decision != "" || !strings.Contains(out.SystemMessage, "feat/behind") {
		t.Errorf("a present stop over an unreadable state: %s; want it allowed with the present line", so)
	}
	if !strings.Contains(se, "state could not be read") {
		t.Errorf("stderr %q does not name the unreadable state", se)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, garbage) {
		t.Errorf("the unreadable state was written over: %q (%v)", got, err)
	}
}

// TestAllowedStopEmitsWhenTheStateCannotBeSaved: a present stop whose wakes'
// save fails is still shown with its line, the failure named, as it was
// before the stop saved anything.
func TestAllowedStopEmitsWhenTheStateCannotBeSaved(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)
	dir := filepath.Join(root, filepath.FromSlash(autonomy.SessionsDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	pinStopClock(t, time.Now().UTC(), time.UTC)
	out, so, se := runStop(t, stopHookPayload(t, "nosave", root, false, "Done.", "*/10 * * * *"))
	if out.Decision != "" || !strings.Contains(out.SystemMessage, "feat/behind") {
		t.Errorf("a present stop whose state could not be saved: stdout %s stderr %s; want the present line", so, se)
	}
	if !strings.Contains(se, "could not be saved") {
		t.Errorf("stderr %q does not name the failed save", se)
	}
}

// TestQuestionVerbReadsTheSessionsStartCopy: run from a worktree other than
// the session's start copy, the verb reads the window where the prompt hook
// wrote it.
func TestQuestionVerbReadsTheSessionsStartCopy(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	setAwayAt(t, root, time.Now().Add(-time.Minute), false)
	t.Setenv(hostSessionIDEnv, "s1")
	if _, err := awayStore(t, root).NoteSessionCopy("s1", root, time.Now()); err != nil {
		t.Fatal(err)
	}
	if ctx := promptAt(t, "s1", root, "let's go through the open questions"); !strings.Contains(ctx, conversationMark) {
		t.Fatalf("the person's prompt did not open the window:\n%s", ctx)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/wt", wt)
	wt = realPath(t, wt)
	if err := os.MkdirAll(filepath.Join(wt, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(wt)
	if res, _, _ := runQuestionVerb(t, "clean-product-chip"); res.Hook == question.OutcomeRefuseAway {
		t.Errorf("from another worktree the verb did not read the session's window in its start copy")
	}
}

// TestConversationWindowEndsWithItsRecord: a prompt the person typed under one
// away record opens nothing under the next, however soon it is set.
func TestConversationWindowEndsWithItsRecord(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)
	setAwayAt(t, root, time.Now().Add(-time.Minute), false)
	if ctx := promptAt(t, "back", root, "I'm back"); !strings.Contains(ctx, conversationMark) {
		t.Fatalf("the person's prompt did not open the window:\n%s", ctx)
	}
	if _, stderr, err := runSplit(t, "", "mode", "back"); err != nil {
		t.Fatalf("mode back: %v\n%s", err, stderr)
	}
	// The next record, set straight after: its start, kept to the second,
	// is no earlier than the prompt's second.
	setAwayHere(t)
	pinStopClock(t, time.Now().UTC().Add(time.Minute), time.UTC)
	out, so, _ := runStop(t, stopHookPayload(t, "back", root, false, "Done.", "*/10 * * * *"))
	if out.Decision != "block" || strings.Contains(out.Reason, "typed a prompt in this session") {
		t.Errorf("the first stop under a new record: %s; want the away block", so)
	}
	if reason := hookDenyOf(t, strings.Replace(chipCall(t, root, "Product Q1"), `"session_id":"s1"`, `"session_id":"back"`, 1)); !strings.Contains(reason, "the person is away") {
		t.Errorf("a question under the new record: %q; want the away refusal", reason)
	}
}

// TestQuestionAdmittedInsideConversationWindow: while away a question is
// refused, but inside the window of the session asking it, it is checked as
// if the person were present, by the hook and by the verb the question
// drafter runs, which reads the session the host's environment names.
func TestQuestionAdmittedInsideConversationWindow(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	root := awayCheckout(t)
	setAwayAt(t, root, time.Now().Add(-time.Minute), false)
	t.Setenv(hostSessionIDEnv, "s1")
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
	// Another session of the checkout, not in its window, is still away, at
	// the hook and at the verb, as is a verb run with no session named.
	other := strings.Replace(chipCall(t, root, "Product Q1"), `"session_id":"s1"`, `"session_id":"s2"`, 1)
	if reason := hookDenyOf(t, other); !strings.Contains(reason, "the person is away") {
		t.Errorf("another session's question: %q; want the away refusal", reason)
	}
	for _, id := range []string{"s2", ""} {
		t.Setenv(hostSessionIDEnv, id)
		if res, _, _ := runQuestionVerb(t, "clean-product-chip"); res.Hook != question.OutcomeRefuseAway {
			t.Errorf("the verb for session %q outside its window: %q; want refuse-away", id, res.Hook)
		}
	}
}
