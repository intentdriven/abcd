package lint

import (
	"os"
	"path/filepath"
	"strings"
)

// checkLinksExtraRoots runs links_resolve over the rule's ExtraRoots: trees whose
// relative links must resolve but whose content no other rule judges. Each tree
// is contained and each leaf read through the guarded read, as the roots walk
// does; a file matching an Exempt glob is skipped. A configured tree that does
// not exist is misconfiguration, for the reason a missing root is: it would
// silently disarm the rule for that tree.
func checkLinksExtraRoots(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	var out []Finding
	for _, root := range cfg.ExtraRoots {
		if err := containedRepoPath(root); err != nil {
			return nil, &configError{"links_resolve extra_roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		rootAbs := filepath.Join(repoRoot, filepath.FromSlash(root))
		if err := resolvedInsideRoot(repoRoot, rootAbs); err != nil {
			return nil, &configError{"links_resolve extra_roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		if _, err := os.Stat(rootAbs); err != nil {
			if os.IsNotExist(err) {
				return nil, &configError{"links_resolve extra_roots entry " + quote(root) +
					" does not exist; a configured tree that does not resolve silently disarms the rule for it"}
			}
			return nil, err
		}
		files, err := markdownFiles(rootAbs)
		if err != nil {
			return nil, err
		}
		for _, fileAbs := range files {
			rel := repoRel(repoRoot, fileAbs)
			if matchesGlob(cfg.Exempt, filepath.ToSlash(rel)) {
				continue
			}
			content, err := readRepoAbs(repoRoot, fileAbs, maxRepoFileBytes)
			if err != nil {
				return nil, err
			}
			lines := strings.Split(string(content), "\n")
			out = append(out, checkLinks(rel, fileAbs, repoRoot, lines, fenceMask(lines), cfg)...)
		}
	}
	return out, nil
}
