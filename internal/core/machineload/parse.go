package machineload

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"time"
)

// The parsers below are pure functions over bytes or an fs.FS, and carry no build
// tag, so both platforms' parsers are exercised on every platform the tests run
// on. Only read_darwin.go and read_linux.go touch the real machine.

// ParseLoadavgSysctl reads macOS's `vm.loadavg` sysctl value: the kernel's
// struct loadavg, three uint32 fixed-point averages, four bytes of padding, then
// an int64 scale, little-endian. syscall.Sysctl returns it as a string with one
// trailing NUL stripped (23 bytes where the struct is 24), so the stripped bytes
// are restored as zeros before the scale is read.
func ParseLoadavgSysctl(raw []byte) (l1, l5, l15 float64, err error) {
	const size = 24
	if len(raw) < 20 || len(raw) > size {
		return 0, 0, 0, fmt.Errorf("vm.loadavg is %d bytes, not the %d-byte loadavg struct", len(raw), size)
	}
	b := make([]byte, size)
	copy(b, raw)
	scale := int64(binary.LittleEndian.Uint64(b[16:24]))
	if scale <= 0 {
		return 0, 0, 0, fmt.Errorf("vm.loadavg carries a scale of %d", scale)
	}
	f := func(off int) float64 { return float64(binary.LittleEndian.Uint32(b[off:off+4])) / float64(scale) }
	return f(0), f(4), f(8), nil
}

// parseDarwinPS reads `/bin/ps -axo pid=,ppid=,pgid=,uid=,etime=,time=,comm=`.
// The first six fields split on whitespace; comm is the rest of the line, so a
// name with spaces survives, and it is reduced to its base name because macOS
// prints the executable's full path, which can carry a home directory.
func parseDarwinPS(out []byte) ([]Proc, error) {
	var procs []Proc
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var fields [6]string
		rest := line
		for i := range fields {
			rest = strings.TrimLeft(rest, " \t")
			cut := strings.IndexAny(rest, " \t")
			if cut < 0 {
				return nil, fmt.Errorf("ps line %d has fewer than seven fields", n)
			}
			fields[i], rest = rest[:cut], rest[cut:]
		}
		comm := strings.TrimSpace(rest)
		if comm == "" {
			return nil, fmt.Errorf("ps line %d has no command name", n)
		}
		var ints [4]int
		for i := range ints {
			v, err := strconv.Atoi(fields[i])
			if err != nil || v < 0 {
				return nil, fmt.Errorf("ps line %d: field %d is not a number", n, i+1)
			}
			ints[i] = v
		}
		age, err := parseClock(fields[4])
		if err != nil {
			return nil, fmt.Errorf("ps line %d: elapsed time: %w", n, err)
		}
		cpu, err := parseClock(fields[5])
		if err != nil {
			return nil, fmt.Errorf("ps line %d: CPU time: %w", n, err)
		}
		procs = append(procs, Proc{
			PID: ints[0], PPID: ints[1], PGID: ints[2], UID: uint32(ints[3]),
			Age: age, CPU: cpu, Name: path.Base(comm),
		})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return procs, nil
}

// parseClock reads the two clock forms ps prints: etime, `[[dd-]hh:]mm:ss`, and
// time, `[dd-][hh:]mm:ss.cc`, where the leading unit may run past its usual
// range (`114:03.49` is 114 minutes). The rightmost field is seconds, with an
// optional fraction; each field to its left is sixty times the one after it, up
// to hours; a `dd-` prefix adds days.
func parseClock(s string) (time.Duration, error) {
	var days int
	if d, rest, ok := strings.Cut(s, "-"); ok {
		v, err := strconv.Atoi(d)
		if err != nil || v < 0 {
			return 0, fmt.Errorf("%q has no day count before its '-'", s)
		}
		days, s = v, rest
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("%q is not [hh:]mm:ss", s)
	}
	secText := parts[len(parts)-1]
	whole, frac, _ := strings.Cut(secText, ".")
	sec, err := strconv.Atoi(whole)
	if err != nil || sec < 0 {
		return 0, fmt.Errorf("%q has no seconds", s)
	}
	total := time.Duration(sec) * time.Second
	if frac != "" {
		f, err := strconv.ParseFloat("0."+frac, 64)
		if err != nil {
			return 0, fmt.Errorf("%q has a malformed fraction", s)
		}
		total += time.Duration(f * float64(time.Second))
	}
	unit := time.Minute
	for i := len(parts) - 2; i >= 0; i-- {
		v, err := strconv.Atoi(parts[i])
		if err != nil || v < 0 {
			return 0, fmt.Errorf("%q has a malformed field", s)
		}
		total += time.Duration(v) * unit
		unit *= 60
	}
	return total + time.Duration(days)*24*time.Hour, nil
}

