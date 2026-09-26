package site

import (
	"errors"
	"strings"
	"testing"
)

// The block walk reads fences by mdrecord's rule, tildes and long runs
// included, while the renderer knew only a three-backtick opener: a `~~~` fence
// arrived as one block and rendered as a PARAGRAPH, delimiters and code inlined
// into prose, with no error. The renderer renders one fence form, so every
// other form is refused loudly rather than rendered as something else.
func TestRenderBlockRefusesAFenceFormItDoesNotRender(t *testing.T) {
	for name, md := range map[string]string{
		"a tilde fence":                   "~~~\ncode\n~~~",
		"a tilde fence with a language":   "~~~sh\nabcd lint\n~~~",
		"a four-backtick fence":           "````md\n```\ninner\n```\n````",
		"a tilde fence under a paragraph": "prose\n~~~\ncode\n~~~",
		"an indented tilde fence":         "  ~~~\n  code\n  ~~~",
	} {
		_, err := testRenderer().RenderBlocks("docs/page.md", Blocks(md, 1))
		var ue *UnsupportedError
		if !errors.As(err, &ue) {
			t.Errorf("%s: rendered without an UnsupportedError (err %v)", name, err)
		}
	}
	// A three-backtick fence quoting a tilde line is still a fence.
	if got := render(t, "```\n~~~\n```"); got == "" {
		t.Errorf("a three-backtick fence quoting a tilde line rendered nothing")
	}
}

// A three-backtick opener indented one to three spaces is a fence in
// CommonMark, and mdrecord's walk reads it as one, but the renderer rendered a
// three-backtick fence only at the left margin or inside a list item it
// dedents: anywhere else the block became a paragraph holding inline code, the
// delimiters and the code inlined into prose with no error. It most often
// arrives as the fence of a loose list item, cut from the item by the blank
// line above it. It renders as a fence, its lines losing up to the opener's
// indent (iss-2609251600023777).
func TestRenderBlockRendersAnIndentedThreeBacktickFence(t *testing.T) {
	for name, tc := range map[string]struct{ md, code string }{
		"one space":                        {" ```\ncode\n ```", "<code>code\n</code>"},
		"two spaces":                       {"  ```\n  code\n    more\n  ```", "<code>code\n  more\n</code>"},
		"three spaces":                     {"   ```\n   code\n   ```", "<code>code\n</code>"},
		"three spaces with a language":     {"   ```sh\n   abcd lint\n   ```", `<code class="language-sh">abcd lint` + "\n</code>"},
		"a body line indented less":        {"   ```\n code\n   ```", "<code>code\n</code>"},
		"a blank line in the body":         {"  ```\n  one\n\n  two\n  ```", "<code>one\n\ntwo\n</code>"},
		"after a blank line under an item": {"- item\n\n  ```\n  code\n  ```", "<code>code\n</code>"},
	} {
		got, err := testRenderer().RenderBlocks("docs/page.md", Blocks(tc.md, 1))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(got, `<div class="cmd"><pre>`+tc.code+`</pre>`) {
			t.Errorf("%s: rendered %s, want the command block holding %q", name, got, tc.code)
		}
		if strings.Contains(got, "<p>") {
			t.Errorf("%s: rendered a paragraph: %s", name, got)
		}
	}
	// Lines after an indented closer run on in the walk's block, and render as
	// the block they are rather than as code.
	got := render(t, "1. item\n\n   ```\n   code\n   ```\n2. next")
	if !strings.Contains(got, "<code>code\n</code>") || !strings.Contains(got, "<li>next</li>") {
		t.Errorf("the item after an indented closer lost its rendering: %s", got)
	}
}

// The tilde twin of an indented fence is refused as the margin tilde fence is,
// and an indented opener under a line of prose is a fence without a blank line
// before it, refused as the margin one is.
func TestRenderBlockRefusesTheIndentedFenceSiblings(t *testing.T) {
	for name, md := range map[string]string{
		"one space, the tilde twin":    " ~~~\ncode\n ~~~",
		"two spaces, the tilde twin":   "  ~~~\ncode\n  ~~~",
		"three spaces, the tilde twin": "   ~~~\ncode\n   ~~~",
		"four backticks, indented":     "  ````\ncode\n  ````",
		"under a paragraph":            "prose\n  ```\ncode\n  ```",
	} {
		_, err := testRenderer().RenderBlocks("docs/page.md", Blocks(md, 1))
		var ue *UnsupportedError
		if !errors.As(err, &ue) {
			t.Errorf("%s: rendered without an UnsupportedError (err %v)", name, err)
		}
	}
}

// A fence at the margin quoting an indented backtick line is still a fence, and
// a fence indented under the list item holding it still renders in the item.
func TestRenderBlockKeepsTheFencesItAlreadyRendered(t *testing.T) {
	for name, md := range map[string]string{
		"a margin fence quoting an indented opener": "```md\n  ```\n```",
		"a fence inside a list item":                "- item\n  ```\n  code\n  ```",
	} {
		if got := render(t, md); !strings.Contains(got, `<div class="cmd">`) {
			t.Errorf("%s lost its command block: %s", name, got)
		}
	}
}
