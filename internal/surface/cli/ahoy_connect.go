package cli

// ahoy_connect.go is the front door of the OpenAI-compatible API adapter
// (itd-2609081951381895): `abcd ahoy --providers`, the read that explains the
// adapter, lists what is configured and says where a key can live, and
// `abcd ahoy connect <provider>`, the write that verifies a provider with one
// call and then stores its block and its key.
//
// The key arrives on stdin and nowhere else: never as a flag (a process
// listing and a shell history keep argv), and never at a prompt that echoes
// (the install prompter echoes every answer into its transcript, and a host's
// question tool would put it in an agent's context). Piped, it is read whole;
// at a terminal it is read on hidden input (term.ReadHidden), echo off, after
// one line on stderr (spc-2610031241482088, step 2). It is not printed, not
// logged and not part of any error.
//
// With no --model, at a terminal (stdin, stdout and stderr all terminals, the
// letter adr-49 sets for drawing), the setup lists the service's models with
// the key, the person picks one in the plain-Terminal list, and a real
// completion to it verifies the connection before anything is written. Off a
// terminal a run with no --model is refused, naming both ways on.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// dispatchNote is what a configured provider does to a delegated step: a verb
// whose agent's role points at it sends the step there itself, and a provider
// that holds a key takes only the self-contained agents (ruling DR5).
var dispatchNote = "a delegating verb whose agent's oracle.roles entry points at a provider sends the step there itself " +
	"and ingests the answer, and every other step runs on the host; a provider whose block names a key takes only " +
	"self-contained agents (" + strings.Join(oracle.SelfContained(), ", ") + "), under ruling DR5 of 2026-09-29, " +
	"and oracle.bundled_context_providers in ~/.abcd/config.json is the person's override for file-reading agents " +
	"whose bundle abcd builds (none yet)"

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
		Dispatch:    dispatchNote,
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

