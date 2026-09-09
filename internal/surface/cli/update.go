package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newUpdater is the seam through which `abcd update` reaches the network — a
// package var like newReleaseFetcher, so a test can prove the dispatch
// refuses BEFORE anything network-capable exists and that no other verb ever
// constructs it (the adr-38 seam extended to the update verb, spc-32 AC8).
var newUpdater = update.NewGitHubUpdater

func newUpdateCommand(asJSON *bool) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "update [tag]",
		Short: "Complete a chosen update: fetch, verify, and swap the PATH-installed binary",
		Long: "Fetches the named release (or resolves the latest, naming it before acting),\n" +
			"verifies the platform binary against the same release's checksums.txt, and\n" +
			"swaps the PATH-installed copy atomically. The verb is the only ask: abcd\n" +
			"never checks for or applies updates on its own (adr-38). A plugin-root\n" +
			"binary, the dev shim, and package-manager installs are refused with the\n" +
			"command that owns them. The file being replaced must be provably abcd's:\n" +
			"the binary running the command, an install ~/.abcd/path-entry records, or\n" +
			"a digest a published release still names. Anything else is refused with a\n" +
			"remedy that reinstalls over it — never one that deletes it.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requested := ""
			if len(args) == 1 {
				requested = args[0]
			}
			// Dispatch first, network never: every refusal shape exits before
			// the updater exists.
			tgt := ahoy.ResolveUpdateTarget()
			if r := update.Plan(tgt); r != nil {
				rep := refusalReport(tgt, r)
				renderUpdateReport(cmd.OutOrStdout(), *asJSON, rep)
				return fmt.Errorf("update refused (%s)", r.Shape)
			}

			u := newUpdater()
			tag, err := u.ResolveTag(requested)
			if err != nil {
				return err
			}
			// Name the tag before acting. On a TTY the bare form asks; an
			// explicit tag was already the user's own naming.
			if requested == "" && !yes && isTTY(os.Stdin) && isTTY(os.Stderr) {
				fmt.Fprintf(cmd.ErrOrStderr(), "resolved latest release: %s — proceed? [y/N] ", termsafe.Sanitize(tag))
				line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
					fmt.Fprintln(cmd.ErrOrStderr(), "declined — nothing was fetched.")
					return nil
				}
			}
			var progress io.Writer
			if isTTY(os.Stderr) {
				progress = cmd.ErrOrStderr()
			}
			rep, err := u.Apply(tgt.Path, tag, progress)
			if err != nil {
				return err
			}
			// The swap changed the bytes an owned-copy PATH entry hashes to, so
			// the provenance record (spc-35) is re-stamped with the digest the
			// release checksums just proved — otherwise the entry this verb
			// refreshed would classify foreign forever after. A no-op when no
			// record names this path (legacy installs, plugin-root refusals).
			if rep.Action == update.ActionSwapped || rep.Action == update.ActionCurrent {
				ahoy.RefreshPathEntryDigest(tgt.Path, rep.Digest)
			}
			renderUpdateReport(cmd.OutOrStdout(), *asJSON, rep)
			if rep.Refusal != nil {
				return fmt.Errorf("update refused (%s)", rep.Refusal.Shape)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the TTY confirmation of a freshly resolved tag")
	return cmd
}

// isTTY reports whether f is a character device — the progress/confirmation
// gate. Piped and hooked invocations are silent except for the receipt.
// Delegates to the canonical check in internal/term.
func isTTY(f *os.File) bool {
	return term.IsTerminal(f)
}

// refusalReport shapes the receipt for a dispatch refusal. The target path is
// redacted at construction: the refusal is a success-shaped envelope (text and
// --json) the CLI error-surface scrub never touches, so an unredacted
// target_path would sit beside the already-redacted refusal detail
// (iss-2608220142158516).
func refusalReport(tgt ahoy.UpdateTarget, r *update.Refusal) update.Report {
	return update.Report{Action: update.ActionRefused, TargetPath: fsutil.RedactHome(tgt.Path), Refusal: r}
}

// renderUpdateReport prints the receipt in both modes. Tags and paths pass
// through termsafe on the text render: the tag is read off HTTP responses.
func renderUpdateReport(w io.Writer, asJSON bool, rep update.Report) {
	_ = render(w, asJSON, rep, func(w io.Writer) {
		switch rep.Action {
		case update.ActionSwapped:
			// A file abcd owned but could not date — its release objects are
			// gone from the forge — has no old version to print, and an empty
			// left-hand side would read as a broken receipt. It is named for
			// what it is, and the digest and the ownership proof that allowed
			// the swap follow on their own line (iss-2609012000222546).
			old := termsafe.Sanitize(rep.OldVersion)
			if rep.OldVersion == "" {
				old = "an unpublished build"
			}
			fmt.Fprintf(w, "updated %s: %s -> %s\n", termsafe.Sanitize(rep.TargetPath), old, termsafe.Sanitize(rep.NewVersion))
			fmt.Fprintf(w, "  origin:   %s\n", rep.Origin)
			if rep.OldDigest != "" {
				fmt.Fprintf(w, "  replaced: sha256 %s — in no published release; %s\n", termsafe.Sanitize(rep.OldDigest), rep.Ownership.Prose())
			}
			fmt.Fprintf(w, "  verified: sha256 %s (release checksums.txt)\n", termsafe.Sanitize(rep.Digest))
		case update.ActionCurrent:
			fmt.Fprintf(w, "already current: %s is %s (verified against the release checksums)\n", termsafe.Sanitize(rep.TargetPath), termsafe.Sanitize(rep.Tag))
		case update.ActionRefused:
			if rep.Refusal != nil {
				fmt.Fprintf(w, "refused (%s): %s\n", rep.Refusal.Shape, termsafe.Sanitize(rep.Refusal.Detail))
				fmt.Fprintf(w, "  remedy: %s\n", rep.Refusal.Remedy)
			}
		}
		if len(rep.EnvIgnored) > 0 {
			fmt.Fprintf(w, "  ignored from the environment: %s\n", strings.Join(rep.EnvIgnored, ", "))
		}
	})
}
