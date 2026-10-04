package cli

// ahoy_credential.go is the front door of the credential store's walkthrough
// (itd-2609221017023290): `abcd ahoy credential` lists the credentials abcd
// reads with whether each is set and in which home, `abcd ahoy credential
// <name>` explains one (what it unlocks, what works without it, the three
// homes with the keychain recommended in the prose), and `--home` runs the
// walkthrough: the reading adapter's own verification call, then the write.
//
// A value arrives on stdin and nowhere else, as for `ahoy connect`: never as a
// flag, never at a prompt, never from a terminal. The external home takes a
// pointer instead (--env, or --file and --field), and abcd stores only that.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// pointerFlags adds the external home's pointer flags to cmd.
func pointerFlags(cmd *cobra.Command, p *credential.Pointer) {
	cmd.Flags().StringVar(&p.Env, "env", "", "for --home external: the environment variable that holds the value")
	cmd.Flags().StringVar(&p.File, "file", "", "for --home external: a tool's JSON configuration file under the home directory, written from ~/")
	cmd.Flags().StringVar(&p.Field, "field", "", "for --home external: the dotted field of --file that holds the value (auth.token)")
}

// credentialService finds the walkthrough's service for name among the
// adapters that read a credential: the site setup's hosting providers, then
// the configured model providers. Reading the provider configuration says its
// diagnostics on stderr (a route skipped under ruling CD2).
func credentialService(stderr io.Writer, roots layered.Roots, name string) (credential.Service, bool, error) {
	if svc, ok := site.CredentialServiceFor(name); ok {
		return svc, true, nil
	}
	svc, ok, diagnostics, err := oracle.CredentialService(roots, name)
	printConfigDiagnostics(stderr, diagnostics)
	return svc, ok, err
}

// credentialView is one credential as a surface shows it: presence and home,
// never the value.
type credentialView struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Home  string `json:"home,omitempty"`
}

// credentialExplanation is `ahoy credential <name>` bare.
type credentialExplanation struct {
	credentialView
	Unlocks   string   `json:"unlocks"`
	WithoutIt string   `json:"without_it"`
	HomesText string   `json:"homes_prose"`
	Homes     []string `json:"homes"`
	Setup     []string `json:"setup"`
}

// newAhoyCredentialCommand builds `ahoy credential [<name>]`.
func newAhoyCredentialCommand(asJSON *bool) *cobra.Command {
	var home string
	var ptr credential.Pointer
	cmd := &cobra.Command{
		Use:  "credential [<name>]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			roots, notes := layered.RootsFor(cwd)
			for _, n := range notes {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
			}
			fail := func(err error, secret string) error {
				msg := err.Error()
				if secret != "" {
					msg = openaiapi.Scrub(msg, secret)
				}
				return &exitError{Code: 2, Msg: "abcd ahoy credential: " + termsafe.Sanitize(fsutil.RedactHome(msg))}
			}
			if len(args) == 0 {
				if home != "" {
					return fail(errors.New("name the credential to store; `abcd ahoy credential` lists them"), "")
				}
				return runCredentialList(cmd, roots, *asJSON)
			}
			name := args[0]
			if !credential.ValidName(name) {
				return fail(errors.New("the name is not a plain credential name"), "")
			}
			svc, ok, err := credentialService(cmd.ErrOrStderr(), roots, name)
			if err != nil {
				return fail(err, "")
			}
			if !ok {
				return fail(fmt.Errorf("no adapter abcd ships reads a credential named %s; `abcd ahoy credential` lists the ones it reads", name), "")
			}
			if home == "" {
				return renderCredentialExplanation(cmd, roots, svc, *asJSON)
			}
			choice := credential.Choice{Home: home, Pointer: ptr}
			if home == credential.HomeABCD || home == credential.HomeKeychain {
				if choice.Value, err = readKey(cmd.InOrStdin()); err != nil {
					return fail(err, "")
				}
			}
			res, err := credential.Walk(context.Background(), roots.Home, svc, choice)
			if err != nil {
				return fail(err, choice.Value)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				line := func(s string) { fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(s)) }
				fmt.Fprintf(w, "abcd ahoy credential — %s verified and kept in the %s home\n", res.Name, res.Home)
				if len(res.Wrote) == 0 {
					line("no change: the same credential was already kept there")
				}
				for _, p := range res.Wrote {
					line("wrote: " + p)
				}
			})
		},
	}
	cmd.Flags().StringVar(&home, "home", "", "where the credential lives: external (--env, or --file and --field) | abcd (read from stdin into the owner-only "+credential.StorePath+") | keychain (read from stdin into the platform keychain)")
	pointerFlags(cmd, &ptr)
	return cmd
}

