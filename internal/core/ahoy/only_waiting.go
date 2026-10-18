package ahoy

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// The setup the yes to the after-update offer runs (spc-2610170635097509,
// step 7): `abcd ahoy install --only-waiting`. It is the ordinary install
// with two changes, both made here before the install's first question: the
// prompter it asks through puts only what Waiting returns for this
// repository, and every kind of change is approved up front, since the yes
// to the offer was the approval. A question that does not wait (one this
// repository answered, or deferred with "decide later") is answered with no
// answer, exactly as a door answers "decide later", so the install treats it
// as it treats every unanswered question.

// onlyWaitingRun is what --only-waiting changes for the install at abs,
// whose detection is det: the options with every category det carries
// approved (unless Yes or an explicit approval set decides them), and the
// filter the prompter is wrapped in. A run without OnlyWaiting is returned
// unchanged, with a nil filter.
func onlyWaitingRun(abs string, det DetectionResult, opts InstallOptions) (InstallOptions, *waitingFilter, error) {
	if !opts.OnlyWaiting {
		return opts, nil, nil
	}
	from, _ := SetupVersion(abs)
	to := ""
	if sv, err := launch.ParseReleaseTag(pluginVersion()); err == nil {
		to = sv.Tag()
	}
	w, err := Waiting(abs, det, from, to)
	if err != nil {
		return opts, nil, err
	}
	f := &waitingFilter{keys: map[string]bool{}, none: opts.Yes}
	for _, it := range w.Questions {
		f.keys[it.Key] = true
	}
	for _, it := range w.Defaults {
		f.keys[it.Key] = true
	}
	if !opts.Yes && opts.ApprovedCategories == nil {
		approved := map[GapCategory]bool{ConfigChange: true}
		for _, g := range det.Gaps {
			approved[g.Category] = true
		}
		opts.ApprovedCategories = approved
	}
	return opts, f, nil
}

// waitingFilter decides which questions an --only-waiting run puts: the keys
// that wait, and none at all under --yes.
type waitingFilter struct {
	keys map[string]bool
	none bool
}

// asks reports whether the question key is put. A README part rides on the
// README question, a status line element on the status line offer, and where
// live sessions run on whether to set them up.
func (f *waitingFilter) asks(key string) bool {
	if f.none || key == "" {
		return false
	}
	switch {
	case strings.HasPrefix(key, readmePartKeyPrefix):
		key = ReadmePromptKey
	case strings.HasPrefix(key, elementPromptPrefix):
		key = StatusLineOfferGapID
	case key == SessionsWherePromptKey:
		key = SessionsPromptKey // where sessions run follows whether to set them up
	}
	return f.keys[key]
}

// prompter wraps p in the filter.
func (f *waitingFilter) prompter(p Prompter) Prompter {
	return &waitingPrompter{inner: p, filter: f}
}

// waitingPrompter is the prompter of an --only-waiting run: a question that
// waits reaches inner; any other value question is answered with no answer
// and any other offer declined. The adoption and a category approval are
// consent, not a setting answered once, so they always reach inner.
type waitingPrompter struct {
	inner  Prompter
	filter *waitingFilter
}

// Confirm puts the adoption, an approval, or an offer that waits.
func (w *waitingPrompter) Confirm(text string) bool {
	id := setupConfirmID(text)
	if id == "adopt" || strings.HasPrefix(id, approvePrefix) || w.filter.asks(id) {
		return w.inner.Confirm(text)
	}
	return false
}

// Prompt puts a value question that waits; any other takes no answer.
func (w *waitingPrompter) Prompt(key string, choices []string, def string) string {
	if !w.filter.asks(key) {
		return ""
	}
	return w.inner.Prompt(key, choices, def)
}

// PromptHelped is Prompt for a question with help built for this run.
func (w *waitingPrompter) PromptHelped(h PromptHelp, choices []string, def string) string {
	if !w.filter.asks(h.Key) {
		return ""
	}
	return promptHelped(w.inner, h, choices, def)
}

// AtTerminal is inner's.
func (w *waitingPrompter) AtTerminal() bool { return atTerminal(w.inner) }

// leftForAPerson reports, noting it, that the question what names is left
// unanswered because this run answers none: an --only-waiting --yes run
// leaves a value question the install would otherwise settle by default.
func (a *applyCtx) leftForAPerson(what string) bool {
	if a.waiting == nil || !a.waiting.none {
		return false
	}
	a.refuse(what + " was left unanswered: an --only-waiting --yes run answers no setup question, so it still waits; " +
		"run abcd ahoy install --only-waiting at a terminal, without --yes, to answer it")
	return true
}
