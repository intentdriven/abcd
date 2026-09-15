package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

const statusPayload = `{"cwd":"%s","model":{"display_name":"Opus"},"context_window":{"used_percentage":8}}`

func payloadFor(root string) string {
	return strings.Replace(statusPayload, "%s", root, 1)
}

// TestStatuslineManagedRendersTheRowBadgeFirst is ac-1 and ac-11 at the verb:
// in a managed checkout the harness gets abcd's row, the badge is element one
// and carries colour, the payload elements follow, and --json is the Row.
func TestStatuslineManagedRendersTheRowBadgeFirst(t *testing.T) {
	root := managedCheckout(t)
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
	if !strings.HasSuffix(stdout, "\n") || strings.Count(stdout, "\n") != 1 {
		t.Errorf("the row is not one newline-terminated line: %q", stdout)
	}
	if !strings.HasPrefix(stdout, "\x1b[") {
		t.Errorf("the row does not open with the badge's colour run: %q", stdout)
	}
	if !strings.Contains(stdout, "waiting: facilitator") {
		t.Errorf("the badge word is missing: %q", stdout)
	}
	if i, j := strings.Index(stdout, "waiting: facilitator"), strings.Index(stdout, "Opus"); j < 0 || i > j {
		t.Errorf("the badge must precede the payload elements: %q", stdout)
	}
	if !strings.Contains(stdout, "ctx 8%") {
		t.Errorf("the context element is missing: %q", stdout)
	}

	so, _, err := runSplit(t, payloadFor(root), "statusline", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var row statusline.Row
	if err := json.Unmarshal([]byte(so), &row); err != nil {
		t.Fatalf("--json: %v\n%s", err, so)
	}
	if len(row.Elements) == 0 || row.Elements[0].Key != statusline.KeyPresence {
		t.Errorf("--json row does not lead with the badge: %+v", row.Elements)
	}
	for _, el := range row.Elements {
		if el.Plain == "" || el.Rendered == "" {
			t.Errorf("element %s has an empty form: %+v", el.Key, el)
		}
	}
}

// TestStatuslineUnmanagedRunsThePreviousCommand is ac-5: outside a managed
// checkout the recorded previous command runs with the SAME stdin, its stdout
// is the verb's stdout, and its exit code is the verb's exit code. abcd
// contributes nothing.
func TestStatuslineUnmanagedRunsThePreviousCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf prev:; cat; printf ' [err]' >&2; exit 3"}`)

	stdin := `{"model":{"display_name":"Opus"}}`
	stdout, stderr, err := runSplit(t, stdin, "statusline")
	if code := exitCodeOf(err); code != 3 {
		t.Errorf("exit = %d (err %v), want the previous command's 3", code, err)
	}
	if err != nil && err.Error() != "" {
		t.Errorf("the previous command's exit carried a message of abcd's own: %q", err.Error())
	}
	if stdout != "prev:"+stdin {
		t.Errorf("stdout = %q, want the previous command's output over the same stdin", stdout)
	}
	if stderr != " [err]" {
		t.Errorf("stderr = %q, want the previous command's stderr verbatim", stderr)
	}
}

// TestStatuslineDisabledBehavesAsUnmanaged is ac-6: the off switch makes a
// managed checkout behave exactly as an unmanaged one — the previous command
// runs, abcd's row does not.
func TestStatuslineDisabledBehavesAsUnmanaged(t *testing.T) {
	root := managedCheckout(t)
	writeUserSettings(t, `{"schema_version":1,"disabled":true,"previous_command":"printf theirs"}`)
	stdout, _, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "theirs" {
		t.Errorf("stdout = %q, want the previous command's output alone", stdout)
	}
}

