package launch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// LockstepTree is the polarity the lockstep check runs under.
type LockstepTree string

const (
	// TreeDev requires the version keys ABSENT (adr-19 dev-stays-unversioned).
	TreeDev LockstepTree = "dev"
	// TreePublic requires the primary present and every secondary to agree.
	TreePublic LockstepTree = "public"
)

// LockstepResult is the outcome of a manifest lockstep check.
type LockstepResult struct {
	Tree       LockstepTree `json:"tree"`
	OK         bool         `json:"ok"`
	Drifts     []string     `json:"drifts,omitempty"`
	Unreadable bool         `json:"unreadable,omitempty"`
	Detail     string       `json:"detail,omitempty"`
	ExitCode   int          `json:"exit_code"` // 0 ok, 1 drift, 2 unreadable
}

// Pinned secondary locations (adr-20 table). The primary is read from
// version-location.json so an adr-19 relocation is absorbed without edits.
const (
	marketplaceFile         = ".claude-plugin/marketplace.json"
	secondaryVersionPointer = "/plugins/0/version"
	secondaryChangelogPtr   = "/plugins/0/changelog"
	changelogVersionPointer = "/plugins/0/changelog/version"
)

// CheckLockstep proves the two manifests plus the version-location contract
// describe one release consistently.
//
// PUBLIC: the primary version (manifest_path + json_pointer from
// version-location.json) must be a present, non-null strict-SemVer string and
// every pinned secondary must AGREE. DEV: those keys must all be ABSENT.
// present-null is distinguished from absent via an explicit sentinel. A
// blocked:true contract, or any unreadable pinned input, yields exit 2.
func CheckLockstep(tree LockstepTree, repoRoot, versionLocationPath string) LockstepResult {
	res := LockstepResult{Tree: tree}

	decision, err := loadJSON(versionLocationPath)
	if err != nil {
		return unreadable(res, "version-location.json not readable: "+err.Error())
	}
	primaryPath, primaryPtr, verr := validateVersionLocation(decision)
	if verr != "" {
		return unreadable(res, verr)
	}

	primaryDoc, err := loadRepoJSON(repoRoot, primaryPath)
	if err != nil {
		return unreadable(res, "primary manifest not readable: "+err.Error())
	}
	marketplace, err := loadRepoJSON(repoRoot, marketplaceFile)
	if err != nil {
		return unreadable(res, "marketplace.json not readable: "+err.Error())
	}

	var drifts []string
	if tree == TreePublic {
		drifts = checkPublic(primaryPath, primaryPtr, primaryDoc, marketplace)
	} else {
		drifts = checkDev(primaryPath, primaryPtr, primaryDoc, marketplace)
	}
	if len(drifts) > 0 {
		res.Drifts = drifts
		res.ExitCode = 1
		return res
	}
	res.OK = true
	res.ExitCode = 0
	return res
}

func unreadable(res LockstepResult, detail string) LockstepResult {
	res.Unreadable = true
	res.Detail = detail
	res.ExitCode = 2
	return res
}

