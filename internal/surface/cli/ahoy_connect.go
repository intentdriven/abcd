package cli

// ahoy_connect.go is the front door of the OpenAI-compatible API adapter
// (itd-2609081951381895): `abcd ahoy --providers`, the read that explains the
// adapter, lists what is configured and says where a key can live, and
// `abcd ahoy connect <provider>`, the write that verifies a provider with one
// call and then stores its block and its key.
//
// The key arrives on stdin and nowhere else: never as a flag (a process
// listing and a shell history keep argv), never at a prompt (the install
// prompter echoes every answer into its transcript, and a host's question tool
// would put it in an agent's context), and never from a terminal, where it
// would be echoed as it is typed. It is not printed, not logged and not part
// of any error.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// dispatchPending is the loud-staging line: the adapter is configured and
// verified, and no delegating verb sends a step through it until provider
// dispatch lands.
const dispatchPending = "no delegating verb sends a step to a provider until provider dispatch lands " +
	"(spc-2609251028149555); until then every delegated step runs on the host"

// providerView is one configured provider as the board shows it: the block,
// whether its key resolves and the home it resolves from (never the key).
type providerView struct {
	oracle.Provider
	KeyState string `json:"key_state"`
	KeyHome  string `json:"key_home,omitempty"`
}

// providersBoard is `ahoy --providers`.
type providersBoard struct {
	Explanation string                `json:"explanation"`
	Providers   []providerView        `json:"providers"`
	Denylist    []oracle.DenyEntry    `json:"denylist"`
	Routes      []oracle.PointedRoute `json:"routes"`
	KeyHomes    string                `json:"key_homes"`
	Homes       []string              `json:"homes"`
	Setup       string                `json:"setup"`
	Dispatch    string                `json:"dispatch"`
	Diagnostics []string              `json:"diagnostics"`
}

// setupExample is the walkthrough's command, the key piped in.
const setupExample = "abcd ahoy connect openrouter --base-url https://openrouter.ai/api/v1 " +
	"--model typesafe/jev-1.13 --home abcd < <a file holding only the key>"

// runAhoyProviders is `ahoy --providers`. It writes nothing and makes no call.
func runAhoyProviders(cmd *cobra.Command, cwd string, asJSON bool) error {
	roots, notes := layered.RootsFor(cwd)
	for _, n := range notes {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
	}
	cfg, err := oracle.LoadAPI(roots)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd ahoy --providers: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	b := providersBoard{
		Explanation: oracle.AdapterExplanation,
		Providers:   []providerView{},
		Denylist:    cfg.Denylist(),
		Routes:      cfg.Routes(),
		KeyHomes:    oracle.KeyHomesProse,
		Homes:       oracle.KeyHomes(),
		Setup:       setupExample,
		Dispatch:    dispatchPending,
		Diagnostics: append([]string{}, cfg.Diagnostics...),
	}
	if b.Routes == nil {
		b.Routes = []oracle.PointedRoute{}
	}
	if b.Denylist == nil {
		b.Denylist = []oracle.DenyEntry{}
	}
	for _, p := range cfg.Providers() {
		state, home := keyState(roots.Home, p.Key)
		b.Providers = append(b.Providers, providerView{Provider: p, KeyState: state, KeyHome: home})
	}
	return render(cmd.OutOrStdout(), asJSON, b, func(w io.Writer) {
		line := func(s string) { fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(s)) }
		fmt.Fprintln(w, "abcd ahoy --providers")
		line(b.Explanation)
		if len(b.Providers) == 0 {
			line("providers: none configured, so every delegated step runs on the host")
		}
		for _, p := range b.Providers {
			key := "no key"
			if p.Key != "" {
				key = "key " + p.Key + " (" + p.KeyState + ")"
				if p.KeyHome != "" {
					key = "key " + p.Key + " (" + p.KeyState + ", " + p.KeyHome + " home)"
				}
			}
			line(fmt.Sprintf("provider %s: %s, %s, models %s", p.Name, p.BaseURL, key, strings.Join(p.Models, ", ")))
		}
		deny := make([]string, len(b.Denylist))
		for i, e := range b.Denylist {
			deny[i] = e.Pattern + " (" + e.Origin + ")"
		}
		if len(deny) == 0 {
			line("denylist (oracle.denylist): none written; a provider serves only the models it lists")
		} else {
			line("denylist (oracle.denylist), which refuses a model even when a provider lists it: " + strings.Join(deny, ", "))
		}
		for _, r := range b.Routes {
			line(fmt.Sprintf("%s %s -> %s (%s)", r.Kind, r.Name, r.Target, r.Target.Origin))
		}
		for _, d := range b.Diagnostics {
			line(d)
		}
		line(b.KeyHomes)
		line("set one up, the key piped in on stdin and never typed at a prompt: " + b.Setup)
		line(b.Dispatch + ".")
	})
}

