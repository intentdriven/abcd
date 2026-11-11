package gitutil

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// CommonDir is git's common dir for the working tree whose top is dir, made
// absolute against dir: the directory every worktree of one repository shares,
// so two working trees belong to one repository exactly when their common dirs
// resolve to one real path. The peers reader and the worktree store's proof of
// belonging both ask it, of the candidate itself, never of this checkout.
//
// It asks without --path-format=absolute, which arrived in git 2.31: an older
// rev-parse echoes an option it does not know to stdout and exits 0, so the
// answer would be the flag's text and a path on two lines, and every candidate
// would compare unequal. The plain form answers relative to the directory it is
// run in (".git" at a main working tree's top) or absolute (a linked
// worktree's), and anything but one line is refused rather than compared.
func CommonDir(dir string) (string, error) {
	out, err := Run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return commonDirAnswer(dir, out)
}

// CommonDirContext is CommonDir bound to ctx: a git still running when ctx ends
// is killed, and its answer is ctx's error.
func CommonDirContext(ctx context.Context, dir string) (string, error) {
	out, err := RunCappedContext(ctx, dir, 4096, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return commonDirAnswer(dir, strings.TrimSpace(out))
}

// commonDirAnswer is CommonDir's reading of git's one-line answer out.
func commonDirAnswer(dir, out string) (string, error) {
	if out == "" || strings.ContainsAny(out, "\r\n") {
		first, _, _ := strings.Cut(out, "\n")
		first, _, _ = strings.Cut(first, "\r")
		return "", fmt.Errorf("git named no single common dir (%q)", first)
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(dir, out)
	}
	return out, nil
}
