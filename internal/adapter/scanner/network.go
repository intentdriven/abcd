package scanner

import (
	"encoding/hex"
	"net/netip"
	"regexp"
	"strings"
)

// Network-identifier detection, built as an ALLOWLIST INVERSION.
//
// Every legitimate illustrative identifier in a committed file comes from a
// reserved documentation range (the examples-use-reserved-identifiers
// principle): RFC 5737 for IPv4, RFC 3849 for IPv6, RFC 2606/6761 for names,
// RFC 7042 for MACs, and the persona registry for device names. That convention
// is what makes detection cheap: instead of trying to recognise a real address —
// impossible without knowing the network — the detector recognises the small,
// closed set of values that are ALLOWED and flags everything else. Private, LAN
// and CGNAT/tailnet addresses are therefore findings, not exceptions: they are
// exactly the class the 2026-07-29 field incident leaked.
//
// This set is the ONE canonical primitive. It is folded into DefaultPatterns, so
// every scanner consumer inherits it (launch dry-run, lifeboat pack, history
// Stage-1 redaction), and the audit privacy-hygiene rule consults it directly
// rather than keeping a second copy.

// Built-in network kinds.
const (
	kindNetIPv4       = "net:ipv4"
	kindNetIPv6       = "net:ipv6"
	kindNetMAC        = "net:mac"
	kindNetLANHost    = "net:lan_hostname"
	kindNetDeviceHost = "net:device_hostname"
)

