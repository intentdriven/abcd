package claim

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The index of unshown announcements (iss-2610161839527371). Each prompt used
// to read every kept announcement to find the few it had not shown. The
// intake now names each announcement it takes in, in a file of its own beside
// the kept ones; the prompt hook reads those files and the announcements they
// name, and removes a file once every announcement in it is shown or can
// never be. `abcd status` reads every kept announcement, as before.
//
// Under the status state directory's unshownDir:
//
//   - new-<when>-<random>: the commits one intake took in, one per line. A
//     file is written whole and never changed, so an intake and a prompt
//     running side by side lose nothing: a prompt removes only files it read.
//   - to-<seconds>: the latest first fetch among the announcements whose
//     files were removed. An announcement first fetched after it is newer
//     than every one that left the index, so the index alone orders it; one
//     that is not sends the prompt back to the full read (unshownRead.widen).
//   - ready: the index holds every announcement kept here. Until it exists,
//     what an earlier build kept is listed first (unshownIndexReady).

// The index's directory and the names in it.
const (
	unshownDir    = "unshown"
	unshownReady  = "ready"
	unshownPrefix = "new-"
	waterPrefix   = "to-"
)

// maxIndexBytes bounds one index file a prompt reads: the first lists at most
// maxKeptAnnouncements commits, and every other at most one intake's.
const maxIndexBytes = 1 << 20

// unshownIndexReady reports whether the index of unshown announcements holds
// every announcement kept here, building it when it does not: the directory,
// then one file naming what is kept already (as the full read takes it, the
// newest maxKeptAnnouncements), then the marker. An index that cannot be
// built answers false, and the caller reads every kept announcement instead.
func unshownIndexReady(r *os.Root, root string) bool {
	dir := statusFile(unshownDir)
	if info, err := r.Lstat(dir + "/" + unshownReady); err == nil && info.Mode().IsRegular() {
		return true
	}
	if err := mkStateDir(r, dir); err != nil {
		return false
	}
	if names, _ := keptNames(root); len(names) > 0 {
		if !indexCommits(r, names) {
			return false
		}
	}
	return writeUnflushedAt(r, dir, unshownReady, nil) == nil
}

// indexCommits names commits in the index, in one new file, unflushed.
func indexCommits(r *os.Root, commits []string) bool {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return false
	}
	// Named by when it was written, so the newest files sort last.
	name := fmt.Sprintf("%s%020d-%s", unshownPrefix, time.Now().UnixNano(), hex.EncodeToString(buf[:]))
	return writeUnflushedAt(r, statusFile(unshownDir), name, []byte(strings.Join(commits, "\n")+"\n")) == nil
}

// unindex marks the index as not holding every kept announcement, after a
// write to it failed, so the next reader builds it again from what is kept.
func unindex(r *os.Root) {
	_ = r.Remove(statusFile(unshownDir + "/" + unshownReady))
}

// unshownRead is one prompt's read of the index.
type unshownRead struct {
	root string
	// held is every status branch this checkout holds (intake.held).
	held map[string]bool
	// files is each index file read whole, by name, with the commits it
	// names: one the cap cut short is left out, and so never removed.
	files map[string][]string
	// dead is every commit read that can never be shown: its kept entry does
	// not read, or its life ended.
	dead map[string]bool
	// water is the latest first fetch among the announcements that left the
	// index, and waters the files that say so.
	water  time.Time
	waters []string
	// mark is the latest first fetch among the announcements this read took
	// from the index.
	mark time.Time
}

