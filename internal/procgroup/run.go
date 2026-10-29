package procgroup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// RunToEOF runs cmd, built with exec.CommandContext(ctx, …), and copies its
// standard output to stdout and its standard error to stderr, reading each
// through a pipe to EOF before it calls Wait. A writer that is an *os.File is
// handed to the command as it is, a nil one is the null device, and a
// cmd.Stdin that is not a file is fed through a pipe of its own.
//
// exec copies into any other writer on goroutines Wait waits for, and under a
// WaitDelay that wait is timed from the moment Wait sees the command exit,
// even one that exited on its own: on a loaded machine the copy of an answer
// the command had already written could miss it, and Wait reported
// exec.ErrWaitDelay over a command that succeeded, its output cut
// (iss-2610272041031418). Read here, nothing is left for exec to copy or to
// time, so a command that exits 0 is read whole, however late its output
// closes.
//
// A process the command started that holds a pipe open therefore holds the
// read until ctx ends. Then Wait is called at once: the command is stopped
// through its Cancel (exec kills it by default; Bound kills its group) and
// Wait closes the pipes, so the holder cannot keep the caller past ctx. The
// command's WaitDelay is left to bound only a command that does not exit
// once stopped. Output cut short by ctx is never answered as the command's:
// the error then wraps ctx's own, with what Wait said in its text.
func RunToEOF(ctx context.Context, cmd *exec.Cmd, stdout, stderr io.Writer) error {
	var reads []func()
	pipe := func(w io.Writer, set *io.Writer, open func() (io.ReadCloser, error)) error {
		if f, ok := w.(*os.File); ok || w == nil {
			if ok {
				*set = f
			}
			return nil
		}
		r, err := open()
		if err != nil {
			return err
		}
		reads = append(reads, func() { _, _ = io.Copy(w, r) })
		return nil
	}
	if err := pipe(stdout, &cmd.Stdout, cmd.StdoutPipe); err != nil {
		return err
	}
	if err := pipe(stderr, &cmd.Stderr, cmd.StderrPipe); err != nil {
		return err
	}
	var feed func()
	if in := cmd.Stdin; in != nil {
		if _, ok := in.(*os.File); !ok {
			cmd.Stdin = nil
			w, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			// A command that stops reading early fails the write; what it
			// read is its business, and its exit status says how it went.
			feed = func() { _, _ = io.Copy(w, in); _ = w.Close() }
		}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	fed := make(chan struct{})
	go func() {
		defer close(fed)
		if feed != nil {
			feed()
		}
	}()
	drained := make(chan struct{})
	var wg sync.WaitGroup
	for _, read := range reads {
		wg.Go(read)
	}
	go func() { wg.Wait(); close(drained) }()

	select {
	case <-drained:
	case <-ctx.Done():
	}
	cut := true
	select {
	case <-drained:
		cut = false
	default:
	}
	err := cmd.Wait()
	<-drained
	<-fed
	switch {
	case !cut:
		return err
	case err == nil:
		return fmt.Errorf("%s exited, but its output was still open when its context ended: %w",
			filepath.Base(cmd.Path), ctx.Err())
	}
	return fmt.Errorf("%s was stopped when its context ended (%v): %w", filepath.Base(cmd.Path), err, ctx.Err())
}
