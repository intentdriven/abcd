package layered

import (
	"strings"
	"testing"
)

// maxMessage bounds every refusal a hostile file can provoke: the file may be
// up to MaxFileBytes, and no key or value it carries may reach a message whole.
const maxMessage = 1024

// TestRefusalsBoundTheKeysTheyEcho is review-tier1 F1: compact() bounded a
// value in a refusal, but a KEY was echoed whole, so a 500 KB key in a
// committed file made every read print a 500 KB line. Each refusal that names
// a key names it bounded.
func TestRefusalsBoundTheKeysTheyEcho(t *testing.T) {
	huge := strings.Repeat("k", 500_000)
	cases := []struct {
		name string
		body string
		act  func(s *Stack) error
	}{
		{"unknown key under a claimed namespace", `{"pace":{"` + huge + `":1}}`,
			func(s *Stack) error { return s.Claim("pace", "work_minutes") }},
		{"many unknown keys", `{"pace":{` + manyKeys(50_000) + `}}`,
			func(s *Stack) error { return s.Claim("pace", "work_minutes") }},
		{"duplicate key", `{"` + huge + `":1,"` + huge + `":2}`, nil},
		{"path through a non-object", `{"pace":{"work_minutes":{"` + huge + `":1}}}`,
			func(s *Stack) error { _, err := Get[int](s, "pace.work_minutes", 1, nil); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.repoFile(Config, tc.body)
			s, err := Load(Config, f.roots)
			if err == nil && tc.act != nil {
				err = tc.act(s)
			}
			if err == nil {
				t.Fatal("want a refusal")
			}
			if n := len(err.Error()); n > maxMessage {
				t.Fatalf("the refusal is %d bytes; a key must be bounded where it is echoed (first 200: %q)", n, err.Error()[:200])
			}
		})
	}
}

// TestBoundKeyIsRuneSafe: truncation never splits a multi-byte rune, so a
// bounded name is still valid UTF-8.
func TestBoundKeyIsRuneSafe(t *testing.T) {
	got := BoundKey(strings.Repeat("é", 200))
	if !strings.HasSuffix(got, "...") || len(got) > 90 {
		t.Fatalf("BoundKey = %q (%d bytes)", got, len(got))
	}
	if strings.ContainsRune(got, '�') {
		t.Fatalf("BoundKey split a rune: %q", got)
	}
	if short := "scribe"; BoundKey(short) != short {
		t.Fatalf("a short key is changed: %q", BoundKey(short))
	}
}

func manyKeys(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"k` + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + itoa(i) + `":1`)
	}
	return b.String()
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var out []byte
	for ; i > 0; i /= 10 {
		out = append([]byte{digits[i%10]}, out...)
	}
	return string(out)
}
