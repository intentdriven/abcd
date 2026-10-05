package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/surface/dashboard"
)

// dashboardHomeFixture is a temp HOME, standing in an unmanaged checkout.
func dashboardHomeFixture(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	repo = filepath.Join(home, "repo")
	gitInitAt(t, repo)
	t.Chdir(repo)
	return home, repo
}

func exitCode(err error) int {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return -1
}

func TestStatusListsDevicesSeen(t *testing.T) {
	home, _ := dashboardHomeFixture(t)

	// Nothing runs: status says so, and so does the bare verb.
	for _, args := range [][]string{{"dashboard"}, {"dashboard", "status"}} {
		out := string(runCLI(t, args...))
		if !strings.Contains(out, "not running") {
			t.Errorf("%v with nothing running:\n%s", args, out)
		}
	}

	// This test process stands in for the server: the run file names it as
	// the kernel sees it, so status reads it as running.
	if err := dashboard.RecordRunForTest(home, os.Getpid(), "dash.example-tailnet.ts.net", 8080,
		[]string{"100.101.102.103:8080", "[fd7a:115c:a1e0::1]:8080"}); err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	seen := `{"devices":[` +
		`{"device":"tablet","person":"Product Thinker","login":"pt@example.com","first_opened":"` + first.Add(time.Hour).Format(time.RFC3339) + `","last_opened":"` + first.Add(time.Hour).Format(time.RFC3339) + `"},` +
		`{"device":"phone","person":"Product Thinker","login":"pt@example.com","first_opened":"` + first.Format(time.RFC3339) + `","last_opened":"` + first.Format(time.RFC3339) + `"},` +
		`{"device":"laptop\u001b[31m","person":"Guest\u202e","login":"guest@example.com","first_opened":"` + first.Add(2*time.Hour).Format(time.RFC3339) + `","last_opened":"` + first.Add(2*time.Hour).Format(time.RFC3339) + `"}` +
		`]}`
	if err := os.WriteFile(abcdhome.Path(home, "dashboard", "seen.json"), []byte(seen), 0o600); err != nil {
		t.Fatal(err)
	}

	out := string(runCLI(t, "dashboard", "status"))
	for _, want := range []string{"running at http://dash.example-tailnet.ts.net:8080", "100.101.102.103:8080", "[fd7a:115c:a1e0::1]:8080", "3 devices", "phone — Product Thinker (pt@example.com)", "tablet", "guest@example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("status does not say %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "phone") > strings.Index(out, "tablet") {
		t.Errorf("devices are not listed in the order they first opened it:\n%s", out)
	}
	if strings.ContainsAny(out, "\x1b\u202e") {
		t.Errorf("a device or person name reached the terminal unsanitised:\n%q", out)
	}

	var st dashboard.Status
	if err := json.Unmarshal(runCLI(t, "dashboard", "status", "--json"), &st); err != nil {
		t.Fatal(err)
	}
	if !st.Running || len(st.Devices) != 3 || st.Devices[0].Device != "phone" || st.URL != "http://dash.example-tailnet.ts.net:8080" {
		t.Errorf("--json status = %+v", st)
	}
}

func TestDashboardStartRefusals(t *testing.T) {
	_, _ = dashboardHomeFixture(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"dashboard", "start", "--port", "0"}, "is not a port"},
		{[]string{"dashboard", "start", "--port", "70000"}, "is not a port"},
		{[]string{"dashboard", "start"}, "not one abcd manages"},
	} {
		_, err := runCLIErr(t, c.args...)
		if exitCode(err) != 2 || err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v = %v (exit %d), want exit 2 saying %q", c.args, err, exitCode(err), c.want)
		}
	}
	// Outside any checkout, start refuses too.
	t.Chdir(t.TempDir())
	if _, err := runCLIErr(t, "dashboard", "start"); exitCode(err) != 2 {
		t.Errorf("start outside a checkout = %v, want exit 2", err)
	}
}

func TestDashboardServeIsHiddenAndRefusesWithoutStart(t *testing.T) {
	root := NewRootCommand()
	serve, _, err := root.Find([]string{"dashboard", "serve"})
	if err != nil || serve.Name() != "serve" {
		t.Fatalf("no dashboard serve: %v", err)
	}
	if !serve.Hidden {
		t.Error("dashboard serve is listed: only start runs it")
	}
	_, err = runCLIErr(t, "dashboard", "serve")
	if exitCode(err) != 2 || !strings.Contains(err.Error(), "started only by `abcd dashboard start`") {
		t.Errorf("serve run directly = %v, want the exit-2 refusal", err)
	}
}

func TestDashboardStopWithNothingRunning(t *testing.T) {
	_, _ = dashboardHomeFixture(t)
	if out := string(runCLI(t, "dashboard", "stop")); !strings.Contains(out, "not running; nothing to stop") {
		t.Errorf("stop with nothing running:\n%s", out)
	}
}
