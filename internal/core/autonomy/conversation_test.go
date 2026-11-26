package autonomy

import (
	"context"
	"strings"
	"testing"
	"time"
)

// conversation_test.go — iss-2611260602137306: a prompt the person types while
// away opens a window in which that session reads as present.

// TestConversationWindowStop: inside the window a stop with work owed is
// allowed with the present line while the session holds a wake that will
// have the checks read it again, and held, naming the window, when it holds
// none; the checks hold as before once the window ends, and a stamp from
// before the away record began, or one later than now, opens nothing.
func TestConversationWindowStop(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	rec := awayRec(now, false)
	owed := owedOf(behindPR)
	seen := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint(), Wakes: 2}
	wake := crons("*/10 * * * *")

	for _, c := range []struct {
		name   string
		prompt time.Time
		crons  []SessionCron
		block  bool
		window bool // the block is the window's, not an away row's
	}{
		{"no prompt", time.Time{}, nil, true, false},
		{"prompt five minutes ago, a wake held", now.Add(-5 * time.Minute), wake, false, false},
		{"prompt five minutes ago, no wake", now.Add(-5 * time.Minute), nil, true, true},
		{"prompt at the window's edge, a wake held", now.Add(-ConversationWindow), wake, false, false},
		{"prompt past the window", now.Add(-ConversationWindow - time.Minute), nil, true, false},
		{"prompt stamped in the future", now.Add(time.Hour), nil, true, false},
		{"prompt before the away record began", rec.Start.Add(-time.Minute), wake, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			st := seen
			st.PersonPromptAt = c.prompt
			if c.name == "prompt before the away record began" {
				// A prompt one minute before the record's start, still
				// inside the window's span of now.
				now := rec.Start.Add(5 * time.Minute)
				v := judgeAt(t, rec, owed, StopInput{StopHookActive: true, SessionCrons: c.crons}, st, now)
				if v.Row == 0 && !v.Block {
					t.Fatalf("a stamp from before the record opened the window: %+v", v)
				}
				return
			}
			v := judgeAt(t, rec, owed, StopInput{StopHookActive: true, SessionCrons: c.crons}, st, now)
			if v.Block != c.block {
				t.Fatalf("Block = %v, want %v (row %d, reason %q)", v.Block, c.block, v.Row, v.Reason)
			}
			if c.prompt.After(now) && !v.State.PersonPromptAt.IsZero() {
				t.Errorf("a stamp later than now was kept: %v", v.State.PersonPromptAt)
			}
			if c.window {
				if !strings.Contains(v.Reason, "typed a prompt in this session") || !strings.Contains(v.Reason, "feat/behind") {
					t.Errorf("reason %q does not name the window and the owed item", v.Reason)
				}
				if !v.State.PersonPromptAt.Equal(c.prompt) {
					t.Errorf("the window's stamp was not kept: %v", v.State.PersonPromptAt)
				}
				return
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
	// With nothing owed the stop inside the window needs no wake.
	st := SessionState{PersonPromptAt: now.Add(-time.Minute)}
	if v := judgeAt(t, rec, Owed{}, StopInput{}, st, now); v.Block {
		t.Errorf("nothing owed inside the window: blocked (%q)", v.Reason)
	}
}

// TestJudgeRemembersCronPrompts: every stop the check reads keeps the prompts
// of the session's wakes, the person present or away, inside the window and
// out, and a reading cut short too, so the prompt hook can tell a wake's
// prompt from the person's whenever the mode is set.
func TestJudgeRemembersCronPrompts(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	away := awayRec(now, false)
	in := StopInput{SessionCrons: []SessionCron{{ID: "a", Schedule: "*/10 * * * *", Recurring: true, Prompt: "  Build, regular look\n"}}}
	short := Owed{Items: []OwedItem{behindPR}, Incomplete: true}
	for _, c := range []struct {
		name string
		rec  *Record
		owed Owed
		st   SessionState
	}{
		{"away", away, owedOf(behindPR), SessionState{}},
		{"away, inside the window", away, owedOf(behindPR), SessionState{PersonPromptAt: now.Add(-time.Minute)}},
		{"present, work owed", nil, owedOf(behindPR), SessionState{}},
		{"present, nothing owed", nil, Owed{}, SessionState{}},
		{"away, the reading cut short", away, short, SessionState{}},
		{"present, the reading cut short", nil, short, SessionState{}},
	} {
		v := judgeAt(t, c.rec, c.owed, in, c.st, now)
		if !v.Save || !knownCronPrompt(v.State, "Build, regular look") {
			t.Errorf("%s: the wake's prompt was not kept (Save %v, %v)", c.name, v.Save, v.State.CronPrompts)
		}
	}
	// A present stop whose wakes are already known writes nothing.
	known := SessionState{SchemaVersion: SchemaVersion, CronPrompts: []string{promptHash("Build, regular look")}}
	if v := judgeAt(t, nil, Owed{}, in, known, now); v.Save {
		t.Errorf("a present stop with nothing new saved the state: %+v", v.State)
	}

	var many []SessionCron
	for i := 0; i < maxCronPrompts+5; i++ {
		many = append(many, SessionCron{ID: "x", Schedule: "0 * * * *", Prompt: strings.Repeat("p", i+1)})
	}
	v := judgeAt(t, away, Owed{}, StopInput{SessionCrons: many}, SessionState{}, now)
	if n := len(v.State.CronPrompts); n != maxCronPrompts {
		t.Errorf("kept %d wake prompts, want the bound %d", n, maxCronPrompts)
	}
	if !knownCronPrompt(v.State, many[len(many)-1].Prompt) {
		t.Error("the newest wake prompt was dropped for an older one")
	}
	// The wake the bound let go opens no window: past the bound no prompt
	// is taken for the person's.
	if personPrompt(v.State, many[0].Prompt) {
		t.Error("a wake past the bound reads as the person's prompt")
	}
	if personPrompt(v.State, "status?") {
		t.Error("a session listing more wakes than the state keeps took a prompt for the person's")
	}
	// Once the session lists no more than the bound, prompts read as before.
	v = judgeAt(t, away, Owed{}, StopInput{SessionCrons: many[:3]}, v.State, now)
	if !personPrompt(v.State, "status?") {
		t.Error("a session back within the bound still opens no window")
	}
}

// TestSlashWakeOpensNoWindow: a wake whose prompt is a slash command reaches
// the prompt hook expanded, not as the stop listed it, so its firing cannot be
// told from the person typing: while the session holds one, no prompt reads
// as the person's (dcn-2611262236204687).
func TestSlashWakeOpensNoWindow(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	away := awayRec(now, false)
	slash := StopInput{SessionCrons: []SessionCron{
		{ID: "a", Schedule: "*/10 * * * *", Recurring: true, Prompt: "plain look"},
		{ID: "b", Schedule: "*/10 * * * *", Recurring: true, Prompt: "  /abcd:status --json"},
	}}
	for _, rec := range []*Record{away, nil} {
		v := judgeAt(t, rec, Owed{}, slash, SessionState{}, now)
		if personPrompt(v.State, "status?") {
			t.Errorf("record %v: a session holding a slash-command wake took a prompt for the person's", rec != nil)
		}
		if personPrompt(v.State, "<command-message>abcd:status</command-message>\n<command-name>/abcd:status</command-name>") {
			t.Errorf("record %v: the slash-command wake's expanded prompt reads as the person's", rec != nil)
		}
	}
	// Once the session holds no such wake, prompts read as before.
	v := judgeAt(t, away, Owed{}, slash, SessionState{}, now)
	v = judgeAt(t, away, Owed{}, StopInput{SessionCrons: slash.SessionCrons[:1]}, v.State, now)
	if !personPrompt(v.State, "status?") {
		t.Error("a session that no longer holds a slash-command wake still opens no window")
	}
}

// TestNotePrompt: only a prompt the person typed opens the window.
func TestNotePrompt(t *testing.T) {
	ctx := context.Background()
	now := at(t, "2026-11-26 06:00")
	rec := awayRec(now, false)
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
			opened, err := NotePrompt(ctx, copyRoot, id, c.prompt, rec, now)
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
			if st.Conversing(rec, now) != c.opens {
				t.Errorf("Conversing = %v after the prompt, want %v", st.Conversing(rec, now), c.opens)
			}
			if len(st.CronPrompts) != 1 {
				t.Errorf("the known wakes were not kept: %v", st.CronPrompts)
			}
		})
	}
	// A stamp later than now is dropped by a prompt that opens nothing, so
	// it cannot open a window once the clock reaches it.
	future := base
	future.PersonPromptAt = now.Add(time.Hour)
	if err := SaveSession(copyRoot, id, future); err != nil {
		t.Fatal(err)
	}
	if opened, err := NotePrompt(ctx, copyRoot, id, "Build, regular look", rec, now); err != nil || opened {
		t.Fatalf("a wake's prompt: opened %v, err %v", opened, err)
	}
	st, err := LoadSession(ctx, copyRoot, id)
	if err != nil {
		t.Fatal(err)
	}
	if !st.PersonPromptAt.IsZero() {
		t.Errorf("a stamp later than now was kept: %v", st.PersonPromptAt)
	}
	// Past the wakes' bound no prompt opens the window.
	if err := SaveSession(copyRoot, id, SessionState{CronPromptsUnmatched: true}); err != nil {
		t.Fatal(err)
	}
	if opened, err := NotePrompt(ctx, copyRoot, id, "status?", rec, now); err != nil || opened {
		t.Errorf("past the wakes' bound: opened %v, err %v; want nothing opened", opened, err)
	}
}

