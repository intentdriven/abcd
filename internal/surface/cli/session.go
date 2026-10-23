package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/tailscale"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	coresession "github.com/intentdriven/abcd/internal/core/session"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/surface/session"
	"github.com/intentdriven/abcd/internal/surface/session/web"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// The environment variables the relay reads the team's GitHub App from. The
// secret is read by name and never printed; the client id is public.
const (
	sessionSecretEnv   = "ABCD_SESSION_GITHUB_CLIENT_SECRET"
	sessionClientIDEnv = "ABCD_SESSION_GITHUB_CLIENT_ID"
)

// sessionLegacyAppTypeEnv once chose between a GitHub App and an OAuth app.
// The relay now signs in through a GitHub App alone, so the variable may be
// unset or say github-app; anything else is a deployment still set up for an
// OAuth app, refused rather than started with a sign-in it cannot honour.
const sessionLegacyAppTypeEnv = "ABCD_SESSION_GITHUB_APP_TYPE"

// sessionOriginsEnv names the page origins serve admits when no --origin is
// given: a comma list. The blueprint writes it.
const sessionOriginsEnv = coresession.OriginsEnv

// sessionOrigins is the page origins serve admits: every --origin, or else
// each entry of ABCD_SESSION_ORIGINS, and the name of where they came from.
func sessionOrigins(flags []string, env string) ([]string, string) {
	if len(flags) > 0 {
		return flags, "--origin"
	}
	var out []string
	for _, e := range strings.Split(env, ",") {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out, sessionOriginsEnv
}

// sessionReposEnv names the repositories the relay serves, besides --repo and
// --owner: a comma list whose owner/name entries are repositories and whose
// bare entries are owners. It is not a secret; the blueprint writes it.
const sessionReposEnv = coresession.ReposEnv

// sessionOriginRepo is the owner/name the checkout at cwd's origin names on
// github.com. It is a package var so a test can answer for it.
var sessionOriginRepo = ahoy.GitHubRepo

// sessionReleaseOrigin is where the blueprint reads a release's checksums.
const sessionReleaseOrigin = "https://github.com/intentdriven/abcd"

// newSessionReleaseFetcher builds the fetcher the blueprint reads a release's
// checksums.txt through. It is a package var so a test serves the release from
// memory: no test reaches the network.
var newSessionReleaseFetcher = func() (coresession.AssetFetcher, error) {
	return update.NewReleaseAssets(sessionReleaseOrigin)
}

// newSessionCommand builds the `session` verb, the front door onto the brief
// session's relay and page (itd-2610091016262335, spc-2610141709430146 steps
// 1 to 3).
func newSessionCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "session",
		Long: "Set up live brief sessions for this project, start one on this computer, run\n" +
			"the relay a team's cloud sessions go through, write the blueprint that deploys\n" +
			"that relay to the team's own Render account, or write the session's static page\n" +
			"for the team to serve.\n" +
			"The relay holds each session's edits in memory only, as opaque bytes, and drops\n" +
			"them when the host closes the session or nobody has been connected for half an\n" +
			"hour; it writes nothing to disk and logs nothing. abcd runs no relay of its own:\n" +
			"the team runs it, on the host's computer reached over Tailscale Serve\n" +
			"(`abcd session start`), or on the team's own relay (`abcd session serve`).\n\n" +
			"Bare `abcd session` prints this help and writes nothing.",
		Args: cobra.NoArgs,
		RunE: helpRunE,
	}
	cmd.AddCommand(newSessionSetupCommand(asJSON), newSessionStartCommand(asJSON), newSessionServeCommand(asJSON), newSessionBlueprintCommand(asJSON), newSessionPageCommand(asJSON))
	return cmd
}

// sessionStart is the one line, or JSON object, serve prints once it listens.
type sessionStart struct {
	URL      string   `json:"url"`
	Loopback bool     `json:"loopback"`
	Proxy    bool     `json:"behind_proxy"`
	SignIn   bool     `json:"sign_in"`
	Repos    []string `json:"repos"`
	Owners   []string `json:"owners"`
	AnyRepo  bool     `json:"any_repository"`
	Hint     string   `json:"hint"`
}

// sessionAllowlist gathers the repositories serve is told to serve: every
// --repo, every --owner, and every entry of ABCD_SESSION_REPOS, refusing one
// that is not owner/name or an owner.
func sessionAllowlist(repos, owners []string, env string) ([]string, []string, error) {
	var outRepos, outOwners []string
	for _, r := range repos {
		if !coresession.ValidRepo(r) {
			return nil, nil, fmt.Errorf("--repo %s is not a repository: give owner/name", strconv.Quote(r))
		}
		outRepos = append(outRepos, r)
	}
	for _, o := range owners {
		if !coresession.ValidOwner(o) {
			return nil, nil, fmt.Errorf("--owner %s is not a GitHub user or organisation", strconv.Quote(o))
		}
		outOwners = append(outOwners, o)
	}
	if strings.TrimSpace(env) == "" {
		return outRepos, outOwners, nil
	}
	for _, e := range strings.Split(env, ",") {
		e = strings.TrimSpace(e)
		switch {
		case strings.Contains(e, "/") && coresession.ValidRepo(e):
			outRepos = append(outRepos, e)
		case !strings.Contains(e, "/") && coresession.ValidOwner(e):
			outOwners = append(outOwners, e)
		default:
			return nil, nil, fmt.Errorf("%s holds %s, which is neither owner/name nor an owner", sessionReposEnv, strconv.Quote(e))
		}
	}
	return outRepos, outOwners, nil
}

// sessionServed is what serve serves: the list it is given, or, with none,
// the origin repository of the checkout at cwd on loopback (every repository
// outside one), and a refusal beyond loopback.
func sessionServed(loopback bool, repos, owners []string, env, cwd string) ([]string, []string, error) {
	r, o, err := sessionAllowlist(repos, owners, env)
	if err != nil || len(r) > 0 || len(o) > 0 {
		return r, o, err
	}
	if !loopback {
		return nil, nil, errors.New("listening beyond this computer, the relay serves only the repositories named with --repo, --owner or " + sessionReposEnv + "; name at least one")
	}
	if cwd != "" {
		if origin, err := sessionOriginRepo(cwd); err == nil && coresession.ValidRepo(origin) {
			return []string{origin}, nil, nil
		}
	}
	return nil, nil, nil
}

