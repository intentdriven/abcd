package ahoy

// The OpenAI-compatible API adapter's explanation at ahoy
// (itd-2609081951381895 criterion 6): on a machine with no provider
// configured, an optional gap says what an aggregator is, what abcd would use
// it for, that everything works without it, and where the walkthrough is.
//
// It is advisory and not resolvable by install: the walkthrough stores a key,
// and a key never passes through the install prompter, which echoes every
// answer into its transcript, or through a host's question tool, which would
// put it in an agent's context. The person runs `abcd ahoy connect` with the
// key piped in instead. Declining is not running it, and changes nothing:
// every delegated step runs on the host.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// ProviderAdapterGapID is the explanation, raised while no provider is
	// configured on this machine.
	ProviderAdapterGapID = "oracle_api.none_configured"
	// ProviderAdapterRefusedGapID names a provider configuration the adapter
	// refuses, so it is never silently unused.
	ProviderAdapterRefusedGapID = "oracle_api.config_refused"
	// ProviderAdapterRouteSkippedGapID names each route the configuration
	// read skipped, one diagnostic per line of its Detail, so a skip is never
	// silent at the bare board.
	ProviderAdapterRouteSkippedGapID = "oracle_api.route_skipped"
)

func detectProviderAdapter(cwd string) []Gap {
	roots, _ := layered.RootsFor(cwd)
	cfg, err := oracle.LoadAPI(roots)
	if err != nil {
		return []Gap{{
			ID: ProviderAdapterRefusedGapID, Category: UserState, Scope: "machine",
			Title:    "the provider configuration is refused",
			Detail:   termsafe.Sanitize(fsutil.RedactHome(err.Error())),
			FixHint:  "Fix the named key in the named file; until then no step reaches a provider, and every delegated step runs on the host.",
			Required: false, Resolvable: false,
		}}
	}
	var gaps []Gap
	if len(cfg.Providers()) == 0 {
		gaps = append(gaps, Gap{
			ID: ProviderAdapterGapID, Category: UserState, Scope: "machine",
			Title:  "no OpenAI-compatible provider configured (optional)",
			Detail: oracle.AdapterExplanation,
			FixHint: "`abcd ahoy --providers` walks through the setup and where the key can live; `abcd ahoy connect` sets one up. " +
				"Declining changes nothing: every delegated step runs on the host, the only route.",
			Required: false, Resolvable: false,
		})
	}
	return append(gaps, skippedRoutes(cfg.Diagnostics)...)
}

// skippedRoutes is the gap naming each route the configuration read skipped
// (a role outside the roster, a route to a provider this machine has not
// configured, a repository's route to a provider that holds a key), one
// diagnostic per line of its Detail. It is advisory: the rest of the
// configuration applies, so nothing is required of the person.
func skippedRoutes(diagnostics []string) []Gap {
	if len(diagnostics) == 0 {
		return nil
	}
	lines := make([]string, len(diagnostics))
	for i, d := range diagnostics {
		lines[i] = termsafe.Sanitize(fsutil.RedactHome(d))
	}
	return []Gap{{
		ID: ProviderAdapterRouteSkippedGapID, Category: UserState, Scope: "machine",
		Title:    "a provider route is skipped",
		Detail:   strings.Join(lines, "\n"),
		FixHint:  "Each line names the route, why it is skipped and where to change it; `abcd ahoy --providers` shows the routes in force.",
		Required: false, Resolvable: false,
	}}
}
