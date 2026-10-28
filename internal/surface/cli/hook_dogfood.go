package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/dogfood"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The dogfood rung of the hook shims (iss-2610070547057816). Each shim under
// hooks/ resolves the installed binary first, and only when the person keeps a
// ~/.abcd.noindex/dogfood-roots list does it ask that binary, through
// `hook dogfood-binary`, whether this checkout runs a build of the reviewed
// branch's tip, as the remote the person declared reports it, instead. The
// decision and the build are internal/core/dogfood's; this file is their front
// door: it prints the build's path for the shim, starts a check and build in
// the background when one is due, and says every refusal and failure on
// stderr. The hook's half never touches the network; the background half does.

// spawnDogfoodBuild starts `hook dogfood-build` detached from the hook, in
// dir. A variable so a test can see the request without starting a build.
var spawnDogfoodBuild = func(dir string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devnull.Close()
	cmd := exec.Command(self, "hook", "dogfood-build")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	// Its own session, so the build outlives the hook that started it and the
	// host's ending of the hook does not reach it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// dogfoodHome is the person's home the rung reads, "" when there is none.
func dogfoodHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func newDogfoodBinaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dogfood-binary",
		Short: "Hook shims: print the build of the reviewed branch's verified tip to run here, if any",
		Long: `Prints, on stdout, the path of the abcd build a hook runs in place of the
installed release, and nothing when the installed release runs. A build is
named only in a checkout listed in ` + dogfood.RootsDisplay + ` with the remote
its reviewed branch lives on (or a worktree that checkout records), and only
once a build of the tip that remote last reported is finished. This verb reads
nothing over the network: a check of the remote, and the build of what it
reports, are started in the background when one is due. The checkout's own
refs and remotes never choose the build. Every refusal and failure is said on
stderr; the exit is 0 whatever happens, so a shim always has the installed
release to fall back to.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return nil
			}
			plan := dogfood.Resolve(dogfoodHome(), cwd)
			for _, n := range plan.Notes {
				fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(n))
			}
			if plan.StartBuild {
				if err := spawnDogfoodBuild(cwd); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "abcd dogfood: the background build could not be started ("+
						termsafe.Sanitize(fsutil.RedactHome(err.Error()))+"); the installed release runs")
				}
			}
			if plan.Binary != "" {
				fmt.Fprintln(cmd.OutOrStdout(), plan.Binary)
			}
			return nil
		},
	}
}

func newDogfoodBuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dogfood-build",
		Short: "Check the declared remote and build its reviewed tip for this listed checkout, now",
		Long: `Asks the remote declared for the checkout the working directory is in, in
` + dogfood.RootsDisplay + `, for the tip of its reviewed branch (git ls-remote,
never prompting for credentials, bounded in time), records the answer in
` + dogfood.StoreDisplay + `/<root-sha>/verified-tip-<hash>.json, one per declared remote and branch, and builds that commit
into ` + dogfood.StoreDisplay + `/<root-sha>/<commit>/abcd once the checkout holds
it. A build already there is kept; a build another process is running is left
to it. The hook shims start this in the background; run it by hand to check
and build ahead of the next hook, or to retry a failed build at once. Exits 1
when the check or the build fails, saying why.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return &exitError{Code: 1, Msg: "abcd dogfood: the working directory cannot be read"}
			}
			home := dogfoodHome()
			plan := dogfood.Resolve(home, cwd)
			if plan.Checkout == nil {
				for _, n := range plan.Notes {
					fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(n))
				}
				return &exitError{Code: 1, Msg: "abcd dogfood: nothing to build here — this checkout is not listed in " + dogfood.RootsDisplay + " with the remote its reviewed branch lives on"}
			}
			bin, err := dogfood.Build(home, *plan.Checkout)
			switch {
			case errors.Is(err, dogfood.ErrBuildRunning):
				fmt.Fprintln(cmd.ErrOrStderr(), "abcd dogfood: another build is running; it finishes this one")
				return nil
			case err != nil:
				return &exitError{Code: 1, Msg: "abcd dogfood: the check and build of " + termsafe.Sanitize(plan.Checkout.Source()) + " FAILED: " +
					termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; the installed release runs"}
			}
			fmt.Fprintln(cmd.OutOrStdout(), bin)
			return nil
		},
	}
}