// parseOnlineRange counts the cores in a Linux cpu range list such as
// `0-3,8-11`, the format of /sys/devices/system/cpu/online, which is what glibc's
// sysconf(_SC_NPROCESSORS_ONLN) reads.
func parseOnlineRange(raw []byte) (int, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return 0, errors.New("the online cpu list is empty")
	}
	n := 0
	for _, part := range strings.Split(s, ",") {
		lo, hi, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(lo)
		if err != nil || a < 0 {
			return 0, fmt.Errorf("the online cpu list %q is malformed", s)
		}
		b := a
		if isRange {
			if b, err = strconv.Atoi(hi); err != nil || b < a {
				return 0, fmt.Errorf("the online cpu list %q is malformed", s)
			}
		}
		n += b - a + 1
	}
	return n, nil
}

// atClkTck is the auxiliary-vector key for the clock-tick rate.
const atClkTck = 17

// defaultClkTck is the clock-tick rate when the auxiliary vector does not carry
// one: USER_HZ on every Linux architecture that ships today.
const defaultClkTck = 100

// parseAuxvClkTck reads AT_CLKTCK out of /proc/self/auxv: pairs of native words,
// key then value, ending at AT_NULL. wordSize and order are the reading
// process's own, since auxv is the process's own vector.
func parseAuxvClkTck(raw []byte, wordSize int, order binary.ByteOrder) int {
	word := func(b []byte) uint64 {
		if wordSize == 4 {
			return uint64(order.Uint32(b))
		}
		return order.Uint64(b)
	}
	for i := 0; i+2*wordSize <= len(raw); i += 2 * wordSize {
		key := word(raw[i:])
		if key == 0 {
			break
		}
		if key == atClkTck {
			if v := word(raw[i+wordSize:]); v > 0 && v < 1<<20 {
				return int(v)
			}
		}
	}
	return defaultClkTck
}

// readProcFS reads a Linux machine from its /proc tree (proc) and its
// /sys/devices/system/cpu tree (cpu). It is the Linux reader's whole logic, kept
// here without a build tag so an fstest.MapFS exercises it on every platform.
//
// A process that exits between the listing and the read is skipped, not an
// error, and so is a zombie. The effective uid comes from the `Uid:` line of
// status, because the owner of /proc/<pid> is root for a non-dumpable process.
func readProcFS(proc, cpu fs.FS, wordSize int, order binary.ByteOrder) (Snapshot, error) {
	var snap Snapshot
	raw, err := fs.ReadFile(proc, "loadavg")
	if err != nil {
		return snap, fmt.Errorf("could not read the load average (/proc/loadavg: %w)", err)
	}
	f := strings.Fields(string(raw))
	if len(f) < 3 {
		return snap, errors.New("could not read the load average (/proc/loadavg is malformed)")
	}
	var loads [3]float64
	for i := range loads {
		if loads[i], err = strconv.ParseFloat(f[i], 64); err != nil || loads[i] < 0 {
			return snap, errors.New("could not read the load average (/proc/loadavg is malformed)")
		}
	}
	snap.Load1, snap.Load5, snap.Load15, snap.HasLoad = loads[0], loads[1], loads[2], true

	if online, err := fs.ReadFile(cpu, "online"); err == nil {
		if n, err := parseOnlineRange(online); err == nil && n > 0 {
			snap.Cores = n
		}
	}

	procs, err := readProcTable(proc, wordSize, order)
	if err != nil {
		return snap, fmt.Errorf("could not read the process table (%w)", err)
	}
	snap.Procs, snap.HasProcs = procs, true
	return snap, nil
}

