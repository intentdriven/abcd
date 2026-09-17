package scanner

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// The network acceptance corpus.
//
// ALLOWED specimens are written as literals on purpose: a reserved documentation
// value (RFC 5737 / 3849 / 2606 / 7042) is safe to commit by definition, and the
// point of the corpus is that those values read plainly.
//
// FLAGGED specimens are ASSEMBLED at runtime instead. They are shape-only values
// (private, CGNAT, ULA, LAN) that identify nobody, but writing them as literals
// would leave non-reserved identifiers sitting in this repo's own tree — the very
// thing the detector exists to stop. The corpus obeys the discipline it enforces.

// v4 assembles a dotted-quad address from its octets.
func v4(a, b, c, d int) string { return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d) }

// v6 assembles an IPv6 address from its hextets; an empty element yields the
// "::" compression.
func v6(groups ...string) string { return strings.Join(groups, ":") }

// mac assembles a colon-separated MAC address from its octets.
func mac(octets ...int) string {
	parts := make([]string, 0, len(octets))
	for _, o := range octets {
		parts = append(parts, fmt.Sprintf("%02x", o))
	}
	return strings.Join(parts, ":")
}

// host assembles a dotted hostname from its labels.
func host(labels ...string) string { return strings.Join(labels, ".") }

// dash assembles a hyphenated device name from its parts.
func dash(parts ...string) string { return strings.Join(parts, "-") }

// scanNet scans one line with the network pattern set only, so a case cannot be
// satisfied by an unrelated secret or identity matcher.
func scanNet(line string) []Finding {
	return ScanText(line, Identity{}, NetworkPatterns(), DefaultIdentitySeverities(), "f")
}

