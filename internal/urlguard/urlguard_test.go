package urlguard

import (
	"net"
	"strings"
	"testing"
)

// TestBlockedIP pins the address ranges no abcd fetch may reach. The NAT64 and
// 6to4 cases are the ones a naive guard misses: they wrap a loopback or metadata
// IPv4 inside an IPv6 address that none of net.IP's own predicates flag.
func TestBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"169.254.169.254",                       // cloud metadata (link-local)
		"10.0.0.1", "192.168.1.1", "172.16.5.5", // private  abcd-audit:allow
		"0.0.0.0", "::", // unspecified
		"224.0.0.1", "ff02::1", // multicast
		"fe80::1",            // link-local unicast
		"fc00::1",            // unique local (private)  abcd-audit:allow
		"64:ff9b::a9fe:a9fe", // NAT64 wrapping 169.254.169.254
		"64:ff9b::7f00:1",    // NAT64 wrapping 127.0.0.1
		"2002:a9fe:a9fe::",   // 6to4 wrapping the metadata endpoint  abcd-audit:allow
		"::ffff:127.0.0.1",   // v4-mapped loopback
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("test bug: %q is not an IP", s)
		}
		if !BlockedIP(ip) {
			t.Errorf("BlockedIP(%s) = false, want true", s)
		}
	}

	allowed := []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946", "8.8.8.8"} // abcd-audit:allow — public specimens the guard must NOT refuse
	for _, s := range allowed {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("test bug: %q is not an IP", s)
		}
		if BlockedIP(ip) {
			t.Errorf("BlockedIP(%s) = true, want false", s)
		}
	}
}

// TestBlockedIPRefusesProviderPlatformMagicIPs pins the fixed, provider-owned
// magic addresses that live in PUBLIC (globally-unicast) space and so match none
// of net.IP's range predicates, yet serve a VM's cloud-platform/metadata plane.
// Azure's 168.63.129.16 (the WireServer / host-platform endpoint delivering abcd-lint:allow
// goal-state and protected extension settings) is the case gh-324 caught. Its
// siblings — Alibaba 100.100.100.200 and Oracle OCI 192.0.0.192 — are already abcd-lint:allow
// refused structurally (CGNAT 100.64/10 and the 192.0.0.0/24 protocol block),
// and are pinned here so the sweep is regression-guarded in one place.
func TestBlockedIPRefusesProviderPlatformMagicIPs(t *testing.T) {
	blocked := []string{
		"168.63.129.16",   // Azure WireServer / host platform (gh-324) abcd-audit:allow
		"100.100.100.200", // Alibaba Cloud metadata (already via CGNAT) abcd-audit:allow
		"192.0.0.192",     // Oracle OCI metadata (already via 192.0.0.0/24) abcd-audit:allow
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("test bug: %q is not an IP", s)
		}
		if !BlockedIP(ip) {
			t.Errorf("BlockedIP(%s) = false, want true (provider platform magic IP)", s)
		}
	}
	// The NAT64 unwrap must refuse the Azure endpoint wrapped in an IPv6 form too.
	if !BlockedIP(net.ParseIP("64:ff9b::168.63.129.16")) { // abcd-audit:allow
		t.Error("BlockedIP failed to refuse the NAT64-embedded Azure WireServer form")
	}
	// The nearest public neighbours in Azure's 168.63/16 stay reachable: only the
	// single /32 magic address is blocked, not Microsoft's surrounding public space.
	for _, s := range []string{"168.63.129.15", "168.63.129.17", "168.63.0.1"} { // abcd-audit:allow
		if BlockedIP(net.ParseIP(s)) {
			t.Errorf("BlockedIP(%s) = true, want false (public neighbour, not the magic IP)", s)
		}
	}
}

// TestCheckHostRefusesNamesAndLiterals covers the two refusal paths that resolve
// without any DNS traffic: the internal/metadata NAME guard, and IP literals.
func TestCheckHostRefusesNamesAndLiterals(t *testing.T) {
	cases := []struct{ host, want string }{
		{"", "refusing to fetch a URL with no host"},
		{"metadata", "refusing to fetch internal/metadata host"},
		{"metadata.google.internal", "refusing to fetch internal/metadata host"},
		{"svc.internal", "refusing to fetch internal/metadata host"},
		{"127.0.0.1", "refusing to fetch"},
		{"169.254.169.254", "refusing to fetch"},
		{"10.0.0.1", "refusing to fetch"}, // abcd-audit:allow
	}
	for _, c := range cases {
		err := CheckHost(c.host)
		if err == nil {
			t.Errorf("CheckHost(%q) = nil, want a refusal", c.host)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("CheckHost(%q) = %q, want it to contain %q", c.host, err, c.want)
		}
	}
}