func loadJSON(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, pathFreeError(err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// loadRepoJSON reads a repo-relative JSON document through an os.Root opened on
// repoRoot, so the read is contained by what the path resolves to rather than by
// its spelling: ValidRelPath is lexical, and a committed symlink inside the
// repository that points out of it would otherwise be read through. A symlink
// that stays inside the repository still reads; one that leaves it is an error,
// and the document outside is never read.
func loadRepoJSON(repoRoot, rel string) (any, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, pathFreeError(err)
	}
	defer root.Close()
	data, err := root.ReadFile(filepath.FromSlash(rel))
	if err != nil {
		return nil, pathFreeError(err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// pathFreeError strips the filesystem path from an os.PathError, leaving only the
// underlying cause. The lockstep detail names the file itself ("version-location.json
// not readable: ..."), and the dry-run report carrying that detail is a SUCCESS
// envelope, so it never passes through cli.scrubPaths — an unstripped *os.PathError
// would leak the cwd/home-rooted absolute path into the report and its --json
// output, against the no-absolute-paths-in-machine-output norm (iss-76/iss-29).
func pathFreeError(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// validateVersionLocation checks the consumed shape and returns the primary
// (manifest_path, json_pointer). A blocked:true decision has no location →
// unreadable.
func validateVersionLocation(decision any) (string, string, string) {
	obj, ok := decision.(map[string]any)
	if !ok {
		return "", "", "version-location decision is not a JSON object"
	}
	if b, ok := obj["blocked"].(bool); ok && b {
		return "", "", "version-location decision is blocked: true — no version location to check against"
	}
	mp, ok := obj["manifest_path"].(string)
	if !ok || mp == "" {
		return "", "", "version-location decision missing a string manifest_path"
	}
	// manifest_path arrives as committed configuration data and is joined onto a
	// trusted root and then WRITTEN (editManifest stamps the derived version), so
	// it must be contained the way launch include patterns are — no absolute path
	// and no ".." traversal, or the stamp escapes the destination (gh-488).
	if !fsutil.ValidRelPath(mp) {
		return "", "", "version-location manifest_path is not a contained repo-relative path: " + mp
	}
	ptr, ok := obj["json_pointer"].(string)
	if !ok || !strings.HasPrefix(ptr, "/") {
		return "", "", "version-location decision missing an RFC-6901 json_pointer"
	}
	return mp, ptr, ""
}

// resolvePointer resolves an RFC-6901 pointer over doc, returning (value,
// present). present is false when any token is a missing key/index, so
// present-null (present=true, value=nil) is distinguished from an absent key.
func resolvePointer(doc any, pointer string) (any, bool) {
	if pointer == "" {
		return doc, true
	}
	cur := doc
	for _, raw := range strings.Split(pointer, "/")[1:] {
		tok := unescapePointerToken(raw)
		switch c := cur.(type) {
		case map[string]any:
			v, ok := c[tok]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			idx, ok := atoiIndex(tok)
			if !ok || idx < 0 || idx >= len(c) {
				return nil, false
			}
			cur = c[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// unescapePointerToken decodes one RFC-6901 reference token (~1 → /, ~0 → ~).
// The order matters and is the spec's: decoding ~0 first would turn "~01" into
// "~1" and then into "/". It is shared by the pointer reader here and the
// pointer writer in render.go so the two can never disagree about a token.
func unescapePointerToken(raw string) string {
	return strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
}

func atoiIndex(tok string) (int, bool) {
	if tok == "" {
		return 0, false
	}
	const maxInt = int(^uint(0) >> 1)
	n := 0
	for _, r := range tok {
		if r < '0' || r > '9' {
			return 0, false
		}
		d := int(r - '0')
		// Guard against overflow: a long all-digit token would otherwise wrap to a
		// negative n and pass the `idx >= len(c)` bound, panicking on c[idx]. No
		// valid slice index is ever this large, so reject rather than wrap.
		if n > (maxInt-d)/10 {
			return 0, false
		}
		n = n*10 + d
	}
	return n, true
}

func fmtValue(v any, present bool) string {
	if !present {
		return "<absent-key>"
	}
	if v == nil {
		return "present-null"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// checkPublic: primary must be a present non-null strict-SemVer string; every
// secondary must agree.
func checkPublic(primaryPath, primaryPtr string, primaryDoc, marketplace any) []string {
	var drifts []string

	primVal, primPresent := resolvePointer(primaryDoc, primaryPtr)
	primStr, isStr := primVal.(string)
	expectedOK := primPresent && primVal != nil && isStr && IsStrictSemver(primStr)
	if !expectedOK {
		reason := "expected a present strict-SemVer version string, got " + fmtValue(primVal, primPresent)
		if primPresent && isStr && !IsStrictSemver(primStr) {
			reason = "version is not strict SemVer: " + fmtValue(primVal, primPresent)
		}
		drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: %s", primaryPath, primaryPtr, reason))
	}

	secVal, secPresent := resolvePointer(marketplace, secondaryVersionPointer)
	if !expectedOK || !secPresent || secVal == nil || secVal != any(primStr) {
		detail := "present but primary version is unreadable, got " + fmtValue(secVal, secPresent)
		if expectedOK {
			detail = "expected " + fmtValue(primStr, true) + " (from primary), got " + fmtValue(secVal, secPresent)
		}
		if expectedOK || secPresent {
			drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: %s", marketplaceFile, secondaryVersionPointer, detail))
		}
	}

	clVal, clPresent := resolvePointer(marketplace, secondaryChangelogPtr)
	if !clPresent || clVal == nil {
		drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: expected a changelog entry, got %s",
			marketplaceFile, secondaryChangelogPtr, fmtValue(clVal, clPresent)))
	} else {
		clv, clvPresent := resolvePointer(marketplace, changelogVersionPointer)
		if !expectedOK || !clvPresent || clv != any(primStr) {
			detail := "present but primary version is unreadable, got " + fmtValue(clv, clvPresent)
			if expectedOK {
				detail = "expected " + fmtValue(primStr, true) + " (from primary), got " + fmtValue(clv, clvPresent)
			}
			if expectedOK || clvPresent {
				drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: %s", marketplaceFile, changelogVersionPointer, detail))
			}
		}
	}
	return drifts
}

// checkDev: every pinned version/changelog key must be ABSENT; a present key
// (even present-null) is drift.
func checkDev(primaryPath, primaryPtr string, primaryDoc, marketplace any) []string {
	var drifts []string
	checks := []struct {
		file string
		ptr  string
		doc  any
	}{
		{primaryPath, primaryPtr, primaryDoc},
		{marketplaceFile, secondaryVersionPointer, marketplace},
		{marketplaceFile, secondaryChangelogPtr, marketplace},
	}
	for _, c := range checks {
		v, present := resolvePointer(c.doc, c.ptr)
		if present {
			drifts = append(drifts, fmt.Sprintf("DRIFT dev %s%s: adr-19 requires this key ABSENT in the dev tree, got %s",
				c.file, c.ptr, fmtValue(v, present)))
		}
	}
	return drifts
}

// CheckDeclaredLockstep is the lockstep check for a non-plugin artefact kind
// (itd-2609150819432059, decision 2): the primary is read from
// version-location.json exactly as CheckLockstep reads it, and the pinned
// plugin-manifest table is replaced by the files the artefact declaration names.
// No plugin manifest is read.
//
// Each declared file is a JSON document carrying the version at its own pointer,
// or at the primary's when it names none. The polarities are CheckLockstep's:
// DEV requires every key ABSENT (adr-19), PUBLIC requires the primary present as
// strict SemVer and every secondary to agree with it. A declared file that cannot
// be read or parsed is unreadable (exit 2) and the detail names it.
//
// A kind that declares no lockstep list and carries no version-location contract
// holds nothing in lockstep, and the result is an OK that says so. A declared
// list with no contract to read the primary from is unreadable: there is nothing
// for the list to agree with.
func CheckDeclaredLockstep(tree LockstepTree, repoRoot, versionLocationPath string, files []LockstepFile) LockstepResult {
	res := LockstepResult{Tree: tree}

	decision, err := loadJSON(versionLocationPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && len(files) == 0 {
			res.OK = true
			res.Detail = "no version-location contract and no declared lockstep list: nothing is held in lockstep"
			return res
		}
		return unreadable(res, "version-location.json not readable: "+err.Error())
	}
	primaryPath, primaryPtr, verr := validateVersionLocation(decision)
	if verr != "" {
		return unreadable(res, verr)
	}
	primaryDoc, err := loadRepoJSON(repoRoot, primaryPath)
	if err != nil {
		return unreadable(res, "primary manifest "+primaryPath+" not readable: "+err.Error())
	}

	type located struct {
		file, ptr string
		doc       any
	}
	secondaries := make([]located, 0, len(files))
	for _, f := range files {
		doc, err := loadRepoJSON(repoRoot, f.Path)
		if err != nil {
			return unreadable(res, "declared lockstep file "+f.Path+" not readable: "+err.Error())
		}
		ptr := f.Pointer
		if ptr == "" {
			ptr = primaryPtr
		}
		secondaries = append(secondaries, located{f.Path, ptr, doc})
	}

	var drifts []string
	primVal, primPresent := resolvePointer(primaryDoc, primaryPtr)
	if tree == TreePublic {
		primStr, isStr := primVal.(string)
		expectedOK := primPresent && isStr && IsStrictSemver(primStr)
		if !expectedOK {
			drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: expected a present strict-SemVer version string, got %s",
				primaryPath, primaryPtr, fmtValue(primVal, primPresent)))
		}
		for _, s := range secondaries {
			v, present := resolvePointer(s.doc, s.ptr)
			switch {
			case expectedOK && (!present || v != any(primStr)):
				drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: expected %s (from primary), got %s",
					s.file, s.ptr, fmtValue(primStr, true), fmtValue(v, present)))
			case !expectedOK && present:
				drifts = append(drifts, fmt.Sprintf("DRIFT public %s%s: present but primary version is unreadable, got %s",
					s.file, s.ptr, fmtValue(v, present)))
			}
		}
	} else {
		if primPresent {
			drifts = append(drifts, fmt.Sprintf("DRIFT dev %s%s: adr-19 requires this key ABSENT in the dev tree, got %s",
				primaryPath, primaryPtr, fmtValue(primVal, primPresent)))
		}
		for _, s := range secondaries {
			if v, present := resolvePointer(s.doc, s.ptr); present {
				drifts = append(drifts, fmt.Sprintf("DRIFT dev %s%s: adr-19 requires this key ABSENT in the dev tree, got %s",
					s.file, s.ptr, fmtValue(v, present)))
			}
		}
	}
	if len(drifts) > 0 {
		res.Drifts = drifts
		res.ExitCode = 1
		return res
	}
	res.OK = true
	return res
}
