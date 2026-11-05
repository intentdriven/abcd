package landing

import (
	"fmt"
	"strings"
)

// recordFolders are the status folders a landing's records lie in, spelled exactly.
var recordFolders = []string{
	".abcd/work/issues/open/", ".abcd/work/issues/resolved/", ".abcd/work/issues/wontfix/",
	".abcd/development/specs/open/", ".abcd/development/specs/closed/",
	".abcd/development/intents/planned/", ".abcd/development/intents/shipped/",
}

// RecordsCommitShape says why the commits from base to head are not what a landing's
// records commit holds, "" when they are: every path a markdown file of mode 100644,
// and none under a status folder spelled in another case. It does not bound which
// markdown: a landing's own relink touches other records, and a pair reads it. The
// error is git's, when it could not say.
func RecordsCommitShape(repoRoot, base, head string) (string, error) {
	raw, err := storedBytes(repoRoot, maxDeltaGitOutput, "diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none", "--raw", "-z", base, head)
	if err != nil {
		return "", fmt.Errorf("git cannot read what %s changes: %w", short(head), err)
	}
	if len(raw) == 0 {
		return "", nil
	}
	fields := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	if len(fields)%2 != 0 {
		return "", fmt.Errorf("git's list of what %s changes does not parse", short(head))
	}
	for i := 0; i < len(fields); i += 2 {
		f, p := strings.Fields(fields[i]), fields[i+1]
		if len(f) < 2 {
			return "", fmt.Errorf("git's list of what %s changes does not parse", short(head))
		}
		if !strings.HasSuffix(p, deltaCoversKind) {
			return "it changes " + clip(p) + ", which is no markdown record", nil
		}
		for _, mode := range []string{strings.TrimPrefix(f[0], ":"), f[1]} {
			if mode != modeAbsent && mode != modeRegular {
				return "it changes " + clip(p) + " at mode " + mode + ", and a record is of mode " + modeRegular, nil
			}
		}
		for _, folder := range recordFolders {
			if len(p) > len(folder) && strings.EqualFold(p[:len(folder)], folder) && p[:len(folder)] != folder {
				return "it changes " + clip(p) + ", under a folder spelled unlike " + folder, nil
			}
		}
	}
	return "", nil
}
