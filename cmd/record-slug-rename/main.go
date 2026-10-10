// Command record-slug-rename renames every committed record file whose slug is
// longer than the record cap (recordid.MaxSlugLen) to the slug the minting verbs
// would cut today, and rewrites every reference to the old name across the
// tracked tree. It is the one-off half of iss-2610100626320367: the minting verbs
// cut new slugs at the cap, and this brings the records minted before it inside
// the same budget. Ids never change.
//
// It is a maintenance tool, run by hand at the moment of landing, on a fresh
// default branch inside an announced window, because the rename touches records
// peer sessions edit all day. Nothing in CI or the Makefile runs it.
//
//	go run ./cmd/record-slug-rename            # dry run: print the plan, change nothing
//	go run ./cmd/record-slug-rename -apply     # rename, rewrite, stage, report; never commits
//
// What -apply does, in order:
//
//  1. Refuses unless the working tree is clean, so nothing uncommitted is folded
//     into the rename.
//  2. Finds every tracked record of the four families whose filenames carry a
//     minted slug — issues, intents, specs and ADRs, every bucket — whose slug
//     exceeds the cap, and computes the new slug with recordid.CapSlug, the same
//     cut the minting verbs apply to a slug that arrives whole.
//  3. Moves each with `git mv`, and rewrites the `slug:` field of its leading
//     frontmatter block to the same value, so the filename and the field stay
//     one value (record-lint's record_schema and the ledger reader compare them
//     exactly).
//  4. Rewrites every reference to an old name in every tracked text file: the
//     name `<id-part>-<old-slug>` (iss-N-, itd-N-, spc-N-, or an ADR's bare
//     number) where it stands whole — not glued to a letter or a hyphen before
//     it, nor to a letter, digit or hyphen after it. The id part is in the
//     match, so the replacement is the path's name, never the slug alone, and a
//     repository-relative path, a relative link, a bare filename and an anchor
//     all follow. A name glued to more text (a derived file such as
//     `<name>-review.json`) is not a reference to the record and is left.
//     The append-only decision log is never rewritten.
//  5. Stages every change (the tree was clean, so every change is the run's),
//     leaving the commit to whoever runs it.
//  6. Reports per family how many records it renamed, how many references it
//     rewrote in how many files, and every place an old slug still appears
//     anywhere in the tracked tree afterwards, with the reason it was left.
//
// A second run over the committed result finds nothing to do. Exit codes: 0 done,
// or a dry run (leftovers are a report to read, not a failure: most are prose
// quoting an old slug, which names no file); 1 a fault, or a refusal (a dirty
// tree, a name collision, a slug field that disagrees with its filename), with
// nothing changed.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// family is one store whose filenames carry a minted slug.
type family struct {
	prefix     string // the family's name in the report
	fileFamily string // recordid.SplitRecordFilename's family ("" for the ADR store)
	dir        string // the store's root, repository-relative
}

// families are the stores the minting verbs write: capture, intent, spec and
// decide. Every bucket under each root is read.
var families = []family{
	{"iss", "iss", recordid.IssuesRelDir},
	{"itd", "itd", intent.IntentsRelDir},
	{"spc", "spc", spec.SpecsRelDir},
	{"adr", "", recordid.ADRsRelDir},
}

// appendOnly are the tracked files whose existing lines no commit may change
// (the decisions-append gate, DA002). A reference in one is reported, never
// rewritten.
var appendOnly = map[string]bool{".abcd/work/DECISIONS.md": true}

// rename is one record's move.
type rename struct {
	Family  string
	OldRel  string
	NewRel  string
	IDPart  string // the filename before the slug: "iss-N-", "itd-N-", "spc-N-" or "NNNN-"
	OldSlug string
	NewSlug string
}

// Leftover is one place an old slug still appears after the rewrite.
type Leftover struct {
	File string
	Line int
	Slug string
	Why  string
}

// Report is what one run did.
type Report struct {
	Plan       []rename       // the renames, in path order
	Renamed    map[string]int // records renamed, per family
	References int            // references rewritten
	Files      int            // files whose content was rewritten
	Leftovers  []Leftover
}

func (r Report) total() int {
	n := 0
	for _, v := range r.Renamed {
		n += v
	}
	return n
}

