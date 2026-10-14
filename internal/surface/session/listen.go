package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// DefaultListen is where the relay listens unless --listen names another
// address: this computer alone (adr-2610032150581128, as amended 2026-10-14).
const DefaultListen = "127.0.0.1:8787"

// The plain-HTTP bounds; a WebSocket is bounded by its pings once upgraded.
const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 16 << 10
	shutdownWait      = 5 * time.Second
)

// ParseListen reads a --listen value: an IP address, or localhost, and a
// port. Port 0 lets the system choose one. A host name other than localhost
// is refused: the relay listens on an address, never on whatever a name
// resolves to.
func ParseListen(s string) (netip.AddrPort, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(s))
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("--listen %q is not host:port", s)
	}
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("--listen %q: the port is not 0 to 65535", s)
	}
	if strings.EqualFold(host, "localhost") {
		host = "127.0.0.1"
	}
	a, err := netip.ParseAddr(host)
	if err != nil || a.Zone() != "" {
		return netip.AddrPort{}, fmt.Errorf("--listen %q: give an IP address (or localhost), not a name", s)
	}
	return netip.AddrPortFrom(a.Unmap(), uint16(n)), nil
}

// Listen opens the relay's listener: abcd's second, admitted by
// adr-2610032150581128's amendment of 2026-10-14.
func Listen(addr netip.AddrPort) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

// Run serves the relay on ln until ctx ends, purging idle sessions as it
// goes. The server's own error log goes nowhere: the relay logs nothing.
func (s *Server) Run(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          log.New(io.Discard, "", 0),
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go s.Sweep(ctx, SweepInterval)
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	select {
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), shutdownWait)
		defer cancel()
		_ = srv.Shutdown(shut)
		<-served
		return nil
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
