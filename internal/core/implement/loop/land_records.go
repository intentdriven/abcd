package loop

// land_records.go is what the landing accepts as the records commit a call killed
// before its state write made: only what a landing makes. The subject and the
// parent say which commit it claims to be; the content says whether it could be.

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The directories a landing's own records live in.
var landingRecordDirs = []string{".abcd/development/specs/", ".abcd/development/intents/", ".abcd/work/issues/"}

// recordsCommitGap says why tip, one commit past the lane's judged head, could
// not be the landing's records commit, "" when it could. Its diff against that
// head must be confined to the records this landing names (the spec, the intent
// and the captures it resolves, moved or edited) and to Markdown elsewhere under
// `.abcd/` in which only a link to one of them moved. A merge brings in more than
// its first parent shows, so it is refused, as is anything that is not a regular
// file.
func recordsCommitGap(repoRoot string, st State, lane Lane, tip string) string {
	if out, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", tip+"^2^{commit}", "--"); err == nil && strings.TrimSpace(out) != "" {
		return "it is a merge"
	}
	raw, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "diff-tree", "-r", "--no-renames", "--raw", "-z", lane.HeadSHA, tip, "--")
	if err != nil {
		return "git could not read what it changed"
	}
	names := []string{}
	for _, id := range []string{st.Spec, st.Intent, st.Issue()} {
		if id != "" {
			names = append(names, id+"-")
		}
	}
	for _, r := range lane.Resolves {
		names = append(names, r.Issue+"-")
	}
	fields := bytes.Split(raw, []byte{0})
	for i := 0; i+1 < len(fields); i += 2 {
		meta := strings.Fields(string(fields[i]))
		p := string(fields[i+1])
		if len(meta) < 5 {
			return "git could not read what it changed"
		}
		if m := meta[1]; m != "000000" && m != "100644" && m != "100755" {
			return fmt.Sprintf("%q is not a regular file", p)
		}
		if !strings.HasPrefix(p, ".abcd/") || !strings.HasSuffix(p, ".md") {
			return fmt.Sprintf("it changes %q, which is no record", p)
		}
		if namedRecord(p, names) {
			continue
		}
		if meta[4] != "M" {
			return fmt.Sprintf("it adds or removes %q, which this landing does not name", p)
		}
		if gap := onlyLinksMoved(repoRoot, lane.HeadSHA, tip, p); gap != "" {
			return gap
		}
	}
	return ""
}

// namedRecord reports whether p is one of the records a landing names, in the
// directory records of its kind live in.
func namedRecord(p string, names []string) bool {
	base := path.Base(p)
	matches := false
	for _, n := range names {
		matches = matches || strings.HasPrefix(base, n)
	}
	if !matches {
		return false
	}
	for _, d := range landingRecordDirs {
		if strings.HasPrefix(p, d) {
			return true
		}
	}
	return false
}

// recordStatusDirs are the folder names a record's status moves it between, each
// a link's only changing part when a landing moves that record.
var recordStatusDirs = strings.NewReplacer("/open/", "/_/", "/resolved/", "/_/", "/wontfix/", "/_/", "/closed/", "/_/",
	"/planned/", "/_/", "/shipped/", "/_/", "/drafts/", "/_/", "/ready/", "/_/")

// recordLinkDir matches a Markdown link's directory, which a record's move can
// add where a link was to a file in the same folder.
var recordLinkDir = regexp.MustCompile(`\]\([^)\s]*/([^/)\s]+\.md)\)`)

// withoutLinkDirs is line as a landing's repointing cannot change it.
func withoutLinkDirs(line string) string {
	return recordStatusDirs.Replace(recordLinkDir.ReplaceAllString(line, "]($1)"))
}

// onlyLinksMoved says why p's change between from and to is more than a link
// following a record to its new folder, "" when it is no more than that.
func onlyLinksMoved(repoRoot, from, to, p string) string {
	out, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "--literal-pathspecs", "diff-tree", "-p", "-U0", "--no-ext-diff", "--no-textconv", "--no-color", from, to, "--", p)
	if err != nil {
		return "git could not read what it changed"
	}
	var removed, added []string
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "):
		case strings.HasPrefix(line, "-"):
			removed = append(removed, line[1:])
		case strings.HasPrefix(line, "+"):
			added = append(added, line[1:])
		}
	}
	if len(removed) != len(added) {
		return fmt.Sprintf("it changes the text of %q, which this landing does not name", p)
	}
	for i := range removed {
		if withoutLinkDirs(removed[i]) != withoutLinkDirs(added[i]) {
			return fmt.Sprintf("it changes the text of %q, which this landing does not name", p)
		}
	}
	return ""
}