var (
	// RFC 5737 IPv4 documentation blocks — the only addresses an example may use.
	docIPv4 = []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
	}
	// RFC 3849 IPv6 documentation prefix.
	docIPv6 = netip.MustParsePrefix("2001:db8::/32")

	// IANA special-use ranges that name no INDIVIDUAL host (maintainer,
	// 2026-07-29). They extend the loopback/unspecified carve-out on the same
	// rationale: an address here identifies a link, a group, a test harness or a
	// protocol mechanism, never a machine, so committing one leaks no topology.
	//
	// What is deliberately NOT here is the whole point of the detector. The
	// private ranges (RFC 1918), CGNAT/tailnet 100.64.0.0/10 and IPv6
	// unique-local fc00::/7 all name real hosts on a real private network — the
	// incident class — and stay flagged. So does 6to4 (2002::/16), which embeds a
	// routable IPv4 address and therefore identifies a host.
	namesNoHostIPv4 = []netip.Prefix{
		netip.MustParsePrefix("169.254.0.0/16"), // link-local autoconfiguration
		netip.MustParsePrefix("224.0.0.0/4"),    // multicast group addresses
		netip.MustParsePrefix("198.18.0.0/15"),  // benchmarking (RFC 2544)
		netip.MustParsePrefix("192.0.0.0/24"),   // IETF protocol assignments
	}
	// nat64WKP is named separately because its low 32 bits are a real IPv4
	// destination, which embeddedIPv4 has to recover.
	nat64WKP = netip.MustParsePrefix("64:ff9b::/96")

	namesNoHostIPv6 = []netip.Prefix{
		netip.MustParsePrefix("fe80::/10"),   // link-local unicast
		netip.MustParsePrefix("ff00::/8"),    // multicast
		nat64WKP,                             // NAT64 well-known prefix
		netip.MustParsePrefix("2001:2::/48"), // benchmarking
	}

	// A dotted quad. Octet-range validation is done by netip in the skip, not by
	// the regex: a regex that spells out 0-255 is unreadable and no more correct.
	ipv4Re = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	// An IPv6 candidate, in two spellings. The second is a hextet followed by at
	// least two more colon-separated groups, whose leading \b is load-bearing:
	// without it the tail of a scope-resolution token (std::string) parses as a
	// valid address by itself. That anchor cannot fire on a "::"-LEADING
	// compressed form, though, which is how a v4-mapped or otherwise compressed
	// address is usually written — so the first alternative carries those, and
	// requires two hextets after the "::" so a bare "::1" is not a candidate.
	// Both spellings carry an optional dotted-quad tail so a v4-mapped address
	// written the usual way ("::ffff:127.0.0.1") is matched WHOLE — a candidate
	// truncated at the first dot parses as a different, and non-exempt, address.
	// Correctness is netip's job; the regex only bounds the candidate.
	ipv6Re = regexp.MustCompile(`::[0-9A-Fa-f]{1,4}(?::[0-9A-Fa-f]{1,4}){1,6}(?:(?:\.\d{1,3}){3})?|\b[0-9A-Fa-f]{1,4}(?::[0-9A-Fa-f]{0,4}){2,7}(?:(?:\.\d{1,3}){3})?`)

	// A MAC address in either separator style. No trailing \b: a fixed-length,
	// pure-word-char token followed by '\b' silently drops the whole match when
	// the next char is a word char (a MAC suffixed with "_eth0" or an alnum), the
	// dead corner the token patterns already retired. The whole-match suppression that
	// the trailing \b incidentally gave — a hex digit right after the address,
	// meaning a longer hex group was truncated — is preserved by
	// insideLongerColonRun's trailing isHexDigit test.
	macRe = regexp.MustCompile(`\b[0-9A-Fa-f]{2}(?::[0-9A-Fa-f]{2}){5}|\b[0-9A-Fa-f]{2}(?:-[0-9A-Fa-f]{2}){5}`)

	// A hostname under a LAN/private suffix: mDNS (.local), the plain .lan
	// convention, and the home-router default (.fritz.box, whose leading label is
	// optional because the bare name is itself the host). At least one label must
	// precede .local/.lan, so a bare "~/.local" directory cannot match. A label
	// may carry a leading underscore so an mDNS SERVICE INSTANCE — the form
	// avahi/dns-sd actually print, "alice-laptop._ipp._tcp.local" — is not broken into
	// pieces by the service labels and missed.
	lanHostRe = regexp.MustCompile(`(?i)\b(?:(?:_?[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+(?:local|lan)|(?:_?[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)*fritz\.box)\b`)

	// A device hostname: <name>-<device noun>. The noun set is deliberately narrow
	// — it excludes "server", "host", "machine", "router" and "box", which are
	// overwhelmingly software-role words in a source tree ("mcp-server",
	// "prompt-router", "state-machine") and would drown the signal. "desktop" and
	// "workstation" are excluded on the same rule, which they failed as plainly:
	// "docker-desktop" and "ubuntu-desktop" are platform names, not machines. A
	// persona-derived name (alice-laptop, carol-server) is allowed by the skip
	// below whether or not its noun is in this set.
	deviceHostRe = regexp.MustCompile(`(?i)\b[a-z0-9][a-z0-9-]*-(?:laptop|macbook|mbp|imac|thinkpad|netbook|chromebook|nas)\b`)
)

// personaNames is the persona registry's given-name sequence
// (.abcd/development/personas.json). A device name derived from a persona is a
// fixture host by construction, so it is allowed. The list is embedded rather
// than read at run time because the registry lives under .abcd/development/,
// which the released binaries do not carry.
var personaNames = map[string]bool{
	"alice": true, "bob": true, "carol": true, "dave": true, "eve": true,
	"frank": true, "grace": true, "henry": true, "iris": true, "jack": true,
	"kira": true, "liam": true, "maya": true, "nia": true,
}

// IsPersonaName reports whether seg is a given name from the persona registry,
// compared the way a path segment spells it: case-folded, because a home
// directory for the persona the conventions mandate is written `/Users/alice`
// whatever case the registry entry uses.
//
// This is the SAME list personaDerivedHost consults for device names
// (iss-2609100505145554): the record asks for a persona home path to be
// treated "the same way a persona-derived device name is", and one list is what
// makes the two answers agree. A caller applying it to a home path must scope it
// to the username POSITION and must yield to the caller's own home — the case
// folding that is harmless for a fixture hostname is not, on its own, enough for
// a home path, because a real account for a person called Alice is spelled
// exactly like the fixture.
func IsPersonaName(seg string) bool {
	return personaNames[strings.ToLower(seg)]
}