func main() {
	apply := flag.Bool("apply", false, "rename and rewrite (default: print the plan and change nothing)")
	rootFlag := flag.String("root", "", "repository root (default: the git toplevel of the working directory)")
	flag.Parse()
	root := *rootFlag
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "record-slug-rename:", err)
			os.Exit(1)
		}
		if root, err = gitutil.Toplevel(wd); err != nil {
			fmt.Fprintln(os.Stderr, "record-slug-rename: not inside a git checkout:", err)
			os.Exit(1)
		}
	}
	rep, err := run(root, *apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, "record-slug-rename:", err)
		os.Exit(1)
	}
	printReport(os.Stdout, rep, *apply)
}

// run plans the rename and, with apply, carries it out.
func run(root string, apply bool) (Report, error) {
	if apply {
		dirty, err := gitutil.Run(root, "status", "--porcelain", "--untracked-files=all")
		if err != nil {
			return Report{}, fmt.Errorf("cannot read the working tree's status: %w", err)
		}
		if dirty != "" {
			return Report{}, errors.New("the working tree is not clean; commit or set aside every change first, so nothing uncommitted is folded into the rename (nothing was changed)")
		}
	}
	plan, err := buildPlan(root)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Plan: plan, Renamed: map[string]int{}}
	for _, r := range plan {
		rep.Renamed[r.Family]++
	}
	if !apply {
		return rep, nil
	}
	for _, r := range plan {
		if err := gitMove(root, r.OldRel, r.NewRel); err != nil {
			return rep, err
		}
		if err := rewriteSlugField(root, r); err != nil {
			return rep, err
		}
	}
	byID := map[string]rename{}
	for _, r := range plan {
		byID[r.IDPart] = r
	}
	files, err := gitutil.TrackedFiles(root)
	if err != nil {
		return rep, err
	}
	for _, rel := range files {
		data, ok, err := readText(root, rel)
		if err != nil {
			return rep, err
		}
		if !ok || appendOnly[rel] {
			continue
		}
		out, n := rewriteRefs(data, byID)
		if n == 0 {
			continue
		}
		if err := writeKeepingMode(root, rel, out); err != nil {
			return rep, err
		}
		rep.References += n
		rep.Files++
	}
	// Stage what the run changed — the tree was clean when it began, so every
	// change is the run's — so the rename is one `git commit` away and a
	// renamed record's rewritten content travels with its move.
	if err := gitStage(root); err != nil {
		return rep, err
	}
	rep.Leftovers, err = findLeftovers(root, plan)
	return rep, err
}

// buildPlan lists every tracked record whose slug exceeds the cap, sorted by
// path, and refuses a plan in which two records, or a record and a tracked
// file, would land on one path.
func buildPlan(root string) ([]rename, error) {
	files, err := gitutil.TrackedFiles(root)
	if err != nil {
		return nil, err
	}
	tracked := make(map[string]bool, len(files))
	for _, f := range files {
		tracked[f] = true
	}
	var plan []rename
	for _, rel := range files {
		for _, fam := range families {
			if !strings.HasPrefix(rel, fam.dir+"/") {
				continue
			}
			base := path.Base(rel)
			_, slug, ok := recordid.SplitRecordFilename(fam.fileFamily, base)
			if !ok || len(slug) <= recordid.MaxSlugLen {
				continue
			}
			// A slug field that disagrees with the filename is refused here,
			// before anything moves, so the rename is never left part way.
			data, _, err := readText(root, rel)
			if err != nil {
				return nil, err
			}
			if _, field, ok := slugLine(data); ok && field != slug {
				return nil, fmt.Errorf("%s: its slug field reads %q, not the filename's %q; record-lint refuses that record already, so fix it before renaming (nothing was changed)", rel, field, slug)
			}
			newSlug := recordid.CapSlug(slug, recordid.MaxSlugLen)
			idPart := strings.TrimSuffix(base, slug+".md")
			plan = append(plan, rename{
				Family: fam.prefix, OldRel: rel,
				NewRel: path.Join(path.Dir(rel), idPart+newSlug+".md"),
				IDPart: idPart, OldSlug: slug, NewSlug: newSlug,
			})
		}
	}
	sort.Slice(plan, func(i, j int) bool { return plan[i].OldRel < plan[j].OldRel })
	seen := map[string]string{}
	ids := map[string]string{}
	for _, r := range plan {
		if tracked[r.NewRel] {
			return nil, fmt.Errorf("%s would be renamed onto %s, which is already tracked (nothing was changed)", r.OldRel, r.NewRel)
		}
		if prev, dup := seen[r.NewRel]; dup {
			return nil, fmt.Errorf("%s and %s would both be renamed to %s (nothing was changed)", prev, r.OldRel, r.NewRel)
		}
		seen[r.NewRel] = r.OldRel
		// The rewrite keys a reference by its id part; two records sharing one
		// would make it ambiguous, so it is refused rather than guessed.
		if prev, dup := ids[r.IDPart]; dup {
			return nil, fmt.Errorf("%s and %s share the name prefix %q, so a reference to either is ambiguous (nothing was changed)", prev, r.OldRel, r.IDPart)
		}
		ids[r.IDPart] = r.OldRel
	}
	return plan, nil
}

