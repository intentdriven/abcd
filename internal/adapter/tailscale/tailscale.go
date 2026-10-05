// Package tailscale is the `tailscale` command as an adapter: this computer's
// own tailnet addresses and name, the lookup of a connecting device by its
// address, and the ports Tailscale's own Serve or Funnel configuration names
// (spc-2610040741034208, decision D3).
//
// abcd does not speak Tailscale's local API itself: how to reach it differs
// between the macOS app, the macOS standalone build and Linux, and the command
// already knows. The adapter runs three read-only commands and no other:
// `status --json --peers=false`, `whois --json <address>` and
// `serve status --json`. It never asks for a certificate and never changes
// Tailscale's state (decisions 21 and 23 of itd-2610032150577708), which
// TestTheAdapterNeverAsksForACertificateOrChangesTailscale holds.
package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// VersionFloor is the oldest Tailscale release whose command output this
// adapter has been shown to read, as major and minor. Source: the dated live
// receipt of 2026-10-05 (spc-2610040741034208 step 1, kept in the local tier),
// which ran `status --json --peers=false` and `whois --json` against 1.102.4
// on macOS and recorded the keys parsed here; the status JSON's own help warns
// that the format changes between releases, so an older release is refused
// rather than read on trust.
var VersionFloor = [2]int{1, 102}

// The adapter's refusals, each a sentinel a caller can test with errors.Is.
var (
	// ErrNotInstalled: no `tailscale` command on PATH and no app bundle.
	ErrNotInstalled = errors.New("the tailscale command is not installed")
	// ErrUnsafeCommand: the command found sits where someone else could
	// replace it.
	ErrUnsafeCommand = errors.New("the tailscale command found is not safe to run")
	// ErrNotRunning: Tailscale is not running, not logged in, or names no
	// address or name for this computer.
	ErrNotRunning = errors.New("Tailscale is not running on this computer")
	// ErrVersionBelowFloor: the installed release is older than VersionFloor.
	ErrVersionBelowFloor = errors.New("the installed Tailscale is older than this abcd reads")
)

// BundledCommands are the places the macOS app keeps its command, tried in
// order when PATH names none. The app's own wrapper on PATH execs the first.
var BundledCommands = []string{"/Applications/Tailscale.app/Contents/MacOS/Tailscale"}

// The bounds every run is held to: a command that hangs or floods is a
// failure, never a wait.
const (
	runTimeout   = 10 * time.Second
	whoisTimeout = 5 * time.Second
	maxOutput    = 4 << 20
	maxStderr    = 4 << 10
)

// Self is this computer as Tailscale names it.
type Self struct {
	// Name is the computer's MagicDNS name, without the trailing dot.
	Name string
	// Addrs are the computer's own Tailscale addresses, in the order
	// Tailscale lists them.
	Addrs []netip.Addr
	// Version is the release string Tailscale reports.
	Version string
}

// Identity is what Tailscale's lookup says about a connecting address.
type Identity struct {
	// Node is the device's stable id, the key a device is counted by.
	Node string `json:"node"`
	// Device is the device's name on the tailnet.
	Device string `json:"device"`
	// Login is the person's login name.
	Login string `json:"login"`
	// Person is the person's display name.
	Person string `json:"person"`
	// Tagged is set when the device carries a tag: it names a machine, not
	// a person.
	Tagged bool `json:"tagged"`
}

// runner runs the command at path with args and returns its standard output.
type runner func(ctx context.Context, path string, args ...string) ([]byte, error)

// Client runs the resolved command.
type Client struct {
	// Path is the resolved command (Resolve).
	Path string
	run  runner
}

// New is a client for the command at path.
func New(path string) *Client { return &Client{Path: path, run: execRun} }

func (c *Client) runner() runner {
	if c.run != nil {
		return c.run
	}
	return execRun
}

// statusJSON is the subset of `status --json` the adapter reads.
type statusJSON struct {
	Version      string
	BackendState string
	Self         *struct {
		DNSName      string
		TailscaleIPs []string
	}
}

