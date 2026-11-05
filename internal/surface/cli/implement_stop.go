package cli

// implement_stop.go is the front door of the stop (itd-2611041549471712):
// `abcd implement stop`, which a person types from their own account, and
// `abcd ahoy shared-folder`, the option of setup that lets it cross accounts. Each
// write is the person's own act: the guard refuses it to an agent under any
// program name, and the verb refuses a caller with no terminal, which a hook, a
// driver and a host's shell tool each are.

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// personAtTerminal reports whether a person could be typing this command: a
// terminal, in no host session, read from every process above this one. A test
// stands it in.
var personAtTerminal = func() bool { return term.IsTerminal(os.Stdin) && !underHostSession() }

// underHostSession reports whether this process runs under a host session, by its
// environment or by the marker its host's session-start hook wrote.
func underHostSession() bool {
	if os.Getenv(hostSessionIDEnv) != "" {
		return true
	}
	m := sessionname.Live()
	home, err := runstop.AccountHome()
	if err != nil || home == "" {
		return false
	}
	tab, err := m.ReadTable()
	if err != nil {
		return false
	}
	return hostSessionAbove(home, tab, m.From)
}

// hostSessionAbove reports whether any process above from is a host session: every
// ancestor is asked.
func hostSessionAbove(home string, tab sessionname.Table, from int) bool {
	for _, h := range sessionname.Lineage(tab, from) {
		if occupancy.SessionOn(home, h) != "" {
			return true
		}
	}
	return false
}

// stopOutput is the --json shape of `implement stop`.
type stopOutput struct {
	Stopped bool       `json:"stopped"`
	Lifted  bool       `json:"lifted,omitempty"`
	SetAt   *time.Time `json:"set_at,omitempty"`
	Path    string     `json:"path,omitempty"`
	// Reach says which runs the stop reaches.
	Reach string `json:"reach,omitempty"`
	// Unreached names each stop that stands where this lift cannot reach it.
	Unreached []string `json:"unreached,omitempty"`
}

// stopHome is the account's home as the account database has it, where every run
// reads a stop, refused when it cannot be found.
func stopHome(sub string) (string, error) {
	home, err := runstop.AccountHome()
	if err != nil || home == "" {
		return "", &exitError{Code: 2, Msg: sub + ": this account's home cannot be found: nothing was written"}
	}
	return home, nil
}

// newImplementStopCommand builds `abcd implement stop [--lift]`.
func newImplementStopCommand(asJSON *bool) *cobra.Command {
	var lift bool
	cmd := &cobra.Command{
		Use: "stop [--lift]",
		Long: "Stop every unattended run on this machine, from your own account, or with --lift\n" +
			"lift the stop. Every run reads the stop before its next move: it opens no lane,\n" +
			"hands out no stage and starts no agent. An agent in a call finishes that call and\n" +
			"its receipt is collected. Then each lane's uncommitted work is saved aside, the\n" +
			"run's reservation is released so a peer may take the work, and the day's page is\n" +
			"written to the shared folder.\n\n" +
			"It takes hold across accounts where the shared folder was chosen in each account\n" +
			"(`abcd ahoy shared-folder`). Without one the stop lives in " + abcdhome.Display(runstop.HomeStopFile) + "\n" +
			"and reaches only this account's runs.\n\n" +
			"This command lifts a stop, and each lift is recorded in " + abcdhome.Display(runstop.LiftLogFile) + ".\n" +
			"A run reads the folder it started under whatever the setting says, and the platform's\n" +
			"shared folder for the account it expects; a setting that has gone or changed during a run\n" +
			"holds that run until it is put back. A stop this lift cannot reach is named.\n" +
			"A run that let its claims go stays halted after the lift until a session runs\n" +
			"`abcd implement resume --take <run-id>`, which reserves the work again first.\n\n" +
			"It is the person's own act: the guard refuses it to an agent, and it refuses a\n" +
			"caller with no terminal or under a host session. Exit 2 on a refusal, with nothing changed: no terminal, a\n" +
			"file at the stop's path no run would honour, or a folder the stop cannot rest on.\n" +
			"No network request.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd implement stop"
			if !personAtTerminal() {
				return &exitError{Code: 2, Msg: prefix + ": this is the person's own act, typed at a terminal, and this caller has none: nothing was changed"}
			}
			home, err := stopHome(prefix)
			if err != nil {
				return err
			}
			now := time.Now()
			if lift {
				lifted, err := runstop.Lift(home, now)
				if err != nil {
					return &exitError{Code: 2, Msg: prefix + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
				}
				out := stopOutput{Lifted: lifted, Unreached: runstop.Unreached(home)}
				return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
					for _, line := range out.Unreached {
						fmt.Fprintln(w, termsafe.Sanitize(fsutil.RedactHome(line)))
					}
					switch {
					case lifted:
						fmt.Fprintln(w, "the stop is lifted; a run that let its claims go stays halted until a session runs `abcd implement resume --take <run-id>`")
					case len(out.Unreached) == 0:
						fmt.Fprintln(w, "no stop was set")
					}
				})
			}
			s, err := runstop.Set(home, now)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + ": no stop was set: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
			}
			reach := "every run on this machine in an account that chose this shared folder"
			if _, chosen, _ := runstop.ReadSetting(home); !chosen {
				reach = "this account's runs only: no shared folder was chosen (`abcd ahoy shared-folder`)"
			}
			out := stopOutput{Stopped: true, SetAt: &s.SetAt, Path: fsutil.RedactHome(s.Path), Reach: reach}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "stopped at %s: %s\n", s.SetAt.Format(time.RFC3339), reach)
				fmt.Fprintf(w, "each run takes it at its next move; lift it with `abcd implement stop --lift`\n")
				fmt.Fprintf(w, "stop file: %s\n", termsafe.Sanitize(out.Path))
			})
		},
	}
	cmd.Flags().BoolVar(&lift, "lift", false, "lift the stop; a halted run then waits for `abcd implement resume --take`")
	return cmd
}