// sessionTailnetName is this computer's tailnet name, as Tailscale's own
// read-only status reports it. It is a package var so a test can answer for
// it: no test runs Tailscale.
var sessionTailnetName = func(ctx context.Context, root string) (string, error) {
	bin, err := tailscale.Resolve(exec.LookPath, tailscale.BundledCommands, root)
	if err != nil {
		return "", err
	}
	self, err := tailscale.New(bin).Status(ctx)
	if err != nil {
		return "", err
	}
	return self.Name, nil
}

// sessionLocal is the start line, or JSON object, `session start` prints once
// it listens.
type sessionLocal struct {
	URL       string `json:"url"`
	Link      string `json:"link"`
	Repo      string `json:"repo"`
	Tailnet   string `json:"tailnet"`
	ServeStep string `json:"serve_step"`
	SignIn    string `json:"sign_in"`
}

// sessionDefaultPort is where a local session listens unless --port names
// another: the relay's own default.
const sessionDefaultPort = 8787

func newSessionStartCommand(asJSON *bool) *cobra.Command {
	var port int
	var clientID string
	c := &cobra.Command{
		Use: "start",
		Long: "Start a live brief session on this computer, for a project whose brief should\n" +
			"not leave the team's tailnet. It runs the session relay on loopback and serves the\n" +
			"session page from it, so nothing goes to abcdesign.app, and prints the one\n" +
			"Tailscale step that shares it on your tailnet and the link to open.\n\n" +
			"It serves sessions for this checkout's origin repository on github.com. Team\n" +
			"members sign in with GitHub's device flow: the page shows a short code to confirm\n" +
			"on github.com, so no secret sits on this computer. The GitHub App's public client\n" +
			"id comes from --client-id, " + sessionClientIDEnv + ", or\n" +
			".abcd/relay/" + coresession.AppFile + ", which repository setup writes; without\n" +
			"one it refuses, since nobody could sign in, and a person at a terminal is first\n" +
			"asked the live brief session question while it is unanswered. Anyone else joins\n" +
			"as a guest with a\n" +
			"name, and suggests changes.\n\n" +
			"It asks Tailscale for this computer's tailnet name and changes nothing in\n" +
			"Tailscale. It answers only that name and its own loopback address, as hosts and\n" +
			"as page origins. People outside your tailnet join as guests once you share this\n" +
			"computer with them in Tailscale. Never Funnel.\n\n" +
			"It runs in the foreground until interrupted; every session ends with it.\n" +
			"Exit 2 on a refusal, with nothing started.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			// A question drawn at a terminal and stopped by Ctrl-C ends the verb (exit 130).
			defer endOnStop(&err)
			const prefix = "abcd session start: "
			refuse := func(msg string) error { return &exitError{Code: 2, Msg: prefix + msg + " (nothing started)"} }
			if port < 0 || port > 65535 {
				return refuse("--port " + strconv.Itoa(port) + " is not 0 to 65535")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the live session's checkout")
			if err != nil {
				return refuse(err.Error())
			}
			repo, err := sessionOriginRepo(root)
			if err != nil || !coresession.ValidRepo(repo) {
				return refuse("a live session serves this checkout's repository, and its origin does not name one on github.com")
			}
			if clientID == "" {
				clientID = strings.TrimSpace(os.Getenv(sessionClientIDEnv))
			}
			if clientID == "" {
				id, err := coresession.ReadAppClientID(root)
				if err != nil {
					return refuse(err.Error())
				}
				clientID = id
			}
			if clientID == "" {
				return refuse("sign-in is not set up for this project: no GitHub App is recorded in .abcd/relay/" + coresession.AppFile +
					"; " + sessionSetupHint(cmd, root) + ", or name the app's client id with --client-id")
			}
			if !coresession.ValidClientID(clientID) {
				return refuse("--client-id " + strconv.Quote(clientID) + " is not a GitHub App's client id")
			}
			name, err := sessionTailnetName(cmd.Context(), root)
			name = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
			if err != nil || name == "" || strings.ContainsAny(name, "/:@ ") {
				why := "Tailscale names no tailnet name for this computer"
				if err != nil {
					why = err.Error()
				}
				return refuse("a local session is reached over Tailscale, and " + why)
			}
			ln, err := session.Listen(netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(port)))
			if err != nil {
				return refuse("cannot listen on 127.0.0.1:" + strconv.Itoa(port) + ": " + err.Error())
			}
			bound := ln.Addr().String()
			boundPort := bound[strings.LastIndex(bound, ":")+1:]
			// The page and the relay are one server: its origin is the
			// tailnet name Serve answers on, or this computer's own address.
			srv, err := session.New(session.Config{
				Origins:      []string{"https://" + name, "http://127.0.0.1:" + boundPort, "http://localhost:" + boundPort},
				Hosts:        []string{name, "127.0.0.1:" + boundPort, "localhost:" + boundPort},
				LoopbackOnly: true,
				Repos:        []string{repo},
				ClientID:     clientID,
				ServePage:    true,
			})
			if err != nil {
				_ = ln.Close()
				return refuse(err.Error())
			}
			local := sessionLocal{
				URL:       "http://" + bound,
				Link:      "https://" + name + "/" + repo + "/brief/",
				Repo:      repo,
				Tailnet:   name,
				ServeStep: "tailscale serve --bg " + boundPort,
				SignIn:    "device",
			}
			_ = render(cmd.OutOrStdout(), *asJSON, local, func(w io.Writer) {
				fmt.Fprintf(w, "abcd session start: a live brief session for %s is ready on this computer\n", termsafe.Sanitize(repo))
				fmt.Fprintln(w, "  1. share it on your tailnet, once (Tailscale keeps it until you turn it off):")
				fmt.Fprintf(w, "       %s\n", local.ServeStep)
				fmt.Fprintln(w, "  2. open this link, sign in with GitHub and start the session:")
				fmt.Fprintf(w, "       %s\n", termsafe.Sanitize(local.Link))
				fmt.Fprintln(w, "     the page then shows a code and a QR code to invite people on your tailnet")
				fmt.Fprintln(w, "  people outside your tailnet can join as guests once you share this computer with them in Tailscale")
				fmt.Fprintln(w, "  stop it with Ctrl-C; every session ends with it")
			})
			if err := srv.Run(cmd.Context(), ln); err != nil && !errors.Is(err, cmd.Context().Err()) {
				return fmt.Errorf("%s%w", prefix, err)
			}
			return nil
		},
	}
	c.Flags().IntVar(&port, "port", sessionDefaultPort, "the port on this computer to listen on (0 lets the system choose)")
	c.Flags().StringVar(&clientID, "client-id", "", "the team's GitHub App's public client id (default: "+sessionClientIDEnv+", then .abcd/relay/"+coresession.AppFile+")")
	return c
}

