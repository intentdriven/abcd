package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/term"
)

func colourEnv(k string) string {
	switch k {
	case "TERM":
		return "xterm-256color"
	case "COLORTERM":
		return "truecolor"
	case "LANG":
		return "en_GB.UTF-8"
	}
	return ""
}

// TestBakedIdentityInSync is the identity drift gate: the generated
// constants must match the canonical identity block. Regenerate with
// `go generate ./internal/surface/cli`.
func TestBakedIdentityInSync(t *testing.T) {
	const root = "../../.."
	cfg, _, err := positioning.LoadConfig(root)
	if err != nil {
		t.Fatalf("loading positioning config: %v", err)
	}
	block, err := positioning.ParseBlock(root, cfg.Block)
	if err != nil {
		t.Fatalf("parsing identity block: %v", err)
	}
	if bakedTitle != block.Title {
		t.Errorf("bakedTitle drifted from the identity block: %q vs %q — run `go generate ./internal/surface/cli`", bakedTitle, block.Title)
	}
	if bakedTagline != block.Tagline {
		t.Errorf("bakedTagline drifted from the identity block: %q vs %q — run `go generate ./internal/surface/cli`", bakedTagline, block.Tagline)
	}
	// The byte gate: the committed file must be exactly what the generator
	// emits — a hand edit that keeps the values but changes the form (an
	// extra const, an init, a reworded header) fails here, not on the next
	// unrelated go generate run.
	committed, err := os.ReadFile("identity_gen.go")
	if err != nil {
		t.Fatalf("reading identity_gen.go: %v", err)
	}
	if want := IdentityGenSource(block.Title, block.Tagline); !bytes.Equal(committed, want) {
		t.Errorf("identity_gen.go differs from the generator's output — run `go generate ./internal/surface/cli`")
	}
}

// TestBareInvocationMachineStreamClean: with the default seam (a buffer is
// not a TTY) the bare invocation emits no banner and no escape byte, and the
// status board renders as it always has (adr-49's machine-stream assertion).
func TestBareInvocationMachineStreamClean(t *testing.T) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)
	root.SetArgs([]string{})
	if err := root.Execute(); err != nil {
		t.Fatalf("bare invocation: %v", err)
	}
	s := out.String()
	if strings.Contains(s, "\x1b") {
		t.Fatalf("machine stream carries an escape byte: %q", s)
	}
	if strings.Contains(s, bakedTagline) {
		t.Fatalf("banner leaked onto a non-TTY stream")
	}
	if !strings.HasPrefix(s, "abcd — ") {
		t.Fatalf("status board changed shape: %q", bannerFirstLine(s))
	}
}

func bannerFirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestBannerRendersAboveBoard: with the seam forced on, the banner tops the
// board — and the board's bytes are exactly the seam-off output (the AC1
// byte-level guard: the banner may only ever prepend).
func TestBannerRendersAboveBoard(t *testing.T) {
	// Render in a directory of the test's own. The board reads the sibling
	// worktrees of the checkout it runs in, and a peer session creating one
	// between the two renders below would change the board's bytes.
	t.Chdir(t.TempDir())
	bare := func() string {
		root := NewRootCommand()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(io.Discard)
		root.SetArgs([]string{})
		if err := root.Execute(); err != nil {
			t.Fatalf("bare invocation: %v", err)
		}
		return out.String()
	}
	withoutBanner := bare()

	prev := bannerTTY
	bannerTTY = func(io.Writer) bool { return true }
	defer func() { bannerTTY = prev }()
	withBanner := bare()

	if !strings.Contains(withBanner, bakedTagline) {
		t.Fatalf("banner missing from TTY output")
	}
	if !strings.HasSuffix(withBanner, withoutBanner) {
		t.Fatalf("board bytes changed under the banner:\nwith:    %q\nwithout: %q", withBanner, withoutBanner)
	}
	if !strings.Contains(strings.TrimSuffix(withBanner, withoutBanner), bakedTagline) {
		t.Fatalf("banner does not precede the board")
	}
}

// TestBannerJSONNeverDecorated: --json wins over a TTY.
func TestBannerJSONNeverDecorated(t *testing.T) {
	prev := bannerTTY
	bannerTTY = func(io.Writer) bool { return true }
	defer func() { bannerTTY = prev }()

	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--json invocation: %v", err)
	}
	if s := out.String(); strings.Contains(s, "\x1b") || strings.Contains(s, bakedTagline) {
		t.Fatalf("--json output decorated: %q", bannerFirstLine(s))
	}
}

// TestBannerLinesRungs pins the composition per rung.
func TestBannerLinesRungs(t *testing.T) {
	colour := bannerLines(term.TrueColor, true)
	if len(colour) != 5 {
		t.Fatalf("colour banner: %d lines, want 5 (3 strip + tagline + hints)", len(colour))
	}
	if !strings.Contains(colour[0], "▀") || !strings.Contains(colour[0], "abcd") {
		t.Errorf("colour banner top line lacks strip or name: %q", colour[0])
	}
	if colour[3] != bakedTagline {
		t.Errorf("tagline line is %q", colour[3])
	}

	mono := bannerLines(term.Mono, true)
	if len(mono) != 8 {
		t.Fatalf("mono banner: %d lines, want 8 (5 shade rows + 3 text)", len(mono))
	}
	for i, l := range mono {
		if strings.Contains(l, "\x1b") {
			t.Errorf("mono line %d carries an escape: %q", i, l)
		}
	}

	text := bannerLines(term.TrueColor, false)
	if len(text) != 3 {
		t.Fatalf("non-UTF-8 banner: %d lines, want 3", len(text))
	}
	for i, l := range text {
		if strings.Contains(l, "\x1b") || strings.Contains(l, "▀") || strings.Contains(l, "░") {
			t.Errorf("non-UTF-8 line %d carries art or escapes: %q", i, l)
		}
	}
	if text[0] != "abcd (dev build)" {
		t.Errorf("dev build name segment: %q", text[0])
	}
}

// TestWriteBannerHonoursNoColor: the flag forces mono through the wired path.
func TestWriteBannerHonoursNoColor(t *testing.T) {
	var out bytes.Buffer
	writeBanner(&out, true, colourEnv)
	if s := out.String(); strings.Contains(s, "\x1b") {
		t.Fatalf("--no-color output carries an escape byte")
	}
	var coloured bytes.Buffer
	writeBanner(&coloured, false, colourEnv)
	if !strings.Contains(coloured.String(), "\x1b[38;2;") {
		t.Fatalf("truecolor env did not produce a truecolor banner")
	}
}
