package cli

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/text/width"

	"github.com/intentdriven/abcd/internal/core"
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
	for _, l := range bannerTaglineLines() {
		if strings.Contains(s, l) {
			t.Fatalf("banner leaked onto a non-TTY stream")
		}
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

	tagline := strings.Join(bannerTaglineLines(), "\n")
	if !strings.Contains(withBanner, tagline) {
		t.Fatalf("banner missing from TTY output")
	}
	if !strings.HasSuffix(withBanner, withoutBanner) {
		t.Fatalf("board bytes changed under the banner:\nwith:    %q\nwithout: %q", withBanner, withoutBanner)
	}
	if !strings.Contains(strings.TrimSuffix(withBanner, withoutBanner), tagline) {
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
	if s := out.String(); strings.Contains(s, "\x1b") || strings.Contains(s, bannerTaglineLines()[0]) {
		t.Fatalf("--json output decorated: %q", bannerFirstLine(s))
	}
}

// TestBannerLinesRungs pins the composition per rung.
func TestBannerLinesRungs(t *testing.T) {
	tagline := bannerTaglineLines()
	nt := len(tagline)
	colour := bannerLines(term.TrueColor, true)
	if len(colour) != 4+nt {
		t.Fatalf("colour banner: %d lines, want %d (3 strip + %d tagline + hints)", len(colour), 4+nt, nt)
	}
	if !strings.Contains(colour[0], "▀") || !strings.Contains(colour[0], "abcd") {
		t.Errorf("colour banner top line lacks strip or name: %q", colour[0])
	}
	if got := strings.Join(colour[3:3+nt], " "); got != bakedTagline {
		t.Errorf("tagline lines rejoin as %q", got)
	}

	mono := bannerLines(term.Mono, true)
	if len(mono) != 7+nt {
		t.Fatalf("mono banner: %d lines, want %d (5 shade rows + name + %d tagline + hints)", len(mono), 7+nt, nt)
	}
	for i, l := range mono {
		if strings.Contains(l, "\x1b") {
			t.Errorf("mono line %d carries an escape: %q", i, l)
		}
	}

	text := bannerLines(term.TrueColor, false)
	if len(text) != 2+nt {
		t.Fatalf("non-UTF-8 banner: %d lines, want %d", len(text), 2+nt)
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

// sgrEscape matches the SGR sequences the banner path emits; they occupy no
// terminal column.
var sgrEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// terminalColumns is the test's own display-width oracle, independent of the
// production wrap: escapes stripped, East Asian wide and fullwidth runes
// counted as two columns, every other rune as one.
func terminalColumns(s string) int {
	n := 0
	for _, r := range sgrEscape.ReplaceAllString(s, "") {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			n += 2
		default:
			n++
		}
	}
	return n
}

// TestBannerFitsSixtySixColumns is AC1's width bound (itd-112, spc-2609230613206687):
// every banner line, on every rung and for both name segments, renders
// within 66 terminal columns — counted in display columns, not bytes.
func TestBannerFitsSixtySixColumns(t *testing.T) {
	orig := core.Version
	t.Cleanup(func() { core.Version = orig })
	for _, version := range []string{"dev", "v10.10.10"} {
		core.Version = version
		for _, mode := range []term.ColorMode{term.TrueColor, term.Ansi256, term.Ansi16, term.Mono} {
			for _, utf8OK := range []bool{true, false} {
				for i, line := range bannerLines(mode, utf8OK) {
					if w := terminalColumns(line); w > 66 {
						t.Errorf("version %q, mode %d, utf8 %v: line %d is %d columns (> 66): %q",
							version, mode, utf8OK, i, w, sgrEscape.ReplaceAllString(line, ""))
					}
				}
			}
		}
	}
}

// TestWrapWords pins the render-time wrap: words are never broken or
// reordered, no line exceeds the limit unless it is one overlong word, and
// the wrap is balanced rather than stranding a last word.
func TestWrapWords(t *testing.T) {
	cases := []struct {
		in    string
		limit int
		want  []string
	}{
		{"one two three", 66, []string{"one two three"}},
		// Greedy at 14 strands "four"; balanced keeps two lines, evened.
		{"one two three four", 14, []string{"one two", "three four"}},
		{"alpha supercalifragilistic beta", 8, []string{"alpha", "supercalifragilistic", "beta"}},
		{"", 66, nil},
	}
	for _, c := range cases {
		got := wrapWords(c.in, c.limit)
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("wrapWords(%q, %d) = %q, want %q", c.in, c.limit, got, c.want)
		}
	}
	// The baked tagline: every word kept, in order; within the bound; and no
	// line under half the longest, so the break is balanced.
	lines := bannerTaglineLines()
	if got := strings.Join(lines, " "); got != bakedTagline {
		t.Errorf("wrapped tagline rejoins as %q, want %q", got, bakedTagline)
	}
	longest := 0
	for _, l := range lines {
		longest = max(longest, terminalColumns(l))
	}
	for _, l := range lines {
		if w := terminalColumns(l); w > bannerWidth || 2*w < longest {
			t.Errorf("tagline line %q is %d columns (bound %d, longest %d)", l, w, bannerWidth, longest)
		}
	}
}

// TestBannerNeverOnASubcommandOrHook is AC2's other half: with the seam
// forced on — as if every stream were an interactive TTY — a subcommand and a
// hook invocation still carry no banner byte, because only the bare root
// invocation composes one.
func TestBannerNeverOnASubcommandOrHook(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("HOME", t.TempDir())
	prev := bannerTTY
	bannerTTY = func(io.Writer) bool { return true }
	defer func() { bannerTTY = prev }()

	for _, args := range [][]string{{"version"}, {"hook", "prompt-router"}} {
		root := NewRootCommand()
		var out, errOut bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errOut)
		root.SetIn(strings.NewReader(`{"prompt":"hello"}`))
		root.SetArgs(args)
		_ = root.Execute() // the outcome is irrelevant; only the bytes are asserted
		for _, s := range []string{out.String(), errOut.String()} {
			if strings.Contains(s, bannerTaglineLines()[0]) || strings.Contains(s, "▀") || strings.Contains(s, "░") {
				t.Errorf("%v carries banner bytes: %q", args, s)
			}
		}
	}
}
