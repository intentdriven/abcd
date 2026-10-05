package ahoy

import (
	"strconv"
	"strings"
)

// HarnessNotice is the session-start line for the harness's user settings:
// "" when nothing there calls abcd that should not, one line naming the
// finding and its remedy when there is one, and one line counting them and
// pointing at `abcd ahoy` when there are several. It reads the settings file,
// the path-entry record and the plugin's hook manifest — each bounded, none
// written, no network.
func HarnessNotice() string {
	root, ok := resolvePluginRoot()
	fs := harnessFindings(readHarnessSettings(), pluginHookEvents(root, ok))
	switch len(fs) {
	case 0:
		return ""
	case 1:
		f := fs[0]
		switch f.kind {
		case findingStrayHook:
			return "abcd: " + f.settings + " runs abcd for " + f.event + " (`" + f.command + "`); " + f.remedy + " `abcd ahoy` lists every such entry."
		case findingStatusLineDangling:
			return "abcd: the status line in " + f.settings + " runs an abcd that is gone (`" + f.command + "`), so it is blank in every repository; " + f.remedy + "."
		default:
			return "abcd: the status line in " + f.settings + " runs an abcd that fails the trust checks (`" + f.command + "`): " + f.reason + "; " + f.remedy + "."
		}
	}
	keys := make([]string, len(fs))
	for i, f := range fs {
		keys[i] = f.key
	}
	return "abcd: " + fs[0].settings + " has " + strconv.Itoa(len(fs)) + " abcd entries that need attention (" + strings.Join(keys, ", ") +
		"); run `abcd ahoy` to see each and its remedy — abcd changes nothing there but its own status line, and only through `abcd ahoy install`."
}
