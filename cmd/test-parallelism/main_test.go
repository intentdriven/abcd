package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// TestPrintsTheDerivedParallelism: the tool prints the -p make preflight's test
// stages take, one number and a newline, so `go test -p "$p"` reads it whole.
func TestPrintsTheDerivedParallelism(t *testing.T) {
	var out, errOut bytes.Buffer
	run(&out, &errOut, func(string) string { return "" }, func() (machineload.Snapshot, error) {
		return machineload.Snapshot{Load1: 20, HasLoad: true, Cores: 16}, nil
	}, t.TempDir())
	if got := out.String(); got != "12\n" {
		t.Fatalf("printed %q, want %q", got, "12\n")
	}
}

// idle is an idle 16-core machine.
func idle() (machineload.Snapshot, error) {
	return machineload.Snapshot{Load1: 2, HasLoad: true, Cores: 16}, nil
}

// TestDividesTheCoresAmongTheSlots: with two preflights allowed in their test
// stages at once, each stage takes at most half the cores.
func TestDividesTheCoresAmongTheSlots(t *testing.T) {
	var out, errOut bytes.Buffer
	env := func(k string) string {
		if k == implement.PreflightSlotsEnv {
			return "2"
		}
		return ""
	}
	run(&out, &errOut, env, idle, t.TempDir())
	if got := out.String(); got != "8\n" || errOut.Len() != 0 {
		t.Fatalf("printed %q (stderr %q), want %q", got, errOut.String(), "8\n")
	}
}

// TestAnUnusableSlotSettingStillPrintsANumber: the tool always prints one
// number and exits 0, so a bad setting takes one slot and says so on stderr;
// the load gate before the stage is what refuses it.
func TestAnUnusableSlotSettingStillPrintsANumber(t *testing.T) {
	var out, errOut bytes.Buffer
	env := func(k string) string {
		if k == implement.PreflightSlotsEnv {
			return "lots"
		}
		return ""
	}
	run(&out, &errOut, env, idle, t.TempDir())
	if got := out.String(); got != "16\n" || !strings.Contains(errOut.String(), implement.PreflightSlotsEnv) {
		t.Fatalf("printed %q (stderr %q), want %q and a note naming %s", got, errOut.String(), "16\n", implement.PreflightSlotsEnv)
	}
}

// TestAnUnreadableMachineConfigTakesOneSlot: a machine config.json that
// cannot be read is no error here either: the stage takes one slot, and the
// load gate before it is what names the file.
func TestAnUnreadableMachineConfigTakesOneSlot(t *testing.T) {
	home := t.TempDir()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"load": {"preflights": 2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	run(&out, &errOut, func(string) string { return "" }, idle, home)
	if got := out.String(); got != "16\n" || errOut.Len() != 0 {
		t.Fatalf("printed %q (stderr %q), want %q and nothing on stderr", got, errOut.String(), "16\n")
	}
}

// TestDividesTheCoresAmongTheSlotsSetOnTheMachine: the count the load gate
// judges by, load.preflights in the machine's config.json, is the count the
// stage divides the cores by, read through the same reader
// (implement.PreflightSlots), so the gate and -p never disagree.
func TestDividesTheCoresAmongTheSlotsSetOnTheMachine(t *testing.T) {
	home := t.TempDir()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"load": {"preflights": 2}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	run(&out, &errOut, func(string) string { return "" }, idle, home)
	if got := out.String(); got != "8\n" || errOut.Len() != 0 {
		t.Fatalf("printed %q (stderr %q), want %q", got, errOut.String(), "8\n")
	}
}

// TestTimeoutPrintsTheScaledPackageBudget: given -timeout and the stage's base
// budget, the tool prints that budget scaled with the load as the stage starts
// (implement.PreflightTimeout), as one go duration and a newline, so
// `go test -timeout "$t"` reads it whole.
func TestTimeoutPrintsTheScaledPackageBudget(t *testing.T) {
	var out, errOut bytes.Buffer
	code := runArgs([]string{"-timeout", "40m"}, &out, &errOut, func(string) string { return "" }, func() (machineload.Snapshot, error) {
		return machineload.Snapshot{Load1: 24, HasLoad: true, Cores: 16}, nil
	}, t.TempDir())
	if got := out.String(); code != 0 || got != "1h20m0s\n" || errOut.Len() != 0 {
		t.Fatalf("exit %d, printed %q (stderr %q), want 0 and %q", code, got, errOut.String(), "1h20m0s\n")
	}
}

// TestTimeoutRefusesABaseItCannotRead: a base that is not a positive go
// duration is a Makefile mistake, so the tool prints nothing on standard
// output and exits 2, and the stage's `&&` chain fails there, rather than
// handing go test an empty -timeout.
func TestTimeoutRefusesABaseItCannotRead(t *testing.T) {
	for _, base := range []string{"forty", "", "0s", "-5m"} {
		var out, errOut bytes.Buffer
		code := runArgs([]string{"-timeout", base}, &out, &errOut, func(string) string { return "" }, idle, t.TempDir())
		if code != 2 || out.Len() != 0 || !strings.Contains(errOut.String(), "-timeout") {
			t.Errorf("base %q: exit %d, printed %q (stderr %q), want exit 2, nothing printed and a note naming -timeout", base, code, out.String(), errOut.String())
		}
	}
}

// TestArgumentsItDoesNotKnowAreRefused: anything but no argument or
// `-timeout <base>` exits 2 and prints nothing, so a mistyped recipe line fails
// loudly instead of being read as a -p.
func TestArgumentsItDoesNotKnowAreRefused(t *testing.T) {
	for _, args := range [][]string{{"-timeout"}, {"-p", "4"}, {"-timeout", "40m", "extra"}} {
		var out, errOut bytes.Buffer
		if code := runArgs(args, &out, &errOut, func(string) string { return "" }, idle, t.TempDir()); code != 2 || out.Len() != 0 || errOut.Len() == 0 {
			t.Errorf("args %q: exit %d, printed %q (stderr %q), want exit 2 and a usage note", args, code, out.String(), errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := runArgs(nil, &out, &errOut, func(string) string { return "" }, idle, t.TempDir()); code != 0 || out.String() != "16\n" {
		t.Errorf("no args: exit %d, printed %q, want 0 and the -p %q", code, out.String(), "16\n")
	}
}
