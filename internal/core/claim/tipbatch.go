package claim

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// A view reads every reservation tip it lists, and read one at a time each
// tip cost three git processes: its name, its size and its body
// (iss-2610180545134593). readSeenTips reads the ordinary ones, commits
// within the object bound, in three processes however many there are, and
// every other tip exactly as before.
//
// The tips come from origin, written by whoever can push to the reservation
// namespace, so the batch is read as untrusted. It is a shortcut and nothing
// more: a body is taken from it only under the name and the size the check
// read, and any tip the batch does not answer that way (missing, not a
// commit, past the bound, answered out of shape) is read on its own by
// readSeen, with every bound and refusal that read applies. A tip the batch
// cannot read never shifts another's body under its name, since the cut stops
// at the first answer it cannot read. Nothing it reads outlives the call: a
// ref is resolved to its commit afresh every time.

// tipRead is one tip as readSeen reads it.
type tipRead struct {
	t   tip
	ok  bool
	err error
}

// batchWant is one commit the batch asks for, with the size its check read.
type batchWant struct {
	name string
	size int
}

// readSeenTips reads each of refs, reservation refs under TrackingPrefix
// that the caller has already vetted, in order, each as readSeen(root, id,
// ref, tree, fetchedAt, keep) reads it: the same tip, the same ok and the
// same error.
func readSeenTips(root string, refs []string, tree string, fetchedAt time.Time, keep bool) []tipRead {
	bodies := batchedTipBodies(root, refs)
	out := make([]tipRead, len(refs))
	for i, ref := range refs {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		if b, ok := bodies[ref]; ok {
			// As readTip reads a commit within the bound: parsed, then dated.
			h, m := parseTip(b.name, []byte(b.body), tree, reservationParser(id))
			t := tip{head: dated(root, h, fetchedAt, keep), message: m}
			out[i] = tipRead{t: seen(root, t, fetchedAt, keep), ok: true}
			continue
		}
		t, ok, err := readSeen(root, id, ref, tree, fetchedAt, keep)
		out[i] = tipRead{t: t, ok: ok, err: err}
	}
	return out
}

// tipBody is a tip's commit and its object as git stored it.
type tipBody struct {
	name string
	body string
}

// batchedTipBodies reads, for each of refs that names a commit within the
// object bound, that commit's object, in three commands: the refs' names and
// types, the commits' types and sizes from their headers (`cat-file
// --batch-check`), then their bodies (`cat-file --batch`), each read as
// stored and capped. A ref it reads no body for is left out, for the caller
// to read on its own; a command that fails leaves out every ref it would have
// read, never refuses the read.
func batchedTipBodies(root string, refs []string) map[string]tipBody {
	if len(refs) == 0 {
		return nil
	}
	asked := make(map[string]bool, len(refs))
	for _, r := range refs {
		asked[r] = true
	}
	listing, err := storedGit(root, 1<<20, "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)", TrackingPrefix)
	if err != nil {
		return nil
	}
	commitOf := map[string]string{}
	var shas []string
	inBatch := map[string]bool{}
	for _, l := range strings.Split(listing, "\n") {
		f := strings.Fields(l)
		// Only a ref the caller asked for, named exactly, whose object git
		// names in full and calls a commit.
		if len(f) != 3 || !asked[f[0]] || f[2] != "commit" || !gitutil.IsFullSHA(f[1]) {
			continue
		}
		commitOf[f[0]] = f[1]
		if !inBatch[f[1]] {
			inBatch[f[1]] = true
			shas = append(shas, f[1])
		}
	}
	if len(shas) == 0 {
		return nil
	}
	ctx := context.Background()
	check, err := storedGitInput(ctx, root, strings.Join(shas, "\n")+"\n", len(shas)*(len(shas[0])+32)+64,
		"cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return nil
	}
	var wants []batchWant
	total := 64
	for _, l := range strings.Split(strings.TrimRight(string(check), "\n"), "\n") {
		f := strings.Fields(l)
		if len(f) != 3 || !inBatch[f[0]] || f[1] != "commit" {
			continue
		}
		n, err := strconv.Atoi(f[2])
		if err != nil || n < 0 || n > maxObjectBytes {
			continue // read on its own, which says why it is unreadable
		}
		wants = append(wants, batchWant{name: f[0], size: n})
		total += n + len(f[0]) + 32
	}
	if len(wants) == 0 {
		return nil
	}
	names := make([]string, len(wants))
	for i, w := range wants {
		names[i] = w.name
	}
	objs, err := storedGitInput(ctx, root, strings.Join(names, "\n")+"\n", total, "cat-file", "--batch")
	if err != nil {
		return nil
	}
	bodies := splitCommitBatch(string(objs), wants)
	out := make(map[string]tipBody, len(bodies))
	for ref, sha := range commitOf {
		if b, ok := bodies[sha]; ok {
			out[ref] = tipBody{name: sha, body: b}
		}
	}
	return out
}

// splitCommitBatch cuts git's `cat-file --batch` answer into bodies, keyed by
// object name, for the wants it answers, in order, as commits of exactly the
// size the check read. A want answered as missing costs that want alone. The
// cut stops at the first answer it cannot read as the want it is for (another
// name, type or size, a short read, a body not ended by its newline), so a
// mangled answer never puts a later body under the wrong name: every want
// from there on is left out, for its caller to read on its own.
func splitCommitBatch(out string, wants []batchWant) map[string]string {
	got := make(map[string]string, len(wants))
	rest := out
	for _, w := range wants {
		header, after, ok := strings.Cut(rest, "\n")
		if !ok {
			break
		}
		if header == w.name+" missing" {
			rest = after
			continue
		}
		if header != w.name+" commit "+strconv.Itoa(w.size) || len(after) < w.size+1 || after[w.size] != '\n' {
			break
		}
		got[w.name] = after[:w.size]
		rest = after[w.size+1:]
	}
	return got
}
