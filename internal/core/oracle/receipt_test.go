package oracle

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRequestBlockCarriesTierBoundAndSource is spec step 4's request half and
// AC 1: with nothing accepted the request block names host-decides, the
// contract ceiling and the none layer, on the harness, with no fallback.
func TestRequestBlockCarriesTierBoundAndSource(t *testing.T) {
	l := newFx(t).load()
	r, err := Resolve("intent-auditor", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	got := r.Request()
	want := RequestRouting{Agent: "intent-auditor", Tier: HostDecides, FanOut: 1, Source: "none", Origin: "none", Connection: Harness}
	if got != want {
		t.Fatalf("Request() = %+v, want %+v", got, want)
	}
	sec := RenderRequestSection(got)
	for _, line := range []string{"routing:", "  agent: intent-auditor", "  tier: host-decides", "  fan_out: 1", "  source: none", "  connection: harness"} {
		if !strings.Contains(sec, line+"\n") {
			t.Fatalf("request section lacks %q:\n%s", line, sec)
		}
	}
	if strings.Contains(sec, "fallback") || strings.Contains(sec, "override") {
		t.Fatalf("a request with no fallback and no override names one:\n%s", sec)
	}
}

// TestReceiptCarriesTheThreeConnectionFieldsAndTheModel is AC 4's and AC 5's
// receipt half: an accepted row no provider serves falls back, and the receipt
// carries the tier asked, the connection tried and used, the reason, and the
// model the payload reported, verbatim; an override is carried verbatim (AC 7).
func TestReceiptCarriesTheThreeConnectionFieldsAndTheModel(t *testing.T) {
	f := newFx(t)
	f.repo(`{"scribe":{"tier":"economy"}}`)
	l := f.load()
	routes, err := ParseRoutes([]string{"scribe=frontier?seed=7"}, []string{"scribe"}, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(routes); err != nil {
		t.Fatal(err)
	}
	r, err := Resolve("scribe", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	rc := r.Receipt("claude-example-1")
	if rc.TierAsked != Frontier || rc.ConnectionTried != "" || rc.ConnectionUsed != Harness ||
		rc.FallbackReason == "" || rc.Override != "scribe=frontier?seed=7" || rc.ModelReported != "claude-example-1" {
		t.Fatalf("receipt = %+v", rc)
	}
	enc, err := json.Marshal(rc)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{`"tier_asked":"frontier"`, `"connection_tried":""`, `"connection_used":"harness"`,
		`"fallback_reason":"no configured`, `"override":"scribe=frontier?seed=7"`, `"settings_sent":{}`, `"model_reported":"claude-example-1"`} {
		if !strings.Contains(string(enc), k) {
			t.Fatalf("receipt JSON lacks %s: %s", k, enc)
		}
	}
}

// TestReceiptRecordsTheMergedSettingsSent is AC 8's receipt half: on a provider
// leg the receipt records the connection's defaults with the row's and then the
// flag's laid over them.
func TestReceiptRecordsTheMergedSettingsSent(t *testing.T) {
	c := Connection{Name: "lab", Defaults: Settings{"temperature": raw("0.2"), "seed": raw("1"), "top_p": raw("0.9")}}
	conns := &spy{serves: map[Tier]Connection{Local: c}, named: map[string]Connection{"lab": c}}
	f := newFx(t)
	f.repo(`{"scribe":{"tier":"local","settings":{"seed":2}}}`)
	l := f.load()
	routes, err := ParseRoutes([]string{"scribe=local?top_p=0.5"}, []string{"scribe"}, conns)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(routes); err != nil {
		t.Fatal(err)
	}
	r, err := Resolve("scribe", l, conns)
	if err != nil {
		t.Fatal(err)
	}
	rc := r.Receipt("")
	enc, _ := json.Marshal(rc.SettingsSent)
	if string(enc) != `{"seed":2,"temperature":0.2,"top_p":0.5}` || rc.ConnectionTried != "lab" || rc.ConnectionUsed != "lab" || rc.FallbackReason != "" {
		t.Fatalf("receipt = %+v settings %s", rc, enc)
	}
}

// TestModelReportedIsThePayloadsOwnFieldBounded: model_reported is whatever
// the payload's model field carries, the top-level one or a reading's
// instrument.model; absent is "", and an untrusted value is bounded and its
// hidden runes encoded so the receipt cannot smuggle them.
func TestModelReportedIsThePayloadsOwnFieldBounded(t *testing.T) {
	cases := []struct {
		name, payload, want string
	}{
		{"top level", `{"model":"example-model-2"}`, "example-model-2"},
		{"reading instrument", `{"instrument":{"model":"example-reader"}}`, "example-reader"},
		{"absent", `{"verdict":"SHIP"}`, ""},
		{"not an object", `[1,2]`, ""},
		{"not a string", `{"model":42}`, ""},
		{"hidden runes encoded", `{"model":"a\u202eb"}`, "a%E2%80%AEb"},
	}
	for _, tc := range cases {
		if got := ModelReported([]byte(tc.payload)); got != tc.want {
			t.Errorf("%s: ModelReported = %q, want %q", tc.name, got, tc.want)
		}
	}
	long := ModelReported([]byte(`{"model":"` + strings.Repeat("m", 10_000) + `"}`))
	if len(long) > MaxModelBytes+3 || !strings.HasSuffix(long, "...") {
		t.Fatalf("a 10 KB model is reported as %d bytes", len(long))
	}
}

// TestSettingsAreBoundedAndCleanWhereTheyEnter is review-tier1 F6: settings
// are attacker-set from a committed file and reach the request block and the
// receipt, so they are bounded and refused unclean where they are read — a
// value over MaxSettingBytes, more than MaxSettings of them, a string carrying
// a control, bidi or zero-width rune — and a --route text is bounded whole.
func TestSettingsAreBoundedAndCleanWhereTheyEnter(t *testing.T) {
	big := strings.Repeat("x", MaxSettingBytes+1)
	var many []string
	for i := 0; i <= MaxSettings; i++ {
		many = append(many, `"k`+strings.Repeat("a", i%3)+string(rune('a'+i%26))+`":1`)
	}
	for name, agents := range map[string]string{
		"long value":    `{"scribe":{"tier":"local","settings":{"stop":"` + big + `"}}}`,
		"too many":      `{"scribe":{"tier":"local","settings":{` + strings.Join(dedupe(many), ",") + `}}}`,
		"bidi in value": `{"scribe":{"tier":"local","settings":{"stop":"a\u202eb"}}}`,
		"escape":        `{"scribe":{"tier":"local","settings":{"stop":"a\u001b[31mb"}}}`,
	} {
		f := newFx(t)
		f.repo(agents)
		if _, err := Load(f.roots); err == nil {
			t.Errorf("%s: a routing file carrying it was admitted", name)
		}
	}
	for name, text := range map[string]string{
		"long flag value": "scribe=local?stop=" + big,
		"bidi flag value": "scribe=local?stop=a\u202eb",
		"escape flag":     "scribe=local?stop=a\x1bb",
		"long route text": "scribe=local?a=" + strings.Repeat("v", 250) + ",b=" + strings.Repeat("v", 250) +
			",c=" + strings.Repeat("v", 250) + ",d=" + strings.Repeat("v", 250) + ",e=" + strings.Repeat("v", 250),
	} {
		if _, err := ParseRoutes([]string{text}, []string{"scribe"}, NoConnections{}); err == nil {
			t.Errorf("%s: --route admitted", name)
		}
	}
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