// nonHostLabels are names that carry a LAN suffix but name no host. abcd's own
// local tier directory (.abcd/.work.local) appears as the bare string
// "work.local" in product code and its tests, so the canonical set names it once
// here rather than making every occurrence carry a waiver — the iss-153 lesson
// (a detector must not tax legitimate product code) applied to hostnames.
var nonHostLabels = map[string]bool{"work.local": true}

// nonUserHomeSegments are the well-known macOS directories that live under
// /Users but name no user. Flagging them as usernames (iss-153) forced waivers
// onto product code that legitimately writes there.
var nonUserHomeSegments = map[string]bool{
	"shared": true, "guest": true, "public": true,
}

// IsNonUserHomeSegment reports whether seg, the segment immediately after a
// /Users home root, names a system directory rather than a user. The comparison
// is case-insensitive because a case-folding filesystem resolves /users/shared
// to the same directory.
func IsNonUserHomeSegment(seg string) bool {
	return nonUserHomeSegments[strings.ToLower(seg)]
}

// isNetworkKind reports whether a finding kind is one of the network
// identifiers, so a consumer can take that half of a merged pattern set.
func isNetworkKind(kind string) bool {
	switch kind {
	case kindNetIPv4, kindNetIPv6, kindNetMAC, kindNetLANHost, kindNetDeviceHost:
		return true
	}
	return false
}

// NetworkPatterns returns the network-identifier subset of THIS scanner's merged
// patterns — the built-in set with the repo's .abcd/config/pii.json override
// applied. A consumer that wants only the network half (the audit
// privacy-hygiene rule) must come through here rather than through the
// package-level set, or a repo that raises a pattern's severity, exactly as the
// set documents it may, sees no effect at the surface that reports it.
func (s *Scanner) NetworkPatterns() []Pattern {
	var out []Pattern
	for _, p := range s.patterns {
		if isNetworkKind(p.Kind) {
			out = append(out, p)
		}
	}
	return out
}

// NetworkPatterns returns the canonical network-identifier pattern set. It is
// exported so the audit privacy-hygiene rule can run exactly these patterns over
// committed files without duplicating them, and it is included in
// DefaultPatterns so every scanner consumer gets the same detection.
//
// Severity: addresses (IPv4/IPv6/MAC) are hard_fail — the reserved ranges make a
// non-reserved address an authoring error with near-zero ambiguity. The two
// hostname patterns are shape heuristics rather than range checks, so they are
// warn: Stage-1 redaction rewrites them either way (Redact acts on every
// finding), and a repo that wants them blocking can raise the severity through
// .abcd/config/pii.json, which may raise but never lower a floor.
func NetworkPatterns() []Pattern {
	return []Pattern{
		{
			Name: "net_ipv4", Kind: kindNetIPv4, Label: "non-reserved IPv4 address",
			Re: ipv4Re, Severity: SeverityHardFail,
			Skip: func(m string) bool { return allowedIPv4(m) },
			SkipAt: func(line string, start, end int) bool {
				return insideLongerDottedRun(line, start, end) || cidrPrefixDeclaration(line, start, end)
			},
			Suggestion: "replace with an RFC 5737 documentation address (192.0.2.x, 198.51.100.x, 203.0.113.x)",
		},
		{
			Name: "net_ipv6", Kind: kindNetIPv6, Label: "non-reserved IPv6 address",
			Re: ipv6Re, Severity: SeverityHardFail,
			Skip: func(m string) bool { return allowedIPv6(m) },
			SkipAt: func(line string, start, end int) bool {
				return insideLongerColonRun(line, start, end) || cidrPrefixDeclaration(line, start, end)
			},
			Suggestion: "replace with an RFC 3849 documentation address (2001:db8::/32)",
		},
		{
			Name: "net_mac", Kind: kindNetMAC, Label: "non-reserved MAC address",
			Re: macRe, Severity: SeverityHardFail,
			Skip:       func(m string) bool { return allowedMAC(m) },
			SkipAt:     insideLongerColonRun,
			Suggestion: "replace with an RFC 7042 documentation MAC (00:00:5E:00:53:00-FF)",
		},
		{
			Name: "net_lan_hostname", Kind: kindNetLANHost, Label: "LAN hostname",
			Re: lanHostRe, Severity: SeverityWarn,
			Skip: func(m string) bool { return personaDerivedHost(m) },
			SkipAt: func(line string, start, end int) bool {
				return dottedFileOrDirectory(line, start, end) ||
					selectorExpression(line, start, end) ||
					mixedCaseSelector(line, start, end)
			},
			// The suggestion names the exact shape personaDerivedHost accepts,
			// so an author refused on a possessive or capitalised persona host
			// can see why (iss-2609190338409222).
			Suggestion: "replace with a reserved name (example.com, host.test) or a persona fixture host spelled <persona>-<noun> in lower case (alice-mac.local; no possessive, no capitals)",
		},
		{
			Name: "net_device_hostname", Kind: kindNetDeviceHost, Label: "device hostname",
			Re: deviceHostRe, Severity: SeverityWarn,
			Skip: func(m string) bool { return personaDerivedHost(m) },
			SkipAt: func(line string, start, end int) bool {
				return commonNounPhrase(line, start, end) || proseSlug(line, start, end)
			},
			Suggestion: "replace with a persona fixture host spelled <persona>-<noun> in lower case (alice-laptop, bob-macbook; no possessive, no capitals)",
		},
	}
}

