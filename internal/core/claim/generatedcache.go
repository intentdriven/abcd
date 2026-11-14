package claim

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// The generated set kept between reads (iss-2610170833428011). Reading it
// lists every file on origin's default branch and matches each against the
// root .gitattributes, about 0.3 s on a tree of 150,000 files and up to the
// matching budget, and every reservation view with a live footprint read it
// again. The answer depends only on what the key below names, so the answer
// for one key is kept in this checkout's local tier and a later read under
// the same key takes it instead of listing and matching again: a view pays
// the match once each time origin's default branch moves to another tree.
//
// The key is a digest of:
//   - the tree of origin's default branch, as stored: it holds the root
//     .gitattributes and every path and mode the match reads, and the read
//     goes by that tree, never the ref, so the kept answer is that tree's
//     alone, whatever the ref does while it is read;
//   - the ref, whose name the reasons quote;
//   - the matching budgets (generatedSteps, matchSteps), which decide whether
//     the match finishes;
//   - the running binary as the process found it at start (generatedReader):
//     its path, size and time written, with the version and the source
//     revision its build stamped when it carries them, so a build whose reader
//     reads the declaration differently never takes another build's answer;
//     and
//   - the format of the kept file.
//
// Nothing else is read: no checkout config, no case-folding of this file
// system, and git's version changes neither the listing's -z form nor the
// tree. Only a finished read is kept: the set, or a match that ran past its
// budget, which is as certain an answer for the key as the set is. A read
// that failed on git, or stopped at a declaration or a listing it would not
// read, keeps nothing, so a passing fault is never remembered and the cheap
// refusals are made again.
//
// The kept file is untrusted input. It is read only as a regular file within
// generatedKeptCap, never through a symbolic link, never when git tracks it
// (localtier.ReadInRoot), and anything in it that is not an answer for the
// key in hand reads as nothing kept, as any fault in reading it does, and
// the answer is read afresh. A tracked copy is refused, so only this checkout
// supplies it, and it is held to no more than the checkout's own git
// directory, which decides the tree it is keyed on. It holds at most
// generatedKeptEntries paths, each taken only as the clean relative ASCII
// path a listing gives, and the reason a kept overrun gives is written here,
// never read from the file. Writing it is best-effort: atomic, owner-only,
// and a failure leaves the reader's answer as it was.

const (
	// generatedKeptRel is the kept file, in the reservations' state.
	generatedKeptRel = stateRelDir + "/generated"
	// generatedKeptCap bounds the kept file read or written; a set past it is
	// not kept.
	generatedKeptCap = 16 << 20
	// generatedKeptMagic and generatedKeptFormat open the kept file; a change
	// to its form changes the format.
	generatedKeptMagic  = "abcd generated set"
	generatedKeptFormat = "1"
	// The kept outcomes: the set, or a match past its budget.
	keptSet   = "set"
	keptSpent = "spent"
)

// minListingEntry is the fewest bytes one file takes in an ls-tree -r -z
// listing: "100644 blob ", the shortest object name (40 hex digits), a tab,
// a one-byte path and the NUL.
const minListingEntry = len("100644 blob ") + 40 + len("\t") + 1 + 1

// generatedKeptEntries bounds the paths a kept file holds, read or written:
// the most files a listing within treeListCap can name, so no set a read
// computes is past it, and a file naming more is not an answer. A variable,
// so a test can reach its edge.
var generatedKeptEntries = treeListCap / minListingEntry

// generatedReader names the running binary (readerOf) as the process found it
// at start, "" when it cannot be named: then nothing is kept or taken. A test
// replaces it.
var generatedReader = sync.OnceValue(func() string {
	exe, fi := vintage.ExecutableAtStart()
	return readerOf(exe, fi, core.Version, vintage.CurrentBuildVintage())
})

// readerOf names the binary at exe, as fi describes it, built at version and
// cur: its path, size and time written, and the version and source revision
// its build stamped. A build without them (one made without its source
// control stamped, say) is named by the rest; "" when fi is not a regular
// file.
func readerOf(exe string, fi os.FileInfo, version string, cur vintage.Current) string {
	if exe == "" || fi == nil || !fi.Mode().IsRegular() {
		return ""
	}
	return fmt.Sprintf("%s\x00%d\x00%d\x00%s\x00%s\x00%t", exe, fi.Size(), fi.ModTime().UnixNano(), version, cur.Revision, cur.Known)
}

