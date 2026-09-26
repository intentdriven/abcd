package source

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// corpusGit runs git in the corpus repository.
//
// The environment is scrubbed of every repo-selection variable — the load-bearing
// case is a call made from inside another repository's pre-commit hook, where git
// has exported GIT_DIR and GIT_INDEX_FILE for THAT repository and an unscrubbed
// child would commit corpus paths into it. Global config stays in effect, because
// the corpus commits under the person's own identity.
//
// Hooks are switched off (core.hooksPath=/dev/null): the corpus is a local store
// whose whole content is what a name guard exists to keep out of project
// repositories, and a global hook dispatcher applying one here would refuse the
// corpus its own documents.
func corpusGit(dir string, args ...string) (string, error) {
	full := append([]string{"-c", "core.hooksPath=/dev/null", "-C", dir}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = gitutil.ScrubbedEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 2048 {
			msg = msg[:2048]
		}
		return "", fmt.Errorf("git %s: %w (%s)", args[0], err, msg)
	}
	return stdout.String(), nil
}

// commit stages paths (additions, modifications and removals alike) and commits
// them. Nothing to commit is not an error.
func commit(dir, msg string, paths ...string) error {
	if _, err := corpusGit(dir, append([]string{"add", "-A", "--"}, paths...)...); err != nil {
		return fmt.Errorf("%w: staging the change failed: %v", ErrCorpusInvalid, err)
	}
	_, err := corpusGit(dir, "diff", "--cached", "--quiet")
	if err == nil {
		return nil
	}
	// Exit 1 means "there are staged changes"; anything else is a failure.
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 {
		return fmt.Errorf("%w: reading the staged change failed: %v", ErrCorpusInvalid, err)
	}
	if _, err := corpusGit(dir, "commit", "-q", "-m", msg); err != nil {
		return fmt.Errorf("%w: the corpus commit failed (is a git identity configured?): %v", ErrCorpusInvalid, err)
	}
	return nil
}