func newSessionServeCommand(asJSON *bool) *cobra.Command {
	var listen, clientID string
	var origins, repos, owners []string
	var behindProxy bool
	c := &cobra.Command{
		Use: "serve",
		Long: "Run the brief session relay in the foreground until interrupted. It listens on\n" +
			"this computer alone (" + session.DefaultListen + ") unless --listen names another address,\n" +
			"which is the second, explicit choice: the team's own relay deployment listens on\n" +
			"0.0.0.0 behind its host's proxy, with --behind-proxy. On a person's own machine,\n" +
			"teammates reach it through Tailscale Serve, on the tailnet only; the start line\n" +
			"prints the command. Never Funnel: a guest outside the tailnet joins through the\n" +
			"team's relay instead.\n\n" +
			"It serves sessions only for the repositories it is given: each --repo owner/name,\n" +
			"every repository of each --owner, and each entry of " + sessionReposEnv + ", a\n" +
			"comma list of either. Nobody signs in against, or hosts, any other. Listening\n" +
			"beyond this computer it refuses to start without one; on loopback it serves this\n" +
			"checkout's origin repository, or any repository outside a GitHub checkout.\n\n" +
			"It serves the session WebSocket at /session, the sign-in's callback at\n" +
			"/auth/callback (the callback URL the team's GitHub App lists) and its exchange\n" +
			"at /auth/token, the GitHub App's public client id at /config and a health check\n" +
			"at /healthz. It admits a browser only from https://abcdesign.app or a loopback\n" +
			"page, or, when --origin is given, from exactly the origins it names, and otherwise, when\n" +
			sessionOriginsEnv + " is set, from exactly the origins that comma list names.\n" +
			"An entry for a page served below a base path names it, as abcd session page\n" +
			"--base-path wrote it (https://host/base/); the callback forwards a sign-in only\n" +
			"to the page's own base path, the root for an entry that names none.\n" +
			"Sign-in goes through the team's GitHub App only, and nobody pastes a token: the\n" +
			"relay takes only a GitHub App user access token. Its client id is read from\n" +
			"--client-id or " + sessionClientIDEnv + ", and its secret from\n" +
			sessionSecretEnv + "; with both, members sign in with GitHub's web\n" +
			"flow. Without both, sign-in is not offered and only guests can join: GitHub's\n" +
			"device flow, which needs no secret, belongs to abcd session start alone, whose\n" +
			"relay serves its own session page.\n" +
			sessionLegacyAppTypeEnv + " set to anything but github-app is refused: the\n" +
			"relay takes no OAuth app.\n\n" +
			"Exit 2 on a refusal, with nothing started.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd session serve: "
			addr, err := session.ParseListen(listen)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error() + " (nothing started)"}
			}
			loopback := addr.Addr().IsLoopback()
			cwd, _ := os.Getwd()
			allowRepos, allowOwners, err := sessionServed(loopback, repos, owners, os.Getenv(sessionReposEnv), cwd)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error() + " (nothing started)"}
			}
			if clientID == "" {
				clientID = strings.TrimSpace(os.Getenv(sessionClientIDEnv))
			}
			if v := strings.TrimSpace(os.Getenv(sessionLegacyAppTypeEnv)); v != "" && v != session.AppTypeGitHubApp {
				return &exitError{Code: 2, Msg: prefix + sessionLegacyAppTypeEnv + " is " + strconv.Quote(v) +
					"; the relay signs in through a GitHub App only: register one with Contents and Pull requests read and write and Metadata read, and unset the variable (nothing started)"}
			}
			admit, admitFrom := sessionOrigins(origins, os.Getenv(sessionOriginsEnv))
			// On loopback alone, the relay is the host's own machine behind
			// Tailscale Serve, and a visitor is counted by the tailnet address
			// Serve forwards.
			srv, err := session.New(session.Config{
				Origins:      admit,
				BehindProxy:  behindProxy,
				LoopbackOnly: loopback,
				Repos:        allowRepos,
				Owners:       allowOwners,
				ClientID:     clientID,
				ClientSecret: os.Getenv(sessionSecretEnv),
			})
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + admitFrom + " " + err.Error() + " (nothing started)"}
			}
			ln, err := session.Listen(addr)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + "cannot listen on " + addr.String() + ": " + err.Error() + " (nothing started)"}
			}
			bound := ln.Addr().String()
			start := sessionStart{
				URL:      "http://" + bound,
				Loopback: loopback,
				Proxy:    behindProxy,
				SignIn:   clientID != "" && os.Getenv(sessionSecretEnv) != "",
				Repos:    append([]string{}, allowRepos...),
				Owners:   append([]string{}, allowOwners...),
				AnyRepo:  len(allowRepos) == 0 && len(allowOwners) == 0,
			}
			if loopback {
				port := bound[strings.LastIndex(bound, ":")+1:]
				start.Hint = "teammates on your tailnet reach it once you run `tailscale serve --bg " + port +
					"`; open the session page with the https address that prints"
			} else {
				start.Hint = "listening beyond this computer, on an address chosen with --listen"
			}
			_ = render(cmd.OutOrStdout(), *asJSON, start, func(w io.Writer) {
				where := "this computer alone (loopback)"
				if !loopback {
					where = "every address --listen names, beyond this computer"
				}
				fmt.Fprintf(w, "abcd session serve: relay listening at %s, on %s\n", termsafe.Sanitize(start.URL), where)
				fmt.Fprintf(w, "  %s\n", start.Hint)
				if start.AnyRepo {
					fmt.Fprintln(w, "  serves sessions for every repository (loopback only; name one with --repo)")
				} else {
					names := append([]string{}, start.Repos...)
					for _, o := range start.Owners {
						names = append(names, "every repository of "+o)
					}
					fmt.Fprintf(w, "  serves sessions for %s\n", termsafe.Sanitize(strings.Join(names, ", ")))
				}
				if start.SignIn {
					fmt.Fprintln(w, "  sign-in: the team's GitHub App, through GitHub's web flow")
				} else {
					fmt.Fprintf(w, "  sign-in is not set up (set --client-id and %s); only guests can join\n", sessionSecretEnv)
				}
				fmt.Fprintln(w, "  stop it with Ctrl-C; every session it holds ends with it")
			})
			if err := srv.Run(cmd.Context(), ln); err != nil && !errors.Is(err, cmd.Context().Err()) {
				return fmt.Errorf("%s%w", prefix, err)
			}
			return nil
		},
	}
	c.Flags().StringVar(&listen, "listen", session.DefaultListen, "the address to listen on: an IP address (or localhost) and a port; anything but loopback is listening beyond this computer")
	c.Flags().BoolVar(&behindProxy, "behind-proxy", false, "trust the last X-Forwarded-For entry as the client's address; only for the team relay behind its host's proxy")
	c.Flags().StringArrayVar(&origins, "origin", nil, "a page origin to admit, as scheme://host[:port], followed by the page's base path when it is served below one, as https://host/base/ (repeatable; also "+sessionOriginsEnv+"); given, it replaces the default of https://abcdesign.app and loopback pages")
	c.Flags().StringArrayVar(&repos, "repo", nil, "a repository to serve sessions for, as owner/name (repeatable; also "+sessionReposEnv+")")
	c.Flags().StringArrayVar(&owners, "owner", nil, "a GitHub user or organisation whose every repository to serve sessions for (repeatable)")
	c.Flags().StringVar(&clientID, "client-id", "", "the team's GitHub sign-in app's public client id (default: "+sessionClientIDEnv+")")
	return c
}

