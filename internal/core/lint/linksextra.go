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
	return walkExtraRoots(repoRoot, "links_resolve", cfg, nil, func(rel, fileAbs string, lines []string) []Finding {
		return checkLinks(rel, fileAbs, repoRoot, lines, fenceMask(lines), cfg)
	})
}

// checkHarnessLeakExtraRoots runs harness_leak over the rule's ExtraRoots, the
// same way: the working tier's issue ledger is committed free text a verb
// writes from operator input, and a rule rooted at the durable record alone
// never read it (iss-2608301306580014). A file the Roots walk already read is
// not read twice, so a tree both declare draws one finding per leak.
func checkHarnessLeakExtraRoots(repoRoot string, cfg RuleConfig, scanned map[string]bool) ([]Finding, error) {
	return walkExtraRoots(repoRoot, ruleHarnessLeak, cfg, scanned, func(rel, _ string, lines []string) []Finding {
		return checkHarnessLeak(rel, lines, fenceMask(lines), cfg)
	})
}

// walkExtraRoots reads every markdown file under a rule's ExtraRoots, except
// one skip names or an Exempt glob matches, and hands each to check.
func walkExtraRoots(repoRoot, ruleID string, cfg RuleConfig, skip map[string]bool, check func(rel, fileAbs string, lines []string) []Finding) ([]Finding, error) {
	var out []Finding
	for _, root := range cfg.ExtraRoots {
		if err := containedRepoPath(root); err != nil {
			return nil, &configError{ruleID + " extra_roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		rootAbs := filepath.Join(repoRoot, filepath.FromSlash(root))
		if err := resolvedInsideRoot(repoRoot, rootAbs); err != nil {
			return nil, &configError{ruleID + " extra_roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		if _, err := os.Stat(rootAbs); err != nil {
			if os.IsNotExist(err) {
				return nil, &configError{ruleID + " extra_roots entry " + quote(root) +
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
			if skip[fileAbs] || matchesGlob(cfg.Exempt, filepath.ToSlash(rel)) {
				continue
			}
			content, err := readRepoAbs(repoRoot, fileAbs, maxRepoFileBytes)
			if err != nil {
				return nil, err
			}
			out = append(out, check(rel, fileAbs, strings.Split(string(content), "\n"))...)
		}
	}
	return out, nil
}
