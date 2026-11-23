package gitutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// maxAliasBytes caps what an alias read keeps of git's answer: only whether
// there is one matters.
const maxAliasBytes = 4096

// AliasDefinedContext reports whether the configuration git reads in the directory
// root defines alias.<name>: the repository's own files, the global file and
// the system one, read with `git config --get`, which reads and runs nothing
// else. Every variable that selects another repository or injects
// configuration is scrubbed (ScrubbedEnv). Where home is not "", git reads its
// global file under home, as a shell whose HOME is home does, and
// XDG_CONFIG_HOME is left out. A clean "not set" is false; any other failure,
// an invalid key among them, is an error, so a caller cannot read a failed
// read as no alias. git is bound to ctx: it is killed when ctx ends, and the
// error then wraps ctx's (context.DeadlineExceeded for a deadline).
func AliasDefinedContext(ctx context.Context, root, home, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "config", "--get", "alias."+name)
	cmd.WaitDelay = time.Second
	env := ScrubbedEnv()
	if home != "" {
		kept := env[:0]
		for _, kv := range env {
			if strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
				continue
			}
			kept = append(kept, kv)
		}
		env = append(kept, "HOME="+home)
	}
	cmd.Env = env
	out := &capWriter{remaining: maxAliasBytes}
	errOut := &capWriter{remaining: maxAliasBytes}
	cmd.Stdout, cmd.Stderr = out, errOut
	err := cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return false, fmt.Errorf("git config --get alias.%s: %w", name, ctxErr)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 && len(out.buf) == 0 && len(errOut.buf) == 0 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("git config --get alias.%s: %w (stderr: %q)", name, err, strings.TrimSpace(string(errOut.buf)))
	}
	return true, nil
}
