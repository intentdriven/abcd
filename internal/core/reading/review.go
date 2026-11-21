package reading

import (
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// review.go is the include table's answer for a model-matrix reviewer's
// request (adr-2610161733400551, spc-2610161949559050 step 5). A reviewer is
// handed one copy's normalised diff and nothing else, assembled through this
// table rather than a second copy of it: a path the diff touches travels only
// where a row that reads the shipped tree reaches it, labelled with that row's
// material class, so the structural deny holds for a review as it does for a
// reading. The record (`.abcd` at any depth, where a copy's run state and its
// decisions would name its model), the git directory, the agents, the evals
// and this package never travel.
//
// The rows asked are the detection position's rows rooted at the repository
// (Source "."), in table order, so the first row that reaches a path owns it,
// exactly as the walk labels it. The record rows that name a family's leaf
// bucket individually are left out on purpose: they escape the deny for a
// reading of the record, and a reviewer is judging code, not the record.

// reviewPosition is the position whose shipped-tree rows a review is asked
// through: the one that reads the shipped tree as its object.
const reviewPosition = PositionDetection

// ReviewKind reports the material class a path of a reviewed diff travels as,
// and false when the include table admits no such path. rel must be a clean,
// slash-separated path relative to the repository's root, printable and
// without a control character; any other spelling is refused rather than
// cleaned, so a path cannot be spelt into a row it would not reach.
func ReviewKind(rel string) (Kind, bool) {
	if !cleanTreePath(rel) {
		return "", false
	}
	for _, row := range Table {
		if row.Source == "." && row.AdmittedAt(reviewPosition) && row.Reaches(rel) {
			return row.Kind, true
		}
	}
	return "", false
}

// cleanTreePath reports whether rel is a path a tree can hold, as written: not
// empty, not absolute, already clean (no `.` or `..` segment, no doubled or
// trailing slash), valid UTF-8, and free of control characters.
func cleanTreePath(rel string) bool {
	if rel == "" || !utf8.ValidString(rel) || strings.HasPrefix(rel, "/") || path.Clean(rel) != rel ||
		rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "\\") {
		return false
	}
	for _, r := range rel {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
