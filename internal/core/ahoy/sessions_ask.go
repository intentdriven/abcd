package ahoy

// The live brief session questions at repository setup (spc-2610141709430146
// step 6, for itd-2610091016262335, adr-2610161400542916).
//
// The answer is `sessions.live` in .abcd/config.json:
//
//   - absent: not yet answered. Detection raises the optional
//     sessions.offered gap. A person at a terminal, in a run that is not
//     --yes, is asked the two questions; every other run reports the gap
//     under optional_skipped with the flag that answers it. `abcd session
//     start` and `abcd session blueprint` ask it again at a terminal;
//   - `none`: answered no. Never asked again; `abcd session setup` still
//     sets live sessions up by hand;
//   - `local`, `cloud` or `both`: where sessions may run, which `abcd session
//     setup` reads to decide its steps.
//
// Q1, keyed sessions, asks whether to set up live brief sessions: yes asks
// Q2, no stores none, and deciding later (the default, and end of input)
// stores nothing. Q2, keyed sessions.where, asks where they may run.
// `--live-sessions local|cloud|both|none` answers both in any run, --yes
// included: it is the owner's answer itself, so the category has no approval
// question, as the README pair has none.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Sessions covers the live brief session answer. It is never asked as a
// category: the question is the consent.
const Sessions GapCategory = "sessions"

// SessionsOfferGapID is the live-session question, raised while it is
// unanswered.
const SessionsOfferGapID = "sessions.offered"

// SessionsPromptKey keys Q1; SessionsWherePromptKey keys Q2.
const (
	SessionsPromptKey      = "sessions"
	SessionsWherePromptKey = "sessions.where"
)

// LiveSessionsNone is the stored no.
const LiveSessionsNone = "none"

// LiveSessionsChoices is what --live-sessions takes, as a summary names it.
const LiveSessionsChoices = "local|cloud|both|none"

// liveSessionsWhere are Q2's answers, in the order it offers them.
var liveSessionsWhere = []string{"local", "cloud", "both"}

// sessionsChangeLater is both questions' change-later line.
const sessionsChangeLater = "abcd ahoy install --live-sessions " + LiveSessionsChoices

// sessionsQ1Choices are Q1's answers; deciding later is its default.
var sessionsQ1Choices = []string{"yes", "no", SetupLaterValue}

// sessionsQ2Choices are Q2's answers; deciding later is its default.
var sessionsQ2Choices = append(append([]string(nil), liveSessionsWhere...), SetupLaterValue)

var sessionsQ1 = PromptHelp{
	Key: SessionsPromptKey,
	About: "In a live brief session the team edits the brief together during a call. Setting it up " +
		"registers the team's own GitHub sign-in app once, and for the cloud deploys the team's relay " +
		"on Render. Set up live brief sessions?",
	ChangeLater: sessionsChangeLater,
	Choices: []ChoiceHelp{
		{Value: "yes", Meaning: "Asks next where sessions may run; abcd session setup then does the rest."},
		{Value: "no", Meaning: "Never asked again; abcd session setup still sets them up by hand."},
		{Value: SetupLaterValue, Meaning: "Records nothing; setup asks again, and so does abcd session start."},
	},
}

var sessionsQ2 = PromptHelp{
	Key: SessionsWherePromptKey,
	About: "Local runs a session on the host's computer, reached only over the team's Tailscale network. " +
		"Cloud runs it on the team's relay on Render, with the page on abcdesign.app. Where may sessions run?",
	ChangeLater: sessionsChangeLater,
	Choices: []ChoiceHelp{
		{Value: "local", Meaning: "Host's computer only; sign-in by a short code, and no secret is kept."},
		{Value: "cloud", Meaning: "Team's relay only; one-click sign-in, the secret kept on Render."},
		{Value: "both", Meaning: "Either; the sign-in app's device flow is switched on for local sessions."},
		{Value: SetupLaterValue, Meaning: "Records nothing, so the next setup run asks again."},
	},
}

