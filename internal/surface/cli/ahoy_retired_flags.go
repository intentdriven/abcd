package cli

import "github.com/spf13/cobra"

// retiredInstallFlags are `ahoy install` flags removed with no alias, each with
// the refusal that names what replaced it. A flag is retired rather than kept
// as an alias when its meaning changed: --outbound-hook and --outbound-forge
// each turned on half of what is now one switch, by the product thinker's
// ruling of 2026-10-15 (iss-2610152024327989), so an alias would either carry
// the old half-meaning or silently widen what a script asked for. Before 1.0 a
// flag may be removed, but a script passing one is told the replacement rather
// than handed a bare unknown-flag error or a silent success.
var retiredInstallFlags = map[string]string{
	"--outbound-hook":  "--outbound-hook was retired: one switch, `--outbound`, now turns on both the commit-message check and the forge-text check, and also tells this clone's check where the verified abcd is. Run abcd ahoy install --outbound. Nothing was written.",
	"--outbound-forge": "--outbound-forge was retired: one switch, `--outbound`, now turns on both the forge-text check and the commit-message check. Run abcd ahoy install --outbound. Nothing was written.",
}

// applyRetiredInstallFlags answers a retired `ahoy install` flag with its
// refusal at exit 2, the code every usage refusal takes, instead of pflag's
// bare "unknown flag". Flags are parsed before RunE, so the refusal comes before
// anything is read or written. It wraps the flag-error function the generic
// tagging set, and passes every other error through unchanged.
func applyRetiredInstallFlags(root *cobra.Command) {
	install, _, err := root.Find([]string{"ahoy", "install"})
	if err != nil || install == nil || install.Name() != "install" {
		return
	}
	inner := install.FlagErrorFunc()
	install.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if m := unknownFlagRe.FindStringSubmatch(err.Error()); m != nil {
			if why, retired := retiredInstallFlags[m[1]]; retired {
				return &exitError{Code: 2, Msg: "abcd ahoy install: " + why}
			}
		}
		return inner(cmd, err)
	})
}
