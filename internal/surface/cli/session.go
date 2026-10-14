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
)

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
	URL      string `json:"url"`
	Loopback bool   `json:"loopback"`
	Proxy    bool   `json:"behind_proxy"`
	SignIn   bool   `json:"sign_in"`
	Hint     string `json:"hint"`
}

func newSessionServeCommand(asJSON *bool) *cobra.Command {
	var listen, clientID string
	var origins []string
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
			"It serves the session WebSocket at /session, the sign-in exchange at /auth/token,\n" +
			"the sign-in app's public client id at /config and a health check at /healthz. It\n" +
			"admits a browser only from https://abcdesign.app, a loopback page or an --origin.\n" +
			"The team's GitHub sign-in app secret is read from " + sessionSecretEnv + "\n" +
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
			if clientID == "" {
				clientID = strings.TrimSpace(os.Getenv(sessionClientIDEnv))
			}
			secret := os.Getenv(sessionSecretEnv)
			// On loopback alone, the relay is the host's own machine behind
			// Tailscale Serve, and a visitor is counted by the tailnet address
			// Serve forwards.
			srv, err := session.New(session.Config{
				Origins:      origins,
				BehindProxy:  behindProxy,
				LoopbackOnly: addr.Addr().IsLoopback(),
				ClientID:     clientID,
				ClientSecret: secret,
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
			loopback := addr.Addr().IsLoopback()
			start := sessionStart{
				URL:      "http://" + bound,
				Loopback: loopback,
				Proxy:    behindProxy,
				SignIn:   clientID != "" && os.Getenv(sessionSecretEnv) != "",
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
	c.Flags().StringArrayVar(&origins, "origin", nil, "another page origin to admit, as scheme://host[:port] (repeatable); https://abcdesign.app and loopback are always admitted")
	c.Flags().StringVar(&clientID, "client-id", "", "the team's GitHub sign-in app's public client id (default: "+sessionClientIDEnv+")")
	return c
}

func newSessionBlueprintCommand(asJSON *bool) *cobra.Command {
	var release string
	c := &cobra.Command{
		Use: "blueprint",
		Long: "Write the team's own relay deployment into this checkout: .abcd/relay/render.yaml,\n" +
			"a Render Blueprint declaring one free web service, and .abcd/relay/Dockerfile,\n" +
			"which downloads the released abcd binary for linux/amd64 and checks it against\n" +
			"the SHA-256 that release's checksums.txt publishes. The checksum is read from the\n" +
			"release page when the files are written (the one network read this verb makes,\n" +
			"named on stderr) and pinned in the Dockerfile.\n\n" +
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
			fetch, err := newSessionReleaseFetcher()
			if err != nil {
				return fmt.Errorf("%s%w", prefix, err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%sreading release %s's checksums.txt from %s\n", prefix, termsafe.Sanitize(release), sessionReleaseOrigin)
			res, err := coresession.WriteBlueprint(coresession.BlueprintRequest{Root: root, Release: release, Fetch: fetch})
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
				fmt.Fprintf(w, "  from: %s\n", termsafe.Sanitize(res.Source))
				fmt.Fprintln(w, "  next: commit both files; on Render choose New, then Blueprint, pick this repository and set the Blueprint path to .abcd/relay/render.yaml;")
				fmt.Fprintf(w, "        then set %s and %s in the service's Environment\n", sessionClientIDEnv, sessionSecretEnv)
			})
		},
	}
	c.Flags().StringVar(&release, "release", "", "the abcd release tag to pin, as vMAJOR.MINOR.PATCH (default: this abcd's own version, when it is a release)")
	return c
}

func newSessionPageCommand(asJSON *bool) *cobra.Command {
	var out string
	c := &cobra.Command{
		Use: "page",
		Long: "Write the brief session's page into --out: index.html and an identical 404.html,\n" +
			"session.js, session.css, and the vendored editor bundle with its licences under\n" +
			"vendor/. The files are plain and static, embedded in this binary; serve them\n" +
			"from any static host, such as GitHub Pages, whose 404.html fallback is how\n" +
			"/<owner>/<repo>/brief/ reaches the page. The page connects only to the relay the\n" +
			"person names, to api.github.com with their own token, and to github.com for\n" +
			"sign-in. It reads nothing from this checkout.\n\n" +
			"It writes those six files and nothing else, and removes nothing. --out may be\n" +
			"absent, empty, or a directory an earlier run wrote, whose files are replaced; a\n" +
			"directory holding anything else, a symlink or a file is refused.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd session page: "
			res, err := web.Write(out)
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
				fmt.Fprintln(w, "  next: serve the directory from a static host; the page opens at /<owner>/<repo>/brief/")
			})
		},
	}
	c.Flags().StringVar(&out, "out", "", "the directory to write the page into: absent, empty, or one an earlier run wrote")
	return c
}
