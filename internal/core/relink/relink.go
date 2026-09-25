// Package relink is the one link-repoint primitive: after a verb MOVES a record
// between status folders, it rewrites every relative markdown link in the
// working tree that named the record's old path so that it names the new one,
// and reports each rewrite.
//
// A record's status is its folder (open/ → closed/, planned/ → shipped/,
// open/ → resolved/), so every lifecycle transition is a rename, and a rename
// strands every relative link that named the file where it was. The verb that
// moves the record is the one place that knows both paths, so it is the place
// the links are repointed — in the same operation, not by a hand survey the next
// gate run discovers was incomplete (iss-2608311127491949,
// iss-2609091732329046). Three classes of link are repointed by the one rule:
//
//   - a link FROM any other file TO a moved record, from any folder — a spec
//     already closed that names ../open/<the closing spec>, an ADR or a plan
//     naming an intent's planned/ path, a draft naming both;
//   - a link FROM a moved record to a file that did not move, which was written
//     from the old folder and resolves against the new one differently — a
//     closing spec's bare link to a sibling still in open/;
//   - a link from a moved record to another record moved in the same operation.
//
// A link is rewritten only when it resolved before the move, so a link that
// never pointed anywhere stays exactly as written: repairing that is not the
// move's business, and a rewrite would disguise it. Links inside fenced code
// and HTML comments are rewritten too — a destination that resolved to the
// moved record is a reference to it wherever it is written, and record-lint's
// links_resolve judges a commented link as much as a live one.
//
// The walk covers the whole working tree's markdown — the durable record, the
// shared working tier and the user-facing docs alike — because a link to a
// record is broken wherever it is written. It never enters .git, the local
// tier (.abcd/.work.local, which is per-checkout scratch), a node_modules
// directory, or a nested checkout (any directory holding its own .git), whose
// files belong to another working tree. Nor does it write into the two
// append-only logs the repository's gates refuse an in-place edit of — the
// decision log (DA002) and the reviews folder (RD002): a link there is history
// as written, and rewriting it would trade a dead link no lint root reads for a
// refused change. It reads and writes through an os.Root, follows no symlink,
// and skips any file past a byte cap.
//
// The package is transport-free: it writes no output, and reports what it did as
// data for the front door to render.
package relink

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// appendOnly names the committed logs no rewrite may touch: the decision log,
// whose gate refuses a removed line below its header (DA002 in
// scripts/check-decisions-append.sh), and the reviews folder, whose gate
// refuses any change to a file after it is created (RD002 in
// scripts/check-reviews.sh).
var appendOnly = map[string]bool{
	".abcd/work/DECISIONS.md": true,
	".abcd/work/reviews":      true,
}

// maxFileBytes caps a markdown file the walk will read and rewrite. The record
// stores cap their own files far below it; a larger file is not a record.
const maxFileBytes = 4 << 20

