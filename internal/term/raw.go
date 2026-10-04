package term

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	xterm "golang.org/x/term"
)

// RawSession is one stretch of raw mode on a terminal (spc-2610030911534855,
// "The answer loop"), and the only place abcd touches termios: no call site
// handles the terminal's modes itself.
//
// The terminal is restored on every exit. Restore is idempotent (sync.Once)
// and is reached from:
//
//   - the caller's deferred Guard, on every return, an error included;
//   - Guard again on a panic, which restores and then panics on with the same
//     value;
//   - the caller, on Ctrl-C, which raw mode delivers as the byte 0x03 because
//     it clears ISIG;
//   - the session's own handler for SIGINT, SIGTERM and SIGHUP sent from
//     outside, which restores and then re-raises the signal with its default
//     action, so the process ends as that signal ends it. A signal the process
//     was started ignoring (nohup's SIGHUP) stays ignored.
//
// Suspend is Ctrl-Z (0x1a under raw mode): restore, stop with SIGTSTP, and on
// SIGCONT enter raw mode again. A SIGCONT after a stop from outside enters raw
// mode again too. SIGWINCH and SIGCONT reach the caller through Hooks, run on
// the session's signal goroutine, so a caller that draws from a hook
// serialises the drawing itself.
type RawSession struct {
	f     *os.File
	hooks Hooks

	mu        sync.Mutex
	orig      *xterm.State
	closed    bool
	suspended bool

	once  sync.Once
	sigs  chan os.Signal
	done  chan struct{}
	conts chan struct{}
}

// Hooks are what a session calls on its signal goroutine: Resized on SIGWINCH,
// and Continued on a SIGCONT after raw mode is entered again. Either may be
// nil. A hook that panics restores the terminal before the panic goes on.
type Hooks struct {
	Resized   func()
	Continued func()
}

// suspendWait bounds how long Suspend waits for SIGCONT after stopping itself.
// A process group the kernel deems orphaned discards SIGTSTP, and a stop that
// never happens must not hang the loop; a real stop holds the process past it,
// so a resumed session finds it elapsed and goes on.
const suspendWait = 500 * time.Millisecond

// relayed are the signals that end a session from outside.
var relayed = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}

// StartRaw puts f, the terminal a person answers at, into raw mode and
// returns the session that restores it. It refuses a descriptor that is not a
// terminal, or a terminal that refuses raw mode, and then leaves nothing
// behind: no signal registration and no goroutine.
//
// The signal handler is registered before raw mode is entered, so no signal
// can find the terminal raw and unwatched.
func StartRaw(f *os.File, h Hooks) (*RawSession, error) {
	if f == nil {
		return nil, errors.New("no terminal to enter raw mode on")
	}
	s := &RawSession{
		f:     f,
		hooks: h,
		sigs:  make(chan os.Signal, 8),
		done:  make(chan struct{}),
		conts: make(chan struct{}, 1),
	}
	watched := []os.Signal{syscall.SIGWINCH, syscall.SIGCONT}
	for _, sig := range relayed {
		if !signal.Ignored(sig) {
			watched = append(watched, sig)
		}
	}
	s.mu.Lock()
	signal.Notify(s.sigs, watched...)
	go s.watch()
	err := withFd(f, func(fd int) error {
		st, err := xterm.MakeRaw(fd)
		s.orig = st
		return err
	})
	if err != nil {
		s.closed = true
		signal.Stop(s.sigs)
		close(s.done)
		s.once.Do(func() {})
		s.mu.Unlock()
		return nil, fmt.Errorf("raw mode: %w", err)
	}
	s.mu.Unlock()
	return s, nil
}

// Restore puts the terminal back as StartRaw found it and ends the session:
// the signal registration is stopped and the signal goroutine told to end.
// Only the first call does anything; every later one returns nil, so a late
// call never undoes a change made after the session ended.
func (s *RawSession) Restore() error {
	var err error
	s.once.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closed = true
		signal.Stop(s.sigs)
		close(s.done)
		err = withFd(s.f, func(fd int) error { return xterm.Restore(fd, s.orig) })
	})
	return err
}

