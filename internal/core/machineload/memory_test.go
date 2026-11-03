package machineload

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

// TestParsePressureMemory reads the `some` line's avg10 of Linux's
// /proc/pressure/memory, and names what is missing from a malformed one.
func TestParsePressureMemory(t *testing.T) {
	good := "some avg10=14.20 avg60=3.10 avg300=0.80 total=123456\nfull avg10=9.00 avg60=1.00 avg300=0.20 total=6543\n"
	m, err := parsePressureMemory([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if m.Platform != MemoryLinux || m.StallPct != 14.2 || m.Level != "" || !strings.HasPrefix(m.Raw, "some avg10=14.20") {
		t.Fatalf("reading = %+v", m)
	}
	// The full line comes first on no kernel, but the some line is what is
	// read wherever it sits.
	m, err = parsePressureMemory([]byte("full avg10=50.00 avg60=0 avg300=0 total=0\nsome avg10=0.00 avg60=0 avg300=0 total=0\n"))
	if err != nil || m.StallPct != 0 {
		t.Fatalf("reading = %+v, %v", m, err)
	}
	for name, tc := range map[string]struct{ raw, want string }{
		"no some line":   {"full avg10=1.00 avg60=0 avg300=0 total=0\n", "no some line"},
		"empty":          {"", "no some line"},
		"no avg10":       {"some avg60=1.00 avg300=0 total=0\n", "no avg10"},
		"not a number":   {"some avg10=abc avg60=0 avg300=0 total=0\n", "not a percentage"},
		"above a whole":  {"some avg10=100.01 avg60=0 avg300=0 total=0\n", "not a percentage"},
		"negative":       {"some avg10=-1 avg60=0 avg300=0 total=0\n", "not a percentage"},
		"not a number 2": {"some avg10=NaN avg60=0 avg300=0 total=0\n", "not a percentage"},
	} {
		if _, err := parsePressureMemory([]byte(tc.raw)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, tc.want)
		}
	}
}

// TestParseMemoryPressureLevel maps macOS's levels, and reads a value abcd
// does not know as critical, never as normal.
func TestParseMemoryPressureLevel(t *testing.T) {
	for v, want := range map[uint32]string{1: MemoryNormal, 2: MemoryWarn, 3: MemoryUrgent, 4: MemoryCritical, 0: MemoryCritical, 8: MemoryCritical} {
		m := parseMemoryPressureLevel(v)
		if m.Platform != MemoryDarwin || m.Level != want {
			t.Errorf("level %d reads %+v, want %s", v, m, want)
		}
	}
	if MemoryLevelRank("mystery") != MemoryLevelRank(MemoryCritical) || !(MemoryLevelRank(MemoryNormal) < MemoryLevelRank(MemoryWarn)) ||
		!(MemoryLevelRank(MemoryWarn) < MemoryLevelRank(MemoryUrgent)) || !(MemoryLevelRank(MemoryUrgent) < MemoryLevelRank(MemoryCritical)) {
		t.Fatal("the levels are not ranked normal < warn < urgent < critical, an unknown one as critical")
	}
}

// TestReadMemoryLinuxNoFile: a kernel without the pressure stall information
// has no /proc/pressure/memory, which reads as unsupported (not checked),
// naming the kernel requirement; a file there is read.
func TestReadMemoryLinuxNoFile(t *testing.T) {
	_, err := readPressureFS(fstest.MapFS{"loadavg": {Data: []byte("1 1 1 1/1 1\n")}})
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "4.20") {
		t.Fatalf("err = %v, want unsupported naming Linux 4.20", err)
	}
	m, err := readPressureFS(fstest.MapFS{"pressure/memory": {Data: []byte("some avg10=2.50 avg60=0 avg300=0 total=0\n")}})
	if err != nil || m.StallPct != 2.5 {
		t.Fatalf("reading = %+v, %v", m, err)
	}
	_, err = readPressureFS(fstest.MapFS{"pressure/memory": {Data: []byte(strings.Repeat("x", maxPressureBytes+1))}})
	if err == nil || errors.Is(err, ErrUnsupported) {
		t.Fatalf("an oversized file reads %v, want an error that is not unsupported", err)
	}
}

// TestReadMemoryUnsupported: the other-platform reader says it cannot read,
// naming the platform, as an unsupported error.
func TestReadMemoryUnsupported(t *testing.T) {
	_, err := memoryUnsupported("plan9")
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "plan9") || !strings.Contains(err.Error(), "macOS and Linux only") {
		t.Fatalf("err = %v", err)
	}
}

// TestReadMemoryReadsThisMachine runs the real reader where there is one: a
// macOS level abcd knows, or on Linux a stall share or a kernel without one.
func TestReadMemoryReadsThisMachine(t *testing.T) {
	m, err := ReadMemory()
	switch runtime.GOOS {
	case "darwin":
		if err != nil || m.Platform != MemoryDarwin || m.Level == "" {
			t.Fatalf("reading = %+v, %v", m, err)
		}
	case "linux":
		if err != nil && !errors.Is(err, ErrUnsupported) {
			t.Fatalf("err = %v", err)
		}
		if err == nil && (m.Platform != MemoryLinux || m.StallPct < 0) {
			t.Fatalf("reading = %+v", m)
		}
	default:
		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("err = %v, want unsupported", err)
		}
	}
}