// TestConversingNeedsTheHoldingRecord: the window is read only under the away
// record holding now, and only from a stamp made under it.
func TestConversingNeedsTheHoldingRecord(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	rec := awayRec(now, false)
	st := SessionState{PersonPromptAt: now.Add(-time.Minute + 300*time.Millisecond)}
	if !st.Conversing(rec, now) {
		t.Fatal("a stamp a minute old under the holding record reads as no window")
	}
	if st.Conversing(nil, now) {
		t.Error("with no record holding, the stamp reads as a window")
	}
	later := *rec
	later.Start = now.Add(-30 * time.Second)
	if st.Conversing(&later, now) {
		t.Error("a stamp from before the record began reads as a window under it")
	}
	// The record keeps its start to the second: a stamp within that second
	// may come from the record before it, and opens nothing.
	same := *rec
	same.Start = st.PersonPromptAt.Truncate(time.Second)
	if st.Conversing(&same, now) {
		t.Error("a stamp within the record's first second reads as a window under it")
	}
}

// TestConversationLineIsAReading: the agent's line says the prompt reads as
// the person's, which is what the check infers, never that the person typed
// it, which nothing proves.
func TestConversationLineIsAReading(t *testing.T) {
	line := ConversationLine()
	if strings.Contains(line, "the person typed this prompt") {
		t.Errorf("the line states as fact who typed the prompt: %q", line)
	}
	for _, want := range []string{"reads as the person's", "this session reads them as present"} {
		if !strings.Contains(line, want) {
			t.Errorf("the line %q lacks %q", line, want)
		}
	}
}