// sessionsHelp is the help for a live-session question key.
func sessionsHelp(key string) (PromptHelp, bool) {
	switch key {
	case SessionsPromptKey:
		return withNow(sessionsQ1), true
	case SessionsWherePromptKey:
		return withNow(sessionsQ2), true
	}
	return PromptHelp{}, false
}

// withNow is h with its current setting: the questions are asked only while
// nothing is stored.
func withNow(h PromptHelp) PromptHelp {
	h.Choices = append([]ChoiceHelp(nil), h.Choices...)
	h.Now = "not set"
	return h
}

// ParseLiveSessions refuses a --live-sessions value that is not local, cloud,
// both or none.
func ParseLiveSessions(v string) error {
	if v == LiveSessionsNone || slices.Contains(liveSessionsWhere, v) {
		return nil
	}
	return fmt.Errorf("--live-sessions takes local, cloud, both or none, not %q", termsafe.Sanitize(v))
}

// LiveSessions reads the stored answer: local, cloud, both or none, and
// false when nothing is stored. An unreadable config, or a stored value that
// is not one of those, is an error.
func LiveSessions(cwd string) (string, bool, error) {
	cfg, err := readConfig(cwd)
	if err != nil {
		return "", false, err
	}
	raw, ok := subMap(cfg, "sessions")["live"]
	if !ok {
		return "", false, nil
	}
	v, isString := raw.(string)
	if !isString || ParseLiveSessions(v) != nil {
		return "", false, errors.New("sessions.live in .abcd/config.json is not local, cloud, both or none; " +
			"set it with abcd ahoy install --live-sessions " + LiveSessionsChoices)
	}
	return v, true, nil
}

// StoreLiveSessions saves v as sessions.live under the config's lock and
// returns what was stored before ("" for nothing). It refuses a value
// ParseLiveSessions refuses, and a repository with no .abcd/config.json.
func StoreLiveSessions(cwd, v string) (string, error) {
	if err := ParseLiveSessions(v); err != nil {
		return "", err
	}
	var prev string
	err := withConfigLock(cwd, func() error {
		cfg, err := readConfig(cwd)
		if err != nil {
			return err
		}
		if cfg == nil {
			return errors.New(".abcd/config.json does not exist yet; answer y to creating abcd's starter files and run abcd ahoy install again")
		}
		if s, ok := subMap(cfg, "sessions")["live"].(string); ok {
			prev = s
		}
		setSub(cfg, "sessions", "live", v)
		return writeConfig(cwd, cfg)
	})
	return prev, err
}

// detectSessions raises the live-session question while it is unanswered. A
// config abcd cannot read raises its own gap; this one waits for it, and a
// stored value that is not an answer is reported by setup, not re-asked.
func detectSessions(cwd string) []Gap {
	cfg, err := readConfig(cwd)
	if err != nil {
		return nil
	}
	if _, ok := subMap(cfg, "sessions")["live"]; ok {
		return nil
	}
	return []Gap{{
		ID: SessionsOfferGapID, Category: Sessions, Scope: "repo",
		Title: "the live brief session question is not answered",
		Detail: "abcd can set up live brief sessions, where the team edits the brief together during a call, " +
			"on the host's own computer over Tailscale, on the team's own relay on Render, or both.",
		FixHint:  "answer it with abcd ahoy install --live-sessions " + LiveSessionsChoices + "; none answers no and is not asked again.",
		Required: false, Resolvable: true,
	}}
}

// SessionsView is the live-session questions as detection carries them while
// they are open, for a front door that relays them: Q1 alone, then Q2 alone,
// each as the shared question type.
type SessionsView struct {
	Asks []question.Ask `json:"asks"`
}

// sessionsView is the view while the question is open, and false once it is
// answered (or the config cannot be read).
func sessionsView(cwd string) (SessionsView, bool) {
	if len(detectSessions(cwd)) == 0 {
		return SessionsView{}, false
	}
	v := SessionsView{}
	for i, key := range []string{SessionsPromptKey, SessionsWherePromptKey} {
		h, _ := sessionsHelp(key)
		choices := sessionsQ1Choices
		if key == SessionsWherePromptKey {
			choices = sessionsQ2Choices
		}
		v.Asks = append(v.Asks, question.Safe(question.Ask{Questions: []question.Question{
			SetupHelpedQuestion(i+1, h, choices, SetupLaterValue),
		}}))
	}
	return v, true
}