// allowedIPv4 reports whether a dotted quad may appear in committed content: a
// reserved documentation address, loopback, the unspecified address, or a
// netmask. A candidate netip cannot parse is not an address at all (a version
// string, a padded octet), so it is allowed rather than guessed at.
func allowedIPv4(m string) bool {
	addr, err := netip.ParseAddr(m)
	if err != nil || !addr.Is4() {
		return true
	}
	if addr.IsLoopback() || addr.IsUnspecified() {
		return true
	}
	for _, p := range docIPv4 {
		if p.Contains(addr) {
			return true
		}
	}
	for _, p := range namesNoHostIPv4 {
		if p.Contains(addr) {
			return true
		}
	}
	return isNetmask4(addr)
}

// isNetmask4 reports whether an address is a contiguous subnet mask
// (255.255.255.0, 255.255.0.0, 255.255.255.255). A mask is a shape, not a host:
// it identifies nobody, and it is the commonest dotted quad in network prose.
func isNetmask4(addr netip.Addr) bool {
	b := addr.As4()
	if b[0] != 255 {
		return false
	}
	v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	inv := ^v
	return inv&(inv+1) == 0 // ones followed by zeros, no gaps
}

// allowedIPv6 mirrors allowedIPv4 for IPv6: the RFC 3849 documentation prefix,
// loopback (::1) and the unspecified address (::) are allowed. A 4-in-6 mapped
// address is judged by its IPv4 half so ::ffff:192.0.2.1 stays allowed while
// ::ffff:<private> does not.
func allowedIPv6(m string) bool {
	addr, ok := parseIPv6Candidate(m)
	if !ok || addr.Is4() {
		return true // not an address, or a quad the IPv4 pattern already judged
	}
	if addr.Is4In6() {
		return allowedIPv4(addr.Unmap().String())
	}
	if addr.IsLoopback() || addr.IsUnspecified() {
		return true
	}
	if docIPv6.Contains(addr) {
		return true
	}
	for _, p := range namesNoHostIPv6 {
		if !p.Contains(addr) {
			continue
		}
		// A special-use PREFIX says nothing about what the rest of the address
		// carries, and two of these ranges carry a host identifier verbatim.
		// Exempting on the prefix alone let the leak through the front door.
		if mac, ok := eui64MAC(addr); ok {
			return allowedMAC(mac) // the interface id IS the interface's MAC
		}
		if v4, ok := embeddedIPv4(addr); ok {
			return allowedIPv4(v4) // NAT64 carries the real destination
		}
		return true
	}
	return false
}

