package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The budget check at the surface (itd-2609201925079472 criterion 7,
// spc-2609301921521360): `abcd build` hands the runner configuration it read
// to the start, so a route the machine sends a role to is asked for its quota
// beside the host's; neither reports one, and each is named in the text, the
// JSON and the run record, skipped out loud.

func TestBuildNamesEachRouteItsBudgetCheckSkipped(t *testing.T) {
	buildRepo(t)
	home := os.Getenv("HOME")
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	machine := `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(machine), 0o600); err != nil {
		t.Fatal(err)
	}
	out := mustImplement(t, "build", "itd-10", "--json")
	var res struct {
		Budget *struct {
			Rows []struct {
				Route           string   `json:"route"`
				Roles           []string `json:"roles"`
				RemainingTokens *int64   `json:"remaining_tokens"`
				Checked         bool     `json:"checked"`
				Detail          string   `json:"detail"`
			} `json:"rows"`
		} `json:"budget"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Budget == nil || len(res.Budget.Rows) != 2 {
		t.Fatalf("the payload carries a budget row per route: %s", out)
	}
	routes := map[string]bool{}
	for _, r := range res.Budget.Rows {
		routes[r.Route] = true
		if r.Checked || r.RemainingTokens != nil || !strings.Contains(r.Detail, "reports no quota") {
			t.Fatalf("a route reporting no quota is skipped out loud: %+v", r)
		}
	}
	if !routes["claude"] || !routes["host"] {
		t.Fatalf("the routed role's runner and the host are each asked: %s", out)
	}
	status := mustImplement(t, "implement", "status")
	for _, want := range []string{"the claude runner reports no quota", "the host reports no quota"} {
		if !strings.Contains(status, want) {
			t.Fatalf("the run record names %q:\n%s", want, status)
		}
	}
}

func TestBuildTextNamesTheBudgetCheck(t *testing.T) {
	buildRepo(t)
	code, out, errOut := implementCLI(t, "build", "itd-10")
	if code != 0 || !strings.Contains(out, "budget:  the host reports no quota, so its check is skipped") {
		t.Fatalf("the text names the budget check: exit %d\n%s\n%s", code, out, errOut)
	}
}
