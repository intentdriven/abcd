package cli

import (
	"errors"
	"fmt"
	"io/fs"
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

// startDueRefresh starts the refresh the block n asked for, when the hook's deadline
// leaves room, and answers the block to show: stamped first, so a refresh that
// cannot be stamped is not started on every prompt. A stamp that cannot be kept,
// from a full disk or a changed mode, adds a line to the block itself, since the
// hook's stderr reaches only the host's debug log and a kept green past due would
// otherwise say nothing for up to 48 hours: no stamp newer than the answer is kept
// for the read to call a refresh that kept nothing (iss-2611051719396028). The
// stderr line stays, for the operator. A store that stays unwritable shows the line
// on every due prompt, since no stamp starts the cooldown.
func startDueRefresh(cmd *cobra.Command, n claim.Notes, root string, now, deadline time.Time) claim.Notes {
	if time.Now().After(deadline.Add(-scheduledSpawnMargin)) {
		return n
	}
	if err := n.StampRefresh(now); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd status: the background refresh of the scheduled run was not started: its stamp could not be kept (%s)\n", stderrReason(err))
		n.Text = withStatusLine(n.Text, termsafe.SanitizeLine("the background refresh of the scheduled run could not be started: its stamp could not be kept ("+unstampedWords(err)+"), so no kept answer is renewed; `abcd status` asks now"))
		return n
	}
	if err := spawnScheduledRefresh(root); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd status: the background refresh of the scheduled run could not be started (%s)\n", stderrReason(err))
	}
	return n
}

// stderrReason is err as the operator's stderr line says it: home-redacted and
// masked for one line, U+2028 and U+2029 included.
func stderrReason(err error) string {
	return termsafe.SanitizeLine(fsutil.RedactHome(err.Error()))
}

// unstampedWords is why a stamp could not be kept, in fixed words chosen by the
// error's class. The block an agent reads never carries the error's own text,
// which can be a refusal written for a person (a command to run, a link to
// replace) and may stand in a block with no line saying it is not an instruction;
// stderr carries the full reason, home-redacted.
func unstampedWords(err error) string {
	var link *fsutil.HomeScopeLinkError
	var exposed *fsutil.HomeScopeExposedError
	switch {
	case errors.As(err, &link):
		return "a link in the store's path is refused"
	case errors.As(err, &exposed):
		return "a folder of the store is open to another account"
	case errors.Is(err, syscall.ENOSPC), errors.Is(err, syscall.EDQUOT):
		return "the disk is full"
	case errors.Is(err, fsutil.ErrNotRealDir):
		return "something that is not a folder stands in the store's path"
	case errors.Is(err, fs.ErrPermission), errors.Is(err, syscall.EROFS):
		return "the store cannot be written by this account"
	}
	return "the store could not be written"
}

// withStatusLine is the STATUS block text with line added: as a block of its own
// when there is none, among the block's lines when it ends as a rendered block
// does, and after it otherwise.
func withStatusLine(text, line string) string {
	switch {
	case text == "":
		return claim.NotesHeading + "\n\n" + line + "\n"
	case strings.HasSuffix(text, "\n"+claim.GrantsNothing+"\n"):
		return strings.TrimSuffix(text, claim.GrantsNothing+"\n") + "- " + line + "\n" + claim.GrantsNothing + "\n"
	}
	return strings.TrimRight(text, "\n") + "\n\n" + line + "\n"
}

func newScheduledRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scheduled-refresh",
		Short: "Ask the forge for the repository's latest scheduled runs and keep the answer, now",
		Long: `Asks the forge (gh api), within five seconds, for the latest completed scheduled
runs of the workflow this repository names as ` + claim.ScheduledWorkflowKey + ` in the
.abcd/config.json at the tip of origin's default branch, on that branch, and keeps what they say in the
machine's store for the repository, which every checkout of it reads. The prompt
hook starts this in the background when the answer it reads has aged, so a session
that never runs ` + "`abcd status`" + ` still learns of a red; run it by hand to refresh ahead
of the next prompt. It fetches nothing. Why the runs could not be read is said on
stderr, and the failure's time and class are kept in the store, so a prompt can say
so; the exit is 0 whatever happens, since nothing waits on it.`,
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
