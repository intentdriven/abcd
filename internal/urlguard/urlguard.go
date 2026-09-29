// Package urlguard is the canonical SSRF guard for every outbound fetch abcd
// makes.
//
// abcd fetches URLs from two places — memory ingest takes one from the operator,
// and the citation refresh takes them from committed documentation — and in both
// the address is attacker-influenceable content. A guard that lives inside one
// caller is a guard the next caller reimplements slightly differently, which is
// exactly how a metadata endpoint ends up reachable from the second fetch path.
// So the predicate, the name check, and the connect-time re-check live here once,
// and every fetcher composes them.
//
// The address predicate is a PARAMETER of the name check and the dial control
// rather than a hard-wired call, for one reason: a fetch path can then be
// exercised against an httptest server (which binds loopback) by relaxing the
// predicate in the test alone, while the shipped default — BlockedIP — is never
// relaxed anywhere. A guard nobody can test end-to-end is a guard nobody trusts.
package urlguard

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// reservedV4 lists the not-globally-routable IPv4 ranges (RFC 6890) that none
// of Go's net.IP predicates cover: IsPrivate is RFC 1918 + ULA only and
// IsUnspecified is the single zero address. CGNAT 100.64/10 is the one with
// live internal services behind it (Tailscale tailnets, carrier and cloud
// provider internal addressing); the rest close the not-globally-routable
// class in one pass. The RFC 5737 documentation blocks stay reachable on
// purpose: they are guaranteed non-routable and are the fixture vocabulary
// the repo's own tests and docs use.
var reservedV4 = func() []net.IPNet {
	var nets []net.IPNet
	for _, cidr := range []string{
		"0.0.0.0/8",     // "this network" (RFC 791)
		"100.64.0.0/10", // shared address space / CGNAT (RFC 6598)
		"192.0.0.0/24",  // IETF protocol assignments (RFC 6890)
		"198.18.0.0/15", // benchmarking (RFC 2544)
		"240.0.0.0/4",   // reserved (RFC 1112), incl. limited broadcast
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("urlguard: bad reserved range " + cidr)
		}
		nets = append(nets, *n)
	}
	return nets
}()

// platformMagicV4 lists fixed, provider-owned cloud-platform/metadata addresses
// that live in GLOBALLY-ROUTABLE public IPv4 space, so — unlike everything in
// reservedV4 — none of Go's net.IP predicates nor the reserved ranges above flag
// them, yet each serves a VM's platform/metadata plane and is exactly what this
// guard exists to keep out of reach. Azure's 168.63.129.16 is the WireServer / abcd-lint:allow
// host-platform endpoint (goal-state, extension config incl. protected settings,
// provided DNS/DHCP/health-probe) — a single static IP across every Azure region
// and national cloud (gh-324). The siblings the sweep surfaced, Alibaba
// 100.100.100.200 and Oracle OCI 192.0.0.192, need no entry here: they already abcd-lint:allow
// fall inside reservedV4 (CGNAT 100.64/10 and the 192.0.0.0/24 protocol block).
var platformMagicV4 = map[string]struct{}{
	"168.63.129.16": {}, // Azure WireServer / host platform (gh-324) // abcd-lint:allow
}

// BlockedIP reports whether ip is in a range that must never be fetched:
// loopback (127/8, ::1), link-local (169.254/16, fe80::/10 unicast and
// multicast), private (10/8, 172.16/12, 192.168/16, fc00::/7 via
// net.IP.IsPrivate), the unspecified address, any multicast address, the
// reserved non-private IPv4 ranges above — notably CGNAT 100.64/10 — and the
// fixed provider platform magic IPs in public space (platformMagicV4). This is
// what keeps cloud metadata endpoints (e.g. 169.254.169.254, 168.63.129.16) and abcd-lint:allow
// internal services out of reach.
func BlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		if _, ok := platformMagicV4[v4.String()]; ok {
			return true
		}
		for _, n := range reservedV4 {
			if n.Contains(v4) {
				return true
			}
		}
	}
	// NAT64 (64:ff9b::/96) and 6to4 (2002::/16) embed an IPv4 destination in an
	// IPv6 address the checks above do not flag; a metadata/loopback/private IPv4
	// wrapped in one of these would otherwise slip through. Extract the embedded
	// v4 and re-check it (embeddedIPv4 returns nil for a v4, so no recursion loop).
	if v4 := embeddedIPv4(ip); v4 != nil {
		return BlockedIP(v4)
	}
	return false
}