// askSessions asks Q1 and, on a yes, Q2, through p, returning the answer to
// store, and false when the answer is to decide later.
func askSessions(p Prompter) (string, bool) {
	h1, _ := sessionsHelp(SessionsPromptKey)
	switch strings.ToLower(strings.TrimSpace(promptHelped(p, h1, sessionsQ1Choices, SetupLaterValue))) {
	case "yes":
	case "no":
		return LiveSessionsNone, true
	default:
		return "", false
	}
	h2, _ := sessionsHelp(SessionsWherePromptKey)
	v := strings.ToLower(strings.TrimSpace(promptHelped(p, h2, sessionsQ2Choices, SetupLaterValue)))
	if slices.Contains(liveSessionsWhere, v) {
		return v, true
	}
	return "", false
}

// AskLiveSessions puts the open live-session question to a person at a
// terminal, as setup does, and stores the answer: the session verbs call it
// when someone tries to start a session on a project not yet set up. It asks
// nothing, and returns false, off a terminal, once the question is answered,
// and where there is no config to store the answer in.
func AskLiveSessions(cwd string, p Prompter) (string, bool, error) {
	if !atTerminal(p) || !readmeStorable(cwd) {
		return "", false, nil
	}
	if _, set, err := LiveSessions(cwd); err != nil || set {
		return "", false, err
	}
	v, ok := askSessions(p)
	if !ok {
		return "", false, nil
	}
	if _, err := StoreLiveSessions(cwd, v); err != nil {
		return "", false, err
	}
	return v, true, nil
}

// sessionsWouldChange reports whether --live-sessions differs from what is
// stored, which makes an otherwise current repository do the work.
func sessionsWouldChange(cwd string, opts InstallOptions) bool {
	if opts.LiveSessions == "" || ParseLiveSessions(opts.LiveSessions) != nil {
		return false
	}
	v, set, err := LiveSessions(cwd)
	return err != nil || !set || v != opts.LiveSessions
}

// asksSessions reports whether this run puts the questions to a person.
func (a *applyCtx) asksSessions() bool {
	return !a.autoYes && atTerminal(a.prompter) && a.has(SessionsOfferGapID) && readmeStorable(a.cwd)
}

// stepSessions asks the questions while they are open, of a person at a
// terminal, and stores that answer or a --live-sessions one, echoing the
// change. It runs after stepReadme.
func (a *applyCtx) stepSessions() {
	stored, set, err := LiveSessions(a.cwd)
	want := a.sessionsFlag
	if err != nil && want == "" {
		a.refuse(err.Error())
		return
	}
	if want == "" && !set && a.asksSessions() {
		v, ok := askSessions(a.prompter)
		if !ok {
			return // decided later: nothing stored, so the next run asks again
		}
		want = v
	}
	if want == "" || (set && want == stored) {
		return
	}
	if _, serr := StoreLiveSessions(a.cwd, want); serr != nil {
		a.refuse("the live brief session answer was not saved: " + errText(serr))
		return
	}
	from := "not set"
	if set {
		from = stored
	}
	a.echoChange("sessions.live", from, want)
	a.note(writeSettings, configPath(a.cwd))
}

// AskSessionsWhere asks Q2 alone, of a person at a terminal: `abcd session
// setup`, run by hand, asks it where nothing says where sessions may run. It
// stores nothing; false off a terminal and on deciding later.
func AskSessionsWhere(p Prompter) (string, bool) {
	if !atTerminal(p) {
		return "", false
	}
	h, _ := sessionsHelp(SessionsWherePromptKey)
	v := strings.ToLower(strings.TrimSpace(promptHelped(p, h, sessionsQ2Choices, SetupLaterValue)))
	if slices.Contains(liveSessionsWhere, v) {
		return v, true
	}
	return "", false
}
