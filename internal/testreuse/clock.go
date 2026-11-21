package testreuse

// make preflight's clock (iss-2611190439574731): where a run's time went, so its
// last lines can give each stage's time and the whole run's checked time with the
// waiting left out (criterion 6 of itd-2611011325421304 sets a run's check time
// against a baseline).
//
// make preflight names one clock file per run in the environment (ClockEnv),
// under the checkout's local tier, .abcd/.work.local/preflight-clock/<start>-<pid>,
// where <start> is the run's start in Unix seconds. scripts/preflight-receipt.sh
// makes the name and notes the waits at the load gate and the race gate; each
// test stage notes here its checking time and, apart from it, its own waits for
// the check directory's lock and for the race gate again. Each note is one line,
// "<kind> <name> <seconds>", and the receipt script sums them after the receipt
// line. A clock is a local report, never a gate: a note that cannot be written
// loses that figure and stops nothing.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	// ClockEnv names the run's clock file, an absolute path.
	ClockEnv = "ABCD_PREFLIGHT_CLOCK"
	// clockWait is time spent waiting for a turn. The run's checked time leaves
	// it out.
	clockWait = "wait"
	// clockStage is time a test stage spent checking.
	clockStage = "stage"
)

// clockWord is a name the receipt script reads as one column.
var clockWord = regexp.MustCompile(`^[a-z]+$`)

// noteClock appends "<kind> <name> <seconds>" to the clock, making the file and
// its directory if need be. An empty clock is no write: outside make preflight
// none is named. A negative d notes zero.
func noteClock(clock, kind, name string, d time.Duration) error {
	if clock == "" {
		return nil
	}
	if !filepath.IsAbs(clock) {
		return fmt.Errorf("the clock %q is not an absolute path", clock)
	}
	if kind != clockWait && kind != clockStage {
		return fmt.Errorf("%q is not a kind of time the clock keeps", kind)
	}
	if !clockWord.MatchString(name) {
		return fmt.Errorf("%q is not one lower-case word", name)
	}
	if err := os.MkdirAll(filepath.Dir(clock), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(clock, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	// One write of one short line: O_APPEND puts each writer's line whole at the
	// end, so the script's notes and a stage's never cut into each other.
	_, werr := fmt.Fprintf(f, "%s %s %.3f\n", kind, name, max(d, 0).Seconds())
	return errors.Join(werr, f.Close())
}

// formatClock is the report's form of a duration, in whole seconds: 42s, 3m12s,
// 41m07s, 1h02m03s. time.ParseDuration reads it back, and the receipt script
// prints the same form.
func formatClock(d time.Duration) string {
	s := int64(max(d, 0).Round(time.Second) / time.Second)
	h, m := s/3600, s%3600/60
	s %= 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// stageClock is where a stage's time went: since began, lock waiting for another
// check's lock and gate going back through the race gate. The rest is checking.
type stageClock struct {
	began      time.Time
	lock, gate time.Duration
}

// report says the stage's time beside its reused line and notes it on the
// preflight's clock: what it spent checking, and apart from it what it waited.
// A clock that cannot be written loses the figure and fails nothing.
func (sc *stageClock) report(cfg Config, gated bool) {
	if sc == nil {
		return
	}
	waited := sc.lock + sc.gate
	checked := time.Since(sc.began) - waited
	at := "the check directory's lock"
	if gated {
		at += " and the race gate"
	}
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage checked in %s (waited %s at %s)\n",
		cfg.Stage, formatClock(checked), formatClock(waited), at)
	err := noteClock(cfg.Clock, clockStage, cfg.Stage, checked)
	if err == nil && sc.lock > 0 {
		err = noteClock(cfg.Clock, clockWait, "lock", sc.lock)
	}
	if err == nil && sc.gate > 0 {
		err = noteClock(cfg.Clock, clockWait, "race", sc.gate)
	}
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "check-tests: the preflight's clock was not written, so its last lines will lack this stage's time: %v\n", err)
	}
}