// Guard is the caller's deferred restore: `defer s.Guard()` directly, so its
// recover sees the caller's panic. It restores, then panics on with the
// recovered value, if there was one.
func (s *RawSession) Guard() {
	r := recover()
	_ = s.Restore()
	if r != nil {
		panic(r)
	}
}

// Suspend is Ctrl-Z: it restores the terminal, stops the process group with
// SIGTSTP as the terminal's own Ctrl-Z would, and once continued enters raw
// mode again. The caller redraws after it returns. On an ended session it
// does nothing.
func (s *RawSession) Suspend() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	if err := withFd(s.f, func(fd int) error { return xterm.Restore(fd, s.orig) }); err != nil {
		s.mu.Unlock()
		return err
	}
	s.suspended = true
	select {
	case <-s.conts: // a stale continue from before
	default:
	}
	s.mu.Unlock()

	_ = syscall.Kill(0, syscall.SIGTSTP)
	select {
	case <-s.conts:
	case <-time.After(suspendWait):
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.suspended = false
	if s.closed {
		return nil
	}
	return s.reenter()
}

// reenter enters raw mode again after a stop, keeping the attributes the
// session started from as the ones Restore puts back. The caller holds mu.
func (s *RawSession) reenter() error {
	return withFd(s.f, func(fd int) error {
		_, err := xterm.MakeRaw(fd)
		return err
	})
}

// watch is the session's signal goroutine. It ends when the session does.
func (s *RawSession) watch() {
	defer func() {
		if r := recover(); r != nil {
			_ = s.Restore()
			panic(r)
		}
	}()
	for {
		select {
		case <-s.done:
			s.drain()
			return
		case sig := <-s.sigs:
			switch sig {
			case syscall.SIGWINCH:
				if s.live() && s.hooks.Resized != nil {
					s.hooks.Resized()
				}
			case syscall.SIGCONT:
				if !s.continued() {
					continue
				}
				if s.hooks.Continued != nil {
					s.hooks.Continued()
				}
			default:
				s.relay(sig)
				return
			}
		}
	}
}

// drain reads what is left on the signal channel once the session has
// ended, and relays a signal that ends the process. Restore stops the
// registration first, so nothing more arrives; but a signal delivered while
// this goroutine was in a hook can be waiting beside the closed done, and a
// select that took done first would drop it and the process would live on.
func (s *RawSession) drain() {
	for {
		select {
		case sig := <-s.sigs:
			switch sig {
			case syscall.SIGWINCH, syscall.SIGCONT:
			default:
				s.relay(sig)
				return
			}
		default:
			return
		}
	}
}

// live reports whether the session is in raw mode: not ended, and not
// stopped by Suspend.
func (s *RawSession) live() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed && !s.suspended
}

// continued handles a SIGCONT: a Suspend waiting on it is woken and does the
// rest; otherwise, after a stop from outside, raw mode is entered again and
// the caller's hook runs. It reports whether the hook should run.
func (s *RawSession) continued() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	if s.suspended {
		select {
		case s.conts <- struct{}{}:
		default:
		}
		return false
	}
	return s.reenter() == nil
}

// relay ends the session on a signal from outside: restore, then the signal's
// default action, so the process ends as that signal ends it.
func (s *RawSession) relay(sig os.Signal) {
	_ = s.Restore()
	signal.Reset(sig)
	if n, ok := sig.(syscall.Signal); ok {
		_ = syscall.Kill(os.Getpid(), n)
	}
}

// withFd runs fn on f's descriptor, reached through SyscallConn so the file
// keeps its blocking mode, as IsTerminal does.
func withFd(f *os.File, fn func(fd int) error) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var inner error
	if err := rc.Control(func(fd uintptr) { inner = fn(int(fd)) }); err != nil {
		return err
	}
	return inner
}
