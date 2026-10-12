package dashboard

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// readStat and readExe are readProcess's two reads of /proc, variables so a
// test can make one fail the way a read racing the process's exit fails.
var (
	readStat = os.ReadFile
	readExe  = os.Readlink
)

// readProcess reads the identity of process pid from /proc: its start time in
// clock ticks since boot (field 22 of its stat line) and its executable. A pid
// with no process, or only a zombie's entry, is errProcessGone.
func readProcess(pid int) (processID, error) {
	if pid <= 0 {
		return processID{}, errProcessGone
	}
	dir := "/proc/" + strconv.Itoa(pid)
	raw, err := readStat(dir + "/stat")
	if err != nil {
		if processGone(err) {
			return processID{}, errProcessGone
		}
		return processID{}, err
	}
	line := string(raw)
	end := strings.LastIndexByte(line, ')')
	if end < 0 {
		return processID{}, errors.New("an unreadable process status line")
	}
	// After the command name come the fields from the third (state) on, so
	// field 22, the start time, is the twentieth of them.
	fields := strings.Fields(line[end+1:])
	if len(fields) < 20 {
		return processID{}, errors.New("an unreadable process status line")
	}
	if fields[0] == "Z" || fields[0] == "X" {
		return processID{}, errProcessGone
	}
	exe, err := readExe(dir + "/exe")
	if err != nil {
		if processGone(err) {
			return processID{}, errProcessGone
		}
		return processID{}, err
	}
	return processID{Start: fields[19], Exe: executableOf(exe)}, nil
}