func newSessionBlueprintCommand(asJSON *bool) *cobra.Command {
	var release, relayOrigin string
	var pageOrigins []string
	c := &cobra.Command{
		Use: "blueprint",
		Long: "Write the team's own relay deployment into this checkout: render.yaml, a Render\n" +
			"Blueprint declaring one free web service, at the repository's root, where\n" +
			"Render's Deploy button reads it, and .abcd/relay/Dockerfile,\n" +
			"which downloads the released abcd binary for linux/amd64 and checks it against\n" +
			"the SHA-256 that release's checksums.txt publishes. The checksum is read from the\n" +
			"release page when the files are written (the one network read this verb makes,\n" +
			"named on stderr) and pinned in the Dockerfile. The relay serves sessions for this\n" +
			"checkout's origin repository on github.com, named in render.yaml as\n" +
			sessionReposEnv + ", a plain value; it refuses a checkout without one.\n\n" +
			"render.yaml also names, in " + sessionOriginsEnv + ", the page origins the relay\n" +
			"admits: each --origin, an https origin followed by the page's base path when it\n" +
			"is served below one, or " + coresession.DefaultPageOrigin + " by default, and,\n" +
			"once repository setup has recorded the team's GitHub App in\n" +
			".abcd/relay/" + coresession.AppFile + ", its public client id, so Render's form asks\n" +
			"only for the secret. A root render.yaml the team wrote is never touched: the\n" +
			"Blueprint then goes to .abcd/relay/render.yaml, picked by its Blueprint Path on\n" +
			"Render's New Blueprint form.\n\n" +
			"While the live brief session question is unanswered, a person at a terminal is\n" +
			"asked it first, as repository setup asks it.\n\n" +
			"--relay-origin names the deployed relay's https origin, once Render has given it\n" +
			"one; the verb then also writes .abcd/relay/relays.json, naming it as the relay\n" +
			"this repository trusts, which the session page reads from the repository, and\n" +
			"prints the callback URL the team's GitHub App must list for the relay's sign-in:\n" +
			"the relay's origin followed by " + coresession.RelayCallbackPath + ".\n\n" +
			"--release names the release to pin; it defaults to this abcd's own version when\n" +
			"that is a release. A file left exactly as abcd wrote it is replaced; one that was\n" +
			"edited, or that abcd did not write, is never overwritten, and nothing is written.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			// A question drawn at a terminal and stopped by Ctrl-C ends the verb (exit 130).
			defer endOnStop(&err)
			const prefix = "abcd session blueprint: "
			if release == "" {
				release = core.Version
				if !coresession.ValidReleaseTag(release) {
					return &exitError{Code: 2, Msg: prefix + "this abcd (" + release + ") is not a release; name one with --release vMAJOR.MINOR.PATCH (nothing written)"}
				}
			}
			if !coresession.ValidReleaseTag(release) {
				return &exitError{Code: 2, Msg: prefix + strconv.Quote(release) + " is not a release tag; name one as vMAJOR.MINOR.PATCH (nothing written)"}
			}
			if relayOrigin != "" {
				if _, err := coresession.NormalizeRelayOrigin(relayOrigin); err != nil {
					return &exitError{Code: 2, Msg: prefix + "--relay-origin " + err.Error() + " (nothing written)"}
				}
			}
			for _, o := range pageOrigins {
				if _, err := coresession.NormalizePageOrigin(o); err != nil {
					return &exitError{Code: 2, Msg: prefix + "--origin " + err.Error() + " (nothing written)"}
				}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the relay blueprint's checkout")
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error() + " (nothing written)"}
			}
			if !ahoy.Managed(root) {
				return &exitError{Code: 2, Msg: prefix + "this checkout is not one abcd manages; run `abcd ahoy install` first (nothing written)"}
			}
			served, err := sessionOriginRepo(root)
			if err != nil || !coresession.ValidRepo(served) {
				return &exitError{Code: 2, Msg: prefix + "the relay serves this checkout's repository, and its origin does not name one on github.com (nothing written)"}
			}
			// The live-session question, while it is open, is asked again
			// here, of a person at a terminal (spc-2610141709430146 step 6).
			if v, answered, aerr := ahoy.AskLiveSessions(root, newPrompter(cmd)); aerr == nil && answered {
				fmt.Fprintf(cmd.ErrOrStderr(), "%srecorded sessions.live: %s in .abcd/config.json\n", prefix, v)
			}
			clientID, err := coresession.ReadAppClientID(root)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error() + " (nothing written)"}
			}
			fetch, err := newSessionReleaseFetcher()
			if err != nil {
				return fmt.Errorf("%s%w", prefix, err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%sreading release %s's checksums.txt from %s\n", prefix, termsafe.Sanitize(release), sessionReleaseOrigin)
			res, err := coresession.WriteBlueprint(coresession.BlueprintRequest{Root: root, Release: release, Running: core.Version, Fetch: fetch, Repos: []string{served}, RelayOrigin: relayOrigin, Origins: pageOrigins, ClientID: clientID})
			if err != nil {
				var ref *coresession.BlueprintRefusal
				if errors.As(err, &ref) || res.Digest == "" {
					return &exitError{Code: 2, Msg: prefix + err.Error()}
				}
				return fmt.Errorf("%s%w", prefix, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				for _, f := range res.Files {
					fmt.Fprintf(w, "abcd session blueprint: %s %s\n", f.State, f.Path)
				}
				fmt.Fprintf(w, "  pinned: abcd %s, %s sha256 %s\n", termsafe.Sanitize(res.Release), res.Asset, res.Digest)
				fmt.Fprintf(w, "  serves: %s (%s in render.yaml)\n", termsafe.Sanitize(strings.Join(res.Repos, ", ")), sessionReposEnv)
				fmt.Fprintf(w, "  admits pages from: %s (%s in render.yaml)\n", termsafe.Sanitize(strings.Join(res.Origins, ", ")), sessionOriginsEnv)
				fmt.Fprintf(w, "  from: %s\n", termsafe.Sanitize(res.Source))
				if res.Callback != "" {
					fmt.Fprintf(w, "  callback: list %s as a Callback URL on the team's GitHub App (its settings page, under Identifying and authorizing users), with wildcard matching off\n", termsafe.Sanitize(res.Callback))
				}
				for _, line := range blueprintNext(res) {
					fmt.Fprintf(w, "  %s\n", line)
				}
			})
		},
	}
	c.Flags().StringArrayVar(&pageOrigins, "origin", nil, "a session page origin the relay admits, as https://host[:port], followed by the page's base path when it is served below one, as https://host/base/ (repeatable; default "+coresession.DefaultPageOrigin+")")
	c.Flags().StringVar(&relayOrigin, "relay-origin", "", "the deployed relay's https origin, as https://host[:port]; given, .abcd/relay/relays.json names it as the relay this repository trusts")
	c.Flags().StringVar(&release, "release", "", "the abcd release tag to pin, as vMAJOR.MINOR.PATCH (default: this abcd's own version, when it is a release)")
	return c
}

