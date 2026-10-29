package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// capRoots lays the repository's and the machine's config.json, each only
// when given.
func capRoots(t *testing.T, repo, machine string) layered.Roots {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	write := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if repo != "" {
		write(filepath.Join(r.Repo, ".abcd", "config.json"), repo)
	}
	if machine != "" {
		write(abcdhome.Path(r.Home, "config.json"), machine)
	}
	return r
}

func TestTheRunsCapOverridesThePersonsConfiguredCap(t *testing.T) {
	r := capRoots(t, "", `{"matrix":{"spend_cap":{"amount":50,"unit":"usd"}}}`)
	got, err := resolveCap(r, &Money{Amount: 5, Unit: "usd"})
	if err != nil || got.Cap == nil || got.Cap.Amount != 5 || got.Origin != CapFromRun {
		t.Fatalf("cap = %+v, %v; want the run's 5 usd", got, err)
	}
	got, err = resolveCap(r, nil)
	if err != nil || got.Cap == nil || got.Cap.Amount != 50 || got.Origin != CapFromMachine {
		t.Fatalf("cap = %+v, %v; want the machine's 50 usd", got, err)
	}
	if got, err := resolveCap(capRoots(t, "", ""), nil); err != nil || got.Cap != nil || got.Origin != "" {
		t.Fatalf("with no cap anywhere cap = %+v, %v; want none and no default", got, err)
	}
}

func TestACapInTheRepositoryConfigurationIsRefused(t *testing.T) {
	_, err := resolveCap(capRoots(t, `{"matrix":{"spend_cap":{"amount":500,"unit":"usd"}}}`, ""), nil)
	if err == nil || !strings.Contains(err.Error(), ".abcd/config.json") || !strings.Contains(err.Error(), "spending") {
		t.Fatalf("err = %v; want the repository's cap refused naming its file", err)
	}
}

func TestAMachineCapThatIsNotAFigureInUSDIsRefusedNamingItsFile(t *testing.T) {
	for _, body := range []string{
		`{"matrix":{"spend_cap":{"amount":-1,"unit":"usd"}}}`,
		`{"matrix":{"spend_cap":{"amount":5,"unit":"credits"}}}`,
		`{"matrix":{"spend_cap":5}}`,
		`{"matrix":{"spendcap":{"amount":5,"unit":"usd"}}}`,
	} {
		if _, err := resolveCap(capRoots(t, "", body), nil); err == nil || !strings.Contains(err.Error(), abcdhome.Display("config.json")) {
			t.Fatalf("%s: err = %v; want a refusal naming the machine file", body, err)
		}
	}
}