// parseIPv6Candidate parses an IPv6 candidate, recovering the address from a
// candidate the bounding regex ended on trailing punctuation. The inner hextet is
// optional (that is what carries the "::" compression), so a colon immediately
// after an address — the ORDINARY rendering in a tool's error message, where
// ping6 or ssh prints "<address>: Name or service not known" — is swallowed into
// the candidate and netip refuses the result. A parse failure reads as "not an address"
// and ALLOWS it, so exactly the incident class passed through raw. A trailing
// colon is punctuation, never part of an address, so it is trimmed and the
// candidate re-parsed before any allow decision is taken. Anything still
// unparseable (a duration, a scope-resolution token) is genuinely not an address.
func parseIPv6Candidate(m string) (netip.Addr, bool) {
	for {
		if addr, err := netip.ParseAddr(m); err == nil {
			return addr, true
		}
		if !strings.HasSuffix(m, ":") {
			return netip.Addr{}, false
		}
		m = strings.TrimSuffix(m, ":")
	}
}

// eui64MAC returns the MAC address an interface id encodes in modified EUI-64
// form, and whether it is in that form at all. Stateless autoconfiguration
// builds the low 64 bits from the interface's MAC by inserting ff:fe in the
// middle and flipping the universal/local bit, so a link-local address is not
// an anonymous link identifier — it is the hardware address, reversibly. The
// caller judges the recovered MAC by the MAC rules.
func eui64MAC(addr netip.Addr) (string, bool) {
	b := addr.As16()
	iid := b[8:]
	if iid[3] != 0xFF || iid[4] != 0xFE {
		return "", false
	}
	if iid[0] == 0 && iid[1] == 0 && iid[2] == 0 && iid[5] == 0 && iid[6] == 0 && iid[7] == 0 {
		return "", false // an all-zero id that happens to sit on the marker
	}
	mac := []byte{iid[0] ^ 0x02, iid[1], iid[2], iid[5], iid[6], iid[7]}
	parts := make([]string, 0, len(mac))
	for _, o := range mac {
		parts = append(parts, hex.EncodeToString([]byte{o}))
	}
	return strings.Join(parts, ":"), true
}

// embeddedIPv4 returns the IPv4 address a NAT64 well-known-prefix address
// carries in its low 32 bits. The prefix is a translation mechanism, so the
// address it wraps is a real destination — spelled in hex rather than dotted
// quads, which is the only reason it read as exempt.
func embeddedIPv4(addr netip.Addr) (string, bool) {
	if !nat64WKP.Contains(addr) {
		return "", false
	}
	b := addr.As16()
	return netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}).String(), true
}

// allowedMAC reports whether a MAC sits in the RFC 7042 documentation range
// 00:00:5E:00:53:00-FF.
func allowedMAC(m string) bool {
	n := strings.ToLower(strings.ReplaceAll(m, "-", ":"))
	return strings.HasPrefix(n, "00:00:5e:00:53:")
}

// cidrPrefixDeclaration suppresses a match that is the base address of a CIDR
// prefix written immediately after it — "fc00::/7", "100.64.0.0/10",
// "64:ff9b::/96". A prefix names a RANGE, not a host: it identifies nobody, and
// range notation is how the reserved and special-use blocks are documented in
// the first place, including by this detector's own design record. The
// suppression is deliberately narrow: the address must be the MASKED BASE of the
// stated length, so "198.51.100.1/24" (and any URL path that merely begins with a
// digit) is still a finding, and a single-host prefix (/32, /128) is never
// exempt.
func cidrPrefixDeclaration(line string, start, end int) bool {
	if end >= len(line) || line[end] != '/' {
		return false
	}
	i := end + 1
	for i < len(line) && isASCIIDigit(line[i]) {
		i++
	}
	if i == end+1 || i-(end+1) > 3 {
		return false
	}
	// The length must END there. Without this the scan simply stopped at the
	// first non-digit, so "/8x" — a prefix length running into a word, which no
	// range declaration does — was read as "/8" and exempted the base address.
	if i < len(line) && isWordByte(line[i]) {
		return false
	}
	p, err := netip.ParsePrefix(line[start:i])
	if err != nil {
		return false
	}
	if p.Bits() >= p.Addr().BitLen()-1 {
		// /32 and /128 name one host; RFC 3021 /31 (and its /127 counterpart)
		// name exactly two, which is a point-to-point LINK, not a range.
		return false
	}
	return p.Masked() == p
}

