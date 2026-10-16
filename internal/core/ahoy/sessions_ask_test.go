package ahoy

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The live brief session questions at repository setup
// (spc-2610141709430146 step 6, for itd-2610091016262335).

// sessionsKeys is the live-session questions asked, in order.
func (p *helpedPrompter) sessionsKeys() []string {
	var out []string
	for _, a := range p.asked {
		if a.key == SessionsPromptKey || a.key == SessionsWherePromptKey {
			out = append(out, a.key)
		}
	}
	return out
}

// storedLiveSessions is sessions.live in the repository's config.
func storedLiveSessions(t *testing.T, repo string) (string, bool) {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := subMap(cfg, "sessions")["live"]
	if !ok {
		return "", false
	}
	s, _ := v.(string)
	return s, true
}

// TestSessionsQuestionsAskedAtTerminal: at a terminal, setup asks whether to
// set up live brief sessions, then, on a yes, where they may run; each with
// its current setting, its change-later line and deciding later. The answer
// is stored as sessions.live.
func TestSessionsQuestionsAskedAtTerminal(t *testing.T) {
	for _, where := range []string{"local", "cloud", "both"} {
		t.Run(where, func(t *testing.T) {
			setupHermetic(t)
			repo := installedRepo(t)
			p := &helpedPrompter{confirm: approveCategories, answers: map[string]string{
				SessionsPromptKey: "yes", SessionsWherePromptKey: where,
			}}
			res := terminalInstall(t, repo, p)
			if got := p.sessionsKeys(); !reflect.DeepEqual(got, []string{SessionsPromptKey, SessionsWherePromptKey}) {
				t.Fatalf("live-session questions asked %q", got)
			}
			q1 := p.find(t, SessionsPromptKey)
			if !reflect.DeepEqual(q1.choices, []string{"yes", "no", SetupLaterValue}) || q1.def != SetupLaterValue {
				t.Errorf("Q1 offers %q (default %q)", q1.choices, q1.def)
			}
			q2 := p.find(t, SessionsWherePromptKey)
			if !reflect.DeepEqual(q2.choices, []string{"local", "cloud", "both", SetupLaterValue}) || q2.def != SetupLaterValue {
				t.Errorf("Q2 offers %q (default %q)", q2.choices, q2.def)
			}
			for _, a := range []helpedAsk{q1, q2} {
				if !a.helped || a.help.Now == "" || a.help.ChangeLater == "" {
					t.Errorf("%s asked without its current setting or change-later line: %+v", a.key, a.help)
				}
			}
			if !strings.HasSuffix(q1.help.About, "Set up live brief sessions?") {
				t.Errorf("Q1 does not end on its question: %q", q1.help.About)
			}
			if !strings.HasSuffix(q2.help.About, "Where may sessions run?") {
				t.Errorf("Q2 does not end on its question: %q", q2.help.About)
			}
			if got, ok := storedLiveSessions(t, repo); !ok || got != where {
				t.Fatalf("sessions.live = %q (stored %v), want %q", got, ok, where)
			}
			if !containsString(res.Changes, "sessions.live: not set -> "+where) {
				t.Errorf("changes = %q", res.Changes)
			}
			again := &helpedPrompter{confirm: approveCategories}
			terminalInstall(t, repo, again)
			if got := again.sessionsKeys(); len(got) != 0 {
				t.Errorf("an answered question was asked again: %q", got)
			}
		})
	}
}

// TestSessionsNoIsNeverAskedAgain: a no is stored as none and never asked
// again, at setup or anywhere.
func TestSessionsNoIsNeverAskedAgain(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	p := &helpedPrompter{confirm: approveCategories, answers: map[string]string{SessionsPromptKey: "no"}}
	terminalInstall(t, repo, p)
	if got := p.sessionsKeys(); !reflect.DeepEqual(got, []string{SessionsPromptKey}) {
		t.Fatalf("asked %q, want Q1 alone", got)
	}
	if got, ok := storedLiveSessions(t, repo); !ok || got != LiveSessionsNone {
		t.Fatalf("a no stored %q (stored %v), want none", got, ok)
	}
	again := &helpedPrompter{confirm: approveCategories}
	res := terminalInstall(t, repo, again)
	if got := again.sessionsKeys(); len(got) != 0 {
		t.Errorf("a stored no was asked again: %q", got)
	}
	if containsString(res.OptionalSkipped, SessionsOfferGapID) {
		t.Errorf("a stored no is still reported skipped: %v", res.OptionalSkipped)
	}
	asked := &helpedPrompter{}
	if _, answered, err := AskLiveSessions(repo, asked); err != nil || answered || len(asked.asked) != 0 {
		t.Errorf("AskLiveSessions after a no: answered %v, err %v, asked %q", answered, err, asked.keys())
	}
}

// TestSessionsLaterAsksAgain: deciding later, at either question, stores
// nothing, so the next setup run asks again.
func TestSessionsLaterAsksAgain(t *testing.T) {
	for _, answers := range []map[string]string{
		{SessionsPromptKey: SetupLaterValue},
		{SessionsPromptKey: "", SessionsWherePromptKey: ""},
		{SessionsPromptKey: "yes", SessionsWherePromptKey: SetupLaterValue},
	} {
		setupHermetic(t)
		repo := installedRepo(t)
		terminalInstall(t, repo, &helpedPrompter{confirm: approveCategories, answers: answers})
		if got, ok := storedLiveSessions(t, repo); ok {
			t.Fatalf("%v stored %q, want nothing", answers, got)
		}
		again := &helpedPrompter{confirm: approveCategories}
		terminalInstall(t, repo, again)
		if got := again.sessionsKeys(); !slices.Contains(got, SessionsPromptKey) {
			t.Errorf("%v: the next run did not ask again: %q", answers, again.keys())
		}
	}
}