// embeddedIPv4 returns the IPv4 address a NAT64 (64:ff9b::/96) or 6to4
// (2002::/16) IPv6 address embeds, or nil when ip is not one of those transition
// forms. Scope is the WELL-KNOWN prefixes only: deprecated IPv4-compatible
// (::/96, non-routable) and site-specific NAT64 prefixes (RFC 8215) are out of
// scope — the DNS64 default is the well-known /96 covered here. v4-mapped
// ::ffff:/96 needs no extraction (the standard IsPrivate/IsLoopback checks
// already fold through To4()).
func embeddedIPv4(ip net.IP) net.IP {
	v6 := ip.To16()
	if v6 == nil || ip.To4() != nil {
		return nil // not IPv6 (a plain v4 or v4-mapped needs no extraction)
	}
	// NAT64 well-known prefix 64:ff9b::/96 → last 4 bytes are the v4.
	if v6[0] == 0x00 && v6[1] == 0x64 && v6[2] == 0xff && v6[3] == 0x9b &&
		v6[4] == 0 && v6[5] == 0 && v6[6] == 0 && v6[7] == 0 &&
		v6[8] == 0 && v6[9] == 0 && v6[10] == 0 && v6[11] == 0 {
		return net.IPv4(v6[12], v6[13], v6[14], v6[15])
	}
	// 6to4 prefix 2002::/16 → bytes 2..5 are the v4.
	if v6[0] == 0x20 && v6[1] == 0x02 {
		return net.IPv4(v6[2], v6[3], v6[4], v6[5])
	}
	return nil
}

// CheckHost refuses a host that is an internal/metadata name or that resolves to
// a blocked address, under the shipped default policy.
func CheckHost(host string) error { return CheckHostWith(host, BlockedIP) }

// CheckHostWith is CheckHost with the address predicate supplied by the caller.
// It runs before the initial request and on every redirect hop. An IP literal is
// checked directly (no DNS); a name is rejected outright when it is an
// *.internal / metadata name, otherwise every resolved address is checked.
//
// The NAME half is policy-independent: an *.internal or metadata name never gets
// as far as an address, so no predicate can wave it through.
func CheckHostWith(host string, blocked func(net.IP) bool) error {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "" {
		return errors.New("refusing to fetch a URL with no host")
	}
	if h == "metadata" || h == "metadata.google.internal" || strings.HasSuffix(h, ".internal") {
		return errors.New("refusing to fetch internal/metadata host " + quote(host))
	}
	if ip := net.ParseIP(h); ip != nil {
		if blocked(ip) {
			return errors.New("refusing to fetch " + quote(host) + ": address " + ip.String() +
				" is link-local, loopback, private, or metadata range")
		}
		return nil
	}
	ips, err := net.LookupIP(h)
	if err != nil {
		return errors.New("cannot resolve host " + quote(host) + ": " + err.Error())
	}
	for _, ip := range ips {
		if blocked(ip) {
			return errors.New("refusing to fetch " + quote(host) + ": it resolves to " + ip.String() +
				" (link-local, loopback, private, or metadata range)")
		}
	}
	return nil
}

// DialControl builds a net.Dialer Control hook that re-checks the ACTUAL
// resolved IP of every dialled connection, closing the DNS-rebinding gap between
// a name-based guard and the transport's own resolution.
func DialControl(blocked func(net.IP) bool) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			host = address
		}
		if ip := net.ParseIP(host); ip != nil && blocked(ip) {
			return errors.New("refusing to connect to " + ip.String() +
				": link-local, loopback, private, or metadata range")
		}
		return nil
	}
}

// quote renders a host inside a message the way the callers' own errors do,
// without pulling in fmt for one verb.
func quote(s string) string { return `"` + s + `"` }
