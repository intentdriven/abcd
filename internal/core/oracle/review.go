package oracle

// review.go reads the reviewer route (itd-6, spc-2609211950427074 scope 1)
// through the layered resolver the model-tier and pacing intents share:
//
//   - oracle.review names the route a build lane's reviews take: host, the
//     bundled default, or rp, RepoPrompt over MCP. The repository's file or
//     the machine's may set it; the repository's wins, as every layered key's
//     does. No flag sets it in this build.
//   - oracle.rp holds the MCP server the rp route reaches: url (http or https
//     to this machine) and an optional tool (the RepoPrompt tool a review is
//     sent to, repoprompt.DefaultTool when unset). Only the machine's file
//     declares it: the address a review request (the diff and the record) is
//     sent to is the person's own machine's to say, as a provider block's is,
//     so a repository declaring it is refused naming the machine's file.
//
// oracle.rp is read only when the route is rp: on the host's route it is not
// read at all, so a fault in it never refuses a step that does not use it.
//
// The rp route selected with no oracle.rp.url on this machine is not a fault:
// the route reads as rp with no server (RP nil), so a repository that opts its
// reviews in never breaks a collaborator who runs no RepoPrompt. Nothing here
// reaches RepoPrompt, starts it, or looks for it: the configuration is read,
// and checked, and that is all.

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// The reviewer route's keys and values, as every refusal names them.
const (
	ReviewKey = "oracle.review"
	RPKey     = "oracle.rp"
	// ReviewHost keeps a lane's reviews on the host's own agents (the
	// default); ReviewRP sends them to RepoPrompt over MCP.
	ReviewHost = "host"
	ReviewRP   = "rp"
)

var toolNameRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,63}$`)

// ReviewRoute is the reviewer route one invocation read.
type ReviewRoute struct {
	// Route is host or rp, with the layer and the file that set it.
	Route  string
	Layer  layered.Layer
	Origin string
	// RP is the MCP server this machine names under oracle.rp; nil when it
	// names none.
	RP *RPServer
}

// RPServer is the MCP server the rp route reaches.
type RPServer struct {
	URL  string
	Tool string
}

// Client is the adapter that reaches the server. It connects when a review is
// sent, never before.
func (s RPServer) Client() (*repoprompt.Client, error) { return repoprompt.New(s.URL, s.Tool) }

// LoadReview reads and checks the reviewer route. A fault is an error naming
// the file and the key; none falls through to the default.
func LoadReview(r layered.Roots) (ReviewRoute, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %w", err)
	}
	v, err := layered.Get(s, ReviewKey, ReviewHost, func(name string) error {
		if name != ReviewHost && name != ReviewRP {
			return fmt.Errorf("it takes %s (the host's own agents) or %s (RepoPrompt over MCP)", ReviewHost, ReviewRP)
		}
		return nil
	})
	if err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %w", err)
	}
	rv := ReviewRoute{Route: v.V, Layer: v.Layer, Origin: v.Origin}
	if rv.Route != ReviewRP {
		// The server block is the rp route's alone: on the host's route it is
		// not read, so a fault in it never refuses a step that does not use it.
		return rv, nil
	}
	// oracle.rp must be an object before the claim reads it: the claim's
	// refusal quotes a value that is not one, and a URL written there in place
	// of the block can carry a credential no refusal may echo.
	found, err := s.Lookup(RPKey)
	if err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %w", err)
	}
	for _, f := range found {
		if _, err := layered.Decode[map[string]json.RawMessage](f.Raw); err != nil {
			return ReviewRoute{}, fmt.Errorf("review route: %s (%s layer): %s is not an object; it takes url and tool",
				f.Origin, f.Layer, RPKey)
		}
	}
	if err := s.Claim(RPKey, "url", "tool"); err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %w", err)
	}
	for _, f := range found {
		if f.Layer != layered.Machine {
			return ReviewRoute{}, fmt.Errorf("review route: %s (%s layer): %s is set here, but the address a review "+
				"request is sent to is this machine's alone to say; set it in %s and remove it from %s",
				f.Origin, f.Layer, RPKey, layered.Config.MachineOrigin(), f.Origin)
		}
	}
	if len(found) == 0 {
		return rv, nil
	}
	// Read and checked here rather than through Get, whose refusal quotes the
	// value: a URL can carry a credential, and no refusal may echo it.
	urls, err := s.Lookup(RPKey + ".url")
	if err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %w", err)
	}
	if len(urls) == 0 {
		return ReviewRoute{}, fmt.Errorf("review route: %s (machine layer): %s names no url; it takes url "+
			"(the MCP server's address on this machine) and an optional tool", layered.Config.MachineOrigin(), RPKey)
	}
	url, err := layered.Decode[string](urls[0].Raw)
	if err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %s (machine layer): %s.url is not a string; it takes the MCP server's address",
			urls[0].Origin, RPKey)
	}
	if err := repoprompt.CheckURL(url); err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %s (machine layer): %s.url: %v", urls[0].Origin, RPKey, err)
	}
	tool, err := layered.Get(s, RPKey+".tool", repoprompt.DefaultTool, func(t string) error {
		if !toolNameRe.MatchString(t) {
			return errors.New("a tool's name is letters, digits, _, . and -, at most 64")
		}
		return nil
	})
	if err != nil {
		return ReviewRoute{}, fmt.Errorf("review route: %w", err)
	}
	rv.RP = &RPServer{URL: url, Tool: tool.V}
	return rv, nil
}