// presence is a credential's state and home, never its value.
func presence(home, name string) credentialView {
	state, from := keyState(home, name)
	return credentialView{Name: name, State: state, Home: from}
}

func (v credentialView) text() string {
	if v.Home != "" {
		return v.Name + ": " + v.State + ", " + v.Home + " home"
	}
	return v.Name + ": " + v.State
}

// runCredentialList is `ahoy credential` bare: every credential an adapter
// reads, with its presence and home. It writes nothing and makes no call.
func runCredentialList(cmd *cobra.Command, roots layered.Roots, asJSON bool) error {
	names := map[string]bool{}
	for _, n := range site.CredentialNames() {
		names[n] = true
	}
	cfg, err := oracle.LoadAPI(roots)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd ahoy credential: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	// A route the read skipped (a repository's route to a provider that holds
	// a key, ruling CD2) is said on stderr, and the listing goes on.
	printConfigDiagnostics(cmd.ErrOrStderr(), cfg.Diagnostics)
	for _, p := range cfg.Providers() {
		if p.Key != "" {
			names[p.Key] = true
		}
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	out := struct {
		Credentials []credentialView `json:"credentials"`
	}{Credentials: []credentialView{}}
	for _, n := range sorted {
		out.Credentials = append(out.Credentials, presence(roots.Home, n))
	}
	return render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) {
		fmt.Fprintln(w, "abcd ahoy credential")
		for _, c := range out.Credentials {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(c.text()))
		}
		fmt.Fprintln(w, "  `abcd ahoy credential <name>` explains one and where it can live.")
	})
}

// renderCredentialExplanation is `ahoy credential <name>` without --home: the
// explanation first, the homes after, and the command for each. Writes
// nothing and makes no call.
func renderCredentialExplanation(cmd *cobra.Command, roots layered.Roots, svc credential.Service, asJSON bool) error {
	e := credentialExplanation{
		credentialView: presence(roots.Home, svc.Name),
		Unlocks:        svc.Unlocks,
		WithoutIt:      svc.WithoutIt,
		HomesText:      credential.HomesProse,
		Homes:          credential.Homes(),
		Setup: []string{
			"abcd ahoy credential " + svc.Name + " --home external --env <VARIABLE>",
			"abcd ahoy credential " + svc.Name + " --home external --file ~/<tool's configuration>.json --field <dotted.field>",
			"abcd ahoy credential " + svc.Name + " --home abcd < <a file holding only the value>",
			"abcd ahoy credential " + svc.Name + " --home keychain < <a file holding only the value>",
		},
	}
	return render(cmd.OutOrStdout(), asJSON, e, func(w io.Writer) {
		fmt.Fprintf(w, "abcd ahoy credential %s — %s\n", svc.Name, termsafe.Sanitize(e.credentialView.text()))
		for _, l := range svc.Explain() {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(l))
		}
		fmt.Fprintln(w, "  store it, verified first with the adapter's own call, the value piped in on stdin and never typed at a prompt:")
		for _, s := range e.Setup {
			fmt.Fprintf(w, "    %s\n", s)
		}
	})
}
