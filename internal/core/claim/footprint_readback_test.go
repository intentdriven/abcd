package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestAFootprintPathCarriesNoLeadingOrTrailingSpace: a footprint path must
// read back exactly as it was written, so the writer refuses one that begins
// or ends with a space, and nothing reaches origin.
func TestAFootprintPathCarriesNoLeadingOrTrailingSpace(t *testing.T) {
	for name, path := range map[string]string{
		"a trailing space": "docs ",
		"a leading space":  " docs",
		"spaces both ends": " docs/a.md ",
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			if _, err := p.reserve("iss-5", path); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "is not a repository-relative path") {
				t.Fatalf("the writer takes the footprint %q: %v", path, err)
			}
			if !ValidFootprintPath(strings.TrimSpace(path)) || ValidFootprintPath(path) {
				t.Fatalf("ValidFootprintPath(%q) should refuse it and take it trimmed", path)
			}
			q := w.clone("99+quinn@users.noreply.github.com", "")
			if _, err := Fetch(q.dir, q.opts()); err != nil {
				t.Fatal(err)
			}
			snap, err := View(q.dir, t0)
			if err != nil {
				t.Fatal(err)
			}
			if len(snap.Reservations) != 0 {
				t.Fatalf("a refused reservation reached origin: %+v", snap.Reservations)
			}
		})
	}
}

// TestAComposedMessageWhoseFootprintReadsBackOtherwiseIsNeverWritten: the
// writer reads the message it composed back through the reader and writes
// nothing when the footprint read back is not the one composed.
func TestAComposedMessageWhoseFootprintReadsBackOtherwiseIsNeverWritten(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 0, 0, 0, time.UTC)
	m := message{Record: "iss-5", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55",
		Footprint: []string{"docs/a.md", "internal"}, ClaimedAt: at, ExpiresAt: at.Add(2 * time.Hour)}
	if why := footprintReadBack(m, m.text()); why != "" {
		t.Fatalf("a footprint that reads back is refused: %s", why)
	}
	for _, fp := range [][]string{{"internal "}, {"docs/a.md", " internal"}} {
		m.Footprint = fp
		if why := footprintReadBack(m, m.text()); why == "" {
			t.Fatalf("the footprint %q reads back otherwise, and is not refused", fp)
		}
	}
	m.Footprint = []string{"docs/a.md"}
	if why := footprintReadBack(m, m.text()+"Abcd-Footprint: internal\n"); why == "" {
		t.Fatal("a message carrying a footprint path it was not composed with is not refused")
	}
}
