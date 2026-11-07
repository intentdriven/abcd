package ahoy

import (
	"runtime/debug"
	"testing"
)

// TestRunningReleaseReadsThisProcesssOwnBuild: a release build names its tag, and
// a source build, a pseudo-version, a modified tree and no metadata name none.
func TestRunningReleaseReadsThisProcesssOwnBuild(t *testing.T) {
	prev := runningBuild
	t.Cleanup(func() { runningBuild = prev })
	build := func(version string, settings ...debug.BuildSetting) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Path: abcdMainPackage, Main: debug.Module{Version: version}, Settings: settings}, true
		}
	}
	runningBuild = build("v0.17.0")
	if tag, err := RunningRelease(); err != nil || tag != "v0.17.0" {
		t.Fatalf("got %q, %v", tag, err)
	}
	for name, read := range map[string]func() (*debug.BuildInfo, bool){
		"source build":  build("(devel)"),
		"pseudo":        build("v0.17.1-0.20261106120000-abcdefabcdef"),
		"modified tree": build("v0.17.0", debug.BuildSetting{Key: "vcs.modified", Value: "true"}),
		"no metadata":   func() (*debug.BuildInfo, bool) { return nil, false },
	} {
		runningBuild = read
		if tag, err := RunningRelease(); err == nil {
			t.Errorf("%s names release %q", name, tag)
		}
	}
}
