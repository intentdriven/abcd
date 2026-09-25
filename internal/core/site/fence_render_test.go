package site

import (
	"errors"
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
