package machineload

import (
	"encoding/binary"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestParseLoadavgSysctlBytes reads the 23-byte form syscall.Sysctl returns for
// vm.loadavg (one trailing NUL stripped from the 24-byte struct): three
// fixed-point averages, padding, and the scale.
func TestParseLoadavgSysctlBytes(t *testing.T) {
	full := make([]byte, 24)
	binary.LittleEndian.PutUint32(full[0:], 28242) // 13.79 * 2048
	binary.LittleEndian.PutUint32(full[4:], 37683) // 18.40 * 2048
	binary.LittleEndian.PutUint32(full[8:], 38502) // 18.80 * 2048
	binary.LittleEndian.PutUint64(full[16:], 2048) // the scale, high bytes zero
	stripped := full[:23]                          // syscall.Sysctl drops the trailing NUL

	for name, raw := range map[string][]byte{"stripped": stripped, "full": full} {
		l1, l5, l15, err := ParseLoadavgSysctl(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		near := func(got, want float64) bool { return got > want-0.005 && got < want+0.005 }
		if !near(l1, 13.79) || !near(l5, 18.40) || !near(l15, 18.80) {
			t.Fatalf("%s: loads = %v %v %v, want 13.79 18.40 18.80", name, l1, l5, l15)
		}
	}
	if _, _, _, err := ParseLoadavgSysctl(full[:12]); err == nil {
		t.Fatal("a 12-byte value parsed as a loadavg struct")
	}
	zeroScale := make([]byte, 24)
	if _, _, _, err := ParseLoadavgSysctl(zeroScale); err == nil {
		t.Fatal("a zero scale parsed")
	}
}

// TestParseDarwinPS reads a macOS ps listing: names with spaces, full paths
// reduced to base names, etime with days, and CPU time with minutes past 59.
func TestParseDarwinPS(t *testing.T) {
	out := []byte(`    1     0     1     0 13:43:52  17:56.08 /sbin/launchd
  151 99661 99661   501 13:34:43   1:57.36 /Applications/Some Browser.app/Contents/MacOS/Some Browser Helper
41233     1 41230   501 2-07:00:00 3288:00.00 yes
  900   899   900   502    05:03   114:03.49 zsh
  901   899   900   502  1-00:00:00 1-02:03:04.50 /usr/bin/burner
`)
	procs, err := parseDarwinPS(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) != 5 {
		t.Fatalf("%d processes, want 5", len(procs))
	}
	want := []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 13*time.Hour + 43*time.Minute + 52*time.Second, CPU: 17*time.Minute + 56*time.Second + 80*time.Millisecond, Name: "launchd"},
		{PID: 151, PPID: 99661, PGID: 99661, UID: 501, Age: 13*time.Hour + 34*time.Minute + 43*time.Second, CPU: time.Minute + 57*time.Second + 360*time.Millisecond, Name: "Some Browser Helper"},
		{PID: 41233, PPID: 1, PGID: 41230, UID: 501, Age: 55 * time.Hour, CPU: 3288 * time.Minute, Name: "yes"},
		{PID: 900, PPID: 899, PGID: 900, UID: 502, Age: 5*time.Minute + 3*time.Second, CPU: 114*time.Minute + 3*time.Second + 490*time.Millisecond, Name: "zsh"},
		{PID: 901, PPID: 899, PGID: 900, UID: 502, Age: 24 * time.Hour, CPU: 26*time.Hour + 3*time.Minute + 4*time.Second + 500*time.Millisecond, Name: "burner"},
	}
	for i, w := range want {
		g := procs[i]
		// Fractions pass through a float; a millisecond is well inside what matters.
		if d := g.CPU - w.CPU; d > time.Millisecond || d < -time.Millisecond {
			t.Errorf("proc %d CPU = %v, want %v", w.PID, g.CPU, w.CPU)
		}
		g.CPU = w.CPU
		if g != w {
			t.Errorf("proc %d = %+v, want %+v", w.PID, g, w)
		}
	}
	for _, bad := range []string{"1 0 1 0 00:01\n", "x 0 1 0 00:01 0:00.00 a\n", "1 0 1 0 bad 0:00.00 a\n"} {
		if _, err := parseDarwinPS([]byte(bad)); err == nil {
			t.Errorf("parsed a malformed line %q", bad)
		}
	}
}

