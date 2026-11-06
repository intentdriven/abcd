package landing

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// recordRoots are the folders whose subfolders hold a record's status.
var recordRoots = []string{recordid.IssuesRelDir, ".abcd/development/specs", ".abcd/development/intents"}

// issueRoot is the record root an issue's move is read under, and issueIDName the id
// a record's file name opens with.
const issueRoot = recordid.IssuesRelDir + "/"

var issueIDName = regexp.MustCompile(`^iss-[0-9]+`)

// RecordsCommitShape says why the commits from base to head are not what a landing's
// records commit holds, "" when they are: every path a plain-text markdown file of
// mode 100644, none under a record folder spelled in another case, and no issue's
// record moved but those named in issues, what the full round was told of. A record changed where it stands is not
// bounded: a landing's own relink touches other records, and a pair reads it. History
// is read as stored. The error is git's, when it could not say.
func RecordsCommitShape(repoRoot, base, head string, issues []string) (string, error) {
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
	folders, err := recordFolders(repoRoot, base)
	if err != nil {
		return "", err
	}
	for i := 0; i < len(fields); i += 2 {
		f, p := strings.Fields(fields[i]), fields[i+1]
		if len(f) < 4 {
			return "", fmt.Errorf("git's list of what %s changes does not parse", short(head))
		}
		if !strings.HasSuffix(p, deltaCoversKind) {
			return "it changes " + clip(p) + ", which is no markdown record", nil
		}
		for side, mode := range []string{strings.TrimPrefix(f[0], ":"), f[1]} {
			switch mode {
			case modeAbsent:
			case modeRegular:
				data, err := storedBytes(repoRoot, maxDeltaBlobBytes, "cat-file", "blob", f[2+side])
				if err != nil {
					return "", fmt.Errorf("%s changes %s, which cannot be read whole: %w", short(head), clip(p), err)
				}
				if why := notPlainText(data); why != "" {
					return "it changes " + clip(p) + ", which holds " + why, nil
				}
			default:
				return "it changes " + clip(p) + " at mode " + mode + ", and a record is of mode " + modeRegular, nil
			}
		}
		if folder := spelledUnlike(p, folders); folder != "" {
			return "it changes " + clip(p) + ", under a folder spelled unlike " + folder, nil
		}
		if strings.HasPrefix(p, issueRoot) && (strings.HasPrefix(f[0], ":"+modeAbsent) || f[1] == modeAbsent) {
			if id := issueIDName.FindString(path.Base(p)); !slices.Contains(issues, id) {
				return "it moves the record " + clip(p) + ", which the lane's full round was not told of", nil
			}
		}
	}
	return "", nil
}

// recordFolders are the record roots and every status folder under one that commit
// holds, as stored, spelled as it spells them.
func recordFolders(repoRoot, commit string) ([]string, error) {
	folders := slices.Clone(recordRoots)
	for _, root := range recordRoots {
		out, err := storedBytes(repoRoot, maxDeltaGitOutput, "ls-tree", "-d", "--name-only", "-z", commit, "--", root+"/")
		if err != nil {
			return nil, fmt.Errorf("git cannot list %s at %s: %w", root, short(commit), err)
		}
		for _, name := range strings.Split(string(out), "\x00") {
			if name != "" {
				folders = append(folders, name)
			}
		}
	}
	return folders, nil
}

// spelledUnlike is the folder p lies under part by part, folded, but does not spell
// exactly, "" when there is none.
func spelledUnlike(p string, folders []string) string {
	parts := strings.Split(p, "/")
	for _, folder := range folders {
		want := strings.Split(folder, "/")
		if len(parts) <= len(want) {
			continue
		}
		same, exact := true, true
		for k, w := range want {
			same = same && strings.EqualFold(parts[k], w)
			exact = exact && parts[k] == w
		}
		if same && !exact {
			return folder
		}
	}
	return ""
}
