package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The shapes below are the live receipt's (2026-10-05, Tailscale 1.102.4 on
// macOS), with every name, address and key replaced by a reserved or invented
// value: the keys and nesting are what the parser reads.
const statusRunning = `{
  "Version": "1.102.4-tabc-gdef",
  "BackendState": "Running",
  "TailscaleIPs": ["100.101.102.103", "fd7a:115c:a1e0::1"],
  "Self": {
    "DNSName": "dash.example-tailnet.ts.net.",
    "HostName": "dash",
    "TailscaleIPs": ["100.101.102.103", "fd7a:115c:a1e0::1"],
    "Online": true
  },
  "MagicDNSSuffix": "example-tailnet.ts.net"
}`

const whoisPerson = `{
  "Node": {
    "ID": 1,
    "StableID": "nStable1",
    "Name": "phone.example-tailnet.ts.net.",
    "ComputedName": "phone",
    "Addresses": ["100.101.102.104/32"]
  },
  "UserProfile": {"ID": 2, "LoginName": "pt@example.com", "DisplayName": "Product Thinker"},
  "CapMap": null
}`

const whoisTagged = `{
  "Node": {
    "ID": 3,
    "StableID": "nStable3",
    "Name": "ci.example-tailnet.ts.net.",
    "ComputedName": "ci",
    "Tags": ["tag:ci"]
  },
  "UserProfile": {"ID": 4, "LoginName": "tagged-devices", "DisplayName": "Tagged Devices"}
}`

// whoisShared is a device shared into the tailnet from another account: its
// Sharer names that account's user (tailcfg.Node.Sharer, omitted when zero).
const whoisShared = `{
  "Node": {
    "ID": 5,
    "StableID": "nStable5",
    "Name": "laptop.example-tailnet.ts.net.",
    "ComputedName": "laptop",
    "User": 6,
    "Sharer": 7
  },
  "UserProfile": {"ID": 6, "LoginName": "someone@example.org", "DisplayName": "Someone Else"}
}`

// fakeRun answers each argument list from a table, recording every call.
type fakeRun struct {
	out   map[string]string
	fail  map[string]error
	calls []string
}

func (f *fakeRun) run(_ context.Context, _ string, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if err := f.fail[key]; err != nil {
		return nil, err
	}
	out, ok := f.out[key]
	if !ok {
		return nil, errors.New("unexpected call: " + key)
	}
	return []byte(out), nil
}

func client(f *fakeRun) *Client { return &Client{Path: "/fake/tailscale", run: f.run} }

func TestStatusReadsThisComputersAddressesAndName(t *testing.T) {
	f := &fakeRun{out: map[string]string{"status --json --peers=false": statusRunning}}
	self, err := client(f).Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []netip.Addr{netip.MustParseAddr("100.101.102.103"), netip.MustParseAddr("fd7a:115c:a1e0::1")}
	if !reflect.DeepEqual(self.Addrs, want) {
		t.Errorf("addresses = %v, want %v", self.Addrs, want)
	}
	if self.Name != "dash.example-tailnet.ts.net" {
		t.Errorf("name = %q, want the DNS name without its trailing dot", self.Name)
	}
	if self.Version != "1.102.4-tabc-gdef" {
		t.Errorf("version = %q", self.Version)
	}
}

func TestStatusRefusesWhenTailscaleIsNotRunning(t *testing.T) {
	for name, body := range map[string]string{
		"stopped":     strings.Replace(statusRunning, `"Running"`, `"Stopped"`, 1),
		"needs login": strings.Replace(statusRunning, `"Running"`, `"NeedsLogin"`, 1),
		"no name":     strings.Replace(statusRunning, `"dash.example-tailnet.ts.net."`, `""`, 1),
		"no address":  `{"Version":"1.102.4","BackendState":"Running","Self":{"DNSName":"dash.example-tailnet.ts.net.","TailscaleIPs":[]}}`,
		"not json":    `Tailscale is stopped.`,
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeRun{out: map[string]string{"status --json --peers=false": body}}
			if _, err := client(f).Status(context.Background()); !errors.Is(err, ErrNotRunning) {
				t.Fatalf("Status() = %v, want ErrNotRunning", err)
			}
		})
	}
}

