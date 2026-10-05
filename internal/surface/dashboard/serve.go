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
// is listening, and serves until SIGTERM or SIGINT.
func Serve(ctx context.Context) error {
	if os.Getenv(serveMarkerEnv) != "1" || !isPipe(readyFD) || !isPipe(configFD) {
		return errNotFromStart
	}
	ready := os.NewFile(readyFD, "ready")
	cfgPipe := os.NewFile(configFD, "config")
	report := func(m readyMessage) {
		data, _ := json.Marshal(m)
		_, _ = ready.Write(append(data, '\n'))
		_ = ready.Close()
	}
	cfg, err := readChildConfig(cfgPipe)
	_ = cfgPipe.Close()
	if err != nil {
		report(readyMessage{Error: err.Error()})
		return err
	}
	var addrs []netip.Addr
	for _, s := range cfg.Addrs {
		a, err := netip.ParseAddr(s)
		if err != nil {
			report(readyMessage{Error: "an address that is not one: " + s})
			return err
		}
		addrs = append(addrs, a)
	}
	raws, err := listenTailnet(addrs, cfg.Port)
	if err != nil {
		report(readyMessage{Error: err.Error()})
		return err
	}
	port := raws[0].Addr().(*net.TCPAddr).Port
	gl := newGatedListener(raws, newGate(tailscale.New(cfg.Tailscale).WhoIs, tailnetPrefixes))
	seen := newSeenRecorder(cfg.Home)
	srv := newHTTPServer(newHandler(handlerConfig{name: cfg.Name, port: port, selfCheck: []byte(cfg.SelfCheck), seen: seen.record}))

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	signal.Ignore(syscall.SIGHUP)
	served := make(chan error, 1)
	go func() { served <- srv.Serve(gl) }()
	report(readyMessage{OK: true, Port: port})

	select {
	case <-ctx.Done():
	case err := <-served:
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
	_ = gl.Close()
	return nil
}

// readChildConfig reads start's configuration, bounded in size and time.
func readChildConfig(r io.Reader) (childConfig, error) {
	type result struct {
		cfg childConfig
		err error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := bufio.NewReader(io.LimitReader(r, 64<<10)).ReadBytes('\n')
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
