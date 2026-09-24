package banlist

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ErrNoEngine is MatchPrivate's refusal when no grep can be run: without the
// enforcing engine there is no answer that agrees with the guard, so the caller
// withholds what it would have asked about.
var ErrNoEngine = errors.New("banlist: the private layer's engine (grep) cannot be run")

// MatchPrivate reports which of names the private layer matches, one bool per
// name. It is the one read-only matcher this package holds: the load check
// (itd-2609231434459890) prints the caller's own program names, and a name the
// private layer bans must not reach a terminal or the run log.
//
// It asks the enforcing engine itself, exactly as grepAccepts and the pre-commit
// guard do, so the scrub and the guard cannot disagree about a match: `grep -inE`
// under LC_ALL=C, each pattern on STDIN through `-f -` (never in argv, where
// /proc/<pid>/cmdline would expose it), against the names in a 0600 temporary
// file that is removed afterwards. Each name must be one line: a caller passes
// names already made terminal-safe.
//
// An absent store matches nothing and is no error. A store that cannot be read
// for what it is (unreadable, a damaged declaration, a line that does not parse),
// a pattern the engine refuses, or no engine at all is an error, and the caller
// must treat every name as matched.
func MatchPrivate(repoRoot string, names []string) ([]bool, error) {
	grepBinOnce.Do(func() { grepBin, _ = exec.LookPath("grep") })
	return matchPrivateWith(repoRoot, names, grepBin)
}

// matchPrivateWith is MatchPrivate with the engine's path given; "" is no engine.
func matchPrivateWith(repoRoot string, names []string, grep string) ([]bool, error) {
	out := make([]bool, len(names))
	if len(names) == 0 {
		return out, nil
	}
	data, err := readPrivate(repoRoot)
	switch {
	case errors.Is(err, ErrNoStore):
		return out, nil
	case err != nil:
		return nil, err
	}
	entries, _, err := parse(data)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.unparsed {
			return nil, fmt.Errorf("%w: line %d of %s does not parse", ErrMalformedStore, e.line, PrivateRelPath)
		}
	}
	if len(entries) == 0 {
		return out, nil
	}
	if grep == "" {
		return nil, ErrNoEngine
	}
	for _, n := range names {
		if strings.ContainsAny(n, "\r\n") {
			return nil, errors.New("banlist: a name to match spans more than one line")
		}
	}

	f, err := os.CreateTemp("", "abcd-names-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.WriteString(strings.Join(names, "\n") + "\n"); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	for _, e := range entries {
		cmd := exec.Command(grep, "-inE", "-f", "-", "--", f.Name())
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		cmd.Stdin = strings.NewReader(e.pattern + "\n")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		// grep's diagnostics quote the expression, so they are discarded.
		cmd.Stderr = nil
		err := cmd.Run()
		var exit *exec.ExitError
		switch {
		case err == nil:
		case errors.As(err, &exit) && exit.ExitCode() == 1:
			continue
		case errors.As(err, &exit):
			return nil, fmt.Errorf("%w: the guard's grep refuses the pattern on line %d of %s", ErrMalformedStore, e.line, PrivateRelPath)
		default:
			return nil, fmt.Errorf("%w: %v", ErrNoEngine, err)
		}
		sc := bufio.NewScanner(&stdout)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			num, _, ok := strings.Cut(sc.Text(), ":")
			if i, err := strconv.Atoi(num); ok && err == nil && i >= 1 && i <= len(out) {
				out[i-1] = true
			}
		}
	}
	return out, nil
}