// TestSessionsOffTerminalIsReportedSkipped: off a terminal the questions are
// never asked, and the run reports them skipped, naming the flag.
func TestSessionsOffTerminalIsReportedSkipped(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(res.OptionalSkipped, SessionsOfferGapID) {
		t.Fatalf("optional_skipped = %v, want %s", res.OptionalSkipped, SessionsOfferGapID)
	}
	if _, ok := storedLiveSessions(t, repo); ok {
		t.Fatal("an unasked question stored an answer")
	}
	h, ok := optionalSkippedHelp[SessionsOfferGapID]
	if !ok || !strings.Contains(h.Action, "--live-sessions") {
		t.Errorf("the skipped help does not name the flag: %+v", h)
	}
}

// TestSessionsFlagAnswers: --live-sessions answers in any run, --yes
// included, and is echoed as a change; an unknown value refuses the run
// before anything is written.
func TestSessionsFlagAnswers(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	opts := installOpts()
	opts.LiveSessions = "local"
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := storedLiveSessions(t, repo); got != "local" {
		t.Fatalf("sessions.live = %q after --live-sessions local (status %s, notes %v)", got, res.Status, res.Notes)
	}
	if !containsString(res.Changes, "sessions.live: not set -> local") {
		t.Errorf("changes = %q", res.Changes)
	}
	opts.LiveSessions = "none"
	if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := storedLiveSessions(t, repo); got != "none" {
		t.Fatalf("sessions.live = %q after --live-sessions none", got)
	}
	opts.LiveSessions = "everywhere"
	res, err = Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" || len(res.Notes) == 0 || !strings.Contains(res.Notes[0], "local, cloud, both or none") {
		t.Errorf("an unknown value: status %q notes %q", res.Status, res.Notes)
	}
	if got, _ := storedLiveSessions(t, repo); got != "none" {
		t.Errorf("a refused value changed sessions.live to %q", got)
	}
}

// TestSessionsViewPassesTheLimits: while the question is open, detection
// carries both questions as the host's question tool takes them, each passing
// every asking limit, Q1 and Q2 in calls of their own.
func TestSessionsViewPassesTheLimits(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	det, err := DryRun(repo)
	if err != nil {
		t.Fatal(err)
	}
	if det.Sessions == nil || len(det.Sessions.Asks) != 2 {
		t.Fatalf("detection carries no live-session view: %+v", det.Sessions)
	}
	for i, a := range det.Sessions.Asks {
		if len(a.Questions) != 1 {
			t.Fatalf("call %d holds %d questions", i, len(a.Questions))
		}
		for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
			t.Errorf("call %d: %+v", i, f)
		}
		q := a.Questions[0]
		if q.Later.Label != question.Default.LaterLabels[0] {
			t.Errorf("call %d's last answer is %q", i, q.Later.Label)
		}
	}
	if det.Sessions.Asks[0].Questions[0].ID != SessionsPromptKey || det.Sessions.Asks[1].Questions[0].ID != SessionsWherePromptKey {
		t.Errorf("questions out of order: %+v", det.Sessions.Asks)
	}
	if err := ParseLiveSessions("both"); err != nil {
		t.Fatal(err)
	}
	if _, err := StoreLiveSessions(repo, "cloud"); err != nil {
		t.Fatal(err)
	}
	det, err = DryRun(repo)
	if err != nil {
		t.Fatal(err)
	}
	if det.Sessions != nil {
		t.Errorf("an answered question still carries its view")
	}
}

// TestAskLiveSessions: the session verbs ask the open question of a person at
// a terminal through the same words, store the answer, and ask nothing off a
// terminal.
func TestAskLiveSessions(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	if v, answered, err := AskLiveSessions(repo, RefusingPrompter{}); err != nil || answered || v != "" {
		t.Fatalf("off a terminal: %q %v %v", v, answered, err)
	}
	p := &helpedPrompter{answers: map[string]string{SessionsPromptKey: "yes", SessionsWherePromptKey: "both"}}
	v, answered, err := AskLiveSessions(repo, p)
	if err != nil || !answered || v != "both" {
		t.Fatalf("at a terminal: %q %v %v (asked %q)", v, answered, err, p.keys())
	}
	if got, _ := storedLiveSessions(t, repo); got != "both" {
		t.Errorf("sessions.live = %q", got)
	}
	if got, set, err := LiveSessions(repo); err != nil || !set || got != "both" {
		t.Errorf("LiveSessions = %q %v %v", got, set, err)
	}
}

// TestAskSessionsWhere: running the setup by hand asks where sessions may
// run alone, of a person at a terminal, with the setup's own words; off a
// terminal, and on deciding later, it has no answer.
func TestAskSessionsWhere(t *testing.T) {
	if _, ok := AskSessionsWhere(RefusingPrompter{}); ok {
		t.Fatal("answered off a terminal")
	}
	p := &helpedPrompter{answers: map[string]string{SessionsWherePromptKey: "cloud"}}
	if v, ok := AskSessionsWhere(p); !ok || v != "cloud" || !reflect.DeepEqual(p.keys(), []string{SessionsWherePromptKey}) {
		t.Fatalf("at a terminal: %q %v, asked %q", v, ok, p.keys())
	}
	later := &helpedPrompter{answers: map[string]string{SessionsWherePromptKey: SetupLaterValue}}
	if _, ok := AskSessionsWhere(later); ok {
		t.Error("deciding later answered")
	}
}