// defaultTree is the tree ref names, read as stored, "" when it cannot be
// named.
func defaultTree(g viewGit, root, ref string) string {
	tree, err := g.stored(root, 256, "rev-parse", "--verify", "--quiet", ref+"^{tree}", "--")
	if err != nil || !gitutil.IsFullSHA(tree) {
		return ""
	}
	return tree
}

// generatedKey is the key the generated set on tree, ref's tree, is kept
// under, "" when the binary cannot be named.
func generatedKey(ref, tree string) string {
	reader := generatedReader()
	if reader == "" {
		return ""
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "%s\x00%s\x00%s\x00%s\x00%d\x00%d",
		generatedKeptFormat, reader, ref, tree, generatedSteps, matchSteps))
	return hex.EncodeToString(sum[:])
}

// ctxOf is g's context, the background one when g holds none.
func ctxOf(g viewGit) context.Context {
	if g.bound() {
		return g.ctx
	}
	return context.Background()
}

// keptGenerated is the answer kept under key: the set, or spent for a match
// past its budget; ok is false when nothing usable is kept.
func keptGenerated(g viewGit, root, key string) (files map[string]bool, spent, ok bool) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, false, false
	}
	defer r.Close()
	data, err := localtier.ReadInRootContext(ctxOf(g), r, generatedKeptRel, generatedKeptCap)
	if err != nil {
		return nil, false, false
	}
	return decodeKept(data, key)
}

// keepGenerated keeps the answer under key, best-effort: nothing is kept in
// a checkout without its local tier, where git tracks the kept file, or for
// a set past either cap.
func keepGenerated(g viewGit, root, key string, files map[string]bool, spent bool) {
	if len(files) > generatedKeptEntries {
		return
	}
	data := encodeKept(key, files, spent)
	if len(data) > generatedKeptCap || tierPresent(root) != nil {
		return
	}
	if localtier.UntrackedContext(ctxOf(g), root, generatedKeptRel) != nil {
		return
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	if mkStateDir(r, stateRelDir) != nil {
		return
	}
	_ = fsutil.WriteRebuildableInRoot(r, generatedKeptRel, data, stateMode)
}

// encodeKept is the kept file: the magic, the format, the key and the
// outcome, then each path in order, every field ended by a NUL, which no
// path holds.
func encodeKept(key string, files map[string]bool, spent bool) []byte {
	var b bytes.Buffer
	outcome := keptSet
	if spent {
		outcome = keptSpent
	}
	for _, f := range []string{generatedKeptMagic, generatedKeptFormat, key, outcome} {
		b.WriteString(f)
		b.WriteByte(0)
	}
	paths := make([]string, 0, len(files))
	for p, in := range files {
		if in {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	for _, p := range paths {
		b.WriteString(p)
		b.WriteByte(0)
	}
	return b.Bytes()
}

// decodeKept reads a kept file for key; ok is false for anything else: a
// file of another form or key, a spent outcome with paths, more paths than
// generatedKeptEntries, or a path no listing gives.
func decodeKept(data []byte, key string) (files map[string]bool, spent, ok bool) {
	text, ended := strings.CutSuffix(string(data), "\x00")
	if !ended {
		return nil, false, false
	}
	// The four fields before the paths, then the paths: a file naming more
	// than generatedKeptEntries is refused before it is split.
	if strings.Count(text, "\x00") > 3+generatedKeptEntries {
		return nil, false, false
	}
	f := strings.Split(text, "\x00")
	if len(f) < 4 || f[0] != generatedKeptMagic || f[1] != generatedKeptFormat || f[2] != key {
		return nil, false, false
	}
	switch f[3] {
	case keptSpent:
		return nil, true, len(f) == 4
	case keptSet:
	default:
		return nil, false, false
	}
	files = make(map[string]bool, len(f)-4)
	for _, p := range f[4:] {
		if !listedPath(p) {
			return nil, false, false
		}
		files[p] = true
	}
	return files, false, true
}

// listedPath reports whether p is a path an ls-tree listing could name as a
// candidate: ASCII, relative and clean.
func listedPath(p string) bool {
	return p != "" && isASCII(p) && path.Clean(p) == p && !path.IsAbs(p) &&
		p != "." && p != ".." && !strings.HasPrefix(p, "../")
}