func TestStatusRefusesAVersionBelowTheFloor(t *testing.T) {
	for v, ok := range map[string]bool{
		"1.102.4-tabc-gdef": true,
		"1.102.0":           true,
		"1.110.1":           true,
		"2.0.0":             true,
		"1.101.9":           false,
		"1.98.0":            false,
		"0.200.0":           false,
		"garbage":           false,
		"":                  false,
	} {
		f := &fakeRun{out: map[string]string{"status --json --peers=false": strings.Replace(statusRunning, "1.102.4-tabc-gdef", v, 1)}}
		_, err := client(f).Status(context.Background())
		if ok && err != nil {
			t.Errorf("version %q refused: %v", v, err)
		}
		if !ok && !errors.Is(err, ErrVersionBelowFloor) {
			t.Errorf("version %q: Status() = %v, want ErrVersionBelowFloor", v, err)
		}
	}
}

func TestWhoIsNamesThePersonAndDevice(t *testing.T) {
	addr := netip.MustParseAddr("100.101.102.104")
	f := &fakeRun{out: map[string]string{"whois --json 100.101.102.104": whoisPerson}}
	id, err := client(f).WhoIs(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
	want := Identity{Node: "nStable1", Device: "phone", Login: "pt@example.com", Person: "Product Thinker"}
	if id != want {
		t.Errorf("identity = %+v, want %+v", id, want)
	}
}

func TestWhoIsReportsTags(t *testing.T) {
	addr := netip.MustParseAddr("100.101.102.105")
	f := &fakeRun{out: map[string]string{"whois --json 100.101.102.105": whoisTagged}}
	id, err := client(f).WhoIs(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
	if !id.Tagged {
		t.Errorf("a node with tags read as untagged: %+v", id)
	}
}

func TestWhoIsReportsASharer(t *testing.T) {
	addr := netip.MustParseAddr("100.101.102.106")
	f := &fakeRun{out: map[string]string{
		"whois --json 100.101.102.106": whoisShared,
		"whois --json 100.101.102.104": whoisPerson,
	}}
	id, err := client(f).WhoIs(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
	if !id.Shared || id.Tagged {
		t.Errorf("a device shared in from another account read as %+v, want shared and untagged", id)
	}
	own, err := client(f).WhoIs(context.Background(), netip.MustParseAddr("100.101.102.104"))
	if err != nil {
		t.Fatal(err)
	}
	if own.Shared {
		t.Errorf("a device of the tailnet's own read as shared: %+v", own)
	}
}

func TestWhoIsUnmapsAndFailsClosed(t *testing.T) {
	f := &fakeRun{
		out:  map[string]string{"whois --json fd7a:115c:a1e0::2": `{"Node":{}}`},
		fail: map[string]error{"whois --json 100.64.0.9": errors.New("exit status 1: no match for IP:port")},
	}
	c := client(f)
	if _, err := c.WhoIs(context.Background(), netip.MustParseAddr("::ffff:100.64.0.9")); err == nil {
		t.Error("a failing lookup was not an error")
	}
	if got := f.calls[0]; got != "whois --json 100.64.0.9" {
		t.Errorf("a v4-mapped address was asked as %q, want the unmapped form", got)
	}
	if _, err := c.WhoIs(context.Background(), netip.MustParseAddr("fd7a:115c:a1e0::2")); err == nil {
		t.Error("an answer naming no node and no person was not an error")
	}
	if _, err := c.WhoIs(context.Background(), netip.Addr{}); err == nil {
		t.Error("an invalid address was looked up")
	}
}

func TestServedPortsReadsEveryPlaceAConfigNamesAPort(t *testing.T) {
	const cfg = `{
  "TCP": {"443": {"HTTPS": true}, "10000": {"TCPForward": "127.0.0.1:5432"}},
  "Web": {"dash.example-tailnet.ts.net:8443": {"Handlers": {"/": {"Proxy": "http://127.0.0.1:3000"}}}},
  "AllowFunnel": {"dash.example-tailnet.ts.net:8080": true},
  "Foreground": {"sess": {"TCP": {"9090": {"HTTP": true}}}},
  "Services": {"svc:web": {"TCP": {"7070": {"HTTP": true}}, "Web": {"web.example-tailnet.ts.net:7443": {}}}}
}`
	f := &fakeRun{out: map[string]string{"serve status --json": cfg}}
	ports, err := client(f).ServedPorts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{443, 7070, 7443, 8080, 8443, 9090, 10000}
	if !reflect.DeepEqual(ports, want) {
		t.Errorf("ports = %v, want %v", ports, want)
	}
}

func TestServedPortsReadsNoConfigAsNone(t *testing.T) {
	for _, body := range []string{"", "null\n", "{}\n"} {
		f := &fakeRun{out: map[string]string{"serve status --json": body}}
		ports, err := client(f).ServedPorts(context.Background())
		if err != nil || len(ports) != 0 {
			t.Errorf("%q: ports = %v, err = %v; want none", body, ports, err)
		}
	}
	f := &fakeRun{out: map[string]string{"serve status --json": "No serve config"}}
	if _, err := client(f).ServedPorts(context.Background()); err == nil {
		t.Error("an answer that is not JSON was read as no configuration")
	}
}

func TestTheAdapterNeverAsksForACertificateOrChangesTailscale(t *testing.T) {
	f := &fakeRun{out: map[string]string{
		"status --json --peers=false":  statusRunning,
		"whois --json 100.101.102.104": whoisPerson,
		"serve status --json":          "{}",
	}}
	c := client(f)
	_, _ = c.Status(context.Background())
	_, _ = c.WhoIs(context.Background(), netip.MustParseAddr("100.101.102.104"))
	_, _ = c.ServedPorts(context.Background())
	for _, call := range f.calls {
		first := strings.Fields(call)[0]
		if first != "status" && first != "whois" && !strings.HasPrefix(call, "serve status") {
			t.Errorf("the adapter ran %q: only status, whois and serve status are read-only", call)
		}
	}
}

func writeExe(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "tailscale")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveRefusesAnUnsafeCommand(t *testing.T) {
	tree := t.TempDir()
	safeDir := t.TempDir()
	safe := writeExe(t, safeDir)

	inside := filepath.Join(tree, "bin")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	inTree := writeExe(t, inside)

	open := t.TempDir()
	if err := os.Chmod(open, 0o777); err != nil {
		t.Fatal(err)
	}
	inOpen := writeExe(t, open)

	writable := t.TempDir()
	loose := writeExe(t, writable)
	if err := os.Chmod(loose, 0o757); err != nil {
		t.Fatal(err)
	}

	lookAt := func(p string) func(string) (string, error) {
		return func(string) (string, error) { return p, nil }
	}
	if got, err := Resolve(lookAt(safe), nil, tree); err != nil || got != safe {
		t.Errorf("Resolve(safe) = %q, %v", got, err)
	}
	for name, p := range map[string]string{"inside the working tree": inTree, "world-writable folder": inOpen, "world-writable command": loose, "relative": "tailscale"} {
		if _, err := Resolve(lookAt(p), nil, tree); !errors.Is(err, ErrUnsafeCommand) {
			t.Errorf("%s: Resolve = %v, want ErrUnsafeCommand", name, err)
		}
	}
}

func TestResolveFallsBackToTheAppsBundledCommand(t *testing.T) {
	missing := func(string) (string, error) { return "", errors.New("not found") }
	bundled := writeExe(t, t.TempDir())
	got, err := Resolve(missing, []string{filepath.Join(t.TempDir(), "absent"), bundled}, t.TempDir())
	if err != nil || got != bundled {
		t.Errorf("Resolve = %q, %v; want the bundled command", got, err)
	}
	if _, err := Resolve(missing, nil, t.TempDir()); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("Resolve with nothing installed = %v, want ErrNotInstalled", err)
	}
}

// TestCommandEnvCarriesATerminal: the macOS app's bundled CLI takes a
// launch with no terminal-shaped variable for a GUI launch and prints a GUI
// error instead of answering, so every command runs with TERM set: the
// caller's own when it has one, else "dumb".
func TestCommandEnvCarriesATerminal(t *testing.T) {
	got := commandEnv([]string{"PATH=/usr/bin", "HOME=/h"})
	if !reflect.DeepEqual(got, []string{"PATH=/usr/bin", "HOME=/h", "TERM=dumb"}) {
		t.Fatalf("no TERM in the environment: commandEnv = %q, want TERM=dumb added", got)
	}
	kept := commandEnv([]string{"PATH=/usr/bin", "TERM=xterm-256color"})
	if !reflect.DeepEqual(kept, []string{"PATH=/usr/bin", "TERM=xterm-256color"}) {
		t.Fatalf("a TERM already set: commandEnv = %q, want it kept as it is", kept)
	}
	empty := commandEnv([]string{"PATH=/usr/bin", "TERM="})
	if !reflect.DeepEqual(empty, []string{"PATH=/usr/bin", "TERM=dumb"}) {
		t.Fatalf("an empty TERM: commandEnv = %q, want it replaced by TERM=dumb", empty)
	}
}
