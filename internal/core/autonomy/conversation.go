package autonomy

import (
	"context"
	"slices"
	"strings"
	"time"
)

// conversation.go — the person talking while away (iss-2611260602137306). The
// away record holds for every session, but a prompt the person types in one
// session opens a window in which that session reads as present: its
// questions go to the person, and its stops are allowed with the present line
// when nothing is owed or the session holds a wake that has the checks read it
// again (dcn-2611262057069959). Every other session stays away. The window
// closes on its own, and with it the record it opened under, so autonomous
// work resumes without the person ending the mode and setting it again: at the
// session's first stop past the window, which a wake held at the window's last
// stop makes sure of while work is owed.
//
// Anything running as the same account can write the session's state or run
// the prompt hook by hand, so the window holds against other accounts and
// stale state, not against this account's own agent; nothing enforces more.

// ConversationWindow is how long after the person's last prompt a session reads
// as present while the away record holds.
const ConversationWindow = 15 * time.Minute

// maxCronPrompts bounds the wake prompts a session's state keeps. A session
// whose stop lists more wakes than this is marked (CronPromptsUnmatched), and
// no prompt opens its window until it lists no more, so a wake the bound let
// go is never read as the person typing.
const maxCronPrompts = 32

// unmatchableWakes reports whether a stop's wakes cannot all be told from the
// person typing by their prompts' text: more of them than the state keeps, or
// one whose prompt the host turns into other text when it fires, a slash
// command (expanded) or a loop's sentinel such as `<<autonomous-loop>>`
// (resolved to the loop's instructions), so it reaches the prompt hook other
// than as the stop lists it (dcn-2611262236204687). A plain-text wake's prompt
// arrives as it was set.
func unmatchableWakes(crons []SessionCron) bool {
	if distinctPrompts(crons) > maxCronPrompts {
		return true
	}
	for _, c := range crons {
		p := strings.TrimSpace(c.Prompt)
		if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "<<") {
			return true
		}
	}
	return false
}

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
// ConversationWindow of now, under the away record rec holding now: no record
// holding is no window, and a stamp from before rec began opens nothing under
// it, so a prompt typed under one record opens nothing under the next. The
// record keeps its start to the second, so a stamp within that second is taken
// as before it: a prompt typed in a record's first second opens nothing, and
// one typed under the record before it in that second never does. A stamp
// later than now (the clock stepped back) opens nothing, and the next write of
// the state drops it (KeepWakes, NotePrompt).
func (s SessionState) Conversing(rec *Record, now time.Time) bool {
	if !rec.Holds(now) || s.PersonPromptAt.IsZero() || s.PersonPromptAt.After(now) || s.PersonPromptAt.Before(rec.Start.Add(time.Second)) {
		return false
	}
	return now.Sub(s.PersonPromptAt) <= ConversationWindow
}

// KeepWakes is st as a stop check leaves it for the prompt hook: the wakes the
// stop's payload lists remembered (and the mark of wakes that cannot all be
// matched set or cleared, unmatchableWakes), and a stamp later than now
// dropped.
// changed reports whether that differs from st, so a check with nothing new
// writes nothing. Every stop the check reads keeps them, the person present or
// away, so a wake set before the mode was is known at its first prompt.
func KeepWakes(st SessionState, crons []SessionCron, now time.Time) (next SessionState, changed bool) {
	next = st
	next.CronPrompts = rememberCrons(st.CronPrompts, crons)
	// A payload listing no wakes may be one whose host did not reach its task
	// list, so the mark stands until a stop lists the wakes again.
	if len(crons) > 0 {
		next.CronPromptsUnmatched = unmatchableWakes(crons)
	}
	if next.PersonPromptAt.After(now) {
		next.PersonPromptAt = time.Time{}
	}
	changed = !slices.Equal(next.CronPrompts, st.CronPrompts) ||
		next.CronPromptsUnmatched != st.CronPromptsUnmatched ||
		!next.PersonPromptAt.Equal(st.PersonPromptAt)
	if changed {
		next.SchemaVersion = SchemaVersion
	}
	return next, changed
}

// distinctPrompts counts the different non-empty prompts crons lists.
func distinctPrompts(crons []SessionCron) int {
	seen := map[string]bool{}
	for _, c := range crons {
		if strings.TrimSpace(c.Prompt) != "" {
			seen[promptHash(c.Prompt)] = true
		}
	}
	return len(seen)
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
// not one of the host's own envelopes, and not a wake the stop checks have
// seen. In a session whose last stop listed wakes that cannot all be matched
// (unmatchableWakes), no prompt is.
func personPrompt(st SessionState, prompt string) bool {
	p := strings.TrimSpace(prompt)
	if p == "" || st.CronPromptsUnmatched || knownCronPrompt(st, p) {
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
// calls it only while the away record holds. The wakes are those every stop
// check of the session has kept (KeepWakes), present or away. A prompt that
// opens nothing drops a stamp later than now, so a clock stepped back cannot
// open a window later. It reports the window open only when the stamp counts
// under rec, the record holding now (Conversing): a prompt in the record's
// first second is stamped but opens nothing.
func NotePrompt(ctx context.Context, copyRoot, sessionID, prompt string, rec *Record, now time.Time) (bool, error) {
	st, err := LoadSession(ctx, copyRoot, sessionID)
	if err != nil {
		return false, err
	}
	if !personPrompt(st, prompt) {
		if st.PersonPromptAt.After(now) {
			st.PersonPromptAt = time.Time{}
			if err := SaveSession(copyRoot, sessionID, st); err != nil {
				return false, err
			}
		}
		return false, nil
	}
	st.PersonPromptAt = now
	if err := SaveSession(copyRoot, sessionID, st); err != nil {
		return false, err
	}
	return st.Conversing(rec, now), nil
}

// ConversationLine is the agent's line on a prompt that opened or renewed the
// window: the person reads as here in this session, for now. It says the
// prompt reads as theirs, since that is inferred from what the prompt is not
// (personPrompt), never proved.
func ConversationLine() string {
	return "abcd: this prompt reads as the person's (it is not a wake of this session's nor one of the host's own prompts), so for " + shortDuration(ConversationWindow) +
		" after it this session reads them as present although the away record holds: answer them, and put a question to them on the host's question tool rather than routing it. Every other session stays away. While work is owed, a turn's end here is allowed only holding a scheduled wake (a cron that fires within the away check's deadline), so the away checks read this session again once the window ends: carry on with owed work after answering."
}
