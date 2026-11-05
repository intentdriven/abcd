package runner

// limit_shapes_test.go is how a headless limit is read: every shape the claude
// runner knows reads as a limit and keeps its reset, whatever the exit, and a
// failed launch keeps what it showed for the run's evidence file.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// limitReset is the reset the fake harness's rejected event names.
var limitReset = time.Unix(1791000000, 0).UTC()

// TestAUsageLimitResultIsALimit: a result carrying the API's limit status reads
// as a limit, on a non-zero exit and on a zero one, and it says no reset.
func TestAUsageLimitResultIsALimit(t *testing.T) {
	for _, mode := range []string{"ratelimit-apierror", "ratelimit-apierror-zero", "ratelimit-apistatus"} {
		t.Run(mode, func(t *testing.T) {
			f := newFake(t, mode, Claude)
			_, _, err := newClaude("").Run(context.Background(), f.request("implementer"))
			var fl *Failure
			if !errors.As(err, &fl) || fl.Reason != ReasonRateLimited {
				t.Fatalf("err = %v, want a %s failure", err, ReasonRateLimited)
			}
			if !fl.ResetsAt.IsZero() {
				t.Fatalf("this shape names no reset: %v", fl.ResetsAt)
			}
		})
	}
}

// TestAnExitZeroLimitKeepsItsReset: the limit a result reports with a zero exit
// names the same reset as one reported with a non-zero exit.
func TestAnExitZeroLimitKeepsItsReset(t *testing.T) {
	for _, mode := range []string{"ratelimit-exit", "ratelimit-result"} {
		t.Run(mode, func(t *testing.T) {
			f := newFake(t, mode, Claude)
			_, _, err := newClaude("").Run(context.Background(), f.request("implementer"))
			var fl *Failure
			if !errors.As(err, &fl) || fl.Reason != ReasonRateLimited {
				t.Fatalf("err = %v, want a %s failure", err, ReasonRateLimited)
			}
			if !fl.ResetsAt.Equal(limitReset) {
				t.Fatalf("the reset is %v, want %v", fl.ResetsAt, limitReset)
			}
		})
	}
}

// TestAResetWrittenAsAFractionIsStillRead: the field is read leniently, so a
// number with a fraction does not lose the reset.
func TestAResetWrittenAsAFractionIsStillRead(t *testing.T) {
	out := []byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1791000000.0}}`)
	if got := claudeRateLimitReset(out); !got.Equal(limitReset) {
		t.Fatalf("the reset is %v, want %v", got, limitReset)
	}
}

// TestAFailedLaunchKeepsWhatItShowed: the exit, the stream's shape and the first
// stderr line ride on the failure, and the shape carries no identifier and no
// free text but a failed result's.
func TestAFailedLaunchKeepsWhatItShowed(t *testing.T) {
	f := newFake(t, "ratelimit-exit", Claude)
	_, _, err := newClaude("").Run(context.Background(), f.request("implementer"))
	var fl *Failure
	if !errors.As(err, &fl) || fl.Seen == nil {
		t.Fatalf("err = %v, want a failure that says what it saw", err)
	}
	if fl.Seen.ExitCode != 1 {
		t.Fatalf("the exit is %d", fl.Seen.ExitCode)
	}
	shape := strings.Join(fl.Seen.Shape, "\n")
	for _, want := range []string{`"status":"rejected"`, `"resetsAt":1791000000`, `"rateLimitType":"five_hour"`, `"is_error":true`} {
		if !strings.Contains(shape, want) {
			t.Errorf("the shape lacks %s:\n%s", want, shape)
		}
	}
	for _, never := range []string{"fake-session-1", "session_id"} {
		if strings.Contains(shape, never) {
			t.Errorf("the shape carries %s:\n%s", never, shape)
		}
	}
}

// TestTheShapeKeepsNoIdentifierOrFreeText: a key that names a person or a session
// is dropped, an unknown string is reduced to its length, and a line that is no
// event is reduced to its length too.
func TestTheShapeKeepsNoIdentifierOrFreeText(t *testing.T) {
	out := []byte(`{"type":"system","subtype":"status","session_id":"s-1","uuid":"u-1","account_email":"pat@example.invalid","organization":"Example Org","note":"a free sentence","nested":{"user_id":"u-2","count":3}}
