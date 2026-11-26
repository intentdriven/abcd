package fsutil

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// exclusiveRoot opens a fresh temp directory as a root for one test.
func exclusiveRoot(t *testing.T) (*os.Root, string) {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return root, dir
}

// onlyEntries fails the test unless dir holds exactly the names want, so a temp
// name left behind is caught whatever it is called.
func onlyEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range ents {
		got = append(got, e.Name())
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("directory holds %q, want exactly %q (a temp name was left behind, or the file is missing)", got, want)
	}
}

// swapExclusiveWrite replaces the content write for one test.
func swapExclusiveWrite(t *testing.T, fn func(f *os.File, data []byte) error) {
	t.Helper()
	prev := exclusiveWrite
	exclusiveWrite = fn
	t.Cleanup(func() { exclusiveWrite = prev })
}

// halfThen writes the first half of the content and then calls stop, which
// sees the directory exactly as a process killed at that instant leaves it.
func halfThen(stop func() error) func(f *os.File, data []byte) error {
	return func(f *os.File, data []byte) error {
		if _, err := f.Write(data[:len(data)/2]); err != nil {
			return err
		}
		return stop()
	}
}

var errKilled = errors.New("the write stopped part-way")

// TestCreateExclusiveInWriteStoppedPartWayLeavesNoFile: a write-once file must
// appear only complete. A crash part-way through the content leaves nothing at
// the final name, so the next exclusive create of it is not refused by a short
// file; a write that fails part-way removes what it made.
func TestCreateExclusiveInWriteStoppedPartWayLeavesNoFile(t *testing.T) {
	root, dir := exclusiveRoot(t)
	data := bytes.Repeat([]byte("record "), 512)
	swapExclusiveWrite(t, halfThen(func() error {
		if _, err := os.Lstat(dir + "/rec.md"); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("part-way through the write the final name already stands (lstat: %v): a crash here leaves a short record no exclusive create can replace", err)
		}
		return errKilled
	}))
	if err := CreateExclusiveIn(root, "rec.md", data, 0o644); !errors.Is(err, errKilled) {
		t.Fatalf("CreateExclusiveIn = %v, want the write's own error", err)
	}
	onlyEntries(t, dir)

	exclusiveWrite = func(f *os.File, data []byte) error { _, err := f.Write(data); return err }
	if err := CreateExclusiveIn(root, "rec.md", data, 0o644); err != nil {
		t.Fatalf("a create after the stopped write is refused: %v", err)
	}
	if got, _ := os.ReadFile(dir + "/rec.md"); !bytes.Equal(got, data) {
		t.Errorf("the record holds %d bytes, want the whole %d", len(got), len(data))
	}
	onlyEntries(t, dir, "rec.md")
}

// TestCreateExclusiveInRefusesAnExistingName: the exclusive contract holds. An
// existing name, a dangling symlink included, is refused with an error that is
// fs.ErrExist and names the caller's path, the file is left as it was, and no
// temp name is left behind.
func TestCreateExclusiveInRefusesAnExistingName(t *testing.T) {
	root, dir := exclusiveRoot(t)
	if err := os.WriteFile(dir+"/rec.md", []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nowhere", dir+"/dangling"); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"rec.md", "dangling"} {
		err := CreateExclusiveIn(root, rel, []byte("second"), 0o644)
		if !errors.Is(err, fs.ErrExist) || !os.IsExist(err) {
			t.Fatalf("CreateExclusiveIn over %s = %v, want fs.ErrExist", rel, err)
		}
		var pe *fs.PathError
		if !errors.As(err, &pe) || pe.Path != rel {
			t.Errorf("the refusal %v does not name %q as its path", err, rel)
		}
	}
	if got, _ := os.ReadFile(dir + "/rec.md"); string(got) != "first" {
		t.Errorf("the existing file now holds %q, want it untouched", got)
	}
	onlyEntries(t, dir, "dangling", "rec.md")
}

// TestCreateExclusiveInOneWinnerWhileAWriteIsInFlight: a creator refused by the
// name never reads a file another creator is still writing. While the first
// creator is part-way through, the second wins; the first is then refused, and
// the record holds the winner's whole content.
func TestCreateExclusiveInOneWinnerWhileAWriteIsInFlight(t *testing.T) {
	root, dir := exclusiveRoot(t)
	slow := bytes.Repeat([]byte("a"), 4096)
	fast := bytes.Repeat([]byte("b"), 4096)
	halfway, resume := make(chan struct{}), make(chan struct{})
	var first error
	var wg sync.WaitGroup
	wg.Add(1)
	swapExclusiveWrite(t, func(f *os.File, data []byte) error {
		if data[0] == 'a' {
			if _, err := f.Write(data[:len(data)/2]); err != nil {
				return err
			}
			close(halfway)
			<-resume
			_, err := f.Write(data[len(data)/2:])
			return err
		}
		_, err := f.Write(data)
		return err
	})
	go func() {
		defer wg.Done()
		first = CreateExclusiveIn(root, "rec.md", slow, 0o644)
	}()
	<-halfway
	second := CreateExclusiveIn(root, "rec.md", fast, 0o644)
	got, _ := os.ReadFile(dir + "/rec.md")
	close(resume)
	wg.Wait()
	if second != nil {
		t.Errorf("the second creator was refused (%v) by a file still being written; it read %d of %d bytes", second, len(got), len(slow))
	}
	if !errors.Is(first, fs.ErrExist) && second == nil {
		t.Errorf("the first creator = %v after the second won, want fs.ErrExist", first)
	}
	final, _ := os.ReadFile(dir + "/rec.md")
	if !bytes.Equal(final, fast) && !bytes.Equal(final, slow) {
		t.Errorf("the record holds %d bytes of mixed or short content, want one creator's whole content", len(final))
	}
	onlyEntries(t, dir, "rec.md")
}

