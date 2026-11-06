package cli

// implement_stop.go is the front door of the stop (itd-2611041549471712):
// `abcd implement stop`, which a person types from their own account, and
// `abcd ahoy shared-folder`, the option of setup that lets it cross accounts. Each
// write is the person's own act: the guard's rule refuses it to an agent, and the
// verb refuses a caller with no terminal or under a host session.

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

// stdinIsTerminal reports whether standard input is a terminal; a test stands it in.
var stdinIsTerminal = func() bool { return term.IsTerminal(os.Stdin) }

// hostReading is what a look above this process says of a host session.
type hostReading int

const (
	// noHostSession: the processes above were read, and none is a host session.
	noHostSession hostReading = iota
	// hostSessionFound: this process runs under a host session.
	hostSessionFound
	// hostSessionUnknown: what was needed to tell could not be read.
	hostSessionUnknown
)

// hostLook is one look above this process: what it says, why it could not tell, with
// what clears that, and the process the walk stopped at where the table did not hold one.
type hostLook struct {
	state hostReading
	why   string
	gap   int
}

// caller is who runs a person's act, read once for that act: a second read could say
// something else, and an act must stand on the reading it was judged by.
type caller struct {
	terminal bool
	look     hostLook
}

// person reports whether a person could be typing this command: a terminal, and no host
// session found above it. Each act decides what an unknown means: setting a stop goes
// ahead, since it has to work on a machine too loaded to read its processes and costs
// only pace; a lift and choosing the folder are refused (cannotTell).
func (c caller) person() bool { return c.terminal && c.look.state != hostSessionFound }

// readCaller reads the caller for one act; a test stands it in.
var readCaller = func() caller {
	if !stdinIsTerminal() {
		return caller{}
	}
	return caller{terminal: true, look: hostSessionReading()}
}

// cannotTell is the refusal of an act that opens what a stop or its folder closes, where
// the look above it could not be finished.
func cannotTell(prefix string, look hostLook, nothing string) error {
	return &exitError{Code: 2, Msg: prefix + ": whether this runs under a host session could not be told: " +
		termsafe.Sanitize(fsutil.RedactHome(look.why)) + ": " + nothing}
}

// hostSessionReading reads whether this process runs under a host session, by its
// environment or by the marker its host's session-start hook wrote. A test stands it in.
var hostSessionReading = func() hostLook {
	if os.Getenv(hostSessionIDEnv) != "" {
		return hostLook{state: hostSessionFound}
	}
	m := sessionname.Live()
	home, err := runstop.AccountHome()
	if err != nil || home == "" {
		return hostLook{state: hostSessionUnknown, why: "this account's home cannot be found"}
	}
	return hostSessionLook(home, m.ReadTable, m.From)
}

// hostSessionLook looks above from. A walk the table cut short has it read once more: a
// process that ended has its children moved to another parent at once, so a gap in the
// same place twice is a process hidden from this account, as a root-owned one can be, and
// the walk reads as far as it can see. A stop no one could ever lift is worse than what
// stands above a hidden process. A gap that moved is a chain that changed under the read,
// and tells nothing until it settles.
func hostSessionLook(home string, read func() (sessionname.Table, error), from int) hostLook {
	const unreadable = "the process table could not be read, as on a machine too loaded to run it: run it again"
	tab, err := read()
	if err != nil {
		return hostLook{state: hostSessionUnknown, why: unreadable}
	}
	first := hostSessionAbove(home, tab, from)
	if first.gap == 0 || first.state == hostSessionFound {
		return first
	}
	tab, err = read()
	if err != nil {
		return hostLook{state: hostSessionUnknown, why: unreadable}
	}
	again := hostSessionAbove(home, tab, from)
	if again.state != hostSessionFound && again.gap != first.gap {
		return hostLook{state: hostSessionUnknown, why: "the processes above this one changed while they were read: run it again"}
	}
	return again
}

// hostSessionAbove says whether any process above from in this table is a host session:
// every ancestor is asked. Only a walk that went all the way up, or started at the first
// process, as a container's shell does, can say none outright. A from the table does not
// hold, or a chain with no end, tells nothing; nor do markers that cannot be read, unless
// another ancestor is found to be one.
func hostSessionAbove(home string, tab sessionname.Table, from int) hostLook {
	walk := sessionname.Ancestors(tab, from)
	look := hostLook{gap: walk.Missing}
	switch {
	case walk.Finished:
	case walk.Missing == 0:
		look = hostLook{state: hostSessionUnknown, why: "the processes above this one do not end at the machine's first process: run it again"}
	case len(walk.Hosts) == 0:
		look = hostLook{state: hostSessionUnknown, why: "this command's parent is not in the process table: run it again"}
	}
	for _, h := range walk.Hosts {
		s, err := occupancy.ReadSessionOn(home, h)
		if err != nil {
			look.state, look.why = hostSessionUnknown, err.Error()+"; fix that folder, then run it again"
			continue
		}
		if s != "" {
			return hostLook{state: hostSessionFound}
		}
	}
	return look
}

