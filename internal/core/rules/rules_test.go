package rules

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestLoadRefusesSymlinkedRulesFile (iss-66 C19) proves a symlinked rules.json
// LEAF is refused via O_NOFOLLOW. The directory-symlink test covers only the
// ancestor guard; reverting O_NOFOLLOW makes Load follow the leaf link and read
// an arbitrary target, so this test fails on that revert.
func TestLoadRefusesSymlinkedRulesFile(t *testing.T) {
	dir := t.TempDir()
	abcd := filepath.Join(dir, ".abcd")
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "real-rules.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(abcd, "rules.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("a symlinked rules.json leaf must be refused, not followed")
	}
}

// TestLoadRefusesFifoRulesFile (iss-66 F1) proves a FIFO leaf is rejected
// promptly rather than hung on. O_NONBLOCK lets the open return so the
// regular-file check fails closed; without it, open(FIFO, O_RDONLY) blocks
// forever and wedges the (non-blocking-by-contract) prompt-router hook.
func TestLoadRefusesFifoRulesFile(t *testing.T) {
	dir := t.TempDir()
	abcd := filepath.Join(dir, ".abcd")
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(abcd, "rules.json"), 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Load(dir)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a FIFO rules.json must be refused, not read")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Load hung on a FIFO rules.json leaf (must never block the prompt-router hook)")
	}
}

func names(ds []ResolvedDomain) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Name
	}
	return out
}

func has(ds []ResolvedDomain, name string) bool {
	for _, d := range ds {
		if d.Name == name {
			return true
		}
	}
	return false
}

func TestDefaultsParseAndValidate(t *testing.T) {
	rs := Defaults()
	if rs.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", rs.SchemaVersion)
	}
	if err := Validate(rs); err != nil {
		t.Fatalf("bundled defaults fail validation: %v", err)
	}
	for _, want := range []string{"COMMITTING", "DOCUMENTATION", "ROADMAP", "ISSUES", "INTENTS", "LIFEBOAT", "PII", "OPINIONS"} {
		if _, ok := rs.Domains[want]; !ok {
			t.Errorf("default domain %q missing", want)
		}
	}
}

func TestOpinionsDomainPointsAtPrinciplesNotCopies(t *testing.T) {
	rs := Defaults()
	// Recall on an opinion/convention/SOTA prompt.
	if !has(rs.Match("what's the SOTA approach and our convention here"), "OPINIONS") {
		t.Fatal("OPINIONS did not recall-match a conventions prompt")
	}
	op := rs.Domains["OPINIONS"]
	// Every rule points at a principle file (one-canonical-primitive): it names a
	// path under .abcd/development/principles/, it does not inline the principle.
	pointers := 0
	for _, r := range op.Rules {
		if strings.Contains(r, ".abcd/development/principles/") {
			pointers++
		}
	}
	if pointers < len(op.Rules)-1 { // allow the one index rule to name the dir
		t.Fatalf("OPINIONS rules should point at principle files, got %d/%d", pointers, len(op.Rules))
	}
}

// TestPIIDomainRecallsNetworkContexts (iss-156) proves the PII domain fires on
// the network/infra prompts that most need it. The original recall set (secret,
// token, credential, pii, redact, hostname, email) missed every one of these, so
// an agent writing up a mesh-VPN or firewall investigation got zero injection.
// Recall matching is word-bounded (see indexPrompt/termHit), so the bare "ip"
// keyword is safe: it matches only a standalone token, never "script" or "zip".
func TestPIIDomainRecallsNetworkContexts(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"investigating the tailscale outage",
		"basic reachability looks fine from here",
		"the vpn drops every hour",
		"the firewall is dropping the connection",
		"write up what the network scan found",
		"note the ip we connected to",
	} {
		if !has(rs.Match(prompt), "PII") {
			t.Errorf("network-context prompt %q did not recall PII, got %v", prompt, names(rs.Match(prompt)))
		}
	}
}

