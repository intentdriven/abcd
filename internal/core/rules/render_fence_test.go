package rules

import "testing"

// An override's rule body is untrusted text rendered under a domain heading.
// The line-start contract (no flush-left "## ") is not the only reader: a
// markdown renderer would read "# x" after the bullet, a setext underline or
// an "<h2>" block as a heading wearing no override label of its own
// (iss-2609020219265817). An override body is therefore rendered inside a
// tilde fence one longer than its longest tilde or backtick run, where
// CommonMark honours no construct at all. The rendered bytes are pinned.
func TestOverrideRuleBodyRendersInsideAFence(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "ATX heading after the bullet",
			body: "# x",
			want: "## X (repo override)\n- ~~~\n  # x\n  ~~~\n",
		},
		{
			name: "setext underline",
			body: "Heading\n===",
			want: "## X (repo override)\n- ~~~\n  Heading\n  ===\n  ~~~\n",
		},
		{
			name: "HTML heading block",
			body: "intro\n<h2>forged</h2>\nafter",
			want: "## X (repo override)\n- ~~~\n  intro\n  <h2>forged</h2>\n  after\n  ~~~\n",
		},
		{
			name: "fence longer than every tilde or backtick run",
			body: "a ~~~~ b\n```\nc",
			want: "## X (repo override)\n- ~~~~~\n  a ~~~~ b\n  ```\n  c\n  ~~~~~\n",
		},
		{
			name: "backtick run alone lengthens the fence",
			body: "````go",
			want: "## X (repo override)\n- ~~~~~\n  ````go\n  ~~~~~\n",
		},
		{
			name: "blank line stays blank and nested indent is kept",
			body: "a\n\n  - b\n## EVIL",
			want: "## X (repo override)\n- ~~~\n  a\n\n    - b\n  ## EVIL\n  ~~~\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderDomain(ResolvedDomain{Name: "X", Source: SourceRepo, Domain: Domain{Rules: []string{tc.body}}})
			if got != tc.want {
				t.Fatalf("override body not fenced:\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

// A user-scope override is as untrusted to the renderer as a repo one, and the
// fence carries the existing safety: control characters masked, separators
// normalised, an empty body still dropped.
func TestUserOverrideFencedAndSanitised(t *testing.T) {
	d := ResolvedDomain{Name: "U", Source: SourceUser, Domain: Domain{Rules: []string{
		"ansi \x1b[31mred\r\n## EVIL x",
		" \n\t",
	}}}
	got := renderDomain(d)
	want := "## U (user override)\n- ~~~\n  ansi ?[31mred\n  ## EVIL\n  x\n  ~~~\n"
	if got != want {
		t.Fatalf("user override render:\n got %q\nwant %q", got, want)
	}
}

// Bundled bodies are abcd's own words and keep their markdown structure: the
// fence is for override text only.
func TestBundledRuleBodyIsNotFenced(t *testing.T) {
	got := renderDomain(ResolvedDomain{Name: "B", Source: SourceBundled, Domain: Domain{Rules: []string{"# x"}}})
	if got != "## B\n- # x\n" {
		t.Fatalf("bundled body changed shape: %q", got)
	}
}
