package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/tailscale"
)

// The descriptors start hands the server: the readiness pipe it reports on,
// and the config pipe it reads its configuration from.
const (
	readyFD  = 3
	configFD = 4
)

// serveMarkerEnv is set by start in the server's environment. It is no
// secret and grants nothing: it only keeps a serve that start did not launch
// from looking at descriptors 3 and 4 at all, which in any other process may
// be the runtime's own.
const serveMarkerEnv = "ABCD_DASHBOARD_SERVE"

// configWait bounds the server's wait for its configuration.
const configWait = 5 * time.Second

// keepWait bounds the server's wait, once it listens, for start to name the
// addresses to keep: start's self-check connects to each address and then
// fetches from it, each within selfCheckWait, and configWait more is slack.
func keepWait(addrs int) time.Duration { return time.Duration(2*addrs)*selfCheckWait + configWait }

// errNotFromStart is the refusal of a serve that start did not launch.
var errNotFromStart = &Refusal{msg: "the dashboard server is started only by `abcd dashboard start`, which hands it its pipes; nothing else starts it"}

// isPipe reports whether fd is an open pipe, judged without taking ownership
// of the descriptor: one that is not a pipe may be the runtime's own.
func isPipe(fd int) bool {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return false
	}
	return st.Mode&syscall.S_IFMT == syscall.S_IFIFO
}

// Serve is the hidden `dashboard serve`: the server process start launches.
// It refuses unless start's marker is in its environment and both of start's
// pipes are open (so nothing but start starts the server, P8), reads its configuration from one, listens on the
// tailnet addresses it names and nothing else, reports on the other that it
// is listening, keeps only the addresses start's self-check reached, and
// serves until SIGTERM or SIGINT.
func Serve(ctx context.Context) error {
	if os.Getenv(serveMarkerEnv) != "1" || !isPipe(readyFD) || !isPipe(configFD) {
		return errNotFromStart
	}
	ready := os.NewFile(readyFD, "ready")
	cfgPipe := os.NewFile(configFD, "config")
	defer cfgPipe.Close()
	report := func(m readyMessage, last bool) {
		data, _ := json.Marshal(m)
		_, _ = ready.Write(append(data, '\n'))
		if last {
			_ = ready.Close()
		}
	}
	cfgIn := bufio.NewReader(io.LimitReader(cfgPipe, 64<<10))
	cfg, err := readChildConfig(cfgIn)
	if err != nil {
		report(readyMessage{Error: err.Error()}, true)
		return err
	}
	var addrs []netip.Addr
	for _, s := range cfg.Addrs {
		a, err := netip.ParseAddr(s)
		if err != nil {
			report(readyMessage{Error: "an address that is not one: " + s}, true)
			return err
		}
		addrs = append(addrs, a)
	}
	raws, err := listenTailnet(addrs, cfg.Port)
	if err != nil {
		report(readyMessage{Error: err.Error()}, true)
		return err
	}
	port := raws[0].Addr().(*net.TCPAddr).Port
	self := newSelfCheck([]byte(cfg.SelfCheck), addrs)
	gl := newGatedListener(raws, newGate(tailscale.New(cfg.Tailscale).WhoIs, tailnetPrefixes, self))
	seen := newSeenRecorder(cfg.Home)
	srv := newHTTPServer(newHandler(handlerConfig{name: cfg.Name, port: port, self: self, seen: seen.record}))

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	signal.Ignore(syscall.SIGHUP)
	served := make(chan error, 1)
	go func() { served <- srv.Serve(gl) }()
	shut := func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
		_ = gl.Close()
	}
	report(readyMessage{OK: true, Port: port}, false)

	// Start's self-check runs now. Once it has, start names the addresses it
	// reached; every other address stops being listened on, and the server
	// says which it kept. No answer, a closed pipe, or a keep that names an
	// address whose self-check has not answered stops the server.
	kept, err := keepOnly(ctx, cfgIn, addrs, raws, self)
	_ = cfgPipe.Close()
	if err != nil {
		report(readyMessage{Error: err.Error()}, true)
		shut()
		return err
	}
	keptText := make([]string, len(kept))
	for i, a := range kept {
		keptText[i] = a.String()
	}
	report(readyMessage{OK: true, Port: port, Addrs: keptText}, true)

	select {
	case <-ctx.Done():
	case err := <-served:
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shut()
	return nil
}

// keepOnly reads start's keepMessage from r, settles the self-check on it,
// and closes the listener of every address it does not name, raws[i] being
// the listener on addrs[i]. It returns the addresses kept, in addrs' order.
func keepOnly(ctx context.Context, r *bufio.Reader, addrs []netip.Addr, raws []net.Listener, self *selfCheck) ([]netip.Addr, error) {
	type result struct {
		keep keepMessage
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			ch <- result{err: errors.New("start named no address to keep")}
			return
		}
		var k keepMessage
		if err := json.Unmarshal(line, &k); err != nil || len(k.Keep) == 0 {
			ch <- result{err: errors.New("start's addresses to keep are not ones this server reads")}
			return
		}
		ch <- result{keep: k}
	}()
	var k keepMessage
	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		k = res.keep
	case <-time.After(keepWait(len(addrs))):
		return nil, errors.New("start named no address to keep in time")
	case <-ctx.Done():
		return nil, errors.New("stopped before start named the addresses to keep")
	}
	keep := map[netip.Addr]bool{}
	var named []netip.Addr
	for _, s := range k.Keep {
		a, err := netip.ParseAddr(s)
		if err != nil || keep[a] {
			return nil, errors.New("start named an address to keep that is not one, or twice: " + s)
		}
		keep[a] = true
		named = append(named, a)
	}
	if !self.settle(named) {
		return nil, errors.New("start named an address to keep whose self-check has not answered")
	}
	var kept []netip.Addr
	for i, a := range addrs {
		if keep[a] {
			kept = append(kept, a)
			continue
		}
		_ = raws[i].Close()
	}
	return kept, nil
}

// readChildConfig reads start's configuration, bounded in size (by r) and
// time.
func readChildConfig(r *bufio.Reader) (childConfig, error) {
	type result struct {
		cfg childConfig
		err error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			ch <- result{err: errors.New("start handed no configuration")}
			return
		}
		var c childConfig
		if err := json.Unmarshal(line, &c); err != nil || c.Wire != childConfigWire || c.Name == "" || c.Tailscale == "" || c.Home == "" || len(c.SelfCheck) < 32 {
			ch <- result{err: errors.New("start's configuration is not one this server reads")}
			return
		}
		ch <- result{cfg: c}
	}()
	select {
	case res := <-ch:
		return res.cfg, res.err
	case <-time.After(configWait):
		return childConfig{}, errors.New("start handed no configuration in time")
	}
}