// TestPIIDomainRecallsNetworkVocabularyGaps pins the coverage the first keyword
// pass missed, each row a distinct hole in the matcher rather than a synonym:
//
//   - "ips" — stem() has a 3-character floor, so stem("ips") == "ips" and the
//     bare "ip" keyword never sees the plural.
//   - "ipv4"/"ipv6" — version-qualified tokens share no stem with "ip".
//   - "reachable" — stem() does not bridge "-ability" to "-able", so the
//     "reachability" keyword cannot reach the adjective.
//   - "unreachable" — stem() strips no "un-" prefix either, and unreachable is
//     the ICMP-canonical failure word a network write-up actually uses.
//   - "mac addresses" — the rule text forbids MACs, so the domain must recall
//     that vocabulary; bare "mac" is unsafe (an Apple Mac), hence the phrase
//     alias, which matches via the stemmed-phrase path in termHit
//     ("addresses" stems to "address").
//   - dns / ssh / subnet / tailnet / wireguard — the ordinary vocabulary of the
//     incident class, none of it reachable from the credential keywords.
//
// Prompt values are deliberately identifier-free (no real host, address or
// network is named) so the test corpus itself obeys the rule under test.
func TestPIIDomainRecallsNetworkVocabularyGaps(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"list the IPs of every node",
		"an IPv4 address in the config",
		"is there an IPv6 literal in this file",
		"confirm the node is reachable over the tunnel",
		"the host is unreachable, write up what we found",
		"the mac addresses of both nics",
		"dns is not resolving for the node",
		"note the tailnet address of the laptop",
		"the ssh config for the build box",
		"which subnet does the host sit on",
		"wireguard keeps dropping the peer",
	} {
		if !has(rs.Match(prompt), "PII") {
			t.Errorf("network-context prompt %q did not recall PII, got %v", prompt, names(rs.Match(prompt)))
		}
	}
}

// TestPIIDomainRecallIPIsWordBounded guards the "ip" keyword against the
// over-matching it would cause under substring semantics: a prompt about a
// script, a description, or a zip file must not inject PII.
func TestPIIDomainRecallIPIsWordBounded(t *testing.T) {
	rs := Defaults()
	if got := rs.Match("unzip the script and edit its description"); has(got, "PII") {
		t.Fatalf("bare 'ip' keyword over-matched a non-network prompt, got %v", names(got))
	}
}

// TestPIIDomainForbidsCommittingNetworkIdentifiers (iss-156) proves the injected
// rule text carries the never-commit-hostnames/IPs rule. It previously lived only
// in a parent CLAUDE.md privacy section, so the rules loader never injected it.
// The rule must name MAC addresses too, and cite the full reserved-value set the
// scanner and the audit privacy-hygiene rule already cite (RFC 5737 IPv4, 3849
// IPv6, 2606 names, 7042 MACs) — pinning all four here means dropping or drifting
// a citation, or deleting the clause, fails the test rather than passing quietly.
// The remedy wording is pinned too: redact-or-omit must stay the default and
// documentation values must stay scoped to illustrative examples — the first
// draft's unconditional "use the reserved ranges instead" read as an instruction
// to silently substitute plausible-looking fake identifiers into factual records.
func TestPIIDomainForbidsCommittingNetworkIdentifiers(t *testing.T) {
	pii, ok := Defaults().Lookup("PII")
	if !ok {
		t.Fatal("PII domain missing from the bundled defaults")
	}
	for _, rule := range pii.Rules {
		low := strings.ToLower(rule)
		if !strings.Contains(low, "hostname") || !strings.Contains(low, "ip address") ||
			!strings.Contains(low, "mac address") || !strings.Contains(low, "network identifier") {
			continue
		}
		var missing []string
		for _, rfc := range []string{"5737", "3849", "2606", "7042"} {
			if !strings.Contains(low, rfc) {
				missing = append(missing, rfc)
			}
		}
		if len(missing) > 0 {
			t.Fatalf("PII network-identifier rule omits RFC citation(s) %v: %q", missing, rule)
		}
		for _, want := range []string{"redact", "omit", "illustrative"} {
			if !strings.Contains(low, want) {
				t.Fatalf("PII network-identifier rule lost the %q remedy wording: %q", want, rule)
			}
		}
		return
	}
	t.Fatalf("no PII rule forbids committing hostnames, IP addresses, MAC addresses, or live network identifiers: %v", pii.Rules)
}

