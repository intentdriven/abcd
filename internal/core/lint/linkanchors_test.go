package lint

import (
	"path/filepath"
	"testing"
)

// links_resolve stripped a link's #fragment before resolving it and skipped a
// same-file #link outright, so no gate validated a heading anchor and ~18
// broken ones sat on a green tree (iss-303). link_anchors checks the fragment
// against the target's ATX heading slugs (GitHub's slugging, duplicates
// suffixed -1, -2, and explicit HTML id/name anchors), landing warn-first
// through its own severity.
func TestLinkAnchorsValidatesFragmentsAgainstHeadingSlugs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/b.md", "# Title\n\n## Two Words, `code` & more\n\n## Repeat\n\n## Repeat\n\n"+
		"<a id=\"custom-anchor\"></a>\n\n```\n## Fenced Heading\n```\n")
	writeFile(t, root, "rec/a.md", "# A\n\n## Local Head\n\n"+
		"[ok](b.md#two-words-code--more)\n"+ // line 5
		"[dup](b.md#repeat-1)\n"+ // 6
		"[html](b.md#custom-anchor)\n"+ // 7
		"[local](#local-head)\n"+ // 8
		"[bad](b.md#missing)\n"+ // 9
		"[fenced](b.md#fenced-heading)\n"+ // 10
		"[badlocal](#nowhere)\n"+ // 11
		"[nonmd](c.txt#frag)\n") // 12
	writeFile(t, root, "rec/c.txt", "text\n")
	cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{
		"links_resolve": {Enabled: true, Severity: "blocker"},
		"link_anchors":  {Enabled: true, Severity: "warn"},
	}}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	a := filepath.Join("rec", "a.md")
	for _, line := range []int{9, 10, 11} {
		if !hasFinding(fs, a, "link_anchors", line) {
			t.Errorf("a.md:%d: the broken anchor was not reported: %+v", line, fs)
		}
	}
	if n := countRule(fs, "link_anchors"); n != 3 {
		t.Errorf("want exactly 3 link_anchors findings, got %d: %+v", n, fs)
	}
	if n := countRule(fs, "links_resolve"); n != 0 {
		t.Errorf("an anchor is not a file: links_resolve must not report it: %+v", fs)
	}
	for _, f := range fs {
		if f.RuleID == "link_anchors" && f.Severity != "warn" {
			t.Errorf("link_anchors lands warn-first through its own severity: %+v", f)
		}
	}
}