// TestWakesBound: up to maxCronPrompts distinct wakes are all kept and leave
// the session's prompts readable; one more marks it, so no prompt reads as the
// person's. A stop whose payload lists no wakes (the host's task list not
// reached) leaves the mark as it was, since the wakes it let go may still
// fire.
func TestWakesBound(t *testing.T) {
	now := at(t, "2026-11-26 06:00")
	listed := func(n int) []SessionCron {
		var out []SessionCron
		for i := 0; i < n; i++ {
			out = append(out, SessionCron{ID: "x", Schedule: "0 * * * *", Prompt: strings.Repeat("p", i+1)})
		}
		return out
	}
	at32, _ := KeepWakes(SessionState{}, listed(maxCronPrompts), now)
	if at32.CronPromptsUnmatched || !personPrompt(at32, "status?") {
		t.Errorf("%d wakes marked the session", maxCronPrompts)
	}
	for _, c := range listed(maxCronPrompts) {
		if !knownCronPrompt(at32, c.Prompt) {
			t.Fatalf("at the bound the wake %q was let go", c.Prompt)
		}
	}
	at33, _ := KeepWakes(SessionState{}, listed(maxCronPrompts+1), now)
	if !at33.CronPromptsUnmatched || personPrompt(at33, "status?") {
		t.Errorf("%d wakes left the session's prompts readable", maxCronPrompts+1)
	}
	if after, _ := KeepWakes(at33, nil, now); !after.CronPromptsUnmatched {
		t.Error("a stop listing no wakes cleared the mark")
	}
}

// TestNotePromptInTheRecordsFirstSecond: a prompt the window would not count
// (stamped within the record's first second) is not reported as opening it.
func TestNotePromptInTheRecordsFirstSecond(t *testing.T) {
	ctx := context.Background()
	now := at(t, "2026-11-26 06:00").Add(400 * time.Millisecond)
	rec := awayRec(now, false)
	rec.Start = now.Truncate(time.Second)
	copyRoot := t.TempDir()
	if opened, err := NotePrompt(ctx, copyRoot, "sess-1", "status?", rec, now); err != nil || opened {
		t.Errorf("a prompt in the record's first second: opened %v, err %v; want nothing reported open", opened, err)
	}
	later := now.Add(2 * time.Second)
	if opened, err := NotePrompt(ctx, copyRoot, "sess-1", "status?", rec, later); err != nil || !opened {
		t.Errorf("a prompt two seconds in: opened %v, err %v; want it open", opened, err)
	}
}
