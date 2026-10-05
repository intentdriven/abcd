package dashboard

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// psPath is the system's ps, named absolutely so PATH cannot choose it.
const psPath = "/bin/ps"

// readProcess reads the identity of process pid from the kernel's process
// table through ps: its start time, to the second, and its executable. A pid
// with no process, or only a zombie's entry, is errProcessGone.
func readProcess(pid int) (processID, error) {
	if pid <= 0 {
		return processID{}, errProcessGone
	}
	field := func(name string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var out bytes.Buffer
		cmd := exec.CommandContext(ctx, psPath, "-o", name+"=", "-p", strconv.Itoa(pid))
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				return "", errProcessGone
			}
			return "", err
		}
		return strings.TrimSpace(out.String()), nil
	}
	state, err := field("stat")
	if err != nil {
		return processID{}, err
	}
	if state == "" || strings.HasPrefix(state, "Z") {
		return processID{}, errProcessGone
	}
	start, err := field("lstart")
	if err != nil {
		return processID{}, err
	}
	exe, err := field("comm")
	if err != nil {
		return processID{}, err
	}
	if start == "" {
		return processID{}, errProcessGone
	}
	return processID{Start: start, Exe: exe}, nil
}
