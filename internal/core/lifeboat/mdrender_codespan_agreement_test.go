package lifeboat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mdrender"
)

// TestBlockEscaperAgreesWithTheRendererOnCodeSpans runs the shared code-span
// agreement table (termsafe's testdata, read by termsafe.BlockText and the site
// renderer's own test too) through escapeLeadingMarker and then the site
// renderer. A value whose leading run is balanced is left as written and
// renders as that span; an unbalanced one is escaped and renders no leading
// span. An escaper and a renderer that pair runs differently disagree here
// (iss-2609262322244502).
func TestBlockEscaperAgreesWithTheRendererOnCodeSpans(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "termsafe", "testdata", "codespan_agreement.json"))
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Cases []struct {
			In       string `json:"in"`
			Balanced bool   `json:"balanced"`
			Code     string `json:"code"`
		}
	}
	if err := json.Unmarshal(data, &table); err != nil || len(table.Cases) == 0 {
		t.Fatalf("the code-span agreement table did not load: %v", err)
	}
	for _, c := range table.Cases {
		got := escapeLeadingMarker(c.In)
		if c.Balanced != (got == c.In) || !c.Balanced && !strings.HasPrefix(got, `\`) {
			t.Errorf("escapeLeadingMarker(%q) = %q; the table says balanced = %v", c.In, got, c.Balanced)
			continue
		}
		html, err := siteRender(t, got)
		opens := err == nil && strings.HasPrefix(html, "<p><code>")
		if opens != c.Balanced {
			t.Errorf("escapeLeadingMarker(%q) = %q: the renderer opens a leading span = %v (err %v); the table says balanced = %v", c.In, got, opens, err, c.Balanced)
			continue
		}
		if c.Balanced && !strings.HasPrefix(html, "<p><code>"+mdrender.EscapeText(c.Code)+"</code>") {
			t.Errorf("escapeLeadingMarker(%q) = %q rendered %q, want a leading span holding %q", c.In, got, html, c.Code)
		}
	}
}
