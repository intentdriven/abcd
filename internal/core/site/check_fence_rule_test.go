package site

import "testing"

// parseCLIReference credits a command's flags from its fenced block, and reads
// that block by mdrecord's rule: a tilde fence is a fence, and a longer fence
// quoting a three-backtick line is one block, not two toggles.
func TestParseCLIReferenceReadsFencesByTheCommonMarkRule(t *testing.T) {
	md := "## `abcd x`\n\n**Flags:**\n\n~~~\n--tilde\n~~~\n\n" +
		"## `abcd y`\n\n**Flags:**\n\n````\n```\n--quoted\n```\n````\n\nProse naming --gone.\n"
	ref := parseCLIReference(md)
	if !ref.flags["abcd x"]["--tilde"] {
		t.Errorf("a flag in a tilde fence was not credited: %v", ref.flags)
	}
	if !ref.flags["abcd y"]["--quoted"] {
		t.Errorf("a flag in a four-backtick fence was not credited: %v", ref.flags)
	}
	if ref.flags["abcd y"]["--gone"] {
		t.Errorf("a flag named in prose was credited: %v", ref.flags)
	}
}
