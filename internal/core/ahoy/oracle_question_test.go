package ahoy

import (
	"strings"
	"testing"
)

// askedRecorder answers every value question from answers (the default
// when a key has none) and remembers which keys it was asked.
type askedRecorder struct {
	answers map[string]string
	asked   []string
}

func (r *askedRecorder) Confirm(string) bool { return true }

func (r *askedRecorder) Prompt(key string, _ []string, def string) string {
	r.asked = append(r.asked, key)
	if v, ok := r.answers[key]; ok {
		return v
	}
	return def
}

func (r *askedRecorder) wasAsked(key string) bool {
	for _, k := range r.asked {
		if k == key {
			return true
		}
	}
	return false
}

// oracleCollectCtx builds the collect-missing applyCtx for a first install: the
// visibility, docs target and oracle backend are all unset.
func oracleCollectCtx(dir string, p Prompter, overrides map[string]string) *applyCtx {
	return &applyCtx{
		cwd:      dir,
		approved: map[GapCategory]bool{ConfigChange: true},
		gapPresent: map[string]bool{
			"config.visibility_missing":     true,
			"config.docs_target_missing":    true,
			"config.oracle_backend_missing": true,
		},
		prompter:  p,
		overrides: overrides,
	}
}

func persistedOracleBackend(t *testing.T, dir string) string {
	t.Helper()
	m, err := readConfig(dir)
	if err != nil || m == nil {
		t.Fatalf("config.json was not written: %v", err)
	}
	v, _ := stringVal(subMap(m, "oracle"), "backend")
	return v
}

// TestSetupRecordsTheOnlyReviewerWithoutAskingForIt is the 2026-10-03 ruling on
// iss-2610031236155833: while host-delegated is the only oracle answer with an
// adapter, the install does not ask the question. It records host-delegated and
// says so in one line of its report, naming the flag that chooses another
// reviewer once one arrives.
func TestSetupRecordsTheOnlyReviewerWithoutAskingForIt(t *testing.T) {
	dir := t.TempDir()
	p := &askedRecorder{answers: map[string]string{"visibility": "private", "docs_target": "skip"}}
	a := oracleCollectCtx(dir, p, nil)

	cfg := a.stepConfigValues()
	if cfg == nil {
		t.Fatal("stepConfigValues returned nil for a fully answerable install")
	}
	if p.wasAsked("oracle_backend") {
		t.Errorf("the install asked oracle_backend while only host-delegated has an adapter (asked %v)", p.asked)
	}
	if cfg.OracleBackend != oracleBackendDefault {
		t.Errorf("OracleBackend = %q, want %q", cfg.OracleBackend, oracleBackendDefault)
	}
	if got := persistedOracleBackend(t, dir); got != oracleBackendDefault {
		t.Errorf("oracle.backend persisted as %q, want %q", got, oracleBackendDefault)
	}
	lines := 0
	for _, n := range a.notes {
		if n == oracleBackendRecordedNote {
			lines++
		}
	}
	if lines != 1 {
		t.Errorf("the report carries the oracle line %d times, want once: %q", lines, a.notes)
	}
	if !strings.Contains(oracleBackendRecordedNote, "--oracle-backend") {
		t.Errorf("the oracle line does not name the flag that chooses another reviewer: %q", oracleBackendRecordedNote)
	}
}

// TestTheOracleFlagStillChoosesTheReviewer: a non-interactive install that
// passes --oracle-backend keeps working exactly as before the ruling. The value
// it names is recorded, nothing is asked, and no line claims a default.
func TestTheOracleFlagStillChoosesTheReviewer(t *testing.T) {
	dir := t.TempDir()
	p := &askedRecorder{answers: map[string]string{"visibility": "private", "docs_target": "skip"}}
	a := oracleCollectCtx(dir, p, map[string]string{"oracle_backend": "mcp"})

	cfg := a.stepConfigValues()
	if cfg == nil {
		t.Fatal("stepConfigValues returned nil")
	}
	if p.wasAsked("oracle_backend") {
		t.Errorf("the flag answered oracle_backend, yet it was asked")
	}
	if cfg.OracleBackend != "mcp" || persistedOracleBackend(t, dir) != "mcp" {
		t.Errorf("the flag's value was not recorded: cfg %q", cfg.OracleBackend)
	}
	for _, n := range a.notes {
		if n == oracleBackendRecordedNote {
			t.Errorf("a flag-chosen reviewer is reported as the recorded default: %q", a.notes)
		}
	}
}

// TestTheOracleQuestionReturnsWithASecondAdapter pins the rule that brings the
// question back: once a second answer is marked as having an adapter, the
// install asks again, with no other change needed.
func TestTheOracleQuestionReturnsWithASecondAdapter(t *testing.T) {
	if oracleBackendAsked() {
		t.Fatal("the oracle question is asked with only host-delegated shipped; the precondition is wrong")
	}
	oracleAdapterShipped["native"] = true
	t.Cleanup(func() { delete(oracleAdapterShipped, "native") })
	if !oracleBackendAsked() {
		t.Fatal("a second answer with an adapter does not bring the question back")
	}

	dir := t.TempDir()
	p := &askedRecorder{answers: map[string]string{"visibility": "private", "docs_target": "skip", "oracle_backend": "native"}}
	a := oracleCollectCtx(dir, p, nil)
	cfg := a.stepConfigValues()
	if cfg == nil {
		t.Fatal("stepConfigValues returned nil")
	}
	if !p.wasAsked("oracle_backend") {
		t.Errorf("the question did not return with two answers shipped (asked %v)", p.asked)
	}
	if cfg.OracleBackend != "native" {
		t.Errorf("the answer was not recorded: %q", cfg.OracleBackend)
	}
	for _, n := range a.notes {
		if n == oracleBackendRecordedNote {
			t.Errorf("an asked question is reported as recorded without asking: %q", a.notes)
		}
	}
}

// TestOracleMeaningsAgreeWithTheShippedAdapters keeps the question's words and
// the set that decides whether it is asked in one step: an answer says it has
// no adapter exactly when it is not marked as shipped, so marking an adapter
// means rewording its meaning in the same change.
func TestOracleMeaningsAgreeWithTheShippedAdapters(t *testing.T) {
	h, _ := HelpFor("oracle_backend")
	for _, c := range h.Choices {
		says := strings.HasSuffix(c.Meaning, noAdapterYet)
		if says == oracleAdapterShipped[c.Value] {
			t.Errorf("%s: shipped=%v but its meaning says no adapter=%v", c.Value, oracleAdapterShipped[c.Value], says)
		}
	}
	for v := range oracleAdapterShipped {
		if !inSet(v, oracleBackendChoices) {
			t.Errorf("%s is marked shipped but is not an oracle answer", v)
		}
	}
}

// TestTheOracleGapClaimsNoQuestionWhileNoneIsAsked: the missing-value gap's
// fix hint said the install asks for the value; while the question is not
// asked it says what the install records instead, and still names the flag.
func TestTheOracleGapClaimsNoQuestionWhileNoneIsAsked(t *testing.T) {
	g := configValueGap("config.oracle_backend_missing", "oracle_backend", "t", "d")
	if strings.Contains(g.FixHint, "asks") {
		t.Errorf("the oracle gap claims a question the install does not ask: %q", g.FixHint)
	}
	if !strings.Contains(g.FixHint, oracleBackendDefault) || !strings.Contains(g.FixHint, "--oracle-backend") {
		t.Errorf("the oracle gap does not say what is recorded and which flag chooses another: %q", g.FixHint)
	}
}
