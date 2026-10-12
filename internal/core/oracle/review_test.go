package oracle

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// The reviewer route of itd-6 (spc-2609211950427074 scope 1): oracle.review
// read through the layered resolver, host by default and rp selecting
// RepoPrompt over MCP, and oracle.rp, the server the route reaches, read from
// the machine's file alone.

func (f *fx) loadReview() ReviewRoute {
	f.t.Helper()
	rv, err := LoadReview(f.roots)
	if err != nil {
		f.t.Fatalf("LoadReview: %v", err)
	}
	return rv
}

func (f *fx) loadReviewErr() error {
	f.t.Helper()
	_, err := LoadReview(f.roots)
	if err == nil {
		f.t.Fatal("LoadReview succeeded; want a refusal")
	}
	return err
}

// TestTheReviewRouteIsTheHostUnlessConfigured is criterion 3's configuration
// half: with nothing set, or only the keys ahoy writes, the route is the
// host's, from the bundled default, and names no server.
func TestTheReviewRouteIsTheHostUnlessConfigured(t *testing.T) {
	f := newFx(t)
	if rv := f.loadReview(); rv.Route != ReviewHost || rv.Layer != layered.Bundled || rv.Origin != "bundled" || rv.RP != nil {
		t.Fatalf("no configuration is the host's route: %+v", rv)
	}
	f.repoConfig(`{"oracle":{"backend":"host-delegated"}}`)
	f.machineConfig(`{"oracle":{"api":{}}}`)
	if rv := f.loadReview(); rv.Route != ReviewHost || rv.Layer != layered.Bundled || rv.RP != nil {
		t.Fatalf("the keys other readers own change nothing: %+v", rv)
	}
}

// TestOracleReviewIsReadFromTheRepositoryOrTheMachine is criterion 1's
// configuration half: rp in either file selects the adapter, the repository's
// file wins a key both set, and the server comes from the machine's file.
func TestOracleReviewIsReadFromTheRepositoryOrTheMachine(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"review":"rp","rp":{"url":"http://127.0.0.1:7777/mcp"}}}`)
	rv := f.loadReview()
	if rv.Route != ReviewRP || rv.Layer != layered.Machine || rv.Origin != layered.Config.MachineOrigin() {
		t.Fatalf("the machine's rp selects the adapter: %+v", rv)
	}
	if rv.RP == nil || rv.RP.URL != "http://127.0.0.1:7777/mcp" || rv.RP.Tool != repoprompt.DefaultTool {
		t.Fatalf("the server is the machine's, with the default tool: %+v", rv.RP)
	}

	g := newFx(t)
	g.repoConfig(`{"oracle":{"backend":"host-delegated","review":"rp"}}`)
	g.machineConfig(`{"oracle":{"rp":{"url":"http://localhost:7777","tool":"chat_send"}}}`)
	rv = g.loadReview()
	if rv.Route != ReviewRP || rv.Layer != layered.Repo || rv.Origin != ".abcd/config.json" {
		t.Fatalf("the repository's rp selects the adapter: %+v", rv)
	}
	if rv.RP == nil || rv.RP.Tool != "chat_send" {
		t.Fatalf("the machine names the tool: %+v", rv.RP)
	}

	h := newFx(t)
	h.repoConfig(`{"oracle":{"review":"host"}}`)
	h.machineConfig(`{"oracle":{"review":"rp","rp":{"url":"http://127.0.0.1:7777/mcp"}}}`)
	if rv := h.loadReview(); rv.Route != ReviewHost || rv.Layer != layered.Repo {
		t.Fatalf("the repository's file wins a key both set: %+v", rv)
	}

	// rp selected with no server on this machine is not a fault: the route
	// names no server, and the loop records each review it hands to the host.
	k := newFx(t)
	k.repoConfig(`{"oracle":{"review":"rp"}}`)
	if rv := k.loadReview(); rv.Route != ReviewRP || rv.RP != nil {
		t.Fatalf("rp with no server here names none: %+v", rv)
	}
}

// TestAReviewRouteFaultIsRefusedNamingTheFileAndTheKey: every fault is loud,
// names where it sits, and never falls through to the default.
func TestAReviewRouteFaultIsRefusedNamingTheFileAndTheKey(t *testing.T) {
	machine := layered.Config.MachineOrigin()
	for _, c := range []struct {
		name, repo, machine string
		want                []string
	}{
		{"an unknown route", `{"oracle":{"review":"repoprompt"}}`, "",
			[]string{".abcd/config.json", "oracle.review", "host", "rp"}},
		{"a route that is not a string", "", `{"oracle":{"review":true}}`,
			[]string{machine, "oracle.review"}},
		{"a server the repository names", `{"oracle":{"rp":{"url":"http://127.0.0.1:7777/mcp"}}}`, "",
			[]string{".abcd/config.json", "oracle.rp", machine}},
		{"a server on another machine", "", `{"oracle":{"rp":{"url":"http://example.com/mcp"}}}`,
			[]string{machine, "oracle.rp.url", "this machine"}},
		{"a server with no url", "", `{"oracle":{"rp":{"tool":"oracle_send"}}}`,
			[]string{machine, "oracle.rp", "url"}},
		{"an unknown key under oracle.rp", "", `{"oracle":{"rp":{"url":"http://127.0.0.1:7777","port":7777}}}`,
			[]string{machine, "oracle.rp.port", "url, tool"}},
		{"a tool that is not a name", "", `{"oracle":{"rp":{"url":"http://127.0.0.1:7777","tool":"two words"}}}`,
			[]string{machine, "oracle.rp.tool"}},
		{"oracle.rp that is not an object", "", `{"oracle":{"rp":"http://127.0.0.1:7777"}}`,
			[]string{machine, "oracle.rp"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFx(t)
			if c.repo != "" {
				f.repoConfig(c.repo)
			}
			if c.machine != "" {
				f.machineConfig(c.machine)
			}
			err := f.loadReviewErr()
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("the refusal names %q: %v", w, err)
				}
			}
			if strings.Contains(err.Error(), "example.com") {
				t.Fatalf("the refusal never quotes the URL, which may carry a credential: %v", err)
			}
		})
	}
}