// TestCheckHostWithPolicyAllowsLoopback is what lets a fetch path be exercised
// against an httptest server: the predicate is a parameter, so a test can relax
// it for 127.0.0.1 without the shipped default ever being relaxed.
func TestCheckHostWithPolicyAllowsLoopback(t *testing.T) {
	permissive := func(ip net.IP) bool { return !ip.IsLoopback() && BlockedIP(ip) }
	if err := CheckHostWith("127.0.0.1", permissive); err != nil {
		t.Errorf("CheckHostWith(127.0.0.1, permissive) = %v, want nil", err)
	}
	if err := CheckHostWith("169.254.169.254", permissive); err == nil {
		t.Error("CheckHostWith(169.254.169.254, permissive) = nil, want a refusal")
	}
	// The name guard is policy-independent: a *.internal name is refused whatever
	// the address predicate says, because it never gets as far as an address.
	if err := CheckHostWith("svc.internal", permissive); err == nil {
		t.Error("CheckHostWith(svc.internal, permissive) = nil, want a refusal")
	}
}

// TestDialControlRefusesBlockedAddress pins the connect-time re-check that closes
// the DNS-rebinding gap between the name guard and the transport's own lookup.
func TestDialControlRefusesBlockedAddress(t *testing.T) {
	control := DialControl(BlockedIP)
	if err := control("tcp", "169.254.169.254:80", nil); err == nil {
		t.Error("DialControl allowed a metadata address")
	} else if !strings.Contains(err.Error(), "refusing to connect") {
		t.Errorf("got %q, want a refusing-to-connect error", err)
	}
	if err := control("tcp", "93.184.216.34:443", nil); err != nil { // abcd-audit:allow
		t.Errorf("DialControl refused a public address: %v", err)
	}
}

// TestBlockedIPCoversReservedNonPrivateRanges pins the RFC 6890 ranges Go's
// net.IP predicates do not cover (iss-356 item 2): CGNAT 100.64/10 is
// Tailscale's and carrier/cloud-internal addressing, and none of IsPrivate /
// IsLoopback / IsUnspecified sees it, so a redirect there reached "internal
// services" the package doc promises are out of reach.
func TestBlockedIPCoversReservedNonPrivateRanges(t *testing.T) {
	// Non-reserved literals carry waivers: the point of this test is that these
	// exact values are refused, so the fixture cannot use the documentation
	// ranges the privacy lint would wave through.
	blocked := []string{
		"100.64.0.1", "100.127.255.254", // CGNAT (RFC 6598) abcd-audit:allow
		"0.1.2.3",         // "this network" (RFC 791) abcd-audit:allow
		"192.0.0.8",       // IETF protocol assignments (RFC 6890) abcd-audit:allow
		"198.18.0.1",      // benchmarking (RFC 2544) abcd-audit:allow
		"198.19.255.254",  // benchmarking upper half abcd-audit:allow
		"240.0.0.1",       // reserved (RFC 1112) abcd-audit:allow
		"255.255.255.255", // limited broadcast abcd-audit:allow
	}
	for _, s := range blocked {
		if !BlockedIP(net.ParseIP(s)) {
			t.Errorf("BlockedIP(%s) = false, want true", s)
		}
	}
	// The nearest public neighbours stay reachable.
	for _, s := range []string{ // abcd-audit:allow
		"100.63.255.254", "100.128.0.1", "198.17.255.254", "198.20.0.1", "192.0.1.1", // abcd-audit:allow
	} {
		if BlockedIP(net.ParseIP(s)) {
			t.Errorf("BlockedIP(%s) = true, want false (public address)", s)
		}
	}
	// The NAT64 unwrap re-checks the new ranges too.
	if !BlockedIP(net.ParseIP("64:ff9b::100.64.0.1")) { // abcd-audit:allow
		t.Error("BlockedIP failed to refuse the NAT64-embedded CGNAT form")
	}
}
