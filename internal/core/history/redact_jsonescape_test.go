package history

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestCaptureRedactsValuesBesideJSONEscapes is the store boundary of
// iss-2609261647358395 and iss-2609251639263391. A transcript is raw JSONL, so
// a token pasted at the start of an output line sits straight after a "\n"
// escape, and an encoder that escapes the solidus writes the caller's home
// with no '/' on the line. Both reached the stored record verbatim: the
// token's leading anchor read the escape letter as a word byte, and neither
// the home-path detector nor the literal $HOME backstop reads "\/". The
// record must carry neither, and must still carry the lines around them.
func TestCaptureRedactsValuesBesideJSONEscapes(t *testing.T) {
	repoRoot, home := setupStore(t)
	token := "ghp_" + "0123456789abcdefABCDEF0123456789abcd"
	escapedHome := strings.ReplaceAll(home, "/", `\/`)
	other := "zqstoreother"

	transcript := strings.Join([]string{
		`{"type":"user","text":"here it is:\n` + token + `\nthanks"}`,
		`{"type":"tool","text":"wrote ` + escapedHome + `\/notes\/plan.md"}`,
		`{"type":"tool","text":"ls\n/home/` + other + `/src\ndone"}`,
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, []byte(transcript), CaptureMeta{SessionID: "sess-jsonescape", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected Wrote=true: an idempotent no-op would leave every assertion below reading a record this call did not write")
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if bytes.Contains(onDisk, []byte(token)) {
		t.Errorf("a token after a \\n escape survived in the stored record")
	}
	if !bytes.Contains(onDisk, []byte("ghp"+strings.Repeat("*", 8))) {
		t.Errorf("the token must be MASKED, not merely absent — no fingerprint in the stored record")
	}
	// The login alone was already redacted by its bare word; what leaked is
	// the path around it, so the assertion is on the home's parent.
	escapedParent := escapedHome[:strings.LastIndex(escapedHome, `\/`)]
	if escapedParent == "" || bytes.Contains(onDisk, []byte(escapedParent)) {
		t.Errorf("the caller's home written with solidus escapes survived in the stored record")
	}
	if bytes.Contains(onDisk, []byte(other)) {
		t.Errorf("a third-party home between \\n escapes survived in the stored record")
	}
	for _, keep := range []string{"here it is:", "thanks", "plan.md", "done"} {
		if !bytes.Contains(onDisk, []byte(keep)) {
			t.Errorf("redaction took %q with it; the lines around a value must survive", keep)
		}
	}
}
