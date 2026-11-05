package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The background half of the scheduled-run alert (iss-2611050035192370). The
// prompt's block reads from disk alone and never touches the network, so a
// session that never ran `abcd status` would otherwise never hear of a red: when
// the kept answer has aged the hook starts `hook scheduled-refresh` detached, and
// a later prompt reads what it kept. The hook's half never touches the network.

// scheduledSpawnMargin is how long before the hook's deadline it last starts a
// refresh, so starting one never costs the prompt its deadline.
const scheduledSpawnMargin = 100 * time.Millisecond

// spawnScheduledRefresh starts `hook scheduled-refresh` detached from the hook, in
// dir. A variable so a test can see the request without starting one.
var spawnScheduledRefresh = func(dir string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	// A test binary is no abcd: started with these arguments it would run its
	// package's tests, and a hook run from a test in this repository's own tree
	// reads a config that names a workflow.
	if strings.HasSuffix(filepath.Base(self), ".test") {
		return nil
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devnull.Close()
	cmd := exec.Command(self, "hook", "scheduled-refresh")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	// Its own session, so it outlives the hook that started it and the host's
	// ending of the hook does not reach it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// startDueRefresh starts the refresh the block asked for, when the hook's deadline
// leaves room: stamped first, so a refresh that cannot be stamped is not started
// on every prompt.
func startDueRefresh(ctx context.Context, cmd *cobra.Command, root string, o claim.Options, deadline time.Time) {
	if time.Now().After(deadline.Add(-scheduledSpawnMargin)) || claim.NoteScheduledRefresh(ctx, root, o) != nil {
		return
	}
	if err := spawnScheduledRefresh(root); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd status: the background refresh of the scheduled run could not be started (%s)\n",
			termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
}

func newScheduledRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scheduled-refresh",
		Short: "Ask the forge for the repository's latest scheduled runs and keep the answer, now",
		Long: `Asks the forge (gh api), within five seconds, for the latest completed scheduled
runs of the workflow this repository names as ` + claim.ScheduledWorkflowKey + ` in its
.abcd/config.json, on origin's default branch, and keeps what they say in the
machine's store for the repository, which every checkout of it reads. The prompt
hook starts this in the background when the answer it reads has aged, so a session
that never runs ` + "`abcd status`" + ` still learns of a red; run it by hand to refresh ahead
of the next prompt. It fetches nothing. Why the runs could not be read is said on
stderr; the exit is 0 whatever happens, since nothing waits on it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return nil
			}
			root, err := gitutil.CheckoutRoot(cwd, statusStore)
			if err != nil {
				return nil
			}
			if why := claim.RefreshScheduledRun(root, statusScheduledRun, claim.Options{}); why != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "abcd status: "+why)
			}
			return nil
		},
	}
}
