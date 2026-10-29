package cli

// guard_allow.go is `guard allow`: the person's one-shot allow of a branch
// delete or a worktree removal the guard could not prove safe
// (iss-2610241239492159). The record and its guards are core's
// (guard.GrantAllow, guard.TakeAllow); this file is the verb, the hook's use
// of the record, and the delete prover both front doors attach.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/sweep"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// guardDeleteProver is the prover `guard check` and `guard hook` attach, so a
// delete the registry names is refused unless its target is proven safe. A
// test replaces it.
var guardDeleteProver = func() guard.DeleteProver {
	return sweep.NewDeleteProver(sweep.DeleteProofOptions{Forge: worktreeForge})
}

// guardAllowNow is the allow's clock. A test moves it.
var guardAllowNow = time.Now

// allowRoot is the root commit of the repository dir is in, which keys its
// allows.
func allowRoot(dir string) (string, error) {
	top, err := gitutil.Toplevel(dir)
	if err != nil || top == "" {
		return "", errors.New("this is not a git checkout, so there is no repository to allow a command in")
	}
	sha, err := gitutil.RootCommit(top)
	if err != nil {
		return "", fmt.Errorf("the repository's root commit could not be read (%s)", scrubPaths(err))
	}
	return sha, nil
}

// takeGuardAllow uses the person's one-shot allow of candidate, run from dir,
// when one is recorded.
func takeGuardAllow(dir, candidate string) (guard.Allowed, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return guard.Allowed{}, false
	}
	sha, err := allowRoot(dir)
	if err != nil {
		return guard.Allowed{}, false
	}
	return guard.TakeAllow(home, sha, candidate, guardAllowNow())
}

func newGuardAllowCommand(asJSON *bool) *cobra.Command {
	var command string
	var once bool
	cmd := &cobra.Command{
		Use: "allow --once --command <line>",
		Long: "Allows one exact command line through the shell guard's hook, once, within\n" +
			"ten minutes: the person's way past a branch delete or a worktree removal the\n" +
			"guard could not prove safe (merged, with no open pull request, in no\n" +
			"worktree, held by no reservation, and for a worktree clean and in use by\n" +
			"nothing). It lets past nothing else: a line any other hazard blocks stays\n" +
			"blocked.\n\n" +
			"The allow is the person's own act. The guard refuses this verb when an\n" +
			"agent runs it, and refuses a shell write into the store, so run it yourself\n" +
			"in a terminal (or with your host's `!` prefix). The line must match the\n" +
			"agent's command byte for byte. The allow is recorded under\n" +
			guard.AllowStoreDisplay() + "<root-commit>/, keyed on the repository's root\n" +
			"commit, read only while this account owns it and no one else can write it,\n" +
			"and removed as the hook uses it. The store's log.jsonl records each line,\n" +
			"when it was allowed and when it was used, and never who.\n\n" +
			"--once is required: an allow is only ever for one use.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !once {
				return &exitError{Code: 2, Msg: "guard allow: pass --once; an allow is only ever for one use of one exact line"}
			}
			if command == "" {
				return &exitError{Code: 2, Msg: "guard allow: pass --command with the exact line to allow"}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return &exitError{Code: 2, Msg: "guard allow: the working directory could not be read"}
			}
			sha, err := allowRoot(cwd)
			if err != nil {
				return &exitError{Code: 2, Msg: "guard allow: " + err.Error()}
			}
			home, err := os.UserHomeDir()
			if err != nil || home == "" {
				return &exitError{Code: 2, Msg: "guard allow: the home folder could not be found"}
			}
			a, err := guard.GrantAllow(home, sha, command, guardAllowNow())
			if err != nil {
				return &exitError{Code: 2, Msg: "guard allow: the allow was not recorded (" + scrubPaths(err) + ")"}
			}
			return render(cmd.OutOrStdout(), *asJSON, a, func(w io.Writer) {
				fmt.Fprintf(w, "allowed once, until %s: %s\n", a.ExpiresAt.Format("15:04 MST"), termsafe.Sanitize(a.Command))
			})
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "allow the line for one use (required)")
	cmd.Flags().StringVar(&command, "command", "", "the exact command line to allow")
	return cmd
}
