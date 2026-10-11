package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// staleusage_pageverb_test.go — a command page that documents no binary verb
// never makes an up-to-date binary call itself stale (iss-2609200953255336,
// iss-2609240519471816).
//
// commands/abcd.md documents the bare dispatcher, and the host-delegated pages
// (consult, ingest, prepare-this-repo) document workflows the host agent runs
// with no Go verb at all. The stale-usage note reads "a page exists for the
// token" as "the surface is newer than this binary", which is false for every
// one of them: rebuilding or updating adds no verb, because none was ever
// meant to exist. The refusal names what the token actually is instead.

func TestDispatcherPageTokenIsNotAStaleVerb(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "abcd", "# `/abcd` where-am-i\n\n```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" --json\n```\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	code, stdout, stderr := runMain(t, "abcd", "iss-1")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Fatalf("stdout must stay empty, got %q", stdout)
	}
	if strings.Contains(stderr, "predates") || strings.Contains(stderr, "stale") {
		t.Fatalf("a dispatcher-page token must not be called a stale verb:\n%s", stderr)
	}
	if !strings.Contains(stderr, "abcd <record-id>") {
		t.Fatalf("the refusal must name the form that works, `abcd <record-id>`:\n%s", stderr)
	}
}

func TestHostDelegatedPageTokenIsNotAStaleVerb(t *testing.T) {
	for verb := range pagesWithNoVerb {
		if verb == dispatcherPage {
			continue
		}
		t.Run(verb, func(t *testing.T) {
			root := stalePluginRoot(t)
			writeCommandPage(t, root, verb, "# `/abcd:"+verb+"`\n\nRuns in the host agent.\n")
			setExecutable(t, filepath.Join(root, "abcd"))

			code, _, stderr := runMain(t, verb)
			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if strings.Contains(stderr, "predates") || strings.Contains(stderr, "make build") || strings.Contains(stderr, "abcd update") {
				t.Fatalf("a host-delegated page must not send the reader to rebuild or update:\n%s", stderr)
			}
			if !strings.Contains(stderr, "/abcd:"+verb) {
				t.Fatalf("the refusal must name the host invocation /abcd:%s:\n%s", verb, stderr)
			}
		})
	}
}

// A page that does document a verb the binary lacks still earns the stale
// note: the exemption is the named set, not every page.
func TestPageWithNoVerbSetLeavesRealStalenessAlone(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "frobnicate", "```bash\nabcd frobnicate\n```\n")
	setExecutable(t, filepath.Join(root, "abcd"))
	_, _, stderr := runMain(t, "frobnicate")
	if !strings.Contains(stderr, "predates the `frobnicate` command") {
		t.Fatalf("a documented verb the binary lacks must still be named stale:\n%s", stderr)
	}
}

// Every entry names a token the binary really lacks: a verb that later gains a
// Go implementation must leave the set, or its own unknown-command path (a
// genuinely stale binary) would be answered with "no binary verb".
// The same holds for a retired token: one the binary registers again is not
// retired.
func TestPagesWithNoVerbNameNoRegisteredVerb(t *testing.T) {
	root := NewRootCommand()
	for set, tokens := range map[string]map[string]string{"pagesWithNoVerb": pagesWithNoVerb, "retiredTokens": retiredTokens} {
		for verb := range tokens {
			for _, c := range root.Commands() {
				if c.Name() == verb || c.HasAlias(verb) {
					t.Errorf("%s names %q, which the binary registers as a verb", set, verb)
				}
			}
		}
	}
}

// TestRetiredVersionTokenNamesWhatReplacedIt is the version half of
// spc-2610100613109045's old-names rule: with the version page retired,
// `abcd version` is still an unknown command (no alias, adr-40), and its note
// names what answers instead (the root flag and the board) rather than the
// retired page, and never calls the binary stale for not knowing it.
func TestRetiredVersionTokenNamesWhatReplacedIt(t *testing.T) {
	if _, ok := pagesWithNoVerb["version"]; ok {
		t.Fatalf("pagesWithNoVerb still names version, whose page is retired")
	}
	root := stalePluginRoot(t)
	setExecutable(t, filepath.Join(root, "abcd"))
	code, stdout, stderr := runMain(t, "version")
	if code != 2 || stdout != "" {
		t.Fatalf("`abcd version` exit = %d, stdout %q; want exit 2 and an empty stdout", code, stdout)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("`abcd version` is not refused as an unknown command:\n%s", stderr)
	}
	for _, want := range []string{"abcd --version", "board"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, stderr)
		}
	}
	for _, not := range []string{"/abcd:version", "predates", "make build", "abcd update"} {
		if strings.Contains(stderr, not) {
			t.Errorf("the refusal names %q:\n%s", not, stderr)
		}
	}
}
