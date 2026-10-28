package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// noteOccupancy is the session-start hook's record of the checkout this
// session starts in (iss-2610240859590333), so a record-writing verb can tell
// a main working tree another live session also works in. It never fails the
// hook: a session it cannot record is one the count cannot see, which is said
// on stderr whatever the reason.
func noteOccupancy(cmd *cobra.Command, in hookInput, cwd string) {
	if in.SessionID == "" || in.AgentID != "" {
		return
	}
	top, err := gitutil.RevParseAbsPath(cwd, "--show-toplevel")
	if err != nil {
		return // not a checkout: there is no working tree to share
	}
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return
	}
	unseen := func(why string) {
		diagnosticLine(cmd.ErrOrStderr(), "abcd: could not record which checkout this session works in (%s), so a record verb's count of sessions sharing it cannot see this one.",
			termsafe.Sanitize(fsutil.RedactHome(why)))
	}
	tab, err := m.ReadTable()
	if err != nil {
		unseen(err.Error())
		return
	}
	host, err := sessionname.FindHost(tab, m.From)
	if err != nil {
		unseen(err.Error())
		return
	}
	if err := occupancy.Note(m.Home, in.SessionID, top, host, tab); err != nil {
		unseen(err.Error())
	}
}

// forgetOccupancy is the session-end hook's removal of that record: one file
// remove, so it fits the host's short budget for the hook. A session that ends
// without it stops counting when its host process does.
func forgetOccupancy(in hookInput) {
	if in.SessionID == "" || in.AgentID != "" {
		return
	}
	if m, ok := sessionNameMachine(); ok && m.Home != "" {
		_ = occupancy.Forget(m.Home, in.SessionID)
	}
}

// refuseSharedMainCheckout refuses a record write into root when root is a
// main working tree that two or more live sessions started in
// (iss-2610240859590333): a record written there lands under every one of
// them, in the tree, the index and the indexes a peer regenerates. A linked
// worktree is never refused, nor is a checkout one session or none works in,
// so a person working alone is not held up. Exit 3 is abcd's "back off",
// as a held reservation is. A count that cannot be made (no process table on
// this platform) refuses nothing and says so.
func refuseSharedMainCheckout(cmd *cobra.Command, verb, root string) error {
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return nil
	}
	err := occupancy.Check(m.Home, root, m.ReadTable)
	var shared *occupancy.SharedError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &shared):
		return &exitError{Code: 3, Msg: verb + ": " + shared.Error()}
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s: could not count the sessions working in this checkout (%s), so a shared main checkout is not refused this time\n",
		verb, termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	return nil
}
