package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/statusline"
)

// TestBoardPresenceLineIsTheRendersPlainForm is the board half of ac-2 and
// ac-7: in a managed checkout the bare board carries one `presence:` line
// equal to the render's plain form, and its JSON envelope carries the same
// state and elements — from the same composition, never words of its own.
func TestBoardPresenceLineIsTheRendersPlainForm(t *testing.T) {
	root := managedCheckout(t)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	want, err := statusline.Compose(root, statusline.Payload{}, statusline.Defaults())
	if err != nil {
		t.Fatal(err)
	}

	text := string(runCLI(t))
	line := ""
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "presence:") {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("the board has no presence line:\n%s", text)
	}
	if got := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "presence:")); got != want.Row.Plain() {
		t.Errorf("presence line = %q, want the render's plain form %q", got, want.Row.Plain())
	}
	if !strings.HasPrefix(want.Row.Plain(), "waiting: product thinker") {
		t.Errorf("precondition: the render's plain form should lead with the parked badge, got %q", want.Row.Plain())
	}

	var got struct {
		Dir        string `json:"dir"`
		Statusline *struct {
			State    string               `json:"state"`
			Plain    string               `json:"plain"`
			Elements []statusline.Element `json:"elements"`
		} `json:"statusline"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if got.Dir == "" {
		t.Error("the board's own fields went missing from the envelope")
	}
	if got.Statusline == nil {
		t.Fatal("--json has no statusline object in a managed checkout")
	}
	if got.Statusline.State != "product-thinker" || got.Statusline.Plain != want.Row.Plain() {
		t.Errorf("statusline = %+v, want state product-thinker and plain %q", got.Statusline, want.Row.Plain())
	}
	if len(got.Statusline.Elements) != len(want.Row.Elements) || got.Statusline.Elements[0].Key != statusline.KeyPresence {
		t.Errorf("elements = %+v, want the render's %+v", got.Statusline.Elements, want.Row.Elements)
	}
}

// TestBoardOmitsPresenceWhereUnmanaged: a checkout abcd does not manage gets no
// presence line and no statusline field — omitted, not null — and the board
// never runs the previous command.
func TestBoardOmitsPresenceWhereUnmanaged(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeUserSettings(t, `{"schema_version":1,"previous_command":"echo THE-PREVIOUS-COMMAND-RAN"}`)

	text := string(runCLI(t))
	if strings.Contains(text, "presence:") {
		t.Errorf("an unmanaged checkout rendered a presence line:\n%s", text)
	}
	if strings.Contains(text, "THE-PREVIOUS-COMMAND-RAN") {
		t.Errorf("the board ran the previous command:\n%s", text)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if raw, has := got["statusline"]; has {
		t.Errorf("--json carries statusline=%s in an unmanaged checkout; the field is omitted there", raw)
	}
}
