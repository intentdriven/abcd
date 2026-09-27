package site

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// The renderer closed an inline code span where the opening run's TEXT first
// recurred rather than on a run of exactly the opening length, and trimmed
// every space from a multi-backtick span (iss-2609262322244502). So a span a
// two-backtick run opened around a three-backtick run was refused and failed
// the whole page, a span of a lone space rendered empty, and a value the block
// escapers judged balanced could be refused. It pairs by termsafe's pairer and
// renders CommonMark's content rules: a line ending is a space, and one space
// comes off each side only when both are there and the content is not all
// spaces.
func TestInlineCodeSpanPairsRunsOfTheSameLength(t *testing.T) {
	for md, want := range map[string]string{
		"``a```b`` rest":      "<p><code>a```b</code> rest</p>",
		"``a```b``":           "<p><code>a```b</code></p>",
		"`` ``` ``":           "<p><code>```</code></p>",
		"` `":                 "<p><code> </code></p>",
		"``  `` x":            "<p><code>  </code> x</p>",
		"`` a `` x":           "<p><code>a</code> x</p>",
		"``  a  `` x":         "<p><code> a </code> x</p>",
		"`` a`` x":            "<p><code> a</code> x</p>",
		"`` `a` ``":           "<p><code>`a`</code></p>",
		"a `b\nc` d":          "<p>a <code>b c</code> d</p>",
		"a ``b\n`` d":         "<p>a <code>b </code> d</p>",
		"`a` and ``b`c`` end": "<p><code>a</code> and <code>b`c</code> end</p>",
	} {
		got, err := testRenderer().RenderBlocks("docs/page.md", Blocks(md, 1))
		if err != nil {
			t.Errorf("%q: %v", md, err)
			continue
		}
		if got != want {
			t.Errorf("%q rendered %q, want %q", md, got, want)
		}
	}
	// A run no later run of its own length closes is still refused, the
	// renderer's own policy: a longer or a shorter run does not close it.
	for _, md := range []string{"`a`` b", "``a` b", "``a``` b"} {
		_, err := testRenderer().RenderBlocks("docs/page.md", Blocks(md, 1))
		var ue *UnsupportedError
		if !errors.As(err, &ue) || !strings.Contains(err.Error(), "unclosed code span") {
			t.Errorf("%q: want an unclosed code span refusal, got %v", md, err)
		}
	}
}

// fence judged a block's last line a closer by a prefix test, while whether a
// line closes a fence is mdrecord's reading. The last line of an unclosed
// fence, opening with three backticks and an info string, was taken for a
// closer and its text dropped from the page. It is a closer only where
// mdrecord's reading of the block closes a fence on it.
func TestFenceLastLineIsACloserOnlyByMdrecord(t *testing.T) {
	for name, tc := range map[string]struct{ md, code string }{
		"an unclosed fence whose last line opens with backticks": {
			"```\ncode\n```js", "code\n```js",
		},
		"an unclosed fence whose last line is a span": {
			"```\ncode\n```x```", "code\n```x```",
		},
		"a closer longer than the opener": {
			"```\ncode\n````", "code",
		},
		"a closer indented within reach": {
			"```\ncode\n   ```", "code",
		},
		// An indented closer lets the walk's block run on to the next fence
		// closed at the margin; the block renders whole, as it always has.
		"a block the walk ran on past an indented closer": {
			"```\na\n  ```\nb\n  ```\n```", "a\n  ```\nb\n  ```",
		},
	} {
		got, err := testRenderer().RenderBlocks("docs/page.md", Blocks(tc.md, 1))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		want := `<div class="cmd"><pre><code>` + escapeText(tc.code) + "\n</code></pre>"
		if !strings.HasPrefix(got, want) || strings.Count(got, `<div class="cmd">`) != 1 {
			t.Errorf("%s: rendered %q, want one command block holding %q", name, got, tc.code)
		}
	}
}

// codeSpanCase is one row of the shared code-span agreement table.
type codeSpanCase struct {
	In       string `json:"in"`
	Balanced bool   `json:"balanced"`
	Code     string `json:"code"`
}

// loadCodeSpanAgreement reads the table every block escaper and this renderer
// are held to: termsafe's testdata, so the escapers' tests read the same rows.
func loadCodeSpanAgreement(t *testing.T) []codeSpanCase {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "termsafe", "testdata", "codespan_agreement.json"))
	if err != nil {
		t.Fatal(err)
	}
	var table struct{ Cases []codeSpanCase }
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("the code-span agreement table holds no cases")
	}
	return table.Cases
}

// TestCodeSpanAgreementTable holds termsafe.OpensBalancedCodeSpan — the
// predicate lifeboat's and ideate's block escapers leave a value unescaped by —
// to the renderer's own reading of the same value: a leading run it calls
// balanced renders as a leading code span holding the table's text, and one it
// calls unbalanced renders none (iss-2609262322244502). The escapers' own tests
// read the same table.
func TestCodeSpanAgreementTable(t *testing.T) {
	for _, c := range loadCodeSpanAgreement(t) {
		if got := termsafe.OpensBalancedCodeSpan(c.In); got != c.Balanced {
			t.Errorf("OpensBalancedCodeSpan(%q) = %v, want %v", c.In, got, c.Balanced)
		}
		html, err := testRenderer().RenderBlocks("docs/page.md", Blocks(c.In, 1))
		opens := err == nil && strings.HasPrefix(html, "<p><code>")
		if opens != c.Balanced {
			t.Errorf("%q: the renderer opens a leading span = %v (err %v, html %q); the table says balanced = %v", c.In, opens, err, html, c.Balanced)
			continue
		}
		if c.Balanced && !strings.HasPrefix(html, "<p><code>"+escapeText(c.Code)+"</code>") {
			t.Errorf("%q rendered %q, want a leading span holding %q", c.In, html, c.Code)
		}
	}
}