// gitMove moves one tracked file with git, so the index records the rename.
func gitMove(root, from, to string) error {
	cmd := exec.Command("git", "-C", root, "mv", "--", from, to)
	cmd.Env = gitutil.IsolatedEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git mv %s %s: %v: %s", from, to, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// gitStage stages every change to a tracked file.
func gitStage(root string) error {
	cmd := exec.Command("git", "-C", root, "add", "--update", "--", ".")
	cmd.Env = gitutil.IsolatedEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add --update: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// rewriteSlugField sets the `slug:` line of the record's leading frontmatter
// block to the new slug, keeping the line's quoting. A record with no such line
// is left as it is: the field is optional for some families, and absence is
// not disagreement. buildPlan has already held the field to the filename.
func rewriteSlugField(root string, r rename) error {
	data, _, err := readText(root, r.NewRel)
	if err != nil {
		return err
	}
	i, _, ok := slugLine(data)
	if !ok {
		return nil
	}
	lines := strings.SplitAfter(string(data), "\n")
	lines[i] = strings.Replace(lines[i], r.OldSlug, r.NewSlug, 1)
	return writeKeepingMode(root, r.NewRel, []byte(strings.Join(lines, "")))
}

// slugLine finds the `slug:` line of a record's leading frontmatter block and
// returns its index among the record's lines and its value, unquoted.
func slugLine(data []byte) (int, string, bool) {
	lines := strings.SplitAfter(string(data), "\n")
	end := frontmatter.Close(lines)
	for i := 1; i < end; i++ {
		if rest, ok := strings.CutPrefix(lines[i], "slug:"); ok {
			return i, strings.Trim(strings.TrimSpace(rest), `"'`), true
		}
	}
	return 0, "", false
}

// idPartRe finds a candidate record-name prefix: a family tag and number, or an
// ADR's bare number, followed by the hyphen the slug hangs from.
var idPartRe = regexp.MustCompile(`(?:(?:iss|itd|spc)-)?[0-9]+-`)

// rewriteRefs replaces every whole occurrence of an old record name with the new
// one and returns the result and the count. The name is matched with its id
// part, then its old slug, and stands whole only when the byte before it is not
// a letter, digit or hyphen and the byte after it is not a letter, digit or
// hyphen; a glued occurrence names something else (a derived file) and is left.
func rewriteRefs(data []byte, byID map[string]rename) ([]byte, int) {
	if len(byID) == 0 {
		return data, 0
	}
	var out bytes.Buffer
	n, pos := 0, 0
	for pos < len(data) {
		loc := idPartRe.FindIndex(data[pos:])
		if loc == nil {
			break
		}
		s, e := pos+loc[0], pos+loc[1]
		r, ok := byID[string(data[s:e])]
		if !ok || (s > 0 && isNameByte(data[s-1])) || !bytes.HasPrefix(data[e:], []byte(r.OldSlug)) {
			// Resume one byte on, not at e: a candidate can begin inside the
			// rejected one ("x12-" holds "12-").
			out.Write(data[pos : s+1])
			pos = s + 1
			continue
		}
		end := e + len(r.OldSlug)
		if end < len(data) && isNameByte(data[end]) {
			out.Write(data[pos:end])
			pos = end
			continue
		}
		out.Write(data[pos:s])
		out.WriteString(r.IDPart + r.NewSlug)
		pos = end
		n++
	}
	out.Write(data[pos:])
	return out.Bytes(), n
}

// isNameByte reports whether b can continue a record name: a letter, a digit or
// a hyphen.
func isNameByte(b byte) bool {
	return b == '-' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// kebabRunRe finds a maximal kebab run, the only shape an old slug can stand in.
var kebabRunRe = regexp.MustCompile(`[a-z0-9]+(?:-[a-z0-9]+)*`)

// findLeftovers reports every place, in content or in a tracked path, where an
// old slug still stands on hyphen boundaries after the rewrite. An old slug is
// longer than the cap and the new one is its strict prefix, so a rewritten
// reference never matches.
func findLeftovers(root string, plan []rename) ([]Leftover, error) {
	old := map[string]bool{}
	for _, r := range plan {
		old[r.OldSlug] = true
	}
	if len(old) == 0 {
		return nil, nil
	}
	// The renamed records are tracked under their new paths by now, so a path
	// still carrying an old slug is a file that is not one of them.
	files, err := gitutil.TrackedFiles(root)
	if err != nil {
		return nil, err
	}
	var out []Leftover
	for _, rel := range files {
		for _, s := range slugsIn(rel, old) {
			out = append(out, Leftover{File: rel, Slug: s, Why: "a tracked path that is not a record names it"})
		}
	}
	for _, rel := range files {
		data, ok, err := readText(root, rel)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		why := "glued to other text, or the slug without its id"
		if appendOnly[rel] {
			why = "an append-only file, never rewritten"
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, s := range slugsIn(line, old) {
				out = append(out, Leftover{File: rel, Line: i + 1, Slug: s, Why: why})
			}
		}
	}
	return out, nil
}

// slugsIn returns each old slug standing in text on hyphen boundaries.
func slugsIn(text string, old map[string]bool) []string {
	var hits []string
	for _, run := range kebabRunRe.FindAllString(text, -1) {
		if len(run) <= recordid.MaxSlugLen {
			continue
		}
		starts := []int{0}
		ends := []int{}
		for i := 0; i < len(run); i++ {
			if run[i] == '-' {
				starts = append(starts, i+1)
				ends = append(ends, i)
			}
		}
		ends = append(ends, len(run))
		for _, a := range starts {
			for _, b := range ends {
				if b-a > recordid.MaxSlugLen && old[run[a:b]] {
					hits = append(hits, run[a:b])
				}
			}
		}
	}
	return hits
}

// readText reads a tracked regular file, reporting ok=false for one that is not
// text (a NUL in its first 8 KiB), a symlink, or absent from the working tree.
func readText(root, rel string) ([]byte, bool, error) {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	fi, err := os.Lstat(abs)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !fi.Mode().IsRegular() {
		return nil, false, nil
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, false, err
	}
	head := data
	if len(head) > 8192 {
		head = head[:8192]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return nil, false, nil
	}
	return data, true, nil
}

// writeKeepingMode rewrites a tracked file in place with its existing mode.
func writeKeepingMode(root, rel string, data []byte) error {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	return os.WriteFile(abs, data, fi.Mode().Perm())
}

func printReport(w io.Writer, rep Report, applied bool) {
	if !applied {
		for _, r := range rep.Plan {
			fmt.Fprintf(w, "%s -> %s\n", r.OldRel, r.NewRel)
		}
		fmt.Fprintf(w, "dry run: %d record(s) would be renamed (iss %d, itd %d, spc %d, adr %d); nothing was changed. Re-run with -apply.\n",
			rep.total(), rep.Renamed["iss"], rep.Renamed["itd"], rep.Renamed["spc"], rep.Renamed["adr"])
		return
	}
	for _, l := range rep.Leftovers {
		fmt.Fprintf(w, "leftover %s:%d: %s (%s)\n", l.File, l.Line, l.Slug, l.Why)
	}
	fmt.Fprintf(w, "renamed %d record(s) (iss %d, itd %d, spc %d, adr %d); rewrote %d reference(s) in %d file(s); %d leftover(s). Nothing is committed.\n",
		rep.total(), rep.Renamed["iss"], rep.Renamed["itd"], rep.Renamed["spc"], rep.Renamed["adr"], rep.References, rep.Files, len(rep.Leftovers))
}