// personaDerivedHost reports whether a host or device name is built from the
// persona registry — alice-laptop, bob-desktop, maya-workstation.lan. The first
// hyphen-separated token of the first label is the persona.
//
// The token must be spelled as the registry spells it, in lower case. A
// CAPITALISED given name in front of a device noun is how macOS names a real
// person's machine — it builds the default host name from the account holder's
// name — and a fixture host written in the registry's own convention is never
// spelled that way, so folding the case handed the exemption to exactly the
// class it exists to keep out.
func personaDerivedHost(m string) bool {
	if nonHostLabels[strings.ToLower(m)] {
		return true
	}
	label := m
	if i := strings.IndexByte(label, '.'); i >= 0 {
		label = label[:i]
	}
	if i := strings.IndexByte(label, '-'); i >= 0 {
		label = label[:i]
	}
	return personaNames[label]
}

// insideLongerDottedRun suppresses a quad that is part of a longer dotted number
// run: the leading \b already excludes a digit neighbour, so only a '.' on either
// side can extend the run.
//
// The trailing side needs TWO or more further groups before it counts as a
// version string. One further group is the BSD/macOS address.port rendering that
// netstat, tcpdump and lsof print — "<addr>.41641" — which is the incident class
// in the incident's own output format, and suppressing it hid exactly what the
// detector exists to catch. The cost is that a five-part version number now
// flags; six-part and longer runs stay exempt, and a five-part version is
// vanishingly rare next to address.port output.
func insideLongerDottedRun(line string, start, end int) bool {
	if start > 0 && line[start-1] == '.' {
		return true
	}
	// A digit immediately after the last octet means the regex truncated a longer
	// number (a 4+-digit build tail, "192.168.0.1000"): the candidate is a
	// fragment, not a whole address. ipv4Re carries no trailing \b, so this
	// mirrors insideLongerColonRun's trailing isHexDigit test on the dotted side.
	if end < len(line) && isASCIIDigit(line[end]) {
		return true
	}
	return trailingDottedGroups(line, end) >= 2
}

// trailingDottedGroups counts the ".<digits>" groups immediately following pos.
func trailingDottedGroups(line string, pos int) int {
	from, n := pos, 0
	defer func() { scanMeter.charge(stageSkipAt, pos-from) }()
	for pos < len(line) && line[pos] == '.' {
		i := pos + 1
		for i < len(line) && isASCIIDigit(line[i]) {
			i++
		}
		if i == pos+1 {
			break // a '.' with no digits after it ends the run
		}
		n++
		pos = i
	}
	return n
}

// maxAddressGroups is the most colon-separated groups a candidate could carry
// and still be an address: eight IPv6 hextets (a MAC's six pairs are fewer).
//
// maxRunGroups is the most an address can carry TOGETHER WITH THE GROUP A TOOL
// PRINTS BESIDE IT — the port of "<address>:41641" or "<address>:53", which the
// run absorbs because a port is hex-shaped too. A run longer than that is a
// digest: the shortest one in practice (an MD5 fingerprint) is sixteen groups,
// so the principle the threshold rests on — a digest is much longer than an
// address — survives the extra group with room to spare.
const (
	maxAddressGroups = 8
	maxRunGroups     = maxAddressGroups + 1
)

// insideLongerColonRun suppresses an address candidate that is part of a longer
// colon-separated hex run. A certificate or SSH fingerprint is dozens of hex
// pairs, and any eight consecutive groups of one parse as a perfectly valid
// IPv6 address — so without looking at the neighbours the detector reports a
// digest as a leaked address. It mirrors insideLongerDottedRun on the other
// separator.
//
// The test is the LENGTH of the surrounding run, not the mere presence of a
// colon beside the candidate. A lone adjacent colon is the commonest punctuation
// an identifier carries — a label prefix ("IPv6:", "inet6 addr:", "host:") or
// the colon a tool prints after an address or a MAC — and treating any of them
// as evidence of a digest silently exempted the hard_fail classes wholesale.
// A hex digit on either side is a different fact and still suppresses on its own:
// it means the regex truncated a LONGER hex group, so the candidate is a fragment
// of a bigger token rather than a whole address.
func insideLongerColonRun(line string, start, end int) bool {
	if start > 0 && isHexDigit(line[start-1]) {
		return true
	}
	if end < len(line) && isHexDigit(line[end]) {
		return true
	}
	return colonRunGroups(line, start, end) > maxRunGroups
}