func newSessionPageCommand(asJSON *bool) *cobra.Command {
	var out, basePath string
	c := &cobra.Command{
		Use: "page",
		Long: "Write the brief session's page into --out: index.html and an identical 404.html,\n" +
			"session.js, session.css, and the vendored editor bundle with its licences under\n" +
			"vendor/. The files are plain and static, embedded in this binary; serve them\n" +
			"from any static host, such as GitHub Pages, whose 404.html fallback is how\n" +
			"/<owner>/<repo>/brief/ reaches the page. The page connects only to the relay it\n" +
			"finds itself (its own origin, the invite link's, or the first the repository's\n" +
			"relays.json names) and to api.github.com with the person's own token, and it\n" +
			"sends the\n" +
			"person to github.com for sign-in and to open the pre-filled issues a session\n" +
			"sends. It reads nothing from this checkout.\n\n" +
			"--base-path names the path the site is served under, such as a project's\n" +
			"GitHub Pages address owner.github.io/<repo>/: the page then loads its files from\n" +
			"under it and opens at <base-path>/<owner>/<repo>/brief/. It is plain path\n" +
			"segments, and the root by default.\n\n" +
			"It writes those six files and nothing else, and removes nothing. --out may be\n" +
			"absent, empty, or a directory an earlier run wrote, whose files are replaced; a\n" +
			"directory holding anything else, a symlink or a file is refused.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd session page: "
			res, err := web.Write(out, basePath)
			if err != nil {
				var ref *web.Refusal
				if errors.As(err, &ref) {
					return &exitError{Code: 2, Msg: prefix + ref.Error() + " (nothing written)"}
				}
				return fmt.Errorf("%s%w", prefix, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				verb := "wrote"
				if res.Replaced {
					verb = "replaced"
				}
				fmt.Fprintf(w, "abcd session page: %s %d files (%d bytes) in %s\n", verb, len(res.Files), res.Bytes, termsafe.Sanitize(res.Out))
				for _, f := range res.Files {
					fmt.Fprintf(w, "  %s\n", f)
				}
				fmt.Fprintf(w, "  next: serve the directory from a static host; the page opens at %s<owner>/<repo>/brief/\n", res.BasePath)
			})
		},
	}
	c.Flags().StringVar(&out, "out", "", "the directory to write the page into: absent, empty, or one an earlier run wrote")
	c.Flags().StringVar(&basePath, "base-path", "/", "the path the site is served under, as plain path segments")
	return c
}

// sessionSetupHint is the way on for a project whose sign-in is not set up.
// While the live brief session question is unanswered, a person at a
// terminal is asked it here, as repository setup asks it
// (spc-2610141709430146 step 6), and the hint follows the answer.
func sessionSetupHint(cmd *cobra.Command, root string) string {
	if v, answered, err := ahoy.AskLiveSessions(root, newPrompter(cmd)); err == nil && answered {
		if v == ahoy.LiveSessionsNone {
			return "live sessions were declined for this project; abcd session setup still sets them up by hand"
		}
		return "recorded that sessions may run " + sessionWhereWords(v) + "; run abcd session setup to create the team's sign-in app"
	}
	if v, set, err := ahoy.LiveSessions(root); err == nil && !set {
		return "live sessions are not set up yet: answer the question abcd ahoy install asks at a terminal, or run abcd session setup"
	} else if err == nil && v == ahoy.LiveSessionsNone {
		return "live sessions were declined for this project; abcd session setup still sets them up by hand"
	}
	return "run abcd session setup to create the team's sign-in app"
}