func TestMatchRecallKeyword(t *testing.T) {
	rs := Defaults()
	got := rs.Match("let's commit and push this")
	if !has(got, "COMMITTING") {
		t.Fatalf("expected COMMITTING, got %v", names(got))
	}
}

func TestMatchMultiWordAlias(t *testing.T) {
	rs := Defaults()
	got := rs.Match("please open the pull request now")
	if !has(got, "COMMITTING") {
		t.Fatalf("expected COMMITTING via multi-word alias, got %v", names(got))
	}
}

func TestMatchNoHitInjectsNothing(t *testing.T) {
	rs := Defaults()
	got := rs.Match("render a react component with a gradient background")
	if len(got) != 0 {
		t.Fatalf("expected zero domains on no-match, got %v", names(got))
	}
}

func TestRecallStemmingMatchesInflections(t *testing.T) {
	rs := Defaults()
	// Plural/tense variants recall their keyword: "issues"->issue, "pushes"->push.
	if !has(rs.Match("triage the open issues"), "ISSUES") {
		t.Fatal("plural 'issues' did not stem-match ISSUES")
	}
	if !has(rs.Match("nothing pushes to main"), "COMMITTING") {
		t.Fatal("'pushes' did not stem-match COMMITTING")
	}
}

func TestRecallStemmingNoOverMatch(t *testing.T) {
	// The named failure mode from the SOTA verdict: "test"-stemming must not
	// activate on "attestation". A short keyword is left unstemmed.
	rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"QA": {State: StateActive, Recall: []string{"test"}, Rules: []string{"r"}},
	}}
	if has(rs.Match("the attestation manifesto and progress"), "QA") {
		t.Fatal("stemming over-matched: 'attestation' activated a 'test' recall")
	}
	// But genuine inflections still match.
	if !has(rs.Match("running the tests now"), "QA") {
		t.Fatal("'tests' did not stem-match 'test'")
	}
	if !has(rs.Match("testing the flow"), "QA") {
		t.Fatal("'testing' did not stem-match 'test'")
	}
}

// TestRecallStemmingEDropAndDoubledConsonant (B09) covers the inflections stem()
// alone cannot round-trip: an "-ing"/"-ed" form whose base keyword either dropped
// a final "e" (merge->merging, rebase->rebasing) or doubled its final consonant
// (commit->committing). Each of these prompts must inject COMMITTING, whose recall
// keywords are [commit, push, branch, merge, rebase].
func TestRecallStemmingEDropAndDoubledConsonant(t *testing.T) {
	rs := Defaults()
	cases := []string{
		"committing the fix now",             // committ -> commit (undouble)
		"the change was committed yesterday", // committed -> committ -> commit
		"help me with merging",               // merg -> merge (e-restore)
		"i just merged the branch",           // merged -> merg -> merge
		"i finished rebasing, what now",      // rebas -> rebase (e-restore)
		"the topic was rebased cleanly",      // rebased -> rebas -> rebase
	}
	for _, p := range cases {
		if !has(rs.Match(p), "COMMITTING") {
			t.Errorf("inflected prompt %q did not stem-match COMMITTING, got %v", p, names(rs.Match(p)))
		}
	}
}

