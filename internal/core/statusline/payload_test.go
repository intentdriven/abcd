package statusline

import (
	"strings"
	"testing"
)

// fullPayloadJSON is the reference fixture: one harness status payload
// carrying every field the four payload-sourced elements read, in the nesting
// the first harness in the roster supplies (itd-200's scope condition
// cond-2609012158058166).
const fullPayloadJSON = `{
  "hook_event_name": "Status",
  "session_id": "abc123",
  "cwd": "/repo",
  "model": {"id": "claude-opus-4-1", "display_name": "Opus"},
  "workspace": {"current_dir": "/repo", "project_dir": "/repo"},
  "version": "1.0.80",
  "context_window": {
    "context_window_size": 200000,
    "used_percentage": 8,
    "remaining_percentage": 92
  },
  "exceeds_200k_tokens": false,
  "rate_limits": {
    "five_hour": {"used_percentage": 23.5, "resets_at": 1738425600},
    "seven_day": {"used_percentage": 41.2, "resets_at": 1738857600}
  }
}`

func TestParsePayloadReadsEveryField(t *testing.T) {
	p, err := ParsePayload([]byte(fullPayloadJSON))
	if err != nil {
		t.Fatalf("ParsePayload: %v", err)
	}
	if p.Model != "Opus" {
		t.Fatalf("Model = %q, want %q", p.Model, "Opus")
	}
	for _, f := range []struct {
		name string
		got  *float64
		want float64
	}{
		{"ContextPct", p.ContextPct, 8},
		{"FiveHourPct", p.FiveHourPct, 23.5},
		{"SevenDayPct", p.SevenDayPct, 41.2},
	} {
		if f.got == nil {
			t.Fatalf("%s is absent, want %v", f.name, f.want)
		}
		if *f.got != f.want {
			t.Fatalf("%s = %v, want %v", f.name, *f.got, f.want)
		}
	}
}