// sessionWhereWords says where sessions may run, in words.
func sessionWhereWords(where string) string {
	switch where {
	case coresession.WhereLocal:
		return "on the host's computer"
	case coresession.WhereCloud:
		return "on the team's relay"
	}
	return "on the host's computer or the team's relay"
}

// blueprintNext is what to do once the blueprint is written: Render's Deploy
// button where render.yaml sits at the root, else Render's New Blueprint
// form with its Blueprint Path.
func blueprintNext(res coresession.BlueprintResult) []string {
	files := res.BlueprintPath + " and .abcd/relay/"
	if res.DeployButton {
		return []string{
			"next: commit and push " + files + ", then deploy the relay with Render's Deploy button:",
			"      " + res.DeployURL,
			"      Render's form asks for " + sessionSecretEnv + "; generate it on your GitHub App's settings page and paste it there",
		}
	}
	return []string{
		"next: commit and push " + files + "; on Render choose New, then Blueprint, pick this repository and set the Blueprint Path to " + res.BlueprintPath,
		"      (the root render.yaml is your own, so Render's Deploy button would read that one)",
		"      Render's form asks for " + sessionSecretEnv + "; generate it on your GitHub App's settings page and paste it there",
	}
}

// The GitHub API session setup asks, and the browser it sends the person to.
// Package vars so a test answers for GitHub and plays the browser: no test
// reaches the network.
var (
	sessionGitHubAPI  = "https://api.github.com"
	sessionHTTPClient = &http.Client{Timeout: 30 * time.Second}
	// sessionSetupVisit is told the setup page's address once it answers.
	// abcd opens nothing itself: the person opens the address it prints.
	sessionSetupVisit = func(string) {}
)

// sessionAppWait is how long setup waits for the person to create the app:
// GitHub's manifest code lasts an hour.
var sessionAppWait = time.Hour

// sessionSetupResult is what `session setup` did, and the steps left to the
// person, in order.
type sessionSetupResult struct {
	Repo        string                       `json:"repo"`
	Where       string                       `json:"where"`
	Recorded    string                       `json:"recorded,omitempty"`
	ClientID    string                       `json:"client_id"`
	AppCreated  bool                         `json:"app_created"`
	AppSettings string                       `json:"app_settings,omitempty"`
	AppInstall  string                       `json:"app_install,omitempty"`
	Blueprint   *coresession.BlueprintResult `json:"blueprint,omitempty"`
	RelayOrigin string                       `json:"relay_origin,omitempty"`
	Callback    string                       `json:"callback_url,omitempty"`
	Steps       []string                     `json:"steps"`
}