// TestNetworkFlagsNonReservedIdentifiers is the allowlist inversion's positive
// half: every identifier OUTSIDE the reserved documentation ranges is a finding,
// including the CGNAT/tailnet and private/LAN classes named in the incident.
func TestNetworkFlagsNonReservedIdentifiers(t *testing.T) {
	// G1: the EXPANDED spelling of an address carries eight non-empty hextets,
	// which is the most an address can hold — so any colon a tool prints beside
	// it (a label prefix, a trailing colon, a ":<port>") used to push the
	// surrounding run past the digest threshold and suppress the whole class.
	ula8 := v6("fd7a", "115c", "a1e0", "0", "0", "0", "0", "1")
	cases := []struct {
		name string
		kind string
		line string
	}{
		{"cgnat tailnet", "net:ipv4", "peer at " + v4(100, 64, 3, 9) + " responded"},
		{"rfc1918 class c", "net:ipv4", "gateway " + v4(192, 168, 1, 1)},
		{"rfc1918 class a", "net:ipv4", "host " + v4(10, 0, 0, 7)},
		{"rfc1918 class b", "net:ipv4", "host " + v4(172, 16, 4, 4)},
		{"public unicast", "net:ipv4", "resolver " + v4(8, 8, 8, 8)},
		{"ula v6", "net:ipv6", "peer " + v6("fd00", "", "1")},
		{"tailnet v6", "net:ipv6", "peer " + v6("fd7a", "115c", "a1e0", "", "1")},
		{"public v6", "net:ipv6", "host " + v6("2606", "4700", "", "1111")},
		{"mac", "net:mac", "hw addr " + mac(0x02, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e)},
		{"mdns lan suffix", "net:lan_hostname", "ping " + host("printer", "local")},
		{"router lan suffix", "net:lan_hostname", "see " + host("nas", "fritz", "box")},
		{"plain lan suffix", "net:lan_hostname", "ssh " + host("gateway", "lan")},
		{"device name", "net:device_hostname", "backup from " + dash("zeta", "laptop")},
		{"device name mixed case", "net:device_hostname", "on " + dash("Zeta", "Macbook")},
		// A single-host prefix is a host, not a range, so the CIDR exemption
		// below must not reach it; nor may a URL path that starts with a digit.
		{"single host prefix", "net:ipv4", "route " + v4(100, 64, 3, 9) + "/32"},
		{"unmasked prefix", "net:ipv4", "route " + v4(10, 0, 0, 1) + "/24"},
		{"url with numeric path", "net:ipv4", "GET http://" + v4(192, 168, 1, 1) + "/24/x"},
		// Ranges that DO identify private topology stay flagged, even though
		// neighbouring special-use ranges are exempt (maintainer, 2026-07-29).
		{"unique local v6", "net:ipv6", "peer " + v6("fc00", "", "1")},
		{"6to4 embeds a host", "net:ipv6", "peer " + v6("2002", "a9fe", "a9fe", "", "")},
		{"public unicast v6", "net:ipv6", "host " + v6("2606", "2800", "220", "1", "248", "1893", "25c8", "1946")},
		// S1: a link-local interface id in modified EUI-64 form IS the interface
		// MAC, so the link-local exemption must not carry it through.
		{"eui64 link local", "net:ipv6", "peer " + v6("fe80", "", "a683", "e7ff", "fe11", "2233")},
		// S2: NAT64 embeds an IPv4 destination; a private one is still private.
		{"nat64 embedding private", "net:ipv6", "peer " + v6("64", "ff9b", "", "c0a8", "105")},
		// S6: a "::"-leading compressed form must reach the detector at all.
		{"v4 mapped private hex", "net:ipv6", "peer " + v6("", "", "ffff", "c0a8", "105")},
		{"v4 mapped private dotted", "net:ipv4", "peer ::ffff:" + v4(192, 168, 1, 5)},
		// S4: the BSD/macOS address.port rendering is the incident's own output
		// format, so a single trailing group must not read as a version string.
		{"netstat address port", "net:ipv4", "tcp4  0  0  " + v4(100, 64, 3, 9) + ".41641  ESTABLISHED"},
		{"tcpdump address port", "net:ipv4", "IP " + v4(192, 168, 1, 10) + ".22 > 198.51.100.7.443: Flags [S]"},
		// S10: RFC 3021 point-to-point prefixes carry exactly two hosts.
		{"p2p prefix v4", "net:ipv4", "link " + v4(8, 8, 8, 0) + "/31"},
		{"p2p prefix v6", "net:ipv6", "link " + v6("2606", "4700", "", "") + "/127"},
		// S8: an mDNS service instance is a host under a LAN suffix.
		{"mdns service instance", "net:lan_hostname", "found " + host("printer", "_ipp", "_tcp", "local")},
		// F1: the ORDINARY rendering of an address is a tool error message, which
		// puts a colon straight after it. The bounding regex ends such a candidate
		// on an EMPTY hextet, netip refuses that spelling, and a
		// candidate that will not parse used to be read as "not an address" and
		// ALLOWED — so the incident class passed redaction raw.
		{"ping6 error trailing colon", "net:ipv6", "ping6: " + v6("fc00", "", "1") + ": Name or service not known"},
		{"ssh error trailing colon", "net:ipv6", "ssh " + v6("fd7a", "115c", "a1e0", "", "1") + ": Connection refused"},
		{"ula no trailing colon", "net:ipv6", "ping6 " + v6("fc00", "", "1")},
		{"curl error ula", "net:ipv6", "curl: (7) Failed to connect to " + v6("fd12", "3456", "789a", "", "5") + " port 443"},
		// F2: a LONE adjacent colon is the commonest punctuation around an
		// identifier — a label prefix, or the colon a tool prints after it. Reading
		// any colon neighbour as "part of a longer digest" exempted the hard_fail
		// classes wholesale; only a run that is genuinely longer than an address or
		// a MAC could be is a digest.
		{"mac with trailing colon", "net:mac", "HWaddr " + mac(0xa4, 0x83, 0xe7, 0x11, 0x22, 0x33) + ": up"},
		{"labelled v6 no space", "net:ipv6", "IPv6:" + v6("fc00", "", "1")},
		{"host label prefix", "net:ipv6", "peer host:" + v6("fc00", "", "1")},
		{"inet6 addr label", "net:ipv6", "inet6 addr:" + v6("fd12", "3456", "789a", "", "5") + "/64"},
		// F5: the discrimination that keeps a Go selector out (below) must not cost
		// a recorded specimen — an all-uppercase suffix, a service instance on a
		// hyphenated device name, and a bare two-label host all still flag.
		{"uppercase lan suffix", "net:lan_hostname", "ssh " + host(dash("CORP", "NAS"), "LAN")},
		{"device service instance", "net:lan_hostname", "found " + host(dash("zeta", "printer"), "_ipp", "_tcp", "local")},
		// F8: the CIDR exemption stopped reading at the first non-digit, so a
		// prefix length that runs into a word was still taken for a range
		// declaration. A prefix length ends at a non-word boundary.
		{"prefix length runs into a word", "net:ipv4", "net " + v4(10, 0, 0, 0) + "/8x"},
		{"prefix length runs into a letter v6", "net:ipv6", "net " + v6("fc00", "", "") + "/7f"},
		// G1: an EXPANDED address beside a single colon. The digest guard counted
		// the empty group that colon opens, and walked backwards into the hex tail
		// of the label word ("IPv6:"), so an eight-hextet address measured nine
		// groups and was suppressed as a digest — the ordinary output of ping6,
		// getaddrinfo, ssh and ifconfig, wholesale.
		{"expanded ping6 error", "net:ipv6", "ping6: " + ula8 + ": Network is unreachable"},
		{"expanded getaddrinfo error", "net:ipv6", "getaddrinfo " + ula8 + ": Name or service not known"},
		{"expanded ssh error", "net:ipv6", "ssh: connect to host " + ula8 + ": Connection refused"},
		{"expanded label prefix", "net:ipv6", "IPv6:" + ula8},
		{"expanded inet6 addr label", "net:ipv6", "inet6 addr:" + ula8 + "/64"},
		{"expanded all zero ula", "net:ipv6", v6("fd00", "0", "0", "0", "0", "0", "0", "1") + ": unreachable"},
		// The address.port rendering on the colon separator, both spellings a tool
		// prints: one further group is a port, not a digest.
		{"expanded address and port", "net:ipv6", "connect to " + ula8 + ":41641 failed"},
		{"expanded nameserver and port", "net:ipv6", "nameserver=" + ula8 + ":53"},
		{"expanded public unicast", "net:ipv6", "peer " + v6("2606", "2800", "220", "1", "248", "1893", "25c8", "1946") + ": refused"},
		{"expanded 6to4", "net:ipv6", "6to4 " + v6("2002", "0a00", "0001", "0000", "0000", "0000", "0000", "0001") + ": down"},
		// G2: a mixed-case suffix is a one-character bypass anyone can type, so it
		// exempts a match only where CODE puts a value, never in prose, a command
		// or a URL.
		{"mixed case host in prose", "net:lan_hostname", "ping " + host("printer", "Local")},
		{"mixed case plain lan suffix", "net:lan_hostname", "ssh " + host("nas", "Lan")},
		{"mixed case device host", "net:lan_hostname", host(dash("Alice", "MBP"), "Local")},
		{"mixed case host in a url", "net:lan_hostname", "curl http://" + host("Printer", "Local") + "/status"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := scanNet(c.line)
			if !hasKind(got, c.kind) {
				t.Errorf("line %q did not flag %s: %+v", c.line, c.kind, got)
			}
		})
	}
}