// TestCreateExclusiveInConcurrentCreatorsOneWinner: of many creators racing for
// one name exactly one wins, every other is refused with fs.ErrExist, the file
// holds the winner's whole content, and no temp name is left behind.
func TestCreateExclusiveInConcurrentCreatorsOneWinner(t *testing.T) {
	root, dir := exclusiveRoot(t)
	const n = 16
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = CreateExclusiveIn(root, "rec.md", bytes.Repeat([]byte{byte('A' + i)}, 64<<10), 0o644)
		}()
	}
	close(start)
	wg.Wait()
	winner := -1
	for i, err := range errs {
		switch {
		case err == nil:
			if winner >= 0 {
				t.Fatalf("creators %d and %d both won", winner, i)
			}
			winner = i
		case !errors.Is(err, fs.ErrExist):
			t.Errorf("creator %d = %v, want nil or fs.ErrExist", i, err)
		}
	}
	if winner < 0 {
		t.Fatal("no creator won")
	}
	got, _ := os.ReadFile(dir + "/rec.md")
	if !bytes.Equal(got, bytes.Repeat([]byte{byte('A' + winner)}, 64<<10)) {
		t.Errorf("the record holds %d bytes that are not the winner's whole content", len(got))
	}
	onlyEntries(t, dir, "rec.md")
}

// TestCreateExclusiveInKeepsPermAndContent: the success path leaves exactly the
// final name, at the caller's mode under the umask, holding the content.
func TestCreateExclusiveInKeepsPermAndContent(t *testing.T) {
	root, dir := exclusiveRoot(t)
	old := syscall.Umask(0o022)
	defer syscall.Umask(old)
	if err := CreateExclusiveIn(root, "rec.md", []byte("whole"), 0o666); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(dir + "/rec.md")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 || !fi.Mode().IsRegular() {
		t.Errorf("mode %v, want a regular file at 0644 (0666 under umask 022)", fi.Mode())
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Nlink != 1 {
		t.Errorf("the record has %d links, want 1 (the temp name still links it)", st.Nlink)
	}
	if got, _ := os.ReadFile(dir + "/rec.md"); string(got) != "whole" {
		t.Errorf("content %q, want %q", got, "whole")
	}
	onlyEntries(t, dir, "rec.md")
}

// TestCreateExclusiveInWithoutHardLinks: on a filesystem with no hard links the
// file still appears only once its content is whole, the exclusive contract
// still holds, and no temp name is left behind.
func TestCreateExclusiveInWithoutHardLinks(t *testing.T) {
	root, dir := exclusiveRoot(t)
	prev := linkIn
	linkIn = func(r *os.Root, oldname, newname string) error {
		return &os.LinkError{Op: "linkat", Old: oldname, New: newname, Err: syscall.ENOTSUP}
	}
	t.Cleanup(func() { linkIn = prev })
	data := bytes.Repeat([]byte("record "), 512)
	swapExclusiveWrite(t, halfThen(func() error {
		if _, err := os.Lstat(dir + "/rec.md"); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("with no hard links, part-way through the write the final name already stands (lstat: %v)", err)
		}
		return errKilled
	}))
	if err := CreateExclusiveIn(root, "rec.md", data, 0o644); !errors.Is(err, errKilled) {
		t.Fatalf("CreateExclusiveIn = %v, want the write's own error", err)
	}
	onlyEntries(t, dir)
	exclusiveWrite = func(f *os.File, data []byte) error { _, err := f.Write(data); return err }
	if err := CreateExclusiveIn(root, "rec.md", data, 0o644); err != nil {
		t.Fatalf("with no hard links the create failed: %v", err)
	}
	if got, _ := os.ReadFile(dir + "/rec.md"); !bytes.Equal(got, data) {
		t.Errorf("the record holds %d bytes, want the whole %d", len(got), len(data))
	}
	if err := CreateExclusiveIn(root, "rec.md", []byte("again"), 0o644); !errors.Is(err, fs.ErrExist) {
		t.Errorf("with no hard links an existing name = %v, want fs.ErrExist", err)
	}
	onlyEntries(t, dir, "rec.md")
}

// TestCreateExclusiveInAFailedWriteNamesTheCallersPath: a write, flush or close
// that fails names rel, the caller's path, never the temp name the content was
// staged under, as the doc comment promises for every refusal.
func TestCreateExclusiveInAFailedWriteNamesTheCallersPath(t *testing.T) {
	root, dir := exclusiveRoot(t)
	swapExclusiveWrite(t, func(f *os.File, data []byte) error {
		f.Close()
		_, err := f.Write(data)
		return err
	})
	err := CreateExclusiveIn(root, "rec.md", []byte("whole"), 0o644)
	var pe *fs.PathError
	if !errors.As(err, &pe) || pe.Path != "rec.md" || strings.Contains(err.Error(), ".abcd-tmp-") {
		t.Errorf("a failed write = %v, want a *fs.PathError naming %q and not the temp name", err, "rec.md")
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Errorf("a failed write = %v, want its cause kept", err)
	}
	onlyEntries(t, dir)
}