// keyState says whether a named credential resolves through the store, and
// from which home: set, not set, refused (the store is unsafe), or none for a
// keyless provider. Never the value.
func keyState(home, name string) (state, from string) {
	if name == "" {
		return "none", ""
	}
	_, err := credential.Store(home).Resolve(name)
	switch {
	case err == nil:
		from, _ = credential.Where(home, name)
		return "set", from
	case errors.Is(err, credential.ErrNotSet):
		return "not set", ""
	}
	return "refused: " + err.Error(), ""
}

// newAhoyConnectCommand builds `ahoy connect <provider>`.
func newAhoyConnectCommand(asJSON *bool) *cobra.Command {
	var baseURL, home, keyName string
	var models []string
	var ptr credential.Pointer
	cmd := &cobra.Command{
		Use:  "connect <provider>",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return &exitError{Code: 2, Msg: "abcd ahoy connect: name the provider to set up; `abcd ahoy --providers` explains the adapter and where its key can live"}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			roots, notes := layered.RootsFor(cwd)
			for _, n := range notes {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
			}
			req := oracle.ConnectRequest{Roots: roots, Provider: args[0], BaseURL: baseURL, Models: models, Home: home, KeyName: keyName, Pointer: ptr}
			if home == oracle.KeyHomeABCD || home == oracle.KeyHomeKeychain {
				key, err := readKey(cmd.InOrStdin())
				if err != nil {
					return &exitError{Code: 2, Msg: "abcd ahoy connect: " + err.Error()}
				}
				req.Key = key
			}
			res, err := oracle.Connect(context.Background(), req)
			if err != nil {
				// Every representation of the key, not only its literal form:
				// the adapter scrubs first, and this is the last time.
				msg := openaiapi.Scrub(err.Error(), req.Key)
				return &exitError{Code: 2, Msg: "abcd ahoy connect: " + termsafe.Sanitize(fsutil.RedactHome(msg))}
			}
			return render(cmd.OutOrStdout(), *asJSON, withMember{v: res, key: "dispatch", val: dispatchPending}, func(w io.Writer) {
				line := func(s string) { fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(s)) }
				fmt.Fprintf(w, "abcd ahoy connect — %s verified and configured\n", termsafe.Sanitize(res.Provider))
				line(fmt.Sprintf("verified: asked %s, %s reported %s", res.Verified.ModelAsked, res.Verified.Provider, res.Verified.ModelReported))
				for _, p := range res.Wrote {
					line("wrote: " + p)
				}
				// A route to a provider that holds a key is the machine's alone
				// (the ruling AA(b) of 2026-09-29, enforced by oracle.LoadAPI), so
				// the repository's file is offered only for a keyless one.
				where := layered.Config.MachineOrigin()
				if res.KeyHome == oracle.KeyHomeNone {
					where = layered.Config.RepoOrigin() + " or " + where
				}
				line(fmt.Sprintf("point a role or a judgement type at it with oracle.roles.<agent> or oracle.judgements.<type> = %q in %s",
					res.Provider+"/"+res.Models[0], where))
				line(dispatchPending + ".")
			})
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "", "the provider's OpenAI-compatible base URL: https, or http to a server on this machine")
	cmd.Flags().StringArrayVar(&models, "model", nil, "a model the provider may serve, repeated for each (the first allowlist; the verification call asks for the first)")
	cmd.Flags().StringVar(&home, "home", "", "where the key lives: external (--env, or --file and --field) | abcd (read from stdin into the owner-only ~/.abcd/credentials.json) | keychain (read from stdin into the platform keychain) | none (a server that takes no key)")
	cmd.Flags().StringVar(&keyName, "key", "", "the credential's name (default: the provider's name)")
	pointerFlags(cmd, &ptr)
	return cmd
}

// readKey reads the key from stdin: refused from a terminal, where it would
// be echoed as it is typed; one trailing line ending is dropped.
func readKey(in io.Reader) (string, error) {
	if f, ok := in.(*os.File); ok {
		if fi, err := f.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
			return "", errors.New("the key is read from stdin, and stdin is a terminal, where it would be echoed as it is typed; " +
				"pipe it in from a file or a variable instead (" + setupExample + ")")
		}
	}
	raw, err := io.ReadAll(io.LimitReader(in, credential.MaxValueBytes+3))
	if err != nil {
		return "", errors.New("the key could not be read from stdin")
	}
	key := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if key == "" {
		return "", errors.New("the chosen home stores a key, and none arrived on stdin; pipe it in (" + setupExample + ")")
	}
	return key, nil
}