// TestNetworkAllowsReservedIdentifiers is the allowlist inversion's negative
// half: a reserved documentation value, a standard protocol value, and a
// persona-derived device name are all silent. A false positive here is what
// would make the lint unusable.
func TestNetworkAllowsReservedIdentifiers(t *testing.T) {
	lines := []string{
		// RFC 5737 documentation blocks.
		"bind 192.0.2.1", "peer 198.51.100.7", "upstream 203.0.113.42",
		// Loopback and unspecified are protocol values, not leaks.
		"listen 127.0.0.1:8080", "listen 0.0.0.0:8080", "loopback 127.0.0.53",
		// A netmask is not an address.
		"mask 255.255.255.0", "mask 255.255.0.0", "broadcast 255.255.255.255",
		// RFC 3849 documentation prefix, loopback, unspecified.
		"peer 2001:db8::1", "peer 2001:DB8:0:0:0:0:0:2", "listen ::1", "listen ::",
		// RFC 7042 documentation MACs.
		"hw 00:00:5E:00:53:00", "hw 00:00:5e:00:53:ff",
		// RFC 2606/6761 reserved names.
		"see example.com", "see www.example.org", "see api.example", "see host.test",
		"see thing.invalid",
		// Persona-derived device names (and the same name with a LAN suffix).
		"from alice-laptop", "from bob-macbook", "from carol-server",
		"ping alice-laptop.local", "ping maya-workstation.lan",
		// Non-host uses of a LAN suffix: a dot-directory, a filename, a dotfile.
		"artefacts live in .abcd/.work.local/scratch/",
		"overrides go in settings.local.json",
		"the wrapper at ~/.local/bin/abcd",
		`skip: [".abcd/\\.work\\.local"]`,
		// IANA special-use ranges that name no INDIVIDUAL host (maintainer,
		// 2026-07-29): the same rationale as the loopback/unspecified carve-out.
		"metadata 169.254.169.254", "autoconf 169.254.0.1",
		"multicast 224.0.0.1", "multicast 239.255.255.250",
		"benchmark 198.18.0.1", "protocol assignment 192.0.0.170",
		"link-local fe80::1", "multicast ff02::1",
		"NAT64 64:ff9b::a9fe:a9fe", "NAT64 64:ff9b::7f00:1",
		"benchmark 2001:2::1",
		// CIDR prefixes name ranges, not hosts — including the special-use blocks
		// this detector's own design record has to spell out.
		"private is 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16",
		"CGNAT 100.64.0.0/10 and benchmark 198.18.0.0/15",
		"link-local fe80::/10, unique local fc00::/7",
		"NAT64 (64:ff9b::/96) and 6to4 (2002::/16)",
		// A prefix declaration ending in ordinary sentence punctuation is still a
		// range: the boundary the length must respect is a WORD boundary.
		"the private block is 10.0.0.0/8.",
		"see 100.64.0.0/10; the CGNAT range",
		// abcd's own local tier is a directory name, not a host.
		`tiers: {".abcd/.work.local", "work.local"}`,
		// C1: colon-separated hex digests are not addresses. A sub-run of eight
		// groups inside one parses as a valid IPv6 unless the neighbours are seen.
		"MD5:16:27:ac:a5:76:28:2d:36:63:1b:56:4d:eb:df:a6:48",
		"SHA256 fingerprint 2f:d4:e1:c6:7a:2d:28:fc:ed:84:9e:e1:bb:76:e7:39:1b:93:eb:12:2f:d4:e1:c6:7a:2d:28:fc:ed:84:9e:e1",
		"hw a4:83:e7:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd",
		// A real-shaped SSH host-key fingerprint (20 groups) stays suppressed: it
		// is longer than any address or MAC could be, which is the only reason a
		// colon neighbour is allowed to silence a candidate at all.
		"SHA1 fingerprint 6a:71:ff:6f:2c:0a:8e:2f:3b:71:9d:6b:5d:aa:cb:1a:0d:2c:5f:9e",
		// S2/S6: a transition address embedding an EXEMPT IPv4 stays exempt.
		"NAT64 64:ff9b::7f00:1", "mapped ::ffff:127.0.0.1", "mapped ::ffff:7f00:1",
		// Shapes that must not be mistaken for addresses.
		"version v1.2.3", "std::string is not an address", "elapsed 12:34:56",
		"released 2026.07.29", "ratio 1.2.3.4.5.6",
		// F5: a selector expression is not a host. Stage-1 redaction rewrites every
		// finding, so a false positive here silently CORRUPTS a stored transcript —
		// the commonest source line in this repo's own transcripts is Go.
		"t := time.Local",
		"if cfg.local {",
		"m.lan = 1",
		"zone := time.Local",
		"x = time.Local",
		// F5b: a hyphenated device noun behind a determiner is a common noun, not a
		// machine — prose about kit, not a hostname.
		"our build-nas pipeline runs nightly",
		"the pre-imac era ended",
		"a synology-nas is a product line",
	}
	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			if got := scanNet(line); len(got) != 0 {
				t.Errorf("reserved/benign line %q flagged: %+v", line, got)
			}
		})
	}
}