// TestParsePayloadTreatsEveryFieldAsOptional walks the payload one removal at
// a time. Every one of these shapes is real: rate_limits appears only for some
// plans and only after the first API response, each window is independently
// dropped once it resets, and used_percentage is null early in a session and
// again after a compaction.
func TestParsePayloadTreatsEveryFieldAsOptional(t *testing.T) {
	cases := []struct {
		name  string
		json  string
		check func(t *testing.T, p Payload)
	}{
		{
			name: "empty object",
			json: `{}`,
			check: func(t *testing.T, p Payload) {
				if p.Model != "" || p.ContextPct != nil || p.FiveHourPct != nil || p.SevenDayPct != nil {
					t.Fatalf("an empty payload yielded %+v", p)
				}
			},
		},
		{
			name: "model object absent",
			json: `{"context_window":{"used_percentage":8}}`,
			check: func(t *testing.T, p Payload) {
				if p.Model != "" {
					t.Fatalf("Model = %q, want empty", p.Model)
				}
			},
		},
		{
			name: "model falls back to the id when the display name is absent",
			json: `{"model":{"id":"claude-opus-4-1"}}`,
			check: func(t *testing.T, p Payload) {
				if p.Model != "claude-opus-4-1" {
					t.Fatalf("Model = %q, want the id", p.Model)
				}
			},
		},
		{
			name: "context window present but used_percentage null",
			json: `{"context_window":{"context_window_size":200000,"used_percentage":null}}`,
			check: func(t *testing.T, p Payload) {
				if p.ContextPct != nil {
					t.Fatalf("ContextPct = %v, want absent", *p.ContextPct)
				}
			},
		},
		{
			name: "rate limits absent entirely",
			json: `{"model":{"display_name":"Opus"}}`,
			check: func(t *testing.T, p Payload) {
				if p.FiveHourPct != nil || p.SevenDayPct != nil {
					t.Fatalf("usage percentages present without rate_limits: %+v", p)
				}
			},
		},
		{
			name: "one rate-limit window dropped",
			json: `{"rate_limits":{"seven_day":{"used_percentage":41.2}}}`,
			check: func(t *testing.T, p Payload) {
				if p.FiveHourPct != nil {
					t.Fatalf("FiveHourPct = %v, want absent", *p.FiveHourPct)
				}
				if p.SevenDayPct == nil || *p.SevenDayPct != 41.2 {
					t.Fatalf("SevenDayPct = %v, want 41.2", p.SevenDayPct)
				}
			},
		},
		{
			name: "zero is a value, not an absence",
			json: `{"context_window":{"used_percentage":0}}`,
			check: func(t *testing.T, p Payload) {
				if p.ContextPct == nil {
					t.Fatal("ContextPct is absent, but the payload carried 0")
				}
				if *p.ContextPct != 0 {
					t.Fatalf("ContextPct = %v, want 0", *p.ContextPct)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParsePayload([]byte(tc.json))
			if err != nil {
				t.Fatalf("ParsePayload: %v", err)
			}
			tc.check(t, p)
		})
	}
}

// TestParsePayloadRefusesMalformedJSON: the harness hands the payload on
// stdin, so a truncated or empty read is a real state and must be an error the
// front door can fall back on, not a zero-value row.
func TestParsePayloadRefusesMalformedJSON(t *testing.T) {
	for _, in := range []string{``, `{`, `not json`, `[]`, `null`} {
		if _, err := ParsePayload([]byte(in)); err == nil {
			t.Fatalf("ParsePayload(%q) returned no error", in)
		}
	}
}

// TestParsePayloadSanitizesTheModelName: the model name is payload text that
// reaches a terminal, so it goes through the repo's one canonical sanitiser
// rather than being trusted (termsafe, iss-81's sibling discipline).
func TestParsePayloadSanitizesTheModelName(t *testing.T) {
	// A raw ESC, JSON-escaped as \u001b, is exactly what encoding/json passes
	// through untouched — the case termsafe exists for.
	p, err := ParsePayload([]byte(`{"model":{"display_name":"Op\u001b[31mus"}}`))
	if err != nil {
		t.Fatalf("ParsePayload: %v", err)
	}
	if strings.ContainsRune(p.Model, 0x1b) {
		t.Fatalf("Model = %q — the escape reached the row unsanitised", p.Model)
	}
	if want := "Op?[31mus"; p.Model != want {
		t.Fatalf("Model = %q, want %q", p.Model, want)
	}
}

// TestParsePayloadLiftsTheWorkingDirectory: the status verb resolves the
// checkout the harness is standing in from the payload, not from its own
// process directory, because the harness runs the status command from wherever
// it was launched. `cwd` wins; `workspace.current_dir` is the fallback; both
// absent is an empty string the front door replaces with its own Getwd. The
// value is for filesystem resolution only and is never printed, which is why
// it is not sanitised and why its JSON tag hides it from every envelope.
func TestParsePayloadLiftsTheWorkingDirectory(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"cwd wins", `{"cwd":"/a","workspace":{"current_dir":"/b"}}`, "/a"},
		{"workspace fallback", `{"cwd":"","workspace":{"current_dir":"/b"}}`, "/b"},
		{"workspace only", `{"workspace":{"current_dir":"/b"}}`, "/b"},
		{"neither", `{"model":{"id":"m"}}`, ""},
		{"null cwd", `{"cwd":null,"workspace":null}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParsePayload([]byte(tc.in))
			if err != nil {
				t.Fatalf("ParsePayload: %v", err)
			}
			if p.Cwd != tc.want {
				t.Fatalf("Cwd = %q, want %q", p.Cwd, tc.want)
			}
		})
	}
	full, err := ParsePayload([]byte(fullPayloadJSON))
	if err != nil {
		t.Fatal(err)
	}
	if full.Cwd != "/repo" {
		t.Fatalf("fixture Cwd = %q, want /repo", full.Cwd)
	}
}
