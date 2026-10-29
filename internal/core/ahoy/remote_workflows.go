package ahoy

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// workflowFacts is what the settings audit reads from the repository's own
// workflow files (itd-2610290746077716): whether every workflow could live
// under a policy that requires actions pinned to a commit, and whether every
// workflow declares its token's permissions. Both decide whether the apply may
// tighten the Actions policy without breaking a CI that works today.
type workflowFacts struct {
	// Files is how many workflow files were read.
	Files int
	// Unpinned counts the `uses:` references not pinned to a full commit SHA.
	Unpinned int
	// NoPermissions names the workflow files with no top-level `permissions:`.
	NoPermissions []string
}

// maxAuditWorkflowFiles bounds the walk, as the scaffold's reader does.
const maxAuditWorkflowFiles = 64

// maxAuditWorkflowBytes bounds each workflow read.
const maxAuditWorkflowBytes = 256 << 10

// usesRe captures the reference of a `uses:` key, in a step or a job.
var usesRe = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*["']?([^"'\s#]+)`)

// pinnedRefRe is a full 40-hex commit SHA, the only ref GitHub's SHA-pinning
// policy accepts.
var pinnedRefRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

// topPermissionsRe matches a workflow-level `permissions:` key.
var topPermissionsRe = regexp.MustCompile(`(?m)^permissions\s*:`)

// readWorkflowFacts reads root's .github/workflows. A reference to a local
// action (`./…`) or a container (`docker://…`) is outside the pinning policy
// and not counted. An absent directory is zero files, which the audit reads
// as "nothing to break".
func readWorkflowFacts(root string) workflowFacts {
	var f workflowFacts
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return f
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		if f.Files >= maxAuditWorkflowFiles {
			break
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxAuditWorkflowBytes)
		if err != nil {
			continue
		}
		f.Files++
		src := strings.ReplaceAll(string(data), "\r\n", "\n")
		if !topPermissionsRe.MatchString(src) {
			f.NoPermissions = append(f.NoPermissions, name)
		}
		for _, line := range strings.Split(src, "\n") {
			m := usesRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			ref := m[1]
			if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") {
				continue
			}
			at := strings.LastIndex(ref, "@")
			if at < 0 || !pinnedRefRe.MatchString(ref[at+1:]) {
				f.Unpinned++
			}
		}
	}
	sort.Strings(f.NoPermissions)
	return f
}