// readProcTable reads every numeric /proc/<pid>.
func readProcTable(proc fs.FS, wordSize int, order binary.ByteOrder) ([]Proc, error) {
	upRaw, err := fs.ReadFile(proc, "uptime")
	if err != nil {
		return nil, fmt.Errorf("/proc/uptime: %w", err)
	}
	upFields := strings.Fields(string(upRaw))
	if len(upFields) < 1 {
		return nil, errors.New("/proc/uptime is malformed")
	}
	uptime, err := strconv.ParseFloat(upFields[0], 64)
	if err != nil || uptime < 0 {
		return nil, errors.New("/proc/uptime is malformed")
	}
	hz := defaultClkTck
	if auxv, err := fs.ReadFile(proc, "self/auxv"); err == nil {
		hz = parseAuxvClkTck(auxv, wordSize, order)
	}
	entries, err := fs.ReadDir(proc, ".")
	if err != nil {
		return nil, fmt.Errorf("/proc: %w", err)
	}
	var procs []Proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		p, ok := readProcPID(proc, e.Name(), pid, uptime, hz)
		if ok {
			procs = append(procs, p)
		}
	}
	return procs, nil
}

// readProcPID reads one process, or reports it gone (exited, a zombie, or a
// file that no longer parses because the pid was reused mid-read).
func readProcPID(proc fs.FS, dir string, pid int, uptime float64, hz int) (Proc, bool) {
	stat, err := fs.ReadFile(proc, dir+"/stat")
	if err != nil {
		return Proc{}, false
	}
	s := string(stat)
	open, closing := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if open < 0 || closing < open {
		return Proc{}, false
	}
	name := s[open+1 : closing]
	f := strings.Fields(s[closing+1:])
	// Fields after the name, zero-based: 0 state, 1 ppid, 2 pgrp, 11 utime,
	// 12 stime, 19 starttime (proc(5) fields 3, 4, 5, 14, 15 and 22).
	if len(f) < 20 || f[0] == "Z" {
		return Proc{}, false
	}
	num := func(i int) (int64, bool) {
		v, err := strconv.ParseInt(f[i], 10, 64)
		return v, err == nil && v >= 0
	}
	ppid, ok1 := num(1)
	pgrp, ok2 := num(2)
	utime, ok3 := num(11)
	stime, ok4 := num(12)
	start, ok5 := num(19)
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return Proc{}, false
	}
	status, err := fs.ReadFile(proc, dir+"/status")
	if err != nil {
		return Proc{}, false
	}
	uid, ok := effectiveUID(status)
	if !ok {
		return Proc{}, false
	}
	ticks := float64(hz)
	age := uptime - float64(start)/ticks
	if age < 0 {
		age = 0
	}
	return Proc{
		PID: pid, PPID: int(ppid), PGID: int(pgrp), UID: uid,
		Age:  time.Duration(age * float64(time.Second)),
		CPU:  time.Duration(float64(utime+stime) / ticks * float64(time.Second)),
		Name: name,
	}, true
}

// effectiveUID reads the second field of status's `Uid:` line (real, effective,
// saved, filesystem).
func effectiveUID(status []byte) (uint32, bool) {
	for _, line := range strings.Split(string(status), "\n") {
		rest, ok := strings.CutPrefix(line, "Uid:")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 2 {
			return 0, false
		}
		v, err := strconv.ParseUint(f[1], 10, 32)
		if err != nil {
			return 0, false
		}
		return uint32(v), true
	}
	return 0, false
}
