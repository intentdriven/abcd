package runner

// limit_shapes_test.go is how a headless limit is read: every shape the claude
// runner knows reads as a limit and keeps its reset, whatever the exit, and a
// failed launch keeps what it showed for the run's evidence file.

import (
	"context"
	"errors"
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
		f, c := walledConfig(t, "ok")
		renewed, fl := c.ProbeLimit(context.Background(), "implementer")
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
		renewed, fl := c.ProbeLimit(context.Background(), "implementer")
		if renewed || fl == nil || fl.Reason != ReasonRateLimited || !fl.ResetsAt.Equal(limitReset) {
			t.Fatalf("a limit says limited, with its reset: %v %+v", renewed, fl)
		}
	})
	t.Run("no answer", func(t *testing.T) {
		_, c := walledConfig(t, "notloggedin")
		renewed, fl := c.ProbeLimit(context.Background(), "implementer")
		if renewed || fl == nil || fl.Reason == ReasonRateLimited {
			t.Fatalf("a launch that fails otherwise says neither: %v %+v", renewed, fl)
		}
	})
}
