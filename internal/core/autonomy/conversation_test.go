package autonomy

import (
	"context"
	"strings"
	"testing"
	"time"
)

// conversation_test.go — iss-2611260602137306: a prompt the person types while
// away opens a window in which that session reads as present.

// TestConversationWindowAllowsStop: inside the window a stop with work owed is
// allowed with the present line, and the checks hold again once it ends.
func TestConversationWindowAllowsStop(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	rec := awayRec(now, false)
	owed := owedOf(behindPR)
	seen := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint(), Wakes: 2}

	for _, c := range []struct {
		name   string
		prompt time.Time
		block  bool
	}{
		{"no prompt", time.Time{}, true},
		{"prompt five minutes ago", now.Add(-5 * time.Minute), false},
		{"prompt at the window's edge", now.Add(-ConversationWindow), false},
		{"prompt past the window", now.Add(-ConversationWindow - time.Minute), true},
		{"prompt stamped in the future", now.Add(time.Hour), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			st := seen
			st.PersonPromptAt = c.prompt
			v := judgeAt(t, rec, owed, StopInput{StopHookActive: true}, st, now)
			if v.Block != c.block {
				t.Fatalf("Block = %v, want %v (row %d, reason %q)", v.Block, c.block, v.Row, v.Reason)
			}
			if c.block {
				return
			}
			if v.Row != 0 || v.Stop != nil {
				t.Errorf("row %d, stop %+v: a stop inside the window is the present one", v.Row, v.Stop)
			}
			if !strings.Contains(v.SystemMessage, "feat/behind") {
				t.Errorf("SystemMessage %q does not name the owed item", v.SystemMessage)
			}
			if !v.Save || !v.State.PersonPromptAt.Equal(c.prompt) {
				t.Errorf("Save %v, PersonPromptAt %v: the window's stamp must be kept", v.Save, v.State.PersonPromptAt)
			}
			if v.State.Fingerprint != st.Fingerprint || v.State.Wakes != st.Wakes {
				t.Errorf("the away counts moved inside the window: %+v", v.State)
			}
		})
	}
}

// TestJudgeRemembersCronPrompts: every stop the record holds for keeps the
// prompts of the session's wakes, inside the window and out, so the prompt hook
// can tell a wake's prompt from the person's.
func TestJudgeRemembersCronPrompts(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	rec := awayRec(now, false)
	in := StopInput{SessionCrons: []SessionCron{{ID: "a", Schedule: "*/10 * * * *", Recurring: true, Prompt: "  Build, regular look\n"}}}
	for _, st := range []SessionState{{}, {PersonPromptAt: now.Add(-time.Minute)}} {
		v := judgeAt(t, rec, owedOf(behindPR), in, st, now)
		if !v.Save || !knownCronPrompt(v.State, "Build, regular look") {
			t.Errorf("state %+v: the wake's prompt was not kept (Save %v, %v)", st, v.Save, v.State.CronPrompts)
		}
	}
	var many []SessionCron
	for i := 0; i < maxCronPrompts+5; i++ {
		many = append(many, SessionCron{ID: "x", Schedule: "0 * * * *", Prompt: strings.Repeat("p", i+1)})
	}
	v := judgeAt(t, rec, Owed{}, StopInput{SessionCrons: many}, SessionState{}, now)
	if n := len(v.State.CronPrompts); n != maxCronPrompts {
		t.Errorf("kept %d wake prompts, want the bound %d", n, maxCronPrompts)
	}
	if !knownCronPrompt(v.State, many[len(many)-1].Prompt) {
		t.Error("the newest wake prompt was dropped for an older one")
	}
}

// TestNotePrompt: only a prompt the person typed opens the window.
func TestNotePrompt(t *testing.T) {
	ctx := context.Background()
	now := at(t, "2026-11-26 06:00")
	copyRoot := t.TempDir()
	const id = "sess-1"
	base := SessionState{CronPrompts: []string{promptHash("Build, regular look")}}
	for _, c := range []struct {
		name, prompt string
		opens        bool
	}{
		{"typed", "status?", true},
		{"slash command", "<command-message>abcd:ideate</command-message>", true},
		{"a known wake", "Build, regular look\n", false},
		{"background report", "<task-notification>\n<task-id>a1</task-id>", false},
		{"peer message", "Another Claude session sent a message: <cross-session-message from=\"x\">hi</cross-session-message>", false},
		{"usage reset", "Your claude.ai usage limit has reset. Continue the task", false},
		{"continued session", "This session is being continued from a previous conversation that ran out of context.", false},
		{"empty", "   ", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := SaveSession(copyRoot, id, base); err != nil {
				t.Fatal(err)
			}
			opened, err := NotePrompt(ctx, copyRoot, id, c.prompt, now)
			if err != nil {
				t.Fatal(err)
			}
			if opened != c.opens {
				t.Fatalf("opened = %v, want %v", opened, c.opens)
			}
			st, err := LoadSession(ctx, copyRoot, id)
			if err != nil {
				t.Fatal(err)
			}
			if st.Conversing(now) != c.opens {
				t.Errorf("Conversing = %v after the prompt, want %v", st.Conversing(now), c.opens)
			}
			if len(st.CronPrompts) != 1 {
				t.Errorf("the known wakes were not kept: %v", st.CronPrompts)
			}
		})
	}
}

// TestAnyConversing: the question check run with no session (`abcd guard
// question`) reads present while any session of the copy is in its window.
func TestAnyConversing(t *testing.T) {
	ctx := context.Background()
	now := at(t, "2026-11-26 06:00")
	copyRoot := t.TempDir()
	if AnyConversing(ctx, copyRoot, now) {
		t.Fatal("a copy with no sessions reads as conversing")
	}
	if err := SaveSession(copyRoot, "old", SessionState{PersonPromptAt: now.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if AnyConversing(ctx, copyRoot, now) {
		t.Fatal("a prompt an hour old reads as conversing")
	}
	if err := SaveSession(copyRoot, "new", SessionState{PersonPromptAt: now.Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if !AnyConversing(ctx, copyRoot, now) {
		t.Fatal("a session in its window does not read as conversing")
	}
}
