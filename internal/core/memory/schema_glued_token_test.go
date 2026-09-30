package memory

import (
	"strings"
	"testing"
)

// TestPageSchemaKeyRefusalSealsAGluedToken — iss-2609290541525428. An
// undeclared key is named through scanner.RedactRefusal, whose patterns anchor
// on a leading \b, so a token glued behind an underscore or a letter in the
// key came back raw. The key is still named, with the token sealed.
func TestPageSchemaKeyRefusalSealsAGluedToken(t *testing.T) {
	pat := "gh" + "p_" + strings.Repeat("D", 36)
	akia := "AK" + "IA" + strings.Repeat("Q", 16)
	for _, tc := range []struct{ name, key, token, body, keep string }{
		{"pat behind an underscore", "notes_" + pat, pat, strings.Repeat("D", 6), "notes_"},
		{"access key behind an underscore", "notes_" + akia, akia, strings.Repeat("Q", 6), "notes_"},
		{"pat between two letters", "x" + pat + "y", pat, strings.Repeat("D", 6), "unknown key(s) [x"},
		{"access key between two letters", "x" + akia + "y", akia, strings.Repeat("Q", 6), "unknown key(s) [x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]any{
				"type": "topic", "domain": "auth", "slug": "x", "body": "# Subject line",
				"source": map[string]any{"class": "session_memory"},
				tc.key:   1,
			}
			_, err := validateDistilledPage(t.TempDir(), data)
			if err == nil {
				t.Fatal("a page carrying an undeclared key was accepted")
			}
			if strings.Contains(err.Error(), tc.token) || strings.Contains(err.Error(), tc.body) {
				t.Errorf("the refusal echoes the glued token: %v", err)
			}
			if !strings.Contains(err.Error(), tc.keep) {
				t.Errorf("the refusal no longer names the key (want %q): %v", tc.keep, err)
			}
		})
	}
}
