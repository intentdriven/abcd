package launch

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// ScanTree runs the launch bundle's secret and personal-data scan over every
// entry of the tree at dir, git's own directories aside, and returns what the
// bundle's gate would refuse the tree on (scanRefusals): a hard-fail finding,
// a scanner that cannot run or runs without the augmenter repoRoot
// configured, and every entry it could not cover. A model-matrix copy's clone
// passes it before the copy is routed to a provider (spc-2610161949559050
// step 3), so what the release gate keeps off the public never leaves the
// machine in a clone either. The scanner's configuration is read at repoRoot,
// the checkout the run belongs to, whose local tier the clone does not hold.
// Git's own directory is not read: a caller whose repository holds more than
// its checked-out files scans the rest itself (ScanText), as a model-matrix
// clone's admission scans its frozen commit's message and identities.
//
// An entry is never followed: a symbolic link or any other non-regular entry
// is handed to the scanner as itself, which leaves it unscanned and so
// refuses the tree, rather than reading what it points at or passing it
// unread. An error is a walk that could not list the tree.
func ScanTree(repoRoot, dir string) ([]string, error) {
	var files []scanner.BundleFile
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == ".git" {
			// A worktree's or a submodule's pointer file is git's, as the
			// directory is.
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		files = append(files, scanner.BundleFile{LogicalPath: filepath.ToSlash(rel), ResolvedPath: p})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("the tree to scan could not be listed: %w", err)
	}
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return scanRefusals(scanner.ScanResult{Unavailable: true, UnavailableReason: err.Error()}), nil
	}
	res, _ := sc.ScanBundle(files)
	return scanRefusals(res), nil
}

// ScanText is the launch bundle's scan over text that is no file of a tree,
// named name in its findings: what the bundle's gate would refuse it on
// (scanRefusals), with the scanner's configuration read at repoRoot. The
// configuration's rules and identity apply, and its path filters do not, since
// the text has no path: nothing it skips or excludes leaves the text out. A
// scanner that cannot run, runs degraded, or runs without the augmenter the
// repository configured refuses, as it does for a tree.
func ScanText(repoRoot, name string, text []byte) []string {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return scanRefusals(scanner.ScanResult{Unavailable: true, UnavailableReason: err.Error()})
	}
	res := scanner.ScanResult{FilesScanned: 1, Findings: sc.ScanText(string(text), name)}
	if gap := sc.AugmenterGap(); gap != "" {
		res.Unscanned = []string{scanner.AugmenterGapPath}
		res.UnscannedWhy = map[string]string{scanner.AugmenterGapPath: gap}
	}
	if bad, why := sc.Unavailable(); bad {
		res.Unavailable, res.UnavailableReason = true, why
	}
	return scanRefusals(res)
}
