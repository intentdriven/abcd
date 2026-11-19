package loop

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestBuildIntentPutsHandBackInTheReceipt holds the build intent to where the
// loop reads a lane's hand-back: the receipt's `handback` field, which is
// where the drain chapter puts it too, never the implementer's report
// (iss-2610262037311796). Every line of the intent naming `handback:` must
// place it in the lane receipt and none in the lane report.
func TestBuildIntentPutsHandBackInTheReceipt(t *testing.T) {
	field, ok := reflect.TypeOf(LaneReceipt{}).FieldByName("HandBack")
	if !ok {
		t.Fatal("LaneReceipt carries no HandBack field")
	}
	tag := strings.Split(field.Tag.Get("json"), ",")[0]
	path := filepath.Join("..", "..", "..", "..", ".abcd", "development", "intents", "planned",
		"itd-2609201916151817-one-verb-takes-a-single-intent-from.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the build intent: %v", err)
	}
	named := 0
	for i, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, "`"+tag+":`") {
			continue
		}
		named++
		if strings.Contains(line, "lane report") {
			t.Errorf("%s:%d puts `%s:` in the lane report; the loop reads it from the lane receipt", path, i+1, tag)
		}
		if !strings.Contains(line, "lane receipt") {
			t.Errorf("%s:%d names `%s:` without placing it in the lane receipt", path, i+1, tag)
		}
	}
	if named == 0 {
		t.Fatalf("%s names no `%s:` field; decision 10 should", path, tag)
	}
}