var (
	// inlineRe is an inline link or image, `[text](dest)`: the SAME shape
	// record-lint's links_resolve checks (internal/core/lint linkRe), so every
	// link the gate would refuse after a move is one this rewrite sees.
	inlineRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	// refDefRe is a link-reference definition, `[label]: dest`, up to three
	// spaces of indent (CommonMark). The gate does not judge these, but one that
	// named the moved record is exactly as broken as an inline link.
	refDefRe = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:[ \t]+(\S+)`)
	// schemeRe marks a destination with a URL scheme (https:, mailto:), which is
	// never a repository path.
	schemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)
)

// Move is one record rename, as repo-relative slash paths.
type Move struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Rewrite is one link destination this package changed.
type Rewrite struct {
	// File is the repo-relative slash path of the file holding the link, at
	// its location after the move.
	File string `json:"file"`
	// Line is the 1-based line the link is on.
	Line int `json:"line"`
	// From is the destination as it was written, and To as it now reads; both
	// carry any #anchor or ?query the link had.
	From string `json:"from"`
	To   string `json:"to"`
}

// Repoint rewrites every relative markdown link under repoRoot that the given
// moves left pointing at an old path, and returns the rewrites in file then
// line order. It runs AFTER the moves: a move whose destination is absent, or
// whose source is still present, did not happen as described and is ignored,
// which is also what makes a second call over the same moves a no-op — a verb
// may re-run it on a retry without double-rewriting anything.
//
// A move naming a path that is not a clean repo-relative slash path is refused
// before anything is read. An error part-way through the walk returns the
// rewrites already written alongside it, so a caller can report both.
func Repoint(repoRoot string, moves []Move) ([]Rewrite, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("relink: opening the repository root: %w", err)
	}
	defer root.Close()

	movedTo, oldPath, err := activeMoves(root, moves)
	if err != nil {
		return nil, err
	}
	if len(movedTo) == 0 {
		return nil, nil
	}
	// A file that names none of the moved records' filenames cannot hold a
	// link to one, so its lines are never parsed. The moved records themselves
	// are always parsed: their own relative links shifted with them.
	var needles []string
	for from := range movedTo {
		needles = append(needles, path.Base(from))
	}

	files, err := markdownFiles(root)
	if err != nil {
		return nil, err
	}
	var out []Rewrite
	for _, rel := range files {
		data, err := fsutil.ReadGuardedInRoot(root, rel, maxFileBytes)
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, fsutil.ErrTooBig) {
			continue
		}
		if err != nil {
			return out, fmt.Errorf("relink: reading %s: %w", rel, err)
		}
		was, moved := oldPath[rel]
		if !moved {
			if !containsAny(data, needles) {
				continue
			}
			was = rel
		}
		updated, rw := rewriteFile(root, rel, was, string(data), movedTo)
		if len(rw) == 0 {
			continue
		}
		if err := fsutil.WriteFileAtomicPreserveModeInRoot(root, rel, []byte(updated)); err != nil {
			return out, fmt.Errorf("relink: writing %s: %w", rel, err)
		}
		out = append(out, rw...)
	}
	return out, nil
}

// activeMoves validates the moves and keeps the ones the tree shows happened:
// the destination is present and the source is gone. It returns the forward map
// (old path → new) and its inverse (new path → old).
func activeMoves(root *os.Root, moves []Move) (map[string]string, map[string]string, error) {
	movedTo := map[string]string{}
	oldPath := map[string]string{}
	for _, m := range moves {
		from, to := filepath.ToSlash(m.From), filepath.ToSlash(m.To)
		if !fsutil.ValidRelPath(from) || !fsutil.ValidRelPath(to) {
			return nil, nil, fmt.Errorf("relink: move %q -> %q must name clean repo-relative paths", m.From, m.To)
		}
		if from == to {
			continue
		}
		if _, err := root.Lstat(from); err == nil {
			continue
		}
		if fi, err := root.Lstat(to); err != nil || !fi.Mode().IsRegular() {
			continue
		}
		movedTo[from] = to
		oldPath[to] = from
	}
	return movedTo, oldPath, nil
}

// markdownFiles lists every markdown file in the tree, as sorted slash paths,
// outside the directories the package comment names. Symlinks are never
// followed: fs.WalkDir reports a symlinked directory as a non-directory entry,
// and only regular files are kept.
func markdownFiles(root *os.Root) ([]string, error) {
	var files []string
	err := fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("relink: walking %s: %w", p, err)
		}
		if d.IsDir() {
			if p == "." {
				return nil
			}
			if skipDir(root, p, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && strings.EqualFold(path.Ext(p), ".md") && !appendOnly[p] {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// skipDir reports whether the walk stays out of a directory: git's own, the
// local tier, an append-only log, a dependency tree, or a nested checkout.
func skipDir(root *os.Root, p, name string) bool {
	if name == ".git" || name == "node_modules" || p == ".abcd/.work.local" || appendOnly[p] {
		return true
	}
	_, err := root.Lstat(p + "/.git")
	return err == nil
}

func containsAny(data []byte, needles []string) bool {
	s := string(data)
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// rewriteFile repoints the links in one file. rel is where the file is now and
// was is where it was before the moves (the same path unless it moved itself):
// a link is resolved against the folder it was written from, mapped through the
// moves, and re-relativised against the folder the file is in now.
func rewriteFile(root *os.Root, rel, was, content string, movedTo map[string]string) (string, []Rewrite) {
	lines := strings.Split(content, "\n")
	var out []Rewrite
	for i, line := range lines {
		var spans [][2]int
		for _, m := range inlineRe.FindAllStringSubmatchIndex(line, -1) {
			spans = append(spans, [2]int{m[2], m[3]})
		}
		if m := refDefRe.FindStringSubmatchIndex(line); m != nil {
			spans = append(spans, [2]int{m[2], m[3]})
		}
		if len(spans) == 0 {
			continue
		}
		sort.Slice(spans, func(a, b int) bool { return spans[a][0] < spans[b][0] })
		var b strings.Builder
		last := 0
		changed := false
		for _, sp := range spans {
			if sp[0] < last {
				continue // overlapping spans: the earlier match owns these bytes
			}
			raw := line[sp[0]:sp[1]]
			repl, from, to, ok := repointDest(root, rel, was, raw, movedTo)
			if !ok {
				continue
			}
			b.WriteString(line[last:sp[0]])
			b.WriteString(repl)
			last = sp[1]
			changed = true
			out = append(out, Rewrite{File: rel, Line: i + 1, From: from, To: to})
		}
		if changed {
			b.WriteString(line[last:])
			lines[i] = b.String()
		}
	}
	return strings.Join(lines, "\n"), out
}

// repointDest returns the rewritten destination text for one link, the
// destination before and after (anchor included, whitespace and title not),
// and whether anything changed.
func repointDest(root *os.Root, rel, was, raw string, movedTo map[string]string) (string, string, string, bool) {
	trimmed := strings.TrimLeft(raw, " \t")
	lead := raw[:len(raw)-len(trimmed)]
	dest, tail := trimmed, ""
	if k := strings.IndexAny(trimmed, " \t"); k >= 0 {
		dest, tail = trimmed[:k], trimmed[k:]
	}
	if dest == "" || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/") ||
		strings.HasPrefix(dest, "<") || schemeRe.MatchString(dest) {
		return "", "", "", false
	}
	p, suffix := dest, ""
	if k := strings.IndexAny(dest, "#?"); k >= 0 {
		p, suffix = dest[:k], dest[k:]
	}
	if p == "" {
		return "", "", "", false
	}
	resolved := path.Join(path.Dir(was), p)
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", "", "", false
	}
	target, recordMoved := movedTo[resolved]
	if !recordMoved {
		if was == rel {
			return "", "", "", false
		}
		// Only the linking file moved. A link that resolved to nothing from
		// where it was written is left exactly as it is.
		if _, err := root.Stat(resolved); err != nil {
			return "", "", "", false
		}
		target = resolved
	}
	np, err := filepath.Rel(filepath.FromSlash(path.Dir(rel)), filepath.FromSlash(target))
	if err != nil {
		return "", "", "", false
	}
	np = filepath.ToSlash(np)
	if strings.HasPrefix(p, "./") && !strings.HasPrefix(np, "../") {
		np = "./" + np
	}
	if np == p {
		return "", "", "", false
	}
	return lead + np + suffix + tail, p + suffix, np + suffix, true
}
