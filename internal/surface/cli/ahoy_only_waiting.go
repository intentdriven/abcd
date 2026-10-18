package cli

// ahoy_only_waiting.go holds `ahoy install --only-waiting`, the setup the yes
// to the offer `abcd update news` makes runs (spc-2610170635097509, step 7):
// the flag is registered here and read by the install's RunE.

import "github.com/spf13/cobra"

// onlyWaitingFlag is the install's flag for the setup the yes runs.
const onlyWaitingFlag = "only-waiting"

// addOnlyWaitingFlag registers --only-waiting on the install command.
func addOnlyWaitingFlag(installCmd *cobra.Command) {
	installCmd.Flags().Bool(onlyWaitingFlag, false, "ask only the setup that waits after an abcd update: "+
		"the questions this repository never answered and the settings whose default changed, never one answered or deferred with decide later; "+
		"every kind of change is approved up front, as the yes to the offer abcd update news makes approves them; "+
		"with the approve-everything flag no question is answered, so each still waits")
}

// onlyWaiting reports whether the install was asked for --only-waiting.
func onlyWaiting(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool(onlyWaitingFlag)
	return v
}

// onlyWaitingCommand is the setup a yes runs.
const onlyWaitingCommand = "abcd ahoy install --only-waiting"
