package ahoy

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DocsLintRootMissingGapID is a roots entry in the repository's docs-lint
// config that does not resolve (iss-2610100649479892). The documentation check
// refuses to run while one is listed, so every rule it carries, the banned
// names included, checks nothing until the entry is fixed. The usual cause is
// a retired conventions file, CLAUDE.md, still listed after setup removed it
// or the person did. The gap is one per such entry and report-only: the config
// is the person's file, and abcd does not edit it.
const DocsLintRootMissingGapID = "docs_lint.root_missing"

// detectDocsLintRoots raises the gap for each roots entry of
// .abcd/docs-lint.json that names nothing on disk, judged as the documentation
// check judges it (os.Stat, so a root may be a file such as README.md, and a
// link that leads nowhere does not resolve). A missing or unloadable config
// raises nothing here: the first is a repository that has not adopted the
// check, and the second the check's own refusal reports. An entry that is not
// a plain path inside the repository is not looked up: the check refuses it
// with its own words, and this read never leaves the project.
func detectDocsLintRoots(cwd string) []Gap {
	cfg, err := lint.LoadConfig(filepath.Join(cwd, filepath.FromSlash(banlist.PublicConfigRelPath)))
	if err != nil {
		return nil
	}
	var gaps []Gap
	for _, root := range cfg.Roots {
		if root == "" || !filepath.IsLocal(filepath.FromSlash(root)) {
			continue
		}
		if _, err := os.Stat(filepath.Join(cwd, filepath.FromSlash(root))); !os.IsNotExist(err) {
			continue
		}
		shown := termsafe.Sanitize(strconv.Quote(root))
		gaps = append(gaps, Gap{
			ID: DocsLintRootMissingGapID, Category: ConfigChange, Scope: "repo",
			Title: banlist.PublicConfigRelPath + " lists " + shown + " in roots, and it does not exist",
			Detail: "The documentation check refuses to run while a roots entry does not resolve, so none of its rules, " +
				"the banned names included, checks anything: `abcd lint docs` exits 2, and bare `abcd lint` reports the " +
				"refusal as an error. A conventions file retired after it was listed, such as CLAUDE.md, leaves exactly this.",
			FixHint: "Edit " + banlist.PublicConfigRelPath + ": remove " + shown + " from roots, or create it. " +
				"ahoy install never edits that file.",
			Required: false, Resolvable: false,
		})
	}
	return gaps
}
