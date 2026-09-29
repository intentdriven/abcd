package ideate

import (
	"strings"
	"testing"
)

// TestDecoderRefusalRedactsTheKey — iss-2609290218032954. The verdict payload
// decodes strictly, and encoding/json's refusal names an undeclared field by
// the payload's own key. That message was returned raw, so a home path in a key
// reached the terminal and the transcript. The key is still named, redacted
// through the canonical scanner.
func TestDecoderRefusalRedactsTheKey(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["reviewer_notes /Users/zzotherperson/notes"] = "smuggled" // abcd-lint:allow — a planted home path in a KEY
	_, err := Record(root, "the-ideate-gate", encode(t, p), at)
	if err == nil {
		t.Fatal("a payload carrying an undeclared key was accepted")
	}
	if strings.Contains(err.Error(), "zzotherperson") {
		t.Errorf("the refusal echoes the payload's key: %v", err)
	}
	if !strings.Contains(err.Error(), "reviewer_notes") {
		t.Errorf("the refusal no longer names the undeclared field: %v", err)
	}
}
