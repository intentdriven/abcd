package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// offProfilePointer says where an unattended start on a separate login is.
const offProfilePointer = "abcd drain --drive is the unattended start: it runs agents of its own on a separate login, with named tools, checked at its start."

// offProfileHelp is what every warned verb's help says of the check, so the
// reference states one limit in one wording. It says what the line can rely on
// and no more: the definition of a profile is runner.SessionOnProfile's.
const offProfileHelp = "abcd reads one setting in the session's environment, and every doubt reads as off a profile, a host\n" +
	"reporting no such setting included. This reads one setting and proves no wall, and the setting's value\n" +
	"changes nothing the session holds. No abcd command starts a session on a profile yet. A profile is a\n" +
	"separate login and settings on one operating-system account, and a stage's shell reads what the account can.\n" +
	offProfilePointer

// warnOffProfile warns a session whose setting reads as off a profile
// (dcn-2611041550129278): one line on stderr whatever the output mode, as the
// other start notices are, so a reader of --json sees it and the JSON on stdout
// stays whole. It never refuses and changes no exit code. It reports whether it
// printed; a session reading as on a profile hears nothing, which says nothing of
// what it holds.
func warnOffProfile(cmd *cobra.Command, prefix string) bool {
	// A home that cannot be read comes back empty, and reads as off a profile.
	home, _ := os.UserHomeDir()
	// A drive's own stage is no session the person started, and its shell's
	// setting says nothing of one.
	if home != "" && os.Getenv(abcdhome.StageHomeEnv) == home {
		return false
	}
	on, why := runner.SessionOnProfile(home, os.Getenv(runner.ConfigDirEnv))
	if on {
		return false
	}
	fmt.Fprintln(cmd.ErrOrStderr(), offProfileLine(prefix, why))
	return true
}

// offProfileLine is the warning, worded as the setting's reading because a
// reading is all abcd has. It opens in capitals to stand out among a verb's other
// lines, and asks to be passed on (dcn-2611050931229356). It carries no local path,
// no variable and no location for the setting, and it prints no step, since no abcd
// command starts a session on a profile yet; it does say where an unattended start
// on a separate login is. It holds for a first join in the
// managed mode too, which is why it says what the verb can start, not that it
// started.
func offProfileLine(prefix, why string) string {
	return "WARNING: " + prefix + " \u2014 THIS SESSION'S CONFIGURATION DIRECTORY SETTING READS AS OFF A PROFILE (it " + why + "), " +
		"and this verb starts or coordinates work that can run unattended. Pass this line on to the person. " +
		"abcd reads that one setting and proves no wall: it cannot check what the session holds, so the person's connected " +
		"services and personal memory may be in its reach, and the setting's value changes nothing the session holds. " +
		"No abcd command starts a session on a profile yet. " + offProfilePointer + " This is a warning: nothing was refused."
}

// warnOffProfileWhileAway is warnOffProfile for a build or a drain, which start
// unattended work while the away mode holds: in the managed mode each is one move
// at the person's direction. A record or store that cannot be read leaves the mode
// in doubt, and every doubt falls towards the line, so it warns and says why. A
// repository with no root commit is no doubt, since no record can exist for it.
func warnOffProfileWhileAway(cmd *cobra.Command, prefix, root string) {
	_, away, err := autonomy.AwayAt(root, autonomy.Options{})
	if errors.Is(err, autonomy.ErrNoRootCommit) || !away && err == nil {
		return
	}
	if warnOffProfile(cmd, prefix) && err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: warned because the away record could not be read: %s\n", prefix, termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
}