func newSessionSetupCommand(asJSON *bool) *cobra.Command {
	var where, relayOrigin, release string
	c := &cobra.Command{
		Use: "setup",
		Long: "Set up live brief sessions for this project, the step repository setup asks\n" +
			"about (Set up live brief sessions? Where may sessions run?). It runs by hand at any\n" +
			"time, a no at repository setup included. Where sessions may run comes from\n" +
			"--where, else from .abcd/config.json (sessions.live), else, at a terminal, from\n" +
			"the question; the answer is recorded there.\n\n" +
			"It registers the team's own GitHub sign-in app through GitHub's App manifest\n" +
			"flow: it prints an address on this computer whose page opens GitHub with the app\n" +
			"filled in (Contents and Pull requests read and write, Metadata read, no\n" +
			"webhook), and the person clicks Create GitHub App. It keeps only the app's public\n" +
			"client id, in .abcd/relay/" + coresession.AppFile + "; the client secret, the webhook\n" +
			"secret and the private key GitHub hands back are never written or shown. An app\n" +
			"already recorded is kept, and setup says to remove the session page's own\n" +
			"address from its callback URLs if it is listed, as an earlier setup had it.\n\n" +
			"It then says what is left, with GitHub's own pages: installing the app on the\n" +
			"repository; for local sessions, ticking Enable Device Flow on the app's settings\n" +
			"page, which a manifest cannot do; for the cloud only, leaving it off. For the\n" +
			"cloud it writes the relay's blueprint (as abcd session blueprint does) and gives\n" +
			"Render's Deploy button, whose form asks for the client secret, which the person\n" +
			"generates on the app's settings page and pastes straight in. --relay-origin\n" +
			"records the deployed relay's address in .abcd/relay/relays.json and names the\n" +
			"callback URL the app must list, the relay's origin followed by " + coresession.RelayCallbackPath + ";\n" +
			"given before the app is created, the manifest sets it.\n\n" +
			"It asks api.github.com whether the owner is an organisation, and converts the\n" +
			"app GitHub created; for the cloud it reads the release's checksums. It waits up\n" +
			"to an hour for the app to be created. Where sessions may run is recorded once\n" +
			"the app is. Exit 2 on a refusal before then, with nothing written; a blueprint\n" +
			"that cannot be written afterwards exits 2 with the app and the answer kept.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			defer endOnStop(&err)
			const prefix = "abcd session setup: "
			refuse := func(msg string) error { return &exitError{Code: 2, Msg: prefix + msg + " (nothing written)"} }
			if where != "" && !coresession.ValidWhere(where) {
				return refuse("--where takes local, cloud or both, not " + strconv.Quote(where))
			}
			if relayOrigin != "" {
				o, nerr := coresession.NormalizeRelayOrigin(relayOrigin)
				if nerr != nil {
					return refuse("--relay-origin " + nerr.Error())
				}
				relayOrigin = o
			}
			if release != "" && !coresession.ValidReleaseTag(release) {
				return refuse(strconv.Quote(release) + " is not a release tag; name one as vMAJOR.MINOR.PATCH")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the live session's checkout")
			if err != nil {
				return refuse(err.Error())
			}
			if !ahoy.Managed(root) {
				return refuse("this checkout is not one abcd manages; run `abcd ahoy install` first")
			}
			repo, err := sessionOriginRepo(root)
			if err != nil || !coresession.ValidRepo(repo) {
				return refuse("live sessions serve this checkout's repository, and its origin does not name one on github.com")
			}
			stored, set, err := ahoy.LiveSessions(root)
			if err != nil {
				return refuse(err.Error())
			}
			chosen := where
			if chosen == "" && set && stored != ahoy.LiveSessionsNone {
				chosen = stored
			}
			if chosen == "" {
				v, ok := ahoy.AskSessionsWhere(newPrompter(cmd))
				if !ok {
					return refuse("say where sessions may run with --where local|cloud|both, or run it at a terminal and answer the question")
				}
				chosen = v
			}
			if relayOrigin != "" && !coresession.AllowsCloud(chosen) {
				return refuse("--relay-origin is the team relay's address, and sessions here run on the host's computer only")
			}
			if coresession.AllowsCloud(chosen) && release == "" {
				release = core.Version
				if !coresession.ValidReleaseTag(release) {
					return refuse("this abcd (" + release + ") is not a release, and the relay's blueprint pins one; name it with --release vMAJOR.MINOR.PATCH")
				}
			}
			clientID, err := coresession.ReadAppClientID(root)
			if err != nil {
				return refuse(err.Error())
			}
			// For the cloud, the release's checksums are read before GitHub is
			// asked for anything, so nothing after the app's creation can
			// fail on a release that cannot be pinned.
			var fetch *onceFetch
			if coresession.AllowsCloud(chosen) {
				inner, ferr := newSessionReleaseFetcher()
				if ferr != nil {
					return fmt.Errorf("%s%w", prefix, ferr)
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "%sreading release %s's checksums.txt from %s\n", prefix, termsafe.Sanitize(release), sessionReleaseOrigin)
				fetch = &onceFetch{inner: inner}
				_, src, found, ferr := fetch.FetchReleaseAsset(release, "checksums.txt")
				switch {
				case ferr != nil:
					return refuse("reading release " + release + "'s checksums: " + ferr.Error())
				case !found:
					return refuse("release " + release + " publishes no checksums at " + src + ", so the relay cannot be pinned to it")
				}
			}
			res := sessionSetupResult{Repo: repo, Where: chosen, ClientID: clientID}
			stderr := cmd.ErrOrStderr()
			var app coresession.CreatedApp
			if clientID == "" {
				owner, _, _ := strings.Cut(repo, "/")
				org, oerr := session.OwnerIsOrganisation(cmd.Context(), sessionHTTPClient, sessionGitHubAPI, owner)
				if oerr != nil {
					return refuse("could not ask GitHub whether " + owner + " is an organisation, which decides where its app is registered: " + oerr.Error())
				}
				manifest, merr := coresession.NewAppManifest(repo, chosen, "", relayOrigin)
				if merr != nil {
					return refuse(merr.Error())
				}
				ln, lerr := session.LoopbackListener()
				if lerr != nil {
					return refuse("cannot listen on this computer for GitHub's answer: " + lerr.Error())
				}
				flow := &session.AppSetup{
					Manifest: manifest,
					FormURL:  func(state string) string { return coresession.ManifestFormURL(owner, org, state) },
					API:      sessionGitHubAPI,
					Client:   sessionHTTPClient,
				}
				ctx, cancel := context.WithTimeout(cmd.Context(), sessionAppWait)
				defer cancel()
				app, err = flow.Run(ctx, ln, func(page string) {
					fmt.Fprintf(stderr, "%sopen this address in your browser to create the team's sign-in app:\n", prefix)
					fmt.Fprintf(stderr, "  %s\n", page)
					fmt.Fprintln(stderr, "  It opens GitHub with the app filled in for "+repo+": check its name, then click Create GitHub App.")
					fmt.Fprintln(stderr, "  Waiting for GitHub, for up to an hour; Ctrl-C stops without creating anything here.")
					sessionSetupVisit(page)
				})
				switch {
				case errors.Is(err, context.Canceled):
					return &exitError{Code: 1, Msg: prefix + "stopped before GitHub's answer came back; nothing was recorded"}
				case errors.Is(err, context.DeadlineExceeded):
					return &exitError{Code: 1, Msg: prefix + "no app came back from GitHub within the hour; nothing was recorded, so run abcd session setup again"}
				case err != nil:
					return &exitError{Code: 1, Msg: prefix + "the sign-in app was not recorded: " + err.Error()}
				}
				// For an organisation the form posts to the organisation's own
				// page, so the app GitHub hands back must be the organisation's:
				// one any other account owns came from another hand, and is
				// never recorded or offered for installing. For a person the
				// owner is whoever was signed in, so it is named instead.
				if org && !strings.EqualFold(app.OwnerLogin, owner) {
					return &exitError{Code: 1, Msg: prefix + "the app GitHub handed back belongs to " + termsafe.Sanitize(app.OwnerLogin) +
						", not the organisation " + owner + " it was created for, so it was not recorded; run abcd session setup again, and delete that app on GitHub if you made it"}
				}
				if !org {
					fmt.Fprintf(stderr, "%sthe app belongs to the GitHub account %s\n", prefix, termsafe.Sanitize(app.OwnerLogin))
				}
				if err := coresession.WriteAppFile(root, app.ClientID); err != nil {
					return &exitError{Code: 1, Msg: prefix + "GitHub created the app, but its client id could not be written: " + err.Error() +
						"; write {\"client_id\": \"" + app.ClientID + "\"} into .abcd/relay/" + coresession.AppFile}
				}
				res.ClientID, res.AppCreated = app.ClientID, true
				res.AppSettings, res.AppInstall = app.SettingsURL(), app.InstallURL()
			}
			// Where sessions may run is recorded once the app is, so every
			// refusal before this point leaves the checkout as it was.
			if !set || stored != chosen {
				if _, serr := ahoy.StoreLiveSessions(root, chosen); serr == nil {
					from := "not set"
					if set {
						from = stored
					}
					res.Recorded = "sessions.live: " + from + " -> " + chosen
				} else {
					fmt.Fprintf(stderr, "%swhere sessions may run was not recorded: %s\n", prefix, termsafe.Sanitize(serr.Error()))
				}
			}
			settings := res.AppSettings
			if settings == "" {
				settings = "the app's settings page on GitHub (Settings, then Developer settings, then GitHub Apps, then Edit beside the app)"
			}
			if res.AppCreated {
				res.Steps = append(res.Steps, "Install the app on "+repo+": open "+res.AppInstall+", choose Only select repositories, pick "+repo+", and click Install.")
			} else {
				res.Steps = append(res.Steps, "The team's sign-in app is already recorded (client id "+res.ClientID+"); check on GitHub that it is installed on "+repo+".",
					// An earlier setup had the app list the session page's own
					// address as its callback; the sign-in now returns to the
					// relay's own callback alone, so a listed page address is
					// left over and comes off.
					"If the app lists the session page's address, "+coresession.FormerPageCallbackURL(repo)+", as a Callback URL, remove it: on "+settings+
						", under Identifying and authorizing users, delete that Callback URL and click Save changes. An earlier version of setup listed it; the sign-in no longer returns there.")
			}
			if coresession.AllowsLocal(chosen) {
				res.Steps = append(res.Steps, "Switch on the app's device flow, which local sessions sign in with: open "+settings+
					", tick Enable Device Flow under Identifying and authorizing users, and click Save changes. GitHub's manifest cannot tick it for you. "+
					"It is left off unless local sessions are allowed, because anyone holding the app's public client id can start a device sign-in.")
			} else {
				res.Steps = append(res.Steps, "Leave Enable Device Flow off on "+settings+
					": only local sessions sign in with it, and anyone holding the app's public client id could start a device sign-in.")
			}
			if coresession.AllowsCloud(chosen) {
				bp, berr := coresession.WriteBlueprint(coresession.BlueprintRequest{Root: root, Release: release, Running: core.Version, Fetch: fetch,
					Repos: []string{repo}, RelayOrigin: relayOrigin, ClientID: res.ClientID})
				if berr != nil {
					return &exitError{Code: 2, Msg: prefix + "the relay's blueprint was not written: " + berr.Error()}
				}
				res.Blueprint, res.RelayOrigin = &bp, relayOrigin
				if relayOrigin != "" {
					// The cloud's sign-in returns to the relay's own callback,
					// which the app must list: GitHub sends a sign-in only to a
					// callback the app lists. A manifest sets it for an app
					// created now; one recorded before may lack it, and either
					// may have wildcard matching on.
					res.Callback = coresession.RelayCallbackURL(relayOrigin)
					res.Steps = append(res.Steps, "Recorded the relay's address in .abcd/relay/"+coresession.RelaysFile+": commit and push it, and the session page finds the relay itself.",
						"Make sure the team's sign-in returns to the relay: on "+settings+", under Identifying and authorizing users, list "+res.Callback+
							" as a Callback URL, with wildcard matching off, and click Save changes. GitHub sends a sign-in only to a callback the app lists, which ties every sign-in to this relay.")
				} else {
					if bp.DeployButton {
						res.Steps = append(res.Steps, "Commit and push "+bp.BlueprintPath+" and .abcd/relay/, then deploy the relay with Render's Deploy button: "+bp.DeployURL+
							" (a private repository needs Render's own GitHub App installed on it first).")
					} else {
						res.Steps = append(res.Steps, "Commit and push "+bp.BlueprintPath+" and .abcd/relay/, then on Render choose New, then Blueprint, pick this repository and set the Blueprint Path to "+
							bp.BlueprintPath+"; the root render.yaml is your own, so Render's Deploy button would read that one.")
					}
					res.Steps = append(res.Steps, "Render's form asks for "+sessionSecretEnv+": on "+settings+
						", under Client secrets, click Generate a new client secret and paste it straight into that field. abcd never sees it, and GitHub shows it only once.",
						"Once Render shows the relay's address, record it: abcd session setup --relay-origin https://<your relay>.onrender.com",
						"Then list the relay's callback, https://<your relay>.onrender.com"+coresession.RelayCallbackPath+", as a Callback URL on "+settings+
							", under Identifying and authorizing users, with wildcard matching off; setup with --relay-origin names it exactly. Until it is listed, GitHub sign-in on the relay does not work.")
				}
			}
			if coresession.AllowsLocal(chosen) {
				res.Steps = append(res.Steps, "Start a session on your own computer with abcd session start.")
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd session setup: live brief sessions for %s may run %s\n", termsafe.Sanitize(repo), sessionWhereWords(chosen))
				if res.Recorded != "" {
					fmt.Fprintf(w, "  recorded %s in .abcd/config.json\n", termsafe.Sanitize(res.Recorded))
				}
				if res.AppCreated {
					fmt.Fprintf(w, "  created the team's sign-in app; kept only its public client id, %s, in .abcd/relay/%s\n", termsafe.Sanitize(res.ClientID), coresession.AppFile)
				}
				if res.Blueprint != nil {
					for _, f := range res.Blueprint.Files {
						fmt.Fprintf(w, "  %s %s\n", f.State, f.Path)
					}
				}
				fmt.Fprintln(w, "  what is left, in order:")
				for i, step := range res.Steps {
					fmt.Fprintf(w, "    %d. %s\n", i+1, termsafe.Sanitize(step))
				}
			})
		},
	}
	c.Flags().StringVar(&where, "where", "", "where sessions may run: local, cloud or both (default: the answer recorded in .abcd/config.json)")
	c.Flags().StringVar(&relayOrigin, "relay-origin", "", "the deployed relay's https origin, once Render shows it, as https://host[:port]; recorded in .abcd/relay/relays.json")
	c.Flags().StringVar(&release, "release", "", "the abcd release the cloud relay pins, as vMAJOR.MINOR.PATCH (default: this abcd's own version, when it is a release)")
	return c
}

// onceFetch reads each release asset once: session setup checks the
// release's checksums before creating the app, and the blueprint pins the
// same bytes afterwards.
type onceFetch struct {
	inner coresession.AssetFetcher
	got   map[string]onceAsset
}

type onceAsset struct {
	data  []byte
	url   string
	found bool
	err   error
}

func (f *onceFetch) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	key := tag + "/" + name
	if a, ok := f.got[key]; ok {
		return a.data, a.url, a.found, a.err
	}
	data, url, found, err := f.inner.FetchReleaseAsset(tag, name)
	if f.got == nil {
		f.got = map[string]onceAsset{}
	}
	f.got[key] = onceAsset{data, url, found, err}
	return data, url, found, err
}