// printConfigDiagnostics says the provider configuration read's non-fatal
// reports (oracle.APIConfig.Diagnostics: a route skipped, and why) on w, one
// line each. It is the one printer every front door that reads the
// configuration and is not the board uses, so a skipped route is said the
// same way wherever it is met.
func printConfigDiagnostics(w io.Writer, diagnostics []string) {
	for _, d := range diagnostics {
		fmt.Fprintf(w, "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(d)))
	}
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
			// Picking draws a list, so it needs what drawing needs: every
			// stream a terminal (adr-49).
			atTerminal := connectIsTerminal(cmd.InOrStdin()) && connectIsTerminal(cmd.OutOrStdout()) &&
				connectIsTerminal(cmd.ErrOrStderr())
			if len(models) == 0 && !atTerminal {
				return &exitError{Code: 2, Msg: "abcd ahoy connect: no model is named; name one with --model, " +
					"or run the command in a terminal to pick one from the models the service lists"}
			}
			req := oracle.ConnectRequest{Roots: roots, Provider: args[0], BaseURL: baseURL, Models: models, Home: home, KeyName: keyName, Pointer: ptr}
			if len(models) == 0 {
				req.Pick = connectPick(cmd, roots, baseURL)
			}
			// What needs no key is refused before the key is asked for, so
			// nobody pastes a key for a setup that cannot finish.
			if err := oracle.CheckConnect(req); err != nil {
				return &exitError{Code: 2, Msg: "abcd ahoy connect: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
			}
			if home == oracle.KeyHomeABCD || home == oracle.KeyHomeKeychain {
				key, err := readConnectKey(cmd, args[0])
				if errors.Is(err, term.ErrInterrupted) {
					return &exitError{Code: ask.ExitInterrupted, Msg: "abcd ahoy connect: interrupted while the key was pasted; nothing was written"}
				}
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
				code := 2
				if errors.Is(err, ask.ErrInterrupted) {
					code = ask.ExitInterrupted
				}
				return &exitError{Code: code, Msg: "abcd ahoy connect: " + termsafe.Sanitize(fsutil.RedactHome(msg))}
			}
			// A route the configuration read skipped (ruling CD2) is said on
			// stderr, in the text and the JSON form alike, and the setup stands.
			printConfigDiagnostics(cmd.ErrOrStderr(), res.Diagnostics)
			return render(cmd.OutOrStdout(), *asJSON, withMember{v: res, key: "dispatch", val: dispatchNote}, func(w io.Writer) {
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
				line(dispatchNote + ".")
			})
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "", "the provider's OpenAI-compatible base URL: https, or http to a server on this machine")
	cmd.Flags().StringArrayVar(&models, "model", nil, "a model the provider may serve, repeated for each (the first allowlist; the verification call asks for the first); "+
		"omitted at a terminal, the service's models are listed with the key and you pick one")
	cmd.Flags().StringVar(&home, "home", "", "where the key lives: external (--env, or --file and --field) | abcd (read from stdin, hidden at a terminal, into the owner-only ~/.abcd/credentials.json) | keychain (read from stdin, hidden at a terminal, into the platform keychain) | none (a server that takes no key)")
	cmd.Flags().StringVar(&keyName, "key", "", "the credential's name (default: the provider's name)")
	pointerFlags(cmd, &ptr)
	return cmd
}

// readKey reads the key from stdin: refused from a terminal, where it would
// be echoed as it is typed; one trailing line ending is dropped.
func readKey(in io.Reader) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(f) {
		return "", errors.New("the key is read from stdin, and stdin is a terminal, where it would be echoed as it is typed; " +
			"pipe it in from a file or a variable instead (" + setupExample + ")")
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

// readConnectKey reads the key for the setup of provider: piped, as readKey
// reads it; at a terminal, on hidden input, after one line on stderr saying
// so. An empty answer is refused as an empty pipe is. Ctrl-C during the paste
// is term.ErrInterrupted, the terminal restored.
func readConnectKey(cmd *cobra.Command, provider string) (string, error) {
	in := cmd.InOrStdin()
	if !connectIsTerminal(in) {
		return readKey(in)
	}
	errOut := cmd.ErrOrStderr()
	if _, err := fmt.Fprintf(errOut, "Paste the key for %s and press Enter. It is not shown.\n", termsafe.Sanitize(provider)); err != nil {
		return "", err
	}
	key, err := connectReadHidden(in, errOut)
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", errors.New("the chosen home stores a key, and none was pasted; run the command again and paste it, or pipe it in (" + setupExample + ")")
	}
	return key, nil
}

// errPickLater is the picker's decide later: no model was picked, so the
// setup ends with nothing written.
var errPickLater = errors.New("you chose to decide later; run the command again to pick")

// The terminal seams of `ahoy connect`: whether a stream is a terminal, the
// hidden read of the key, and the picker. Tests replace them, since a test's
// streams are buffers.
var (
	connectIsTerminal = func(s any) bool {
		f, ok := s.(*os.File)
		return ok && term.IsTerminal(f)
	}
	connectReadHidden = func(in io.Reader, out io.Writer) (string, error) {
		f, ok := in.(*os.File)
		if !ok {
			return "", errors.New("hidden input needs a terminal, and stdin is not one")
		}
		return term.ReadHidden(f, out)
	}
	connectPick = terminalPick
)

// terminalPick is the picker the setup offers at a terminal: the
// plain-Terminal long list over the listed ids, typing to narrow
// (spc-2610030911534855), drawn on stderr. Ctrl-C is ask.ErrInterrupted;
// decide later is errPickLater.
func terminalPick(cmd *cobra.Command, roots layered.Roots, baseURL string) func(context.Context, []string) (string, error) {
	return func(_ context.Context, ids []string) (string, error) {
		in, ok := cmd.InOrStdin().(*os.File)
		if !ok {
			return "", errors.New("the list needs a terminal, and stdin is not one")
		}
		host := baseURL
		if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
			host = u.Host
		}
		choices := make([]question.Option, len(ids))
		for i, id := range ids {
			choices[i] = question.Option{Value: id, Label: id}
		}
		a := question.Ask{Questions: []question.Question{{
			ID:   "model",
			Chip: "Setup",
			Material: []question.Block{{Kind: question.KindParagraph,
				Text: fmt.Sprintf("%s lists %d models. Type part of a name to narrow the list.", host, len(ids))}},
			Ask:  "Which model should abcd verify and set up?",
			List: &question.List{Choices: choices},
			// No listed id can take this value: validModel admits no '#'.
			Later: question.Option{Value: "#later", Label: "Decide later",
				Meaning: "Nothing is set up or written; run the command again to pick."},
		}}}
		got, err := ask.Terminal{In: in, Out: cmd.ErrOrStderr(), Getenv: os.Getenv,
			Mode: term.ResolveColorMode(os.Getenv, false), ASCII: !term.UTF8Locale(os.Getenv), Roots: roots}.Put(a)
		switch {
		case err != nil:
			return "", err
		case len(got) != 1 || got[0].Later:
			return "", errPickLater
		}
		return got[0].Value, nil
	}
}