// TestRecallStemmingEDropNoOverMatch guards the e-drop/undouble variants against
// over-activation: a token whose "-ing"/"-ed" strip happens to end near a keyword
// must not spuriously match an unrelated domain.
func TestRecallStemmingEDropNoOverMatch(t *testing.T) {
	rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"QA": {State: StateActive, Recall: []string{"merge"}, Rules: []string{"r"}},
	}}
	if has(rs.Match("meridian coordinates and margins"), "QA") {
		t.Fatal("e-drop stemming over-matched an unrelated word against 'merge'")
	}
}

// TestMatchMultiWordAliasStemsInflectedPhrase (B32) proves a plural phrase hits
// its singular multi-word alias. COMMITTING's alias "pull request" must match the
// prompt "please review my pull requests" via the stemmed padded prompt, even
// though no single recall keyword appears.
func TestMatchMultiWordAliasStemsInflectedPhrase(t *testing.T) {
	rs := Defaults()
	if !has(rs.Match("please review my pull requests"), "COMMITTING") {
		t.Fatalf("plural phrase 'pull requests' did not stem-match the 'pull request' alias, got %v",
			names(rs.Match("please review my pull requests")))
	}
}

func TestLoadRefusesSymlinkedAbcdDir(t *testing.T) {
	dir := t.TempDir()
	real := t.TempDir()
	if err := os.WriteFile(filepath.Join(real, "rules.json"),
		[]byte(`{"schema_version":1,"domains":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, ".abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("Load followed a symlinked .abcd directory")
	}
}

func TestMatchWordBoundaryNoSubstringFalsePositive(t *testing.T) {
	rs := Defaults()
	// "scommitted" must not trigger COMMITTING's "commit" keyword.
	got := rs.Match("the discommitted witticism")
	if has(got, "COMMITTING") {
		t.Fatalf("substring false positive: %v", names(got))
	}
}

func TestStarCommandActivatesRegardlessOfKeyword(t *testing.T) {
	rs := Defaults()
	got := rs.Match("*ROADMAP draft the next milestone")
	if !has(got, "ROADMAP") {
		t.Fatalf("star-command did not activate ROADMAP: %v", names(got))
	}
}

func TestStarCommandBoundaries(t *testing.T) {
	// Synthetic domain whose name is NOT one of its recall keywords, so a hit can
	// only come from star-command parsing (not incidental recall on the name).
	rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"ZONK": {State: StateActive, Recall: []string{"zzznomatch"}, Rules: []string{"r"}},
	}}
	// Positive control: a well-formed star-command activates.
	if !has(rs.Match("*ZONK do the thing"), "ZONK") {
		t.Fatal("well-formed *ZONK not activated")
	}
	// Boundary rejections: star preceded by a non-space, glued to a longer token,
	// or not an uppercase name are NOT star-commands.
	for _, p := range []string{"path/*ZONK", "e*ZONK", "*ZONKING now", "list *.py files", "* ZONK bullet"} {
		if got := rs.Match(p); has(got, "ZONK") {
			t.Errorf("%q wrongly parsed as a star-command: %v", p, names(got))
		}
	}
}

func TestStarCommandActivatesDormant(t *testing.T) {
	rs := Defaults()
	d := rs.Domains["ROADMAP"]
	d.State = StateDormant
	rs.Domains["ROADMAP"] = d
	// dormant: no recall activation...
	if has(rs.Match("update the roadmap"), "ROADMAP") {
		t.Fatal("dormant domain activated by recall")
	}
	// ...but star-command overrides dormant.
	if !has(rs.Match("*ROADMAP go"), "ROADMAP") {
		t.Fatal("star-command did not override dormant")
	}
}

func TestKillSwitchSuppressesEverything(t *testing.T) {
	rs := Defaults()
	rs.Disabled = true
	if got := rs.Match("commit and push"); len(got) != 0 {
		t.Fatalf("kill switch did not suppress recall: %v", names(got))
	}
	// Star-command must NOT bypass the kill switch.
	if got := rs.Match("*ROADMAP go"); len(got) != 0 {
		t.Fatalf("star-command bypassed the kill switch: %v", names(got))
	}
}

func TestMergePerFieldOverride(t *testing.T) {
	base := Defaults()
	over := RuleSet{
		SchemaVersion: 1,
		Domains: map[string]Domain{
			"ROADMAP": {State: StateDormant}, // silence, keep recall/rules
			"CUSTOM":  {State: StateActive, Recall: []string{"widget"}, Rules: []string{"do the thing"}},
		},
	}
	merged := Merge(base, over)
	// ROADMAP keeps its default recall but is now dormant.
	if got := merged.Domains["ROADMAP"]; got.State != StateDormant || len(got.Recall) == 0 {
		t.Fatalf("per-field override wrong: state=%q recall=%v", got.State, got.Recall)
	}
	// CUSTOM added.
	if _, ok := merged.Domains["CUSTOM"]; !ok {
		t.Fatal("custom domain not merged in")
	}
	if !has(merged.Match("build a widget"), "CUSTOM") {
		t.Fatal("merged custom domain does not recall-match")
	}
}

func TestMergeKillSwitchIsSticky(t *testing.T) {
	base := Defaults()
	if got := Merge(base, RuleSet{Disabled: true}); !got.Disabled {
		t.Fatal("repo override could not enable the kill switch")
	}
}

// A base with no domains is a valid RuleSet (Validate accepts it), and Merge
// promises new domain keys are added — so merging onto it must add them rather
// than panic on an unallocated map.
func TestMergeNilBaseDomainsAddsNewKeys(t *testing.T) {
	base := RuleSet{SchemaVersion: 1}
	if err := Validate(base); err != nil {
		t.Fatalf("base with nil Domains should validate: %v", err)
	}
	over := RuleSet{
		SchemaVersion: 1,
		Domains: map[string]Domain{
			"CUSTOM": {State: StateActive, Recall: []string{"widget"}, Rules: []string{"do the thing"}},
		},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Merge onto a nil-Domains base panicked: %v", r)
		}
	}()
	merged := Merge(base, over)
	got, ok := merged.Domains["CUSTOM"]
	if !ok {
		t.Fatalf("custom domain not merged in, got %v", merged.Domains)
	}
	if got.State != StateActive {
		t.Fatalf("state = %q, want %q", got.State, StateActive)
	}
	if len(got.Recall) != 1 || got.Recall[0] != "widget" {
		t.Fatalf("recall = %v, want [widget]", got.Recall)
	}
	if len(got.Rules) != 1 || got.Rules[0] != "do the thing" {
		t.Fatalf("rules = %v, want [do the thing]", got.Rules)
	}
	if !has(merged.Match("build a widget"), "CUSTOM") {
		t.Fatal("merged custom domain does not recall-match")
	}
}

func TestValidateRejectsBadDomainName(t *testing.T) {
	rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{"bad-name": {}}}
	if err := Validate(rs); err == nil {
		t.Fatal("expected validation error for lowercase/hyphen domain name")
	}
}

func TestValidateRejectsBadSchemaVersion(t *testing.T) {
	if err := Validate(RuleSet{SchemaVersion: 2}); err == nil {
		t.Fatal("expected validation error for schema_version != 1")
	}
}

func TestValidateRejectsBadState(t *testing.T) {
	rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{"X": {State: "paused"}}}
	if err := Validate(rs); err == nil {
		t.Fatal("expected validation error for unknown state")
	}
}

func TestLoadAbsentFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	rs, err := Load(dir)
	if err != nil {
		t.Fatalf("Load with no rules.json: %v", err)
	}
	if len(rs.Domains) != len(Defaults().Domains) {
		t.Fatalf("absent rules.json should yield defaults, got %d domains", len(rs.Domains))
	}
}

func TestLoadMergesRepoFile(t *testing.T) {
	dir := t.TempDir()
	writeRepoRules(t, dir, `{"schema_version":1,"disabled":false,"domains":{"ROADMAP":{"state":"dormant"}}}`)
	rs, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rs.Domains["ROADMAP"].State != StateDormant {
		t.Fatal("repo override not applied")
	}
	if len(rs.Domains["ROADMAP"].Rules) == 0 {
		t.Fatal("per-field merge dropped default rules")
	}
}

func TestLoadMalformedFileFailsClosed(t *testing.T) {
	dir := t.TempDir()
	writeRepoRules(t, dir, `{ this is not json `)
	if _, err := Load(dir); err == nil {
		t.Fatal("malformed rules.json must fail closed, not silently fall back")
	}
}

func TestRenderContainsDomainsAndHeader(t *testing.T) {
	rs := Defaults()
	out := Render(rs.Match("commit and push"))
	if out == "" {
		t.Fatal("render produced nothing for a match")
	}
	if !contains(out, "COMMITTING") {
		t.Fatalf("render missing domain name:\n%s", out)
	}
}

func TestRenderEmptyIsZeroBytes(t *testing.T) {
	if out := Render(nil); out != "" {
		t.Fatalf("no-match render must be zero bytes (D3), got %q", out)
	}
}

func TestSignatureStableAndDistinct(t *testing.T) {
	rs := Defaults()
	commit := pick(rs, "COMMITTING")
	docs := pick(rs, "DOCUMENTATION")
	if Signature(commit) != Signature(commit) {
		t.Fatal("signature not stable")
	}
	if Signature(commit) == Signature(docs) {
		t.Fatal("distinct domains share a signature")
	}
	// Content drift changes the signature.
	drift := commit
	drift.Rules = append([]string{"a new rule"}, drift.Rules...)
	if Signature(drift) == Signature(commit) {
		t.Fatal("signature did not change on content drift")
	}
}

func TestActiveExcludesDormantAndKillSwitch(t *testing.T) {
	rs := Defaults()
	full := len(rs.Active())
	if full != len(rs.Domains) {
		t.Fatalf("Active() = %d, want all %d default domains", full, len(rs.Domains))
	}
	d := rs.Domains["PII"]
	d.State = StateDormant
	rs.Domains["PII"] = d
	if got := len(rs.Active()); got != full-1 {
		t.Fatalf("dormant domain still active: %d", got)
	}
	if has(rs.Active(), "PII") {
		t.Fatal("Active() returned a dormant domain")
	}
	rs.Disabled = true
	if got := rs.Active(); got != nil {
		t.Fatalf("kill switch: Active() = %v, want nil", names(got))
	}
}

func TestLookup(t *testing.T) {
	rs := Defaults()
	if _, ok := rs.Lookup("NOSUCH"); ok {
		t.Fatal("Lookup returned ok for an absent domain")
	}
	rd, ok := rs.Lookup("PII")
	if !ok || rd.Name != "PII" || len(rd.Rules) == 0 {
		t.Fatalf("Lookup(PII) = %+v ok=%v", rd, ok)
	}
}

func pick(rs RuleSet, name string) ResolvedDomain {
	return ResolvedDomain{Name: name, Domain: rs.Domains[name]}
}

func writeRepoRules(t *testing.T, dir, body string) {
	t.Helper()
	abcd := filepath.Join(dir, ".abcd")
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(abcd, "rules.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// A domain whose merged rules are empty injects a heading-only "## NAME" block:
// suppression wearing the domain's name, and to the agent a domain that appears
// to say nothing (GHSA-22f8-qf5r-gjgq sibling). The documented way to silence a
// domain is {"state": "dormant"}; an override that empties the rules, or a
// custom domain declared without any, is refused loudly at load instead.
func TestValidateRefusesDomainWithoutRules(t *testing.T) {
	emptied := Merge(Defaults(), RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"PII": {Rules: []string{}},
	}})
	err := Validate(emptied)
	if err == nil {
		t.Fatal("an override that empties a bundled domain's rules passed validation (heading-only block)")
	}
	if !strings.Contains(err.Error(), "PII") || !strings.Contains(err.Error(), "dormant") {
		t.Fatalf("refusal must name the domain and the dormant remedy: %v", err)
	}
	ruleless := Merge(Defaults(), RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"CUSTOM": {Recall: []string{"widget"}},
	}})
	if err := Validate(ruleless); err == nil {
		t.Fatal("a custom domain declared without rules passed validation (heading-only block)")
	}
	// The bundled defaults and a dormant state-only override still validate.
	if err := Validate(Defaults()); err != nil {
		t.Fatalf("defaults must validate: %v", err)
	}
	silenced := Merge(Defaults(), RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"PII": {State: StateDormant},
	}})
	if err := Validate(silenced); err != nil {
		t.Fatalf("a dormant state-only override must validate: %v", err)
	}
}

// TestLoadSkipsARulelessDomainAndKeepsTheRest is the proportionality half of
// the ruleless-domain refusal. Validate must still refuse the shape — it is what
// guards the bundled defaults, where a heading-only domain is a build error —
// but a repo's rules.json is a file somebody already has, and `{"rules": []}`
// is a plausible way to have tried to silence a domain. Failing the whole load
// on it stops EVERY domain injecting, safety rules included, on the strength of
// one stderr line and a config that worked yesterday. So Load drops the
// offending domain, keeps the rest, and says which one it dropped and how to
// silence a domain deliberately.
func TestLoadSkipsARulelessDomainAndKeepsTheRest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"schema_version":1,"domains":{
		"COMMITTING":{"rules":[]},
		"CUSTOM":{"recall":["widget"]},
		"MINE":{"recall":["widget"],"rules":["do the thing"]}}}`
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "rules.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	rs, err := Load(dir)
	if err != nil {
		t.Fatalf("a ruleless domain must not fail the whole load: %v", err)
	}
	if _, ok := rs.Domains["COMMITTING"]; ok {
		t.Error("the emptied bundled domain must be dropped, not injected as a heading-only block")
	}
	if _, ok := rs.Domains["CUSTOM"]; ok {
		t.Error("a custom domain declared without rules must be dropped")
	}
	if d, ok := rs.Domains["MINE"]; !ok || len(d.Rules) != 1 {
		t.Error("a well-formed domain in the same file must survive")
	}
	if _, ok := rs.Domains["PII"]; !ok {
		t.Error("the untouched bundled domains must survive: one bad domain must not silence the ruleset")
	}
	notes := rs.Notes()
	if len(notes) != 2 {
		t.Fatalf("one note per dropped domain, got %d: %v", len(notes), notes)
	}
	joined := strings.Join(notes, "\n")
	for _, want := range []string{"COMMITTING", "CUSTOM", "dormant", RepoRelPath} {
		if !strings.Contains(joined, want) {
			t.Errorf("the diagnostic must name the domain, the file and the dormant remedy; missing %q in:\n%s", want, joined)
		}
	}
	// Deterministic order, so the diagnostic does not churn between runs.
	if notes[0] > notes[1] {
		t.Errorf("notes must be ordered by domain name: %v", notes)
	}
	// A clean file carries no notes at all.
	clean := t.TempDir()
	if err := os.MkdirAll(filepath.Join(clean, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clean, ".abcd", "rules.json"), []byte(`{"schema_version":1,"domains":{"PII":{"state":"dormant"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	quiet, err := Load(clean)
	if err != nil {
		t.Fatal(err)
	}
	if len(quiet.Notes()) != 0 {
		t.Errorf("a clean load must carry no notes, got %v", quiet.Notes())
	}
	if _, ok := quiet.Domains["PII"]; !ok {
		t.Error("a dormant state-only override keeps its domain: dormant is the documented way to silence one")
	}
}
