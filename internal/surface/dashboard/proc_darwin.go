package dashboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// psPath is the system's ps, named absolutely so PATH cannot choose it.
const psPath = "/bin/ps"

// psEnv is the whole environment ps runs with. Its start time is local-time
// text in the reader's language, so without a fixed zone and locale one
// process reads differently under start and under a stop run in another time
// zone, and stop would take the server for another process and leave it
// listening.
var psEnv = []string{"TZ=UTC", "LC_ALL=C"}

// psTimeout bounds each ps run, and psCommand builds it; both are variables
// so a test can make ps outlast its deadline the way it does on a stalled
// machine.
var (
	psTimeout = 5 * time.Second
	psCommand = exec.CommandContext
)

// readProcess reads the identity of process pid from the kernel's process
// table through ps: its start time, to the second and in UTC, and its
// executable. A pid
// with no process, or only a zombie's entry, is errProcessGone; a ps that
// outlasts its deadline is an error.
func readProcess(pid int) (processID, error) {
	if pid <= 0 {
		return processID{}, errProcessGone
	}
	field := func(name string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
		defer cancel()
		var out bytes.Buffer
		cmd := psCommand(ctx, psPath, "-o", name+"=", "-p", strconv.Itoa(pid))
		cmd.Env = psEnv
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			// A ps killed by its own deadline exits like one that found no
			// process; it read nothing, so it is an error, never a gone pid.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return "", fmt.Errorf("%s -o %s -p %d: %w", psPath, name, pid, ctxErr)
			}
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
