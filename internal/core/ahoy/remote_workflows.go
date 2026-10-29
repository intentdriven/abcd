package ahoy

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
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
	// Unread names what the reader could not judge: a workflow file it could
	// not read, the files past its bound, and each local action a workflow
	// uses, whose own steps it does not read. Anything unread holds both
	// tightenings at advisory, because steps abcd has not seen may break.
	Unread []string
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

// readWorkflowFacts reads root's .github/workflows. A container reference
// (`docker://…`) is outside the pinning policy and not counted; a local action
// (`./…`) is named as unread, since its own steps may use unpinned actions.
// An absent directory is zero files, which the audit reads as "nothing to
// break"; a directory that cannot be listed is unread.
func readWorkflowFacts(root string) workflowFacts {
	var f workflowFacts
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			f.Unread = append(f.Unread, ".github/workflows (it could not be listed)")
		}
		return f
	}
	local := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		if f.Files >= maxAuditWorkflowFiles {
			f.Unread = append(f.Unread, "the workflow files past the first "+strconv.Itoa(maxAuditWorkflowFiles))
			break
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxAuditWorkflowBytes)
		if err != nil {
			f.Unread = append(f.Unread, name)
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
			if strings.HasPrefix(ref, "./") {
				if !local[ref] {
					local[ref] = true
					f.Unread = append(f.Unread, "the local action "+ref)
				}
				continue
			}
			if strings.HasPrefix(ref, "docker://") {
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

// workflowDrift says why this checkout's workflow files cannot stand for the
// default branch's on GitHub, or "" when they match file for file. Every
// setting drawn from the workflows is applied to the default branch, so it
// must be judged on that branch's workflows: an audit run on a feature branch
// would otherwise require a check only that branch reports, and every pull
// request into the default branch would wait on it. The comparison is by git
// blob id, read from GitHub's contents answer and hashed here from the files
// the audit itself reads, so an uncommitted edit counts as a difference.
func workflowDrift(root, repo, branch string) string {
	if !branchNameRe.MatchString(branch) || strings.Contains(branch, "..") {
		return "the default branch's name cannot be put in an API path, so its workflows were not compared"
	}
	out, err := github.API(root, nil, "repos/"+repo+"/contents/.github/workflows?ref="+url.QueryEscape(branch))
	remote := map[string]string{}
	switch {
	case err == nil:
		var entries []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		}
		if json.Unmarshal(out, &entries) != nil {
			return "GitHub's listing of " + branch + "'s workflows could not be read, so they were not compared"
		}
		for _, e := range entries {
			if e.Type == "file" && isWorkflowFile(e.Name) {
				remote[e.Name] = e.SHA
			}
		}
	case isHTTP(err, 404):
		// The default branch has no workflows directory.
	default:
		return readFailure(branch+"'s workflows on GitHub", err)
	}
	local := map[string]string{}
	var unread []string
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "this checkout's .github/workflows could not be listed, so it was not compared with " + branch + "'s"
	}
	for _, e := range entries {
		if e.IsDir() || !isWorkflowFile(e.Name()) {
			continue
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, e.Name()), maxAuditWorkflowBytes)
		if err != nil {
			unread = append(unread, e.Name())
			continue
		}
		sum := sha1.Sum(append([]byte("blob "+strconv.Itoa(len(data))+"\x00"), data...))
		local[e.Name()] = hex.EncodeToString(sum[:])
	}
	var differ []string
	for name, id := range local {
		if remote[name] != id {
			differ = append(differ, name)
		}
	}
	for name := range remote {
		if _, ok := local[name]; !ok {
			differ = append(differ, name)
		}
	}
	differ = append(differ, unread...)
	if len(differ) == 0 {
		return ""
	}
	sort.Strings(differ)
	return "this checkout's workflows differ from " + branch + "'s on GitHub (" + strings.Join(differ, ", ") +
		"), and the default branch is what this setting governs; run the audit from a checkout of " + branch + "'s tip"
}

// isWorkflowFile reports whether name is a file GitHub runs as a workflow.
func isWorkflowFile(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}
