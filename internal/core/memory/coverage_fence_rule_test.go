package memory

import "testing"

// The quoted-span extractor skips fenced lines by mdrecord's rule. A run
// indented four columns inside a fence is content, not its closer (CommonMark),
// so a blockquote line below it is still inside the fence and is no span.
func TestExtractQuotedSpansReadsFencesByTheCommonMarkRule(t *testing.T) {
	page := "# Page\n\n```\n    ```\n> a quoted example line\n```\n\n> a live quoted line\n"
	spans := extractQuotedSpans(page)
	if len(spans) != 1 || spans[0].normalized != "a live quoted line" {
		t.Fatalf("spans = %+v, want only the live quoted line", spans)
	}
}