// sharedFolderWalkthrough is what setup says before the person chooses.
func sharedFolderWalkthrough() string {
	return "The shared folder is one option of setup, for a person who runs abcd unattended on a\n" +
		"second account of this computer. It holds what the two accounts pass between them: the\n" +
		"stop you set from your own account, which every run reads before its next move, and each\n" +
		"day's digest page. Every account on this machine can read what is in it, so it is meant\n" +
		"for a machine no other person shares. Declining leaves every other part of abcd working,\n" +
		"and a stop then reaches only runs in your own account. To decline, do nothing.\n"
}

// sharedFolderOutput is the --json shape of `ahoy shared-folder`.
type sharedFolderOutput struct {
	Chosen    bool   `json:"chosen"`
	Folder    string `json:"folder,omitempty"`
	Person    string `json:"person,omitempty"`
	Suggested string `json:"suggested"`
	Explains  string `json:"explains"`
}

// newAhoySharedFolderCommand builds `abcd ahoy shared-folder [<folder>] [--person <account>]`.
func newAhoySharedFolderCommand(asJSON *bool) *cobra.Command {
	var person string
	cmd := &cobra.Command{
		Use: "shared-folder [<folder>] [--person <account>]",
		Long: "Say what the shared folder is for and which one this account chose; with <folder>,\n" +
			"choose it. Run it in each account that takes part, naming the same folder, and\n" +
			"--person naming the account you stop from (this account when omitted). A run honours\n" +
			"a stop only from that account.\n\n" +
			sharedFolderWalkthrough() + "\n" +
			"The folder must be a real directory, and one other accounts can write must carry\n" +
			"the sticky bit, and it must be owned by root or by you, each parent above it held\n" +
			"to the same rule, so no account can remove what you wrote there or move it away.\n" +
			"/Users/Shared on macOS and /var/tmp on Linux both qualify. In\n" +
			"your own account this also makes the abcd-stop directory the stop lives in. The choice is written to " + abcdhome.Display(runstop.SettingFile) + ".\n\n" +
			"Choosing is the person's own act: the guard refuses it to an agent, and it refuses\n" +
			"a caller with no terminal or under a host session. Exit 2 on a refusal, with nothing written. No network\n" +
			"request.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd ahoy shared-folder"
			home, err := stopHome(prefix)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				if !personAtTerminal() {
					return &exitError{Code: 2, Msg: prefix + ": choosing the folder is the person's own act, typed at a terminal, and this caller has none: nothing was written"}
				}
				if _, err := runstop.WriteSetting(home, args[0], person); err != nil {
					return &exitError{Code: 2, Msg: prefix + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + ": nothing was written"}
				}
			}
			s, chosen, err := runstop.ReadSetting(home)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
			}
			out := sharedFolderOutput{Chosen: chosen, Suggested: runstop.Suggested(), Explains: sharedFolderWalkthrough()}
			if chosen {
				out.Folder, out.Person = s.Folder, s.Person
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprint(w, out.Explains)
				if !chosen {
					fmt.Fprintf(w, "none chosen; on this machine the folder would be %s: `abcd ahoy shared-folder %s`\n", out.Suggested, out.Suggested)
					return
				}
				fmt.Fprintf(w, "chosen: %s; a stop is honoured from the account %s\n", termsafe.Sanitize(s.Folder), termsafe.Sanitize(s.Person))
			})
		},
	}
	cmd.Flags().StringVar(&person, "person", "", "the account you stop from; this account when omitted")
	return cmd
}
