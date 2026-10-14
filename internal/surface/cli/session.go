package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

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

// The environment variables the relay reads the team's GitHub sign-in app
// from. The secret is read by name and never printed; the client id is public.
const (
	sessionSecretEnv   = "ABCD_SESSION_GITHUB_CLIENT_SECRET"
	sessionClientIDEnv = "ABCD_SESSION_GITHUB_CLIENT_ID"
	sessionAppTypeEnv  = coresession.AppTypeEnv
)

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
		Long: "Run the relay a team's live brief session goes through, write the blueprint\n" +
			"that deploys it to the team's own Render account, or write the session's static\n" +
			"page for the team to serve. The relay holds each session's\n" +
			"edits in memory only, as opaque bytes, and drops them when the host closes the\n" +
			"session or nobody has been connected for half an hour; it writes nothing to disk\n" +
			"and logs nothing. abcd runs no relay of its own: the team runs it, on the host's\n" +
			"machine reached over Tailscale Serve, or on the team's own relay.\n\n" +
			"Bare `abcd session` prints this help and writes nothing.",
		Args: cobra.NoArgs,
		RunE: helpRunE,
	}
	cmd.AddCommand(newSessionServeCommand(asJSON), newSessionBlueprintCommand(asJSON), newSessionPageCommand(asJSON))
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
			"It serves the session WebSocket at /session, the sign-in exchange at /auth/token,\n" +
			"the sign-in app's public client id at /config and a health check at /healthz. It\n" +
			"admits a browser only from https://abcdesign.app or a loopback page, or, when\n" +
			"--origin is given, from exactly the origins it names.\n" +
			"The team's GitHub sign-in app secret is read from " + sessionSecretEnv + ", its\n" +
			"kind from " + sessionAppTypeEnv + " (github-app, the default, or oauth-app)\n" +
			"and its client id from --client-id or " + sessionClientIDEnv + "; without\n" +
			"them, sign-in is not offered and a team member pastes a personal access token.\n\n" +
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
			appType := strings.TrimSpace(os.Getenv(sessionAppTypeEnv))
			if appType == "" {
				appType = session.AppTypeGitHubApp
			}
			if appType != session.AppTypeGitHubApp && appType != session.AppTypeOAuthApp {
				return &exitError{Code: 2, Msg: prefix + sessionAppTypeEnv + " is " + strconv.Quote(appType) + "; give " + session.AppTypeGitHubApp + " or " + session.AppTypeOAuthApp + " (nothing started)"}
			}
			secret := os.Getenv(sessionSecretEnv)
			// On loopback alone, the relay is the host's own machine behind
			// Tailscale Serve, and a visitor is counted by the tailnet address
			// Serve forwards.
			srv, err := session.New(session.Config{
				Origins:      origins,
				BehindProxy:  behindProxy,
				LoopbackOnly: loopback,
				Repos:        allowRepos,
				Owners:       allowOwners,
				ClientID:     clientID,
				ClientSecret: secret,
				AppType:      appType,
			})
			secret = ""
			_ = secret
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + "--origin " + err.Error() + " (nothing started)"}
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
					fmt.Fprintln(w, "  sign-in: the team's GitHub sign-in app is set up")
				} else {
					fmt.Fprintf(w, "  sign-in is not set up (set %s and --client-id); a team member pastes a personal access token instead\n", sessionSecretEnv)
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
	c.Flags().StringArrayVar(&origins, "origin", nil, "a page origin to admit, as scheme://host[:port] (repeatable); given, it replaces the default of https://abcdesign.app and loopback pages")
	c.Flags().StringArrayVar(&repos, "repo", nil, "a repository to serve sessions for, as owner/name (repeatable; also "+sessionReposEnv+")")
	c.Flags().StringArrayVar(&owners, "owner", nil, "a GitHub user or organisation whose every repository to serve sessions for (repeatable)")
	c.Flags().StringVar(&clientID, "client-id", "", "the team's GitHub sign-in app's public client id (default: "+sessionClientIDEnv+")")
	return c
}

func newSessionBlueprintCommand(asJSON *bool) *cobra.Command {
	var release, relayOrigin string
	c := &cobra.Command{
		Use: "blueprint",
		Long: "Write the team's own relay deployment into this checkout: .abcd/relay/render.yaml,\n" +
			"a Render Blueprint declaring one free web service, and .abcd/relay/Dockerfile,\n" +
			"which downloads the released abcd binary for linux/amd64 and checks it against\n" +
			"the SHA-256 that release's checksums.txt publishes. The checksum is read from the\n" +
			"release page when the files are written (the one network read this verb makes,\n" +
			"named on stderr) and pinned in the Dockerfile. The relay serves sessions for this\n" +
			"checkout's origin repository on github.com, named in render.yaml as\n" +
			sessionReposEnv + ", a plain value; it refuses a checkout without one.\n\n" +
			"--relay-origin names the deployed relay's https origin, once Render has given it\n" +
			"one; the verb then also writes .abcd/relay/relays.json, naming it as the relay\n" +
			"this repository trusts, which the session page reads from the repository.\n\n" +
			"--release names the release to pin; it defaults to this abcd's own version when\n" +
			"that is a release. A file left exactly as abcd wrote it is replaced; one that was\n" +
			"edited, or that abcd did not write, is never overwritten, and nothing is written.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			fetch, err := newSessionReleaseFetcher()
			if err != nil {
				return fmt.Errorf("%s%w", prefix, err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%sreading release %s's checksums.txt from %s\n", prefix, termsafe.Sanitize(release), sessionReleaseOrigin)
			res, err := coresession.WriteBlueprint(coresession.BlueprintRequest{Root: root, Release: release, Fetch: fetch, Repos: []string{served}, RelayOrigin: relayOrigin})
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
				fmt.Fprintf(w, "  from: %s\n", termsafe.Sanitize(res.Source))
				fmt.Fprintln(w, "  next: commit both files; on Render choose New, then Blueprint, pick this repository and set the Blueprint path to .abcd/relay/render.yaml;")
				fmt.Fprintf(w, "        then set %s and %s in the service's Environment\n", sessionClientIDEnv, sessionSecretEnv)
			})
		},
	}
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
			"/<owner>/<repo>/brief/ reaches the page. The page connects only to the relay the\n" +
			"person names and to api.github.com with their own token, and it sends the\n" +
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