// TestNetworkPatternsAreInTheDefaultSet proves the detector is built ONCE and
// reaches every scanner consumer: launch dry-run, lifeboat pack, history
// capture. A set that only the audit rule consults would be a second copy.
func TestNetworkPatternsAreInTheDefaultSet(t *testing.T) {
	have := map[string]bool{}
	for _, p := range DefaultPatterns() {
		have[p.Name] = true
	}
	for _, p := range NetworkPatterns() {
		if !have[p.Name] {
			t.Errorf("network pattern %q is missing from DefaultPatterns", p.Name)
		}
	}
}

// TestRedactRemovesNetworkIdentifiers is the Stage-1 redaction guarantee
// (iss-125): a transcript carrying a LAN hostname and a private address is
// rewritten so a re-scan — the store's stage-two verification — finds nothing.
func TestRedactRemovesNetworkIdentifiers(t *testing.T) {
	text := "ssh " + host("printer", "local") + "\n" +
		"route via " + v4(192, 168, 1, 1) + "\n" +
		"hw " + mac(0x02, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e) + "\n"
	findings := ScanText(text, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	if len(findings) == 0 {
		t.Fatal("no findings on a transcript carrying network identifiers")
	}
	redacted, n := Redact(text, findings)
	if n == 0 {
		t.Fatal("Redact rewrote nothing")
	}
	if residual := ScanText(redacted, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t"); len(residual) != 0 {
		t.Errorf("network identifiers survived redaction: %+v", residual)
	}
	for _, raw := range []string{host("printer", "local"), v4(192, 168, 1, 1)} {
		if strings.Contains(redacted, raw) {
			t.Errorf("redacted text still contains %q", raw)
		}
	}
}

// TestNonUserHomeSegments guards the iss-153 fix at its canonical home: the
// macOS system directories under /Users are not usernames.
func TestNonUserHomeSegments(t *testing.T) {
	for _, seg := range []string{"Shared", "shared", "Guest", "Public"} {
		if !IsNonUserHomeSegment(seg) {
			t.Errorf("IsNonUserHomeSegment(%q) = false, want true", seg)
		}
	}
	for _, seg := range []string{"alice", "bob", "sharedstuff", ""} {
		if IsNonUserHomeSegment(seg) {
			t.Errorf("IsNonUserHomeSegment(%q) = true, want false", seg)
		}
	}
}

// C4: the scanner half of the iss-153 fix, observed at its own surface. Without
// the allowlist a macOS system directory reports as a third-party home path.
func TestIdentitySkipsNonUserHomeSegments(t *testing.T) {
	id := Identity{HomePath: "/tmp/not-the-caller", HomeUser: "not-the-caller"}
	scan := func(line string) []Finding {
		return ScanText(line, id, nil, DefaultIdentitySeverities(), "f")
	}
	if got := scan("the installer writes under /Users/Shared"); hasKind(got, kindHomeOther) {
		t.Errorf("/Users/Shared flagged as a third-party home path: %+v", got)
	}
	if got := scan("and never /Users/Guest"); hasKind(got, kindHomeOther) {
		t.Errorf("/Users/Guest flagged as a third-party home path: %+v", got)
	}
	if got := scan("a leak at /Users/sharedstuff/x"); !hasKind(got, kindHomeOther) { // abcd-audit:allow — the specimen IS the leak under test
		t.Errorf("a real username under /Users was not flagged: %+v", got)
	}
}

// S3: an exempt system directory must not shield a username reached by TRAVERSAL
// out of it. The one-segment match stopped at the exempt segment and nothing
// looked past it — a regression against the behaviour before the exemption
// landed.
//
// A segment reached DIRECTLY beneath the system root is no longer flagged
// (iss-2609100505145554): a system root is not a home root, so nothing under it
// sits in the username position, and flagging it refused writes on the product's
// own install docs. The traversal assertions below are the ones that matter, and
// they all still hold.
func TestIdentityNestedUsernameUnderSystemDirectory(t *testing.T) {
	id := Identity{HomePath: "/tmp/not-the-caller", HomeUser: "not-the-caller"}
	scan := func(line string) []Finding {
		return ScanText(line, id, nil, DefaultIdentitySeverities(), "f")
	}
	nested := "keys at /Users/Shared/" + strings.Join([]string{"j", "doe"}, "") + "/keys.txt"
	if got := scan(nested); hasKind(got, kindHomeOther) {
		t.Fatalf("a segment reached directly under a system root is a shared-folder entry, not a home path: %+v", got)
	}
	if got := scan("the tier lives at /Users/Shared"); hasKind(got, kindHomeOther) {
		t.Errorf("a bare system directory flagged: %+v", got)
	}
	if got := scan("the tier lives at /Users/Shared/ and nowhere else"); hasKind(got, kindHomeOther) {
		t.Errorf("a system directory with no further segment flagged: %+v", got)
	}
	// F3b: a dots-only or an empty segment names no user, but it must not end the
	// search — either one between the system directory and a name re-creates the
	// shield the exemption is not allowed to give.
	user := strings.Join([]string{"j", "doe"}, "")
	for _, line := range []string{
		"keys at /Users/Shared/../" + user + "/keys.txt",
		"keys at /Users/Shared/./" + user + "/keys.txt",
		"keys at /Users/Shared//" + user + "/keys.txt",
	} {
		got := scan(line)
		if !hasKind(got, kindHomeOther) {
			t.Errorf("a username behind a dots-only or empty segment was not flagged in %q: %+v", line, got)
		}
		for _, f := range got {
			if f.Kind == kindHomeOther && !strings.Contains(f.Matched, user) {
				t.Errorf("matched span %q does not cover the nested username in %q", f.Matched, line)
			}
		}
	}
	if got := scan("the tier lives at /Users/Shared/.."); hasKind(got, kindHomeOther) {
		t.Errorf("a system directory with only a parent marker after it flagged: %+v", got)
	}
}

// F4: the SERIALIZED finding surface owes the same trade as redaction. `abcd
// launch --json` writes it to stdout and CI may archive it, so a head-and-tail
// fingerprint there leaks a MAC's vendor bytes, an address's prefix and final
// hextet, and a hostname's head and suffix — the re-identifying parts.
func TestSerializedNetworkIdentifiersMaskedWhole(t *testing.T) {
	for _, r := range []string{
		mac(0xa4, 0x83, 0xe7, 0x11, 0x22, 0x33),
		v6("fd7a", "115c", "a1e0", "", "1"),
		host(dash("zeta", "crowd"), "fritz", "box"),
	} {
		findings := ScanText("peer "+r+" seen", Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")
		if len(findings) == 0 {
			t.Fatalf("no finding for %q", r)
		}
		blob, err := json.Marshal(findings)
		if err != nil {
			t.Fatal(err)
		}
		js := string(blob)
		if strings.Contains(js, r) {
			t.Errorf("serialized finding still contains %q: %s", r, js)
		}
		if head := r[:3]; strings.Contains(js, head) {
			t.Errorf("serialized finding leaks the head %q of %q: %s", head, r, js)
		}
		if tail := r[len(r)-3:]; strings.Contains(js, tail) {
			t.Errorf("serialized finding leaks the tail %q of %q: %s", tail, r, js)
		}
	}
}

// S9: network identifiers are IDENTIFIERS, not secrets, so they must be masked
// whole. The secret fingerprint keeps the first three and last two runes, which
// preserves a MAC's vendor bytes and a hostname's head — enough to re-identify.
func TestRedactMasksNetworkIdentifiersWhole(t *testing.T) {
	raw := []string{
		mac(0xa4, 0x83, 0xe7, 0x11, 0x22, 0x33),
		v6("fd7a", "115c", "a1e0", "", "1"),
		host(dash("zeta", "crowd"), "fritz", "box"),
		v4(100, 64, 3, 9),
	}
	text := "peer " + strings.Join(raw, " and ") + "\n"
	findings := ScanText(text, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	redacted, _ := Redact(text, findings)
	for _, r := range raw {
		if strings.Contains(redacted, r) {
			t.Errorf("redacted text still contains %q", r)
		}
		// No fingerprint window either: the leading runes of an identifier are
		// exactly what makes it re-identifiable.
		if head := r[:3]; strings.Contains(redacted, head) {
			t.Errorf("redacted text leaks the head %q of %q: %s", head, r, redacted)
		}
	}
	if residual := ScanText(redacted, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t"); len(residual) != 0 {
		t.Errorf("identifiers survived redaction: %+v", residual)
	}
}

// TestNetworkAddressCaughtWithWordSuffix is the attack-input test for the
// trailing-\b dead corner on the hard_fail address patterns (iss-307). ipv4Re
// and macRe were fixed-length/word-char + trailing \b, so an address followed by
// a word char ('_' or an alnum) had no ASCII boundary and RE2 dropped the whole
// match — a silent hard_fail miss on a redaction path. Each suffixed address
// must be caught; the version/digest controls that the compensating guards keep
// silent must stay silent.
func TestNetworkAddressCaughtWithWordSuffix(t *testing.T) {
	colonMAC := mac(0xa4, 0x83, 0xe7, 0x11, 0x22, 0x33)
	dashMAC := strings.ReplaceAll(colonMAC, ":", "-")
	caught := []struct {
		name, kind, line string
	}{
		{"ipv4 underscore suffix", "net:ipv4", "logs/" + v4(192, 168, 1, 44) + "_2026.pcap"},
		{"ipv4 json key", "net:ipv4", `{"` + v4(192, 168, 1, 44) + `_gw": 1}`},
		{"ipv4 alnum suffix", "net:ipv4", "host " + v4(10, 1, 2, 3) + "gw"},
		{"mac colon underscore suffix", "net:mac", "iface " + colonMAC + "_eth0 up"},
		{"mac dash underscore suffix", "net:mac", "hw " + dashMAC + "_eth0"},
	}
	for _, c := range caught {
		if !hasKind(scanNet(c.line), c.kind) {
			t.Errorf("%s: %q not caught, want %s", c.name, c.line, c.kind)
		}
	}

	// Controls the compensating guards must keep silent: a truncated-number
	// fragment (a 4-digit tail) is not an address, and a dotted version string
	// stays exempt.
	silent := []struct {
		name, line string
	}{
		{"four-digit tail truncation", "addr [::ffff:192.168.0.1000] noted"},
		{"dotted version string", "bumped to 1.2.3.1234 today"},
	}
	for _, c := range silent {
		for _, f := range scanNet(c.line) {
			if f.Kind == "net:ipv4" {
				t.Errorf("%s: %q flagged %s, want silent", c.name, c.line, f.Kind)
			}
		}
	}
}
