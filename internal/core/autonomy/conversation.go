package autonomy

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// conversation.go — the person talking while away (iss-2611260602137306). The
// away record holds for every session, but a prompt the person types in one
// session opens a window in which that session reads as present: its stops are
// allowed with the present line and its questions go to the person. Every
// other session stays away, and the window closes on its own, so autonomous
// work resumes without the person ending the mode and setting it again.

// ConversationWindow is how long after the person's last prompt a session reads
// as present while the away record holds.
const ConversationWindow = 15 * time.Minute

// maxCronPrompts bounds the wake prompts a session's state keeps.
const maxCronPrompts = 32

// hostPromptPrefixes open the prompts the host submits on its own: a
// background task's report, another session's message, a usage-limit
// resumption and a continued session's summary. The host runs the prompt hook
// on each (code.claude.com/docs/en/hooks, UserPromptSubmit) with no field
// naming the prompt's origin, so their envelopes are what tells them apart.
var hostPromptPrefixes = []string{
	"<task-notification>",
	"<cross-session-message",
	"Another Claude session sent a message:",
	"Your claude.ai usage limit has reset.",
	"This session is being continued from a previous conversation",
}

// Conversing reports whether the person typed a prompt in this session within
// ConversationWindow of now. A stamp later than now (the clock stepped back, or
// a state written by hand) opens nothing.
func (s SessionState) Conversing(now time.Time) bool {
	if s.PersonPromptAt.IsZero() || s.PersonPromptAt.After(now) {
		return false
	}
	return now.Sub(s.PersonPromptAt) <= ConversationWindow
}

// promptHash is a prompt's hash as the state keeps it, surrounding space aside.
func promptHash(prompt string) string { return hashText(strings.TrimSpace(prompt)) }

// knownCronPrompt reports whether prompt is one of the session's wakes.
func knownCronPrompt(st SessionState, prompt string) bool {
	return slices.Contains(st.CronPrompts, promptHash(prompt))
}

// rememberCrons adds the wakes' prompts to known, newest last, each once, and
// keeps the newest maxCronPrompts.
func rememberCrons(known []string, crons []SessionCron) []string {
	if len(crons) == 0 {
		return known
	}
	out := slices.Clone(known)
	for _, c := range crons {
		if strings.TrimSpace(c.Prompt) == "" {
			continue
		}
		h := promptHash(c.Prompt)
		out = slices.DeleteFunc(out, func(k string) bool { return k == h })
		out = append(out, h)
	}
	if len(out) > maxCronPrompts {
		out = out[len(out)-maxCronPrompts:]
	}
	return out
}

// personPrompt reports whether a submitted prompt is the person's: not empty,
// not one of the host's own envelopes, and not a wake the stop checks have seen.
func personPrompt(st SessionState, prompt string) bool {
	p := strings.TrimSpace(prompt)
	if p == "" || knownCronPrompt(st, p) {
		return false
	}
	for _, pre := range hostPromptPrefixes {
		if strings.HasPrefix(p, pre) {
			return false
		}
	}
	return true
}

// NotePrompt stamps the session's state with now when prompt is the person's,
// opening the conversation window, and reports whether it did. The prompt hook
// calls it only while the away record holds. A wake's prompt the stop checks
// have not yet seen reads as the person's: the cost is one window in which the
// session is not held, never a stop held against the person.
func NotePrompt(ctx context.Context, copyRoot, sessionID, prompt string, now time.Time) (bool, error) {
	st, err := LoadSession(ctx, copyRoot, sessionID)
	if err != nil {
		return false, err
	}
	if !personPrompt(st, prompt) {
		return false, nil
	}
	st.PersonPromptAt = now
	if err := SaveSession(copyRoot, sessionID, st); err != nil {
		return false, err
	}
	return true, nil
}

// AnyConversing reports whether any session of the copy at copyRoot is in its
// conversation window: what a check run with no session id (`abcd guard
// question`, run by the question drafter) reads, since the session it serves
// is one of them. A state that cannot be read counts for nothing.
func AnyConversing(ctx context.Context, copyRoot string, now time.Time) bool {
	entries, err := os.ReadDir(filepath.Join(copyRoot, filepath.FromSlash(SessionsDir)))
	if err != nil {
		return false
	}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !e.Type().IsRegular() || !ValidSessionID(id) {
			continue
		}
		if ctx.Err() != nil {
			return false
		}
		if st, err := LoadSession(ctx, copyRoot, id); err == nil && st.Conversing(now) {
			return true
		}
	}
	return false
}

// ConversationLine is the agent's line on a prompt that opened or renewed the
// window: the person is here in this session, for now.
func ConversationLine() string {
	return "abcd: the person typed this prompt, so for " + shortDuration(ConversationWindow) +
		" after it this session reads them as present although the away record holds: answer them, and put a question to them on the host's question tool rather than routing it. Every other session stays away, and once the window ends the away checks hold here again, so carry on with owed work after answering."
}