// TestStatuslineWithNoPreviousCommandPrintsNothing: unmanaged and nothing
// recorded is an empty line and exit 0, never an error and never a prompt.
func TestStatuslineWithNoPreviousCommandPrintsNothing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	for _, stdin := range []string{"", `{"model":{"display_name":"Opus"}}`} {
		stdout, stderr, err := runSplit(t, stdin, "statusline")
		if err != nil || stdout != "" || stderr != "" {
			t.Errorf("stdin %q: err=%v stdout=%q stderr=%q, want a quiet exit 0", stdin, err, stdout, stderr)
		}
	}
	plain := t.TempDir()
	t.Chdir(plain)
	stdout, stderr, err := runSplit(t, "", "statusline")
	if err != nil || stdout != "" || stderr != "" {
		t.Errorf("outside a repository: err=%v stdout=%q stderr=%q, want a quiet exit 0", err, stdout, stderr)
	}
}

// TestStatuslineEmptyStdinStillRendersTheBadge: the smoke harness and a
// curious human both run the verb with nothing on stdin, and that is an empty
// payload — every payload element drops, the badge stays, the exit is 0 and
// stderr is silent.
func TestStatuslineEmptyStdinStillRendersTheBadge(t *testing.T) {
	root := managedCheckout(t)
	stdout, stderr, err := runSplit(t, "", "statusline")
	if err != nil {
		t.Fatalf("empty stdin: %v", err)
	}
	if stderr != "" {
		t.Errorf("empty stdin is not a diagnostic; stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "abcd") || !strings.Contains(stdout, filepath.Base(root)) {
		t.Errorf("stdout = %q, want the badge and the repository", stdout)
	}
	if strings.Contains(stdout, "ctx") {
		t.Errorf("a payload element rendered with no payload: %q", stdout)
	}
}

// TestStatuslineBadPayloadStillRendersTheBadge: a payload that does not parse
// is named on stderr and the row still renders without its payload elements —
// a status surface that goes blank on a bad payload hides the parked stop the
// badge exists to show.
func TestStatuslineBadPayloadStillRendersTheBadge(t *testing.T) {
	root := managedCheckout(t)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runSplit(t, "not json at all", "statusline")
	if err != nil {
		t.Fatalf("bad payload: %v", err)
	}
	if !strings.Contains(stdout, "waiting: product thinker") {
		t.Errorf("the badge went missing on a bad payload: %q", stdout)
	}
	if !strings.HasPrefix(stderr, "abcd statusline:") || !strings.Contains(stderr, "payload") {
		t.Errorf("stderr = %q, want the parse failure named with the verb's prefix", stderr)
	}
}

// TestStatuslineOverCapPayloadIsNamed: an over-cap payload is discarded whole
// (never truncated into a severed prefix), named on stderr with the cap, and
// the badge still renders.
func TestStatuslineOverCapPayloadIsNamed(t *testing.T) {
	managedCheckout(t)
	big := `{"model":{"display_name":"` + strings.Repeat("x", maxHookStdinBytes) + `"}}`
	stdout, stderr, err := runSplit(t, big, "statusline")
	if err != nil {
		t.Fatalf("over-cap payload: %v", err)
	}
	if !strings.Contains(stderr, "over the 1048576-byte cap") {
		t.Errorf("stderr = %q, want the cap named", stderr)
	}
	if !strings.Contains(stdout, "abcd") {
		t.Errorf("the badge went missing on an over-cap payload: %q", stdout)
	}
}

// TestStatuslineResolvesTheCheckoutFromThePayload: the harness runs the
// status command from wherever it was launched, so the checkout is the
// payload's cwd, not the process's.
func TestStatuslineResolvesTheCheckoutFromThePayload(t *testing.T) {
	root := managedCheckout(t)
	elsewhere := t.TempDir()
	t.Chdir(elsewhere)
	stdout, _, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, filepath.Base(root)) {
		t.Errorf("stdout = %q, want the payload's checkout rendered", stdout)
	}
}

