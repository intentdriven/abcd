package lint_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// TestSettingsFileShape holds the host's committed settings file
// (spc-2610120941441282 step 4, itd-2610120738075547 C11 and decision 10) to
// the three things it does and nothing more: it enables the abcd plugin from
// this repository's own marketplace, switches off the host's co-author line
// and session link, and runs the connector's setup step from one SessionStart
// hook. The file belongs to the host connector (C12), though it sits where the
// host reads it; TestNoHostMarkerOutsideConnector exempts it by name.
func TestSettingsFileShape(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	var market struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name   string `json:"name"`
			Source struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".claude-plugin/marketplace.json")), &market); err != nil {
		t.Fatalf("read the marketplace: %v", err)
	}
	if market.Name == "" || len(market.Plugins) != 1 || market.Plugins[0].Name == "" {
		t.Fatalf("the marketplace names %q with %d plugins; this test expects one plugin, abcd's", market.Name, len(market.Plugins))
	}
	// The repository that hosts the marketplace, read from where it serves
	// the plugin's release: https://github.com/<owner>/<name>/releases/...
	m := regexp.MustCompile(`^https://github\.com/([A-Za-z0-9-]+/[A-Za-z0-9._-]+)/releases/`).FindStringSubmatch(market.Plugins[0].Source.URL)
	if m == nil {
		t.Fatalf("the marketplace's plugin source %q names no github.com repository", market.Plugins[0].Source.URL)
	}
	repo := m[1]

	var settings map[string]any
	if err := json.Unmarshal([]byte(readRepoFile(t, root, hostSettingsFile)), &settings); err != nil {
		t.Fatalf("%s is not JSON: %v", hostSettingsFile, err)
	}
	keys := []string{}
	for k := range settings {
		keys = append(keys, k)
	}
	want := map[string]any{
		"extraKnownMarketplaces": map[string]any{
			market.Name: map[string]any{"source": map[string]any{"source": "github", "repo": repo}},
		},
		"enabledPlugins": map[string]any{market.Plugins[0].Name + "@" + market.Name: true},
		"attribution":    map[string]any{"commit": "", "sessionUrl": false},
	}
	for k, v := range want {
		if !reflect.DeepEqual(settings[k], v) {
			t.Errorf("%s: %s = %#v, want %#v", hostSettingsFile, k, settings[k], v)
		}
	}
	if len(settings) != len(want)+1 {
		t.Errorf("%s carries the keys %v; it carries the plugin, the attribution and the hooks, and nothing more", hostSettingsFile, keys)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok || len(hooks) != 1 {
		t.Fatalf("%s: hooks = %#v, want the one SessionStart event", hostSettingsFile, settings["hooks"])
	}
	groups, ok := hooks["SessionStart"].([]any)
	if !ok || len(groups) != 1 {
		t.Fatalf("%s: hooks.SessionStart = %#v, want one matcher group", hostSettingsFile, hooks["SessionStart"])
	}
	group, _ := groups[0].(map[string]any)
	handlers, ok := group["hooks"].([]any)
	if !ok || len(handlers) != 1 {
		t.Fatalf("%s: the SessionStart group = %#v, want one handler", hostSettingsFile, group)
	}
	setup := hostConnectorDir + "/" + connectorValue(t, root, "setup")
	if !reflect.DeepEqual(handlers[0], map[string]any{
		"type":    "command",
		"command": `bash "$CLAUDE_PROJECT_DIR"/` + setup,
	}) {
		t.Errorf("%s: the SessionStart handler = %#v, want one command running %s and nothing else", hostSettingsFile, handlers[0], setup)
	}
	for k := range group {
		if k != "hooks" && k != "matcher" {
			t.Errorf("%s: the SessionStart group carries %q", hostSettingsFile, k)
		}
	}
}

// connectorValue reads one key of the host connector's declaration, as data.
func connectorValue(t *testing.T, root, key string) string {
	t.Helper()
	for _, line := range strings.Split(readRepoFile(t, root, hostConnectorDir+"/connector.conf"), "\n") {
		if v, ok := strings.CutPrefix(line, key+"="); ok {
			return v
		}
	}
	t.Fatalf("the host connector declares no %s", key)
	return ""
}