not an event at all`)
	shape := strings.Join(claudeShape(out), "\n")
	for _, never := range []string{"s-1", "u-1", "pat@example.invalid", "Example Org", "a free sentence", "u-2", "not an event"} {
		if strings.Contains(shape, never) {
			t.Errorf("the shape carries %q:\n%s", never, shape)
		}
	}
	for _, want := range []string{`"subtype":"status"`, `"count":3`, `"note":"[text, 15 bytes]"`, "[not json, 19 bytes]"} {
		if !strings.Contains(shape, want) {
			t.Errorf("the shape lacks %s:\n%s", want, shape)
		}
	}
}

// TestTheShapeKeepsTheLimitPastTheNoise: thinking noise and a status repeated turn
// after turn do not push the event that matters out of what is kept.
func TestTheShapeKeepsTheLimitPastTheNoise(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"type":"system","subtype":"init","claude_code_version":"2.1.318"}` + "\n")
	b.WriteString(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected"}}` + "\n")
	for range 40 {
		b.WriteString(`{"type":"system","subtype":"thinking_tokens","estimated_tokens":5}` + "\n")
		b.WriteString(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed"}}` + "\n")
	}
	got := claudeShape([]byte(b.String()))
	shape := strings.Join(got, "\n")
	for _, want := range []string{"2.1.318", `"status":"rejected"`, `"status":"allowed"`} {
		if !strings.Contains(shape, want) {
			t.Errorf("the shape lacks %s:\n%s", want, shape)
		}
	}
	if len(got) > maxShapeLines {
		t.Errorf("the shape keeps %d lines", len(got))
	}
}

// TestAKilledRunSaysHowLongItWasSilent: a run past its time, which wrote nothing
// after its start, is killed; what shows is that it went silent, and no limit is
// guessed at.
func TestAKilledRunSaysHowLongItWasSilent(t *testing.T) {
	f := newFake(t, "hang", Claude)
	req := f.request("implementer")
	req.Timeout = 300 * time.Millisecond
	_, _, err := newClaude("").Run(context.Background(), req)
	var fl *Failure
	if !errors.As(err, &fl) || !errors.Is(err, ErrKilled) || fl.Seen == nil {
		t.Fatalf("err = %v, want a killed failure that says what it saw", err)
	}
	if fl.Reason == ReasonRateLimited {
		t.Fatal("silence alone is never read as a limit")
	}
	if fl.Seen.ExitCode != -1 || fl.Seen.SilentFor <= 0 {
		t.Fatalf("the failure says %+v", fl.Seen)
	}
}

// TestARejectedEventThenSilenceReadsAsALimitAtTheDeadline: a harness that reports
// a limit and then waits without exiting is killed at its deadline and reads as
// the limit it reported, reset included.
func TestARejectedEventThenSilenceReadsAsALimitAtTheDeadline(t *testing.T) {
	f := newFake(t, "ratelimit-silent", Claude)
	req := f.request("implementer")
	req.Timeout = 300 * time.Millisecond
	_, _, err := newClaude("").Run(context.Background(), req)
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonRateLimited || !fl.ResetsAt.Equal(limitReset) {
		t.Fatalf("err = %v, want the limit it reported, with its reset", err)
	}
	if fl.Seen == nil || fl.Seen.ExitCode != -1 {
		t.Fatalf("the failure says %+v", fl.Seen)
	}
}

// walledConfig is a Config on the fake's walled login.
func walledConfig(t *testing.T, mode string) (fakeEnv, *Config) {
	t.Helper()
	f, _, p := walledFake(t, mode)
	return f, mustLoad(t, `{"runner":{"claude":{}}}`, "").Walled(p)
}

// TestAProbeTellsRenewedFromLimited: the probe's three answers, each from one
// short launch on the walled login that holds no tool.
func TestAProbeTellsRenewedFromLimited(t *testing.T) {
	t.Run("renewed", func(t *testing.T) {
		f, c := walledConfig(t, "probe-ok")
		renewed, fl := c.ProbeLimit(context.Background(), "implementer", "")
		if !renewed || fl != nil {
			t.Fatalf("a launch that answers says renewed: %v %v", renewed, fl)
		}
		argv := f.argv(t, Claude)
		if got := argWith(argv, "--tools="); got != "" || !strings.Contains(strings.Join(argv, " "), "--tools=") {
			t.Fatalf("the probe holds no tool: %q in %q", got, argv)
		}
		if strings.Contains(strings.Join(argv, " "), "Brief:") {
			t.Fatalf("the probe names no brief: %q", argv)
		}
	})
	t.Run("limited", func(t *testing.T) {
		_, c := walledConfig(t, "ratelimit-exit")
		renewed, fl := c.ProbeLimit(context.Background(), "implementer", "")
		if renewed || fl == nil || fl.Reason != ReasonRateLimited || !fl.ResetsAt.Equal(limitReset) {
			t.Fatalf("a limit says limited, with its reset: %v %+v", renewed, fl)
		}
	})
	t.Run("no answer", func(t *testing.T) {
		_, c := walledConfig(t, "notloggedin")
		renewed, fl := c.ProbeLimit(context.Background(), "implementer", "")
		if renewed || fl == nil || fl.Reason == ReasonRateLimited {
			t.Fatalf("a launch that fails otherwise says neither: %v %+v", renewed, fl)
		}
	})
}

// TestAProbeThatGotSomethingElseIsNoAnswer: a clean result that is not the word, and
// the word from a launch that held a tool, are neither of them the probe's answer.
func TestAProbeThatGotSomethingElseIsNoAnswer(t *testing.T) {
	for _, mode := range []string{"probe-word", "probe-tools", "ok"} {
		t.Run(mode, func(t *testing.T) {
			_, c := walledConfig(t, mode)
			renewed, fl := c.ProbeLimit(context.Background(), "implementer", "")
			if renewed || fl == nil || fl.Reason == ReasonRateLimited {
				t.Fatalf("this says nothing of the limit: %v %+v", renewed, fl)
			}
		})
	}
}

// TestTheProbeAsksOnTheModelItIsGiven: a limit may be one model's.
func TestTheProbeAsksOnTheModelItIsGiven(t *testing.T) {
	f, c := walledConfig(t, "probe-ok")
	if renewed, fl := c.ProbeLimit(context.Background(), "implementer", "anthropic/a-model"); !renewed {
		t.Fatalf("the probe answered: %+v", fl)
	}
	if got := argWith(f.argv(t, Claude), "--model="); got != "a-model" {
		t.Fatalf("the probe asked on %q", got)
	}
}

// TestAFailedLaunchNamesTheModelItAskedFor: the probe that follows asks on the same.
func TestAFailedLaunchNamesTheModelItAskedFor(t *testing.T) {
	f := newFake(t, "ratelimit-exit", Claude)
	req := f.request("implementer")
	req.Model = "anthropic/a-model"
	_, _, err := newClaude("anthropic/base").Run(context.Background(), req)
	var fl *Failure
	if !errors.As(err, &fl) || fl.Model != "anthropic/a-model" {
		t.Fatalf("the failure names the launch's model: %+v", fl)
	}
}

// TestAProbeRunsNoHarnessFromTheRepository: the probe admits its binary as a stage
// does, against the checkout too.
func TestAProbeRunsNoHarnessFromTheRepository(t *testing.T) {
	f, _, p := walledFake(t, "probe-ok")
	p.Repo = filepath.Dir(f.bin)
	c := mustLoad(t, `{"runner":{"claude":{}}}`, "").Walled(p)
	renewed, fl := c.ProbeLimit(context.Background(), "implementer", "")
	if renewed || fl == nil {
		t.Fatalf("a harness inside the checkout is never run: %v %+v", renewed, fl)
	}
}

// TestTheLatestStateOfTheStreamStands: a rejected event the stream goes on to lift
// is no limit, and a limit on one window is not lifted by another's status.
func TestTheLatestStateOfTheStreamStands(t *testing.T) {
	rejected := func(kind string, at int) string {
		return fmt.Sprintf(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":%d,"rateLimitType":%q}}`, at, kind)
	}
	allowed := func(kind string) string {
		return fmt.Sprintf(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":%q}}`, kind)
	}
	for _, c := range []struct {
		name  string
		lines []string
		limit bool
		reset int64
	}{
		{"lifted", []string{rejected("five_hour", 100), allowed("five_hour")}, false, 0},
		{"another window's status", []string{rejected("five_hour", 100), allowed("seven_day")}, true, 100},
		{"rejected again", []string{rejected("five_hour", 100), allowed("five_hour"), rejected("five_hour", 200)}, true, 200},
		{"the later of two", []string{rejected("five_hour", 100), rejected("seven_day", 900)}, true, 900},
		{"one of two lifted", []string{rejected("five_hour", 100), rejected("seven_day", 900), allowed("seven_day")}, true, 100},
		{"answered after", []string{rejected("five_hour", 100), `{"type":"assistant","message":{"content":[]}}`}, false, 0},
		{"finished after", []string{rejected("five_hour", 100), `{"type":"result","subtype":"success","is_error":false,"result":"done"}`}, false, 0},
		{"an assistant error", []string{`{"type":"assistant","error":"rate_limit"}`}, true, 0},
		{"its error, then an answer", []string{`{"type":"assistant","error":"rate_limit"}`, `{"type":"assistant","message":{"content":[]}}`}, false, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := []byte(strings.Join(c.lines, "\n"))
			if got := claudeRateLimit(out) != ""; got != c.limit {
				t.Fatalf("limit = %v, want %v", got, c.limit)
			}
			var want time.Time
			if c.reset > 0 {
				want = time.Unix(c.reset, 0).UTC()
			}
			if got := claudeRateLimitReset(out); !got.Equal(want) {
				t.Fatalf("reset = %v, want %v", got, want)
			}
		})
	}
}

// TestALaunchThatRecoveredAndFailedIsAnOrdinaryFailure: through a real launch.
func TestALaunchThatRecoveredAndFailedIsAnOrdinaryFailure(t *testing.T) {
	f := newFake(t, "ratelimit-recovered", Claude)
	_, _, err := newClaude("").Run(context.Background(), f.request("implementer"))
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason == ReasonRateLimited {
		t.Fatalf("err = %v, want an ordinary failure", err)
	}
}

// TestAFailureSaysWhetherItReachedTheModel: an init event on a launch that ended by
// itself, or something spent; never a launch that started nothing, and never one
// killed having spent nothing, which may be a limit waited out inside it.
func TestAFailureSaysWhetherItReachedTheModel(t *testing.T) {
	run := func(t *testing.T, mode string, timeout time.Duration) *Failure {
		t.Helper()
		f := newFake(t, mode, Claude)
		req := f.request("implementer")
		req.Timeout = timeout
		_, _, err := newClaude("").Run(context.Background(), req)
		var fl *Failure
		if !errors.As(err, &fl) {
			t.Fatalf("err = %v", err)
		}
		return fl
	}
	if fl := run(t, "started-exit", 0); !fl.Seen.Started || !fl.ReachedModel() {
		t.Errorf("an init event, then a failed exit: %+v %+v", fl, fl.Seen)
	}
	if fl := run(t, "notloggedin", 0); fl.ReachedModel() {
		t.Errorf("no event at all: %+v", fl)
	}
	if fl := run(t, "hang", 300*time.Millisecond); fl.ReachedModel() {
		t.Errorf("killed, nothing spent: %+v", fl)
	}
	if !(&Failure{Reason: ReasonInvalid}).ReachedModel() {
		t.Error("an answer the contract refused came from the model")
	}
}

// TestTheRunnerCutsNoTextTheScannerHasNotSeen: the shape keeps a failed result whole,
// so the redactor sees all of it, and one too long to read reduces to its length.
func TestTheRunnerCutsNoTextTheScannerHasNotSeen(t *testing.T) {
	kept := strings.Repeat("k", 5000)
	out := []byte(`{"type":"result","subtype":"success","is_error":true,"result":"` + kept + `"}`)
	if shape := strings.Join(claudeShape(out), ""); !strings.Contains(shape, kept) {
		t.Fatalf("the result is cut before the scanner sees it: %d bytes", len(shape))
	}
	huge := strings.Repeat("h", 2*maxShapeText)
	out = []byte(`{"type":"result","subtype":"success","is_error":true,"result":"` + huge + `"}`)
	if shape := strings.Join(claudeShape(out), ""); strings.Contains(shape, "hhhh") {
		t.Fatalf("a result too long to read is reduced to its length: %d bytes", len(shape))
	}
}

// TestProveWallSaysALimitIsALimit: met at the proof, on either exit, it is no wall,
// and it carries the reset it named.
func TestProveWallSaysALimitIsALimit(t *testing.T) {
	for mode, reset := range map[string]time.Time{
		"wall-limit-exit": limitReset,
		"wall-limit-zero": limitReset,
		"wall-limit-none": {},
	} {
		t.Run(mode, func(t *testing.T) {
			newFake(t, mode, Claude)
			err := ProveWall(context.Background(), wallProfile(t), wallBin)
			var le *LimitError
			if !errors.As(err, &le) {
				t.Fatalf("err = %v, want a limit", err)
			}
			if !le.ResetsAt.Equal(reset) {
				t.Fatalf("the reset is %v, want %v", le.ResetsAt, reset)
			}
			var we *WallError
			if errors.As(err, &we) {
				t.Fatalf("a limit blames no wall: %v", err)
			}
		})
	}
}