// TestParseOnlineRange counts the cores in /sys/devices/system/cpu/online.
func TestParseOnlineRange(t *testing.T) {
	for in, want := range map[string]int{"0\n": 1, "0-3\n": 4, "0-3,8-11\n": 8, "0,2,4-5": 4} {
		got, err := parseOnlineRange([]byte(in))
		if err != nil || got != want {
			t.Errorf("parseOnlineRange(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "a-b", "3-1", "-1"} {
		if _, err := parseOnlineRange([]byte(bad)); err == nil {
			t.Errorf("parseOnlineRange(%q) accepted", bad)
		}
	}
}

// linuxStat builds a /proc/<pid>/stat line: comm, then state, ppid, pgrp, and
// zeros up to utime (14), stime (15) and starttime (22).
func linuxStat(pid, comm, state string, ppid, pgrp, utime, stime, start int) string {
	f := []string{pid, "(" + comm + ")", state, strconv.Itoa(ppid), strconv.Itoa(pgrp)}
	for i := 6; i <= 13; i++ {
		f = append(f, "0")
	}
	f = append(f, strconv.Itoa(utime), strconv.Itoa(stime))
	for i := 16; i <= 21; i++ {
		f = append(f, "0")
	}
	f = append(f, strconv.Itoa(start), "0", "0")
	return strings.Join(f, " ") + "\n"
}

// TestParseLinuxProc reads a /proc tree: a comm holding `) (`, a zombie, a
// process that vanished between the listing and the read, the Uid line, and
// AT_CLKTCK from auxv.
func TestParseLinuxProc(t *testing.T) {
	auxv := make([]byte, 48)
	binary.LittleEndian.PutUint64(auxv[0:], 6) // AT_PAGESZ
	binary.LittleEndian.PutUint64(auxv[8:], 4096)
	binary.LittleEndian.PutUint64(auxv[16:], atClkTck)
	binary.LittleEndian.PutUint64(auxv[24:], 250)
	// AT_NULL terminates.
	proc := fstest.MapFS{
		"loadavg":    {Data: []byte("1.50 2.25 3.00 2/345 6789\n")},
		"uptime":     {Data: []byte("10000.00 50000.00\n")},
		"self/auxv":  {Data: auxv},
		"self/stat":  {Data: []byte("not a pid directory")},
		"100/stat":   {Data: []byte(linuxStat("100", "weird) (name", "R", 1, 100, 250*600, 250*300, 250*1000))},
		"100/status": {Data: []byte("Name:\tweird\nUid:\t1000\t1001\t1000\t1000\n")},
		"200/stat":   {Data: []byte(linuxStat("200", "gone", "Z", 1, 200, 0, 0, 0))},
		"200/status": {Data: []byte("Uid:\t0\t0\t0\t0\n")},
		"300/stat":   {Data: []byte(linuxStat("300", "vanished", "S", 1, 300, 0, 0, 0))},
		// 300/status is missing: the process exited between the two reads.
		"sys/stat": {Data: []byte("not numeric")},
	}
	cpu := fstest.MapFS{"online": {Data: []byte("0-3,8-11\n")}}

	snap, err := readProcFS(proc, cpu, 8, binary.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.HasLoad || snap.Load1 != 1.5 || snap.Load5 != 2.25 || snap.Load15 != 3 {
		t.Fatalf("loads = %+v", snap)
	}
	if snap.Cores != 8 {
		t.Fatalf("cores = %d, want 8", snap.Cores)
	}
	if !snap.HasProcs || len(snap.Procs) != 1 {
		t.Fatalf("procs = %+v, want only pid 100", snap.Procs)
	}
	p := snap.Procs[0]
	want := Proc{PID: 100, PPID: 1, PGID: 100, UID: 1001, Age: 9000 * time.Second, CPU: 900 * time.Second, Name: "weird) (name"}
	if p != want {
		t.Fatalf("proc = %+v, want %+v", p, want)
	}

	// Without an AT_CLKTCK entry the rate is 100.
	delete(proc, "self/auxv")
	proc["100/stat"] = &fstest.MapFile{Data: []byte(linuxStat("100", "n", "R", 1, 100, 100, 0, 100*500))}
	snap, err = readProcFS(proc, cpu, 8, binary.LittleEndian)
	if err != nil || len(snap.Procs) != 1 || snap.Procs[0].Age != 9500*time.Second || snap.Procs[0].CPU != time.Second {
		t.Fatalf("default clock rate: %+v, %v", snap.Procs, err)
	}

	// A process table that cannot be read still reports the load.
	delete(proc, "uptime")
	snap, err = readProcFS(proc, cpu, 8, binary.LittleEndian)
	if err == nil || !snap.HasLoad || snap.HasProcs {
		t.Fatalf("unreadable table: %+v, %v", snap, err)
	}
}