// Status reads this computer's name and addresses. It refuses with
// ErrNotRunning when the backend is not running or names no address or name,
// and with ErrVersionBelowFloor below VersionFloor.
func (c *Client) Status(ctx context.Context) (Self, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	out, err := c.runner()(ctx, c.Path, "status", "--json", "--peers=false")
	if err != nil {
		return Self{}, fmt.Errorf("%w: %v", ErrNotRunning, err)
	}
	var st statusJSON
	if err := json.Unmarshal(out, &st); err != nil {
		return Self{}, fmt.Errorf("%w: its status is not the JSON this abcd reads", ErrNotRunning)
	}
	if st.BackendState != "Running" {
		return Self{}, fmt.Errorf("%w (it reports %q)", ErrNotRunning, clip(st.BackendState))
	}
	if !versionAtLeast(st.Version, VersionFloor) {
		return Self{}, fmt.Errorf("%w: %q is below %d.%d", ErrVersionBelowFloor, clip(st.Version), VersionFloor[0], VersionFloor[1])
	}
	if st.Self == nil {
		return Self{}, fmt.Errorf("%w: it names no device for this computer", ErrNotRunning)
	}
	self := Self{Name: strings.TrimSuffix(st.Self.DNSName, "."), Version: st.Version}
	for _, s := range st.Self.TailscaleIPs {
		a, err := netip.ParseAddr(s)
		if err != nil {
			continue
		}
		self.Addrs = append(self.Addrs, a.Unmap())
	}
	if self.Name == "" || len(self.Addrs) == 0 {
		return Self{}, fmt.Errorf("%w: it names no address or name for this computer", ErrNotRunning)
	}
	return self, nil
}

// whoisJSON is the subset of `whois --json` the adapter reads.
type whoisJSON struct {
	Node *struct {
		StableID     string
		Name         string
		ComputedName string
		Tags         []string
	}
	UserProfile *struct {
		LoginName   string
		DisplayName string
	}
}

// WhoIs asks Tailscale who is at addr. A failing lookup, or one naming no
// device or no person, is an error: the caller refuses the connection.
func (c *Client) WhoIs(ctx context.Context, addr netip.Addr) (Identity, error) {
	if !addr.IsValid() {
		return Identity{}, errors.New("tailscale whois: no address to look up")
	}
	addr = addr.Unmap()
	ctx, cancel := context.WithTimeout(ctx, whoisTimeout)
	defer cancel()
	out, err := c.runner()(ctx, c.Path, "whois", "--json", addr.String())
	if err != nil {
		return Identity{}, fmt.Errorf("tailscale whois: %w", err)
	}
	var w whoisJSON
	if err := json.Unmarshal(out, &w); err != nil {
		return Identity{}, errors.New("tailscale whois: the answer is not the JSON this abcd reads")
	}
	if w.Node == nil || w.UserProfile == nil || w.Node.StableID == "" || w.UserProfile.LoginName == "" {
		return Identity{}, errors.New("tailscale whois: the answer names no device or no person")
	}
	device := w.Node.ComputedName
	if device == "" {
		device = strings.TrimSuffix(w.Node.Name, ".")
	}
	return Identity{
		Node:   w.Node.StableID,
		Device: device,
		Login:  w.UserProfile.LoginName,
		Person: w.UserProfile.DisplayName,
		Tagged: len(w.Node.Tags) > 0,
	}, nil
}

// serveConfig is the subset of `serve status --json` that names ports: the
// TCP handlers by port, the web handlers and the funnel switch by host:port,
// and the same shape nested under each foreground session and each service.
type serveConfig struct {
	TCP         map[string]json.RawMessage
	Web         map[string]json.RawMessage
	AllowFunnel map[string]json.RawMessage
	Foreground  map[string]serveConfig
	Services    map[string]serveConfig
}

// ServedPorts lists every port Tailscale's own Serve or Funnel configuration
// names, sorted, so the dashboard never takes one. An answer that is not JSON
// is an error, never read as no configuration.
func (c *Client) ServedPorts(ctx context.Context) ([]uint16, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	out, err := c.runner()(ctx, c.Path, "serve", "status", "--json")
	if err != nil {
		return nil, fmt.Errorf("tailscale serve status: %w", err)
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 || string(out) == "null" {
		return nil, nil
	}
	var cfg serveConfig
	if err := json.Unmarshal(out, &cfg); err != nil {
		return nil, errors.New("tailscale serve status: the answer is not the JSON this abcd reads")
	}
	set := map[uint16]bool{}
	collectPorts(cfg, set)
	ports := make([]uint16, 0, len(set))
	for p := range set {
		ports = append(ports, p)
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i] < ports[j] })
	return ports, nil
}