// unshownAnnouncements is the prompt's read of the kept announcements: those
// the index names, read on a branch in held, that still stand at now or are
// unreadable and first read in the last day, in keptAnnouncements' order, and
// how many more the index names past maxKeptAnnouncements, the newest taken
// in read first. The answer is nil when the index cannot be read or built:
// the caller then reads every kept announcement.
func unshownAnnouncements(ctx context.Context, root string, now time.Time, held map[string]bool) ([]Announcement, int, *unshownRead, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, 0, nil, nil
	}
	ready := unshownIndexReady(r, root)
	r.Close()
	if !ready {
		return nil, 0, nil, nil
	}
	x := &unshownRead{root: root, held: held, files: map[string][]string{}, dead: map[string]bool{}}
	var names []string
	for _, e := range statusEntries(root, unshownDir) {
		name := e.Name()
		if !e.Type().IsRegular() {
			continue
		}
		if s, ok := strings.CutPrefix(name, waterPrefix); ok {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
				x.waters = append(x.waters, name)
				if at := time.Unix(n, 0).UTC(); at.After(x.water) {
					x.water = at
				}
			}
			continue
		}
		if strings.HasPrefix(name, unshownPrefix) {
			names = append(names, name)
		}
	}
	// The newest taken in first, as the names order them.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	var out []Announcement
	past := 0
	read := map[string]bool{}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, 0, nil, err
		}
		data := readStatusFile(root, unshownDir+"/"+name, maxIndexBytes)
		if data == nil {
			continue
		}
		var commits []string
		whole := true
		for _, c := range strings.Split(string(data), "\n") {
			if !gitutil.IsFullSHA(c) {
				continue
			}
			commits = append(commits, c)
			if read[c] {
				continue
			}
			if len(read) >= maxKeptAnnouncements {
				past++
				whole = false
				continue
			}
			read[c] = true
			a, ok := keptAnnouncement(root, c)
			if !ok {
				x.dead[c] = true
				continue
			}
			if a.FirstFetched.After(x.mark) {
				x.mark = a.FirstFetched
			}
			switch {
			case a.Unreadable != "" && now.Sub(a.FirstFetched) >= MaxAnnouncementLife, a.Unreadable == "" && !a.Standing(now):
				x.dead[c] = true
			case held != nil && !held[a.Branch]:
				// Not shown while its branch is not held here, and kept in
				// the index: a fetch may bring the branch back.
			default:
				out = append(out, a)
			}
		}
		if whole {
			x.files[name] = commits
		}
	}
	sortAnnouncements(out)
	return out, past, x, nil
}

// widen is as, the index's announcements, when the index alone orders each
// one not yet shown: every such one was first fetched after everything that
// left the index. When one was not (a later intake took in what an earlier
// one left beneath a commit already shown, or two intakes fell in one
// second), an announcement no longer in the index may be newer than it for
// the same item, so the answer is every kept announcement, with as's.
func (x *unshownRead) widen(now time.Time, shown map[string]bool, as []Announcement) []Announcement {
	behind := false
	for _, a := range as {
		if !shown[a.Commit] && !a.FirstFetched.After(x.water) {
			behind = true
			break
		}
	}
	if !behind {
		return as
	}
	all, _ := keptAnnouncementsCounted(x.root, now, x.held)
	have := map[string]bool{}
	for _, a := range all {
		have[a.Commit] = true
	}
	for _, a := range as {
		if !have[a.Commit] {
			all = append(all, a)
		}
	}
	sortAnnouncements(all)
	return all
}

// settle sorts the index files this read took whole, once a block is built
// from as with keys the lines it shows or counts: a file whose every commit
// is already kept shown, or can never be shown, is removed now; one whose
// last commits this block shows is answered, for MarkShown to remove once the
// block is kept shown. It answers too the first fetch the index's mark must
// reach before those are removed.
func (x *unshownRead) settle(shown map[string]bool, keys []string, as []Announcement) (spent []string, mark time.Time) {
	mark = x.mark
	for _, a := range as {
		if a.FirstFetched.After(mark) {
			mark = a.FirstFetched
		}
	}
	keyed := make(map[string]bool, len(keys))
	for _, k := range keys {
		keyed[k] = true
	}
	var gone []string
	for name, commits := range x.files {
		isGone, isSpent := true, true
		for _, c := range commits {
			switch {
			case x.dead[c] || shown[c]:
			case keyed[c]:
				isGone = false
			default:
				isGone, isSpent = false, false
			}
		}
		switch {
		case isGone:
			gone = append(gone, name)
		case isSpent:
			spent = append(spent, name)
		}
	}
	x.remove(gone, mark)
	sort.Strings(spent)
	return spent, mark
}

// remove removes the index files named, best effort, once the index's mark
// has reached mark: with the mark not kept, nothing is removed, and the files
// are read again at the next prompt.
func (x *unshownRead) remove(names []string, mark time.Time) {
	if len(names) == 0 {
		return
	}
	r, err := os.OpenRoot(x.root)
	if err != nil {
		return
	}
	defer r.Close()
	dir := statusFile(unshownDir)
	if mark.After(x.water) {
		name := waterPrefix + strconv.FormatInt(mark.Unix(), 10)
		if err := writeUnflushedAt(r, dir, name, nil); err != nil {
			return
		}
		// Every mark kept before is beneath this one.
		for _, old := range x.waters {
			if old != name {
				_ = r.Remove(dir + "/" + old)
			}
		}
		x.water, x.waters = time.Unix(mark.Unix(), 0).UTC(), []string{name}
	}
	for _, name := range names {
		_ = r.Remove(dir + "/" + name)
	}
}