// colonRunGroups counts the NON-EMPTY colon-separated groups of the maximal
// hex/colon run containing [start,end). An empty group carries no hex at all, so
// counting it (the "::" compression, the colon a tool prints after an address)
// inflated a run by a group it never held: a fully expanded eight-hextet address
// beside one colon measured nine groups and was suppressed as a digest.
func colonRunGroups(line string, start, end int) int {
	start = colonRunStart(line, start)
	for end < len(line) && isColonRunByte(line[end]) {
		end++
	}
	scanMeter.charge(stageSkipAt, 2*(end-start))
	n := 0
	for _, g := range strings.Split(line[start:end], ":") {
		if g != "" {
			n++
		}
	}
	return n
}

// colonRunStart walks back to the beginning of the hex/colon run, refusing to
// absorb hex bytes that belong to a LARGER WORD. "IPv6:" ends in a hex digit,
// but that '6' is the tail of the label, not a group of the run: taking it added
// a phantom group to every labelled address. A hex byte preceded by a non-hex
// word byte is part of a word, not of the run.
func colonRunStart(line string, start int) int {
	for start > 0 {
		if line[start-1] == ':' {
			start--
			continue
		}
		if !isHexDigit(line[start-1]) {
			return start
		}
		i := start
		for i > 0 && isHexDigit(line[i-1]) {
			i--
		}
		if i > 0 && isWordByte(line[i-1]) {
			return start // the hex bytes are a word's tail, not a group
		}
		start = i
	}
	return start
}

func isColonRunByte(b byte) bool { return b == ':' || isHexDigit(b) }

// dottedFileOrDirectory suppresses a LAN-suffix match that is really a filename
// or dot-directory rather than a host: ".work.local/" (the local tier),
// "settings.local.json", "DECISIONS.local.md". A leading '.' means the label
// chain began as a dotfile; a trailing '.' followed by a label means a further
// extension. Sentence-final punctuation ("ping alice-laptop.local.") is not
// suppressed, because the '.' there is not followed by a label character.
func dottedFileOrDirectory(line string, start, end int) bool {
	if start > 0 && line[start-1] == '.' {
		return true
	}
	return end+1 < len(line) && line[end] == '.' && isHostLabelByte(line[end+1])
}

// mixedCaseSelector reports whether a LAN-suffix match is a package-qualified
// exported identifier — the shape of `zone := time.Local` — rather than a host.
//
// Mixed case ALONE cannot decide that, and treating it as decisive was a bypass
// anyone could type: one shifted key in the suffix of a machine's name and the
// pattern went silent on it. The mixed case must ALSO sit where
// code puts a value, which is what a selector expression is; prose, a command
// line and a URL never do. The cost of getting the exemption too wide is not a
// noisy report but a silent leak, and the cost of getting it too narrow is a
// corrupted transcript — so the position is required as well as the case.
func mixedCaseSelector(line string, start, end int) bool {
	scanMeter.charge(stageSkipAt, end-start)
	return mixedCaseHostSuffix(line[start:end]) && valuePosition(line, start)
}

// valuePosition reports whether the byte before a match (skipping horizontal
// space) is one that introduces a VALUE in code: an assignment or comparison,
// an opening parenthesis, an argument separator, or a `return`.
func valuePosition(line string, start int) bool {
	i := start
	for i > 0 && (line[i-1] == ' ' || line[i-1] == '\t') {
		i--
	}
	if i == 0 {
		return false
	}
	switch line[i-1] {
	case '=', '(', ',':
		return true
	}
	j := i
	for j > 0 && isWordByte(line[j-1]) {
		j--
	}
	scanMeter.charge(stageSkipAt, start-j)
	return line[j:i] == "return"
}