// stopOutput is the --json shape of `implement stop`.
type stopOutput struct {
	Stopped bool       `json:"stopped"`
	Lifted  bool       `json:"lifted,omitempty"`
	SetAt   *time.Time `json:"set_at,omitempty"`
	// Stood says the stop was already standing, and was left as it is.
	Stood bool   `json:"stood,omitempty"`
	Path  string `json:"path,omitempty"`
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
		Long: "Stop the unattended runs of each account that chose this shared folder, or with --lift\n" +
			"lift the stop. Every run reads the stop before its next move: it opens no lane,\n" +
			"hands out no stage and starts no agent. An agent in a call finishes that call and\n" +
			"its receipt is collected. Then each lane's uncommitted work is saved aside, the\n" +
			"run's reservation is released so a peer may take the work, and the day's page is\n" +
			"written to this account's directory in the shared folder. A stop that stands is left\n" +
			"as it is, and says since when.\n\n" +
			"It takes hold across accounts where the shared folder was chosen in each account\n" +
			"(`abcd ahoy shared-folder`). Without one the stop lives in " + abcdhome.Display(runstop.HomeStopFile) + "\n" +
			"and reaches only this account's runs.\n\n" +
			"This command lifts a stop, and each lift is recorded in " + abcdhome.Display(runstop.LiftLogFile) + ".\n" +
			"A run reads the folder it started under whatever the setting says, and the platform's\n" +
			"shared folder for the account it expects; a setting that has gone or changed during a run\n" +
			"holds that run until it is put back. A stop this lift cannot reach is named.\n" +
			"A run that let its claims go stays halted after the lift until a session runs\n" +
			"`abcd implement resume --take <run-id>`, which reserves the work again first.\n\n" +
			"It is the person's own act: the guard's rule refuses it to an agent, and it refuses a\n" +
			"caller with no terminal or under a host session. Exit 2 on a refusal, with nothing changed: no terminal, a\n" +
			"file at the stop's path no run would honour, or a folder the stop cannot rest on.\n" +
			"Where it cannot tell whether a host session is above it, a stop is still set, and a\n" +
			"lift is refused and asks to be run again.\n" +
			"No network request.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd implement stop"
			c := readCaller()
			if !c.person() {
				return &exitError{Code: 2, Msg: prefix + ": this is the person's own act, typed at a terminal, and this caller has none: nothing was changed"}
			}
			home, err := stopHome(prefix)
			if err != nil {
				return err
			}
			now := time.Now()
			if lift {
				if c.look.state == hostSessionUnknown {
					return cannotTell(prefix, c.look, "nothing was changed")
				}
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
			reach := "the runs of each account that chose this shared folder; abcd cannot tell from here which accounts chose it"
			if _, chosen, _ := runstop.ReadSetting(home); !chosen {
				reach = "this account's runs only: no shared folder was chosen (`abcd ahoy shared-folder`)"
			}
			out := stopOutput{Stopped: true, SetAt: &s.SetAt, Stood: s.Stood, Path: fsutil.RedactHome(s.Path), Reach: reach}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if s.Stood {
					fmt.Fprintf(w, "a stop has stood since %s, and it stays as it is: %s\n", s.SetAt.Format(time.RFC3339), reach)
				} else {
					fmt.Fprintf(w, "stopped at %s: %s\n", s.SetAt.Format(time.RFC3339), reach)
				}
				fmt.Fprintf(w, "each run takes it at its next move; lift it with `abcd implement stop --lift`\n")
				fmt.Fprintf(w, "stop file: %s\n", termsafe.Sanitize(out.Path))
			})
		},
	}
	cmd.Flags().BoolVar(&lift, "lift", false, "lift the stop; a halted run then waits for a take-over (abcd implement resume --take)")
	return cmd
}

// sharedFolderWalkthrough is what setup says before the person chooses.
func sharedFolderWalkthrough() string {
	return "The shared folder is one option of setup, for a person who runs abcd unattended on a\n" +
		"second account of this computer. It holds what the two accounts pass between them: the\n" +
		"stop you set from your own account, which the runs of each account that chose it read before\n" +
		"their next move, and each\n" +
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
			"to the same rule, which keeps what you wrote there from other accounts' removal or moving.\n" +
			"/Users/Shared on macOS and /var/tmp on Linux both qualify. In\n" +
			"your own account this also makes the directory the stop lives in, named by your account's\n" +
			"number. On Linux an age cleaner may remove what has stood in a temporary tree for weeks, a\n" +
			"stop included; a folder an administrator makes elsewhere avoids that. The choice is written to " + abcdhome.Display(runstop.SettingFile) + ".\n\n" +
			"Choosing is the person's own act: the guard's rule refuses it to an agent, and it refuses\n" +
			"a caller with no terminal, under a host session, or where it cannot tell whether one is above it.\n" +
			"Exit 2 on a refusal, with nothing written. No network request.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd ahoy shared-folder"
			home, err := stopHome(prefix)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				c := readCaller()
				if !c.person() {
					return &exitError{Code: 2, Msg: prefix + ": choosing the folder is the person's own act, typed at a terminal, and this caller has none: nothing was written"}
				}
				if c.look.state == hostSessionUnknown {
					return cannotTell(prefix, c.look, "nothing was written")
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