// TestStatuslineSettingsNotesGoToStderr: a refused presence pair is reported
// out of band with the verb's prefix, and the row still renders.
func TestStatuslineSettingsNotesGoToStderr(t *testing.T) {
	managedCheckout(t)
	writeUserSettings(t, `{"schema_version":1,"presence":{"foreground":"#777777","background":"#888888"}}`)
	stdout, stderr, err := runSplit(t, "", "statusline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stderr, "abcd statusline: statusline: REFUSED") {
		t.Errorf("stderr = %q, want the refusal with the verb's prefix", stderr)
	}
	if !strings.Contains(stdout, "abcd") {
		t.Errorf("the row did not render: %q", stdout)
	}
}

// countingTransport records every HTTP round trip and refuses it. Swapped in
// for http.DefaultTransport so a verb that reached the network — through the
// default client or any client built on the default transport — is counted,
// not merely slow.
type countingTransport struct{ hits *int32 }

func (c countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	atomic.AddInt32(c.hits, 1)
	return nil, os.ErrPermission
}

type countingFetcher struct{ hits *int32 }

func (c countingFetcher) LatestTag() (string, error) {
	atomic.AddInt32(c.hits, 1)
	return "", os.ErrPermission
}

// TestModeAndStatuslineTouchNoNetwork is the zero-network harness the spec
// names: neither verb makes a request, in either state of the repository,
// through the default transport or the release-fetcher seam.
func TestModeAndStatuslineTouchNoNetwork(t *testing.T) {
	var hits int32
	origTransport := http.DefaultTransport
	http.DefaultTransport = countingTransport{&hits}
	origFetcher := newReleaseFetcher
	newReleaseFetcher = func() vintage.ReleaseFetcher { return countingFetcher{&hits} }
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
		newReleaseFetcher = origFetcher
	})

	root := managedCheckout(t)
	runCLI(t, "mode")
	runCLI(t, "mode", "product-thinker")
	runCLI(t, "mode", "--json")
	runSplit(t, payloadFor(root), "statusline")
	runSplit(t, "", "statusline", "--json")
	runCLI(t)

	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeUserSettings(t, `{"schema_version":1,"previous_command":"true"}`)
	runSplit(t, payloadFor(repo), "statusline")
	runCLI(t, "mode")

	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("the mode and statusline verbs made %d network request(s)", n)
	}
}

// TestStatuslineFallbackMarkerRefusesToRecurse is the run-time half of the
// recursion guard: a previous status command that reaches abcd's own status
// verb would run this verb again, which would run the previous command again,
// without end. The child carries a marker in its environment, and a run that
// finds the marker already set is that recursion — it prints nothing, says so
// once on stderr, and exits 0, whatever the previous command would have done.
func TestStatuslineFallbackMarkerRefusesToRecurse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf recursed"}`)
	t.Setenv("ABCD_STATUSLINE_FALLBACK", "1")

	stdout, stderr, err := runSplit(t, "", "statusline")
	if err != nil {
		t.Fatalf("a refused recursion is exit 0, got %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: the previous command must not have run", stdout)
	}
	if !strings.HasPrefix(stderr, "abcd statusline:") || !strings.Contains(stderr, "recurse") {
		t.Errorf("stderr = %q, want one note saying abcd refused to recurse", stderr)
	}
	if strings.Count(stderr, "\n") != 1 {
		t.Errorf("stderr = %q, want exactly one line", stderr)
	}
}

// TestStatuslineFallbackCarriesTheMarkerToTheChild proves the other half: the
// previous command runs with the marker set, so a child that is abcd's own
// status verb finds it and stops.
func TestStatuslineFallbackCarriesTheMarkerToTheChild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	t.Setenv("ABCD_STATUSLINE_FALLBACK", "")
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf '%s' \"$ABCD_STATUSLINE_FALLBACK\""}`)

	stdout, stderr, err := runSplit(t, "", "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if stdout != "1" {
		t.Errorf("stdout = %q, want %q: the child did not carry ABCD_STATUSLINE_FALLBACK", stdout, "1")
	}
}