// mixedCaseHostSuffix reports whether the match's LAST label is written in mixed
// case. A host name is written in ONE case, lower or upper; mixed case is the Go
// exported-identifier convention.
func mixedCaseHostSuffix(m string) bool {
	suffix := m
	if i := strings.LastIndexByte(m, '.'); i >= 0 {
		suffix = m[i+1:]
	}
	return suffix != strings.ToLower(suffix) && suffix != strings.ToUpper(suffix)
}

// selectorExpression reports whether a LAN-suffix match sits where code puts a
// selector rather than where prose or a command line puts a host: it is the
// target of an assignment or comparison ("m.lan = 1"), it heads a block
// ("if cfg.local {"), or it is called or indexed ("cfg.local()", "x.lan[0]").
// A hostname is the OBJECT of a command or the value on the right of a setting,
// never any of those. The test is deliberately narrow — it costs a finding only
// where an identifier is plainly being read as a field.
func selectorExpression(line string, start, end int) bool {
	if end < len(line) && (line[end] == '(' || line[end] == '[') {
		return true
	}
	i := end
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	scanMeter.charge(stageSkipAt, i-end)
	return i < len(line) && (line[i] == '=' || line[i] == '{')
}

// determiners introduce a COMMON NOUN. A device name is a name; "our build-nas",
// "the pre-imac era" and "a synology-nas" are prose about kit.
var determiners = map[string]bool{
	"a": true, "an": true, "the": true, "this": true, "that": true,
	"these": true, "those": true, "my": true, "our": true, "your": true,
	"their": true, "its": true, "every": true, "each": true, "some": true,
	"any": true, "no": true,
}

// slugFunctionWords are the English function words a prose slug strings
// between its hyphens and a machine's name does not carry.
var slugFunctionWords = map[string]bool{
	"a": true, "an": true, "the": true, "to": true, "of": true, "for": true,
	"on": true, "onto": true, "in": true, "into": true, "from": true,
	"with": true, "without": true, "off": true, "over": true, "via": true,
	"and": true, "or": true, "my": true, "our": true, "your": true,
	"their": true, "this": true, "that": true,
}

// proseSlug reports whether a device-hostname match is a hyphenated prose
// slug — a page or file named in words ("migrating-to-the-nas") — rather than
// a machine's name: one of the tokens before its device word is a function
// word. A machine is named <owner>-<device> or <place>-<device>, and a real
// host name built from a sentence is rare enough that sparing it costs less
// than rewriting every slug that ends in a device word, which left a record's
// reproduction step unfollowable (iss-2609240646532741). The token sequence is
// the match's own; a determiner BEFORE the match is commonNounPhrase's case.
func proseSlug(line string, start, end int) bool {
	scanMeter.charge(stageSkipAt, end-start)
	toks := strings.Split(strings.ToLower(line[start:end]), "-")
	for _, t := range toks[:len(toks)-1] {
		if slugFunctionWords[t] {
			return true
		}
	}
	return false
}

// commonNounPhrase reports whether a device-hostname match is preceded by a
// determiner, which makes it a common noun rather than a machine's name. The
// device-noun shape alone cannot tell a machine named after its owner from a
// product line named after its maker, and the determiner is the one signal
// English gives for free.
func commonNounPhrase(line string, start, _ int) bool {
	i := start
	for i > 0 && (line[i-1] == ' ' || line[i-1] == '\t') {
		i--
	}
	if i == start {
		return false // nothing but the match itself before it
	}
	j := i
	for j > 0 && isWordByte(line[j-1]) {
		j--
	}
	scanMeter.charge(stageSkipAt, start-j)
	return determiners[strings.ToLower(line[j:i])]
}

func isASCIIDigit(b byte) bool { return b >= '0' && b <= '9' }

// isWordByte matches the bytes that continue an ASCII word.
func isWordByte(b byte) bool {
	return isASCIIDigit(b) || b == '_' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isHexDigit(b byte) bool {
	return isASCIIDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func isHostLabelByte(b byte) bool {
	return isASCIIDigit(b) || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
