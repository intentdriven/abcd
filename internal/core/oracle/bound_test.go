package oracle

import (
	"fmt"
	"strings"
	"testing"
)

// TestOrphanAndRouteNamesAreBounded is review-tier1 F1 in this package: an
// agent name comes from a committed file or a flag, and a 500 KB name must not
// reach a diagnostic or a refusal whole; nor may ten thousand orphan rows
// become ten thousand stderr lines.
func TestOrphanAndRouteNamesAreBounded(t *testing.T) {
	huge := strings.Repeat("g", 500_000)

	f := newFx(t)
	f.repo(`{"` + huge + `":{"tier":"local"}}`)
	l := f.load()
	if len(l.Diagnostics) != 1 || len(l.Diagnostics[0]) > 1024 {
		t.Fatalf("orphan diagnostic: %d line(s), first %d bytes", len(l.Diagnostics), len(l.Diagnostics[0]))
	}

	var rows []string
	for i := 0; i < 10_000; i++ {
		rows = append(rows, fmt.Sprintf(`"ghost-%d":{"tier":"local"}`, i))
	}
	f = newFx(t)
	f.repo(`{` + strings.Join(rows, ",") + `}`)
	l = f.load()
	total := 0
	for _, d := range l.Diagnostics {
		total += len(d)
	}
	if len(l.Diagnostics) > 10 || total > 8192 {
		t.Fatalf("10000 orphan rows produced %d diagnostic line(s), %d bytes", len(l.Diagnostics), total)
	}
	if !strings.Contains(strings.Join(l.Diagnostics, "\n"), "more") {
		t.Fatalf("the diagnostics do not say how many orphans were not named: %q", l.Diagnostics)
	}

	_, err := ParseRoutes([]string{huge + "=local"}, []string{"scribe"}, NoConnections{})
	if err == nil || len(err.Error()) > 1024 {
		t.Fatalf("a --route naming a huge agent: err %d bytes", len(fmt.Sprint(err)))
	}
	_, err = ParseRoutes([]string{"scribe=local@" + huge}, []string{"scribe"}, NoConnections{})
	if err == nil || len(err.Error()) > 1024 {
		t.Fatalf("a --route naming a huge connection: err %d bytes", len(fmt.Sprint(err)))
	}
	if _, err = Resolve(huge, l, NoConnections{}); err == nil || len(err.Error()) > 1024 {
		t.Fatalf("Resolve of a huge name: err %d bytes", len(fmt.Sprint(err)))
	}
}

// TestSettingRefusalsBoundWhatTheyEcho: a malformed setting's name and value
// are payload, echoed bounded.
func TestSettingRefusalsBoundWhatTheyEcho(t *testing.T) {
	huge := strings.Repeat("S", 500_000)
	for _, agents := range []string{
		`{"scribe":{"tier":"local","settings":{"` + huge + `":1}}}`,
		`{"scribe":{"tier":"local","settings":{"seed":{"x":"` + huge + `"}}}}`,
	} {
		f := newFx(t)
		f.repo(agents)
		_, err := Load(f.roots)
		if err == nil || len(err.Error()) > 1024 {
			t.Fatalf("err %d bytes", len(fmt.Sprint(err)))
		}
	}
}
