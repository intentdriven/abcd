package needs

import (
	"path/filepath"
	"testing"
)

// TestADirectoryLinkTheRepositoryHoldsIsHeldToTheGuards: ~/alias links to
// R/scripts, and R/scripts is a committed link to opt/tools/bin, which holds a
// real go outside R. The path, its final file and ~/alias all lie outside R,
// but the route passes through R/scripts, a link the repository holds, so the
// repository chose which program runs. Both rules refuse it, naming that link.
func TestADirectoryLinkTheRepositoryHoldsIsHeldToTheGuards(t *testing.T) {
	root := t.TempDir()
	repo := tree(t, root, "repo")
	opt := tree(t, root, "opt", "tools", "bin")
	program(t, opt, "go")
	scripts := link(t, opt, filepath.Join(repo, "scripts"))
	alias := link(t, scripts, filepath.Join(tree(t, root, "home"), "alias"))
	p := filepath.Join(alias, "go")
	rule := Rule{Guards: []string{repo}}
	_, err := rule.Admit(p)
	refusedAs(t, err, Inside, scripts)
	_, err = rule.Judge("go", p)
	refusedAs(t, err, Inside, scripts)
}