func collectPorts(cfg serveConfig, set map[uint16]bool) {
	for k := range cfg.TCP {
		if p, err := strconv.ParseUint(k, 10, 16); err == nil {
			set[uint16(p)] = true
		}
	}
	for _, m := range []map[string]json.RawMessage{cfg.Web, cfg.AllowFunnel} {
		for k := range m {
			if _, port, err := net.SplitHostPort(k); err == nil {
				if p, err := strconv.ParseUint(port, 10, 16); err == nil {
					set[uint16(p)] = true
				}
			}
		}
	}
	for _, sub := range cfg.Foreground {
		collectPorts(sub, set)
	}
	for _, sub := range cfg.Services {
		collectPorts(sub, set)
	}
}

// Resolve finds the command once: on PATH through lookPath, then at the first
// of bundled that exists. It refuses, with ErrUnsafeCommand, a command that is
// not an absolute path, that sits inside the working tree (tree), in a
// world-writable folder, or that is itself world-writable: the conditions the
// plugin hooks refuse for abcd's own binary, because whoever can replace the
// command decides who the dashboard lets in.
func Resolve(lookPath func(string) (string, error), bundled []string, tree string) (string, error) {
	path := ""
	if lookPath != nil {
		if p, err := lookPath("tailscale"); err == nil && p != "" {
			path = p
		} else if errors.Is(err, exec.ErrDot) {
			return "", fmt.Errorf("%w: PATH names it relative to the current folder", ErrUnsafeCommand)
		}
	}
	if path == "" {
		for _, b := range bundled {
			if fi, err := os.Stat(b); err == nil && fi.Mode().IsRegular() {
				path = b
				break
			}
		}
	}
	if path == "" {
		return "", ErrNotInstalled
	}
	if err := judgeCommand(path, tree); err != nil {
		return "", err
	}
	return path, nil
}

// judgeCommand applies Resolve's refusals to path.
func judgeCommand(path, tree string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: it did not resolve to an absolute path", ErrUnsafeCommand)
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("%w: its folder could not be resolved", ErrUnsafeCommand)
	}
	if tree != "" {
		if t, err := filepath.EvalSymlinks(tree); err == nil {
			if rel, err := filepath.Rel(t, dir); err == nil && (rel == "." || !strings.HasPrefix(rel, "..")) {
				return fmt.Errorf("%w: it lives inside the working tree", ErrUnsafeCommand)
			}
		}
	}
	di, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("%w: its folder could not be read", ErrUnsafeCommand)
	}
	if di.Mode().Perm()&0o002 != 0 {
		return fmt.Errorf("%w: its folder is world-writable", ErrUnsafeCommand)
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return fmt.Errorf("%w: it is not a regular file", ErrUnsafeCommand)
	}
	if fi.Mode().Perm()&0o002 != 0 {
		return fmt.Errorf("%w: the command itself is world-writable", ErrUnsafeCommand)
	}
	return nil
}

// versionAtLeast reports whether v ("1.102.4-t…") is at least floor.
func versionAtLeast(v string, floor [2]int) bool {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	minorText := parts[1]
	if i := strings.IndexFunc(minorText, func(r rune) bool { return r < '0' || r > '9' }); i >= 0 {
		minorText = minorText[:i]
	}
	minor, err := strconv.Atoi(minorText)
	if err != nil {
		return false
	}
	if major != floor[0] {
		return major > floor[0]
	}
	return minor >= floor[1]
}

// clip bounds a value Tailscale reported before it reaches a message.
func clip(s string) string {
	const max = 64
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

// boundedBuffer keeps at most limit bytes and records that more arrived.
type boundedBuffer struct {
	buf      bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	room := b.limit - b.buf.Len()
	if len(p) > room {
		b.overflow = true
		if room > 0 {
			b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return b.buf.Write(p)
}

// execRun runs the command with no shell, a bounded output and the caller's
// deadline.
func execRun(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdin = nil
	out := &boundedBuffer{limit: maxOutput}
	errOut := &boundedBuffer{limit: maxStderr}
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.buf.String())
		if msg != "" {
			return nil, fmt.Errorf("%v: %s", err, clip(firstLine(msg)))
		}
		return nil, err
	}
	if out.overflow {
		return nil, fmt.Errorf("its answer is larger than %d bytes", maxOutput)
	}
	return out.buf.Bytes(), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
