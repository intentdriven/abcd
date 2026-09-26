package launch

import (
	"errors"
	"strings"
	"testing"
)

// itd-2609150819432059: one reader validates the artefact declaration, and
// every launch verb and ahoy go through it.

func TestLoadArtefactReadsEachAcceptedKind(t *testing.T) {
	for _, kind := range []ArtefactKind{KindPlugin, KindBinary, KindApplication} {
		root := t.TempDir()
		writeFile(t, root, ArtefactRelPath, `{"kind": "`+string(kind)+`"}`)
		art, err := LoadArtefact(root)
		if err != nil {
			t.Fatalf("kind %s: %v", kind, err)
		}
		if art.Kind != kind || len(art.Lockstep) != 0 || art.Site {
			t.Errorf("kind %s: read %+v", kind, art)
		}
	}
}

// The absent declaration is its own named refusal: it names the file as the
// declaration's home and the kinds it accepts, never a missing-file error.
func TestLoadArtefactAbsentNamesTheHomeAndTheKinds(t *testing.T) {
	_, err := LoadArtefact(t.TempDir())
	if !errors.Is(err, ErrNoArtefact) {
		t.Fatalf("err = %v, want ErrNoArtefact", err)
	}
	for _, want := range []string{ArtefactRelPath, "plugin", "binary", "application", "abcd ahoy install"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	for _, not := range []string{"no such file", "not found"} {
		if strings.Contains(err.Error(), not) {
			t.Errorf("the refusal reads as a missing-file error (%q): %v", not, err)
		}
	}
}

func TestLoadArtefactRefusesAnUnknownKindNamingTheAcceptedSet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ArtefactRelPath, `{"kind": "container-image"}`)
	_, err := LoadArtefact(root)
	if err == nil || errors.Is(err, ErrNoArtefact) {
		t.Fatalf("err = %v, want a refusal of the unknown kind", err)
	}
	for _, want := range []string{`"container-image"`, "plugin, binary, application"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

func TestLoadArtefactReadsTheLockstepListAndTheSiteOptIn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ArtefactRelPath,
		`{"kind": "binary", "site": true, "lockstep": ["package.json", {"path": "app/meta.json", "json_pointer": "/release/version"}]}`)
	art, err := LoadArtefact(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []LockstepFile{{Path: "package.json"}, {Path: "app/meta.json", Pointer: "/release/version"}}
	if len(art.Lockstep) != 2 || art.Lockstep[0] != want[0] || art.Lockstep[1] != want[1] || !art.Site {
		t.Fatalf("read %+v, want lockstep %+v and site true", art, want)
	}
}

func TestLoadArtefactRefusesMalformedDeclarations(t *testing.T) {
	cases := map[string]struct{ body, want string }{
		"not json":            {`{kind: plugin`, "not a JSON object"},
		"not an object":       {`["plugin"]`, "not a JSON object"},
		"no kind":             {`{"lockstep": []}`, "declares no kind"},
		"kind not a string":   {`{"kind": 3}`, "declares no kind"},
		"unknown key":         {`{"kind": "binary", "lockstop": []}`, `"lockstop"`},
		"lockstep not a list": {`{"kind": "binary", "lockstep": "a.json"}`, "lockstep"},
		"escaping path":       {`{"kind": "binary", "lockstep": ["../a.json"]}`, "../a.json"},
		"absolute path":       {`{"kind": "binary", "lockstep": ["/etc/a.json"]}`, "/etc/a.json"},
		"record namespace":    {`{"kind": "binary", "lockstep": [".abcd/config/x.json"]}`, ".abcd/config/x.json"},
		"bad pointer":         {`{"kind": "binary", "lockstep": [{"path": "a.json", "json_pointer": "version"}]}`, "json_pointer"},
		"duplicate path":      {`{"kind": "binary", "lockstep": ["a.json", "a.json"]}`, "twice"},
		"site not a boolean":  {`{"kind": "binary", "site": "yes"}`, "site"},
		"plugin with a list":  {`{"kind": "plugin", "lockstep": ["a.json"]}`, "plugin"},
		"misspelt entry key":  {`{"kind": "binary", "lockstep": [{"path": "a.json", "pointer": "/v"}]}`, "lockstep entry"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ArtefactRelPath, c.body)
			_, err := LoadArtefact(root)
			if err == nil {
				t.Fatalf("accepted %s", c.body)
			}
			if errors.Is(err, ErrNoArtefact) {
				t.Fatalf("a malformed declaration read as an absent one: %v", err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal does not name %q: %v", c.want, err)
			}
		})
	}
}

// A declaration is read the way it is written: a repeated key is refused
// rather than read last-wins, and a lockstep entry's keys are matched exactly,
// so a case-folded spelling encoding/json would bind is refused as the top
// level already refuses "Kind" (iss-2609260149249724).
func TestParseArtefactRefusesRepeatedAndCaseFoldedKeys(t *testing.T) {
	cases := map[string]struct{ body, want string }{
		"repeated kind":        {`{"kind": "application", "kind": "binary"}`, `"kind"`},
		"kind case twin":       {`{"kind": "binary", "KIND": "plugin"}`, `"KIND"`},
		"repeated entry path":  {`{"kind": "binary", "lockstep": [{"path": "a.json", "path": "b.json"}]}`, `"path"`},
		"upper-case path":      {`{"kind": "binary", "lockstep": [{"PATH": "a.json"}]}`, "lockstep entry"},
		"upper-case pointer":   {`{"kind": "binary", "lockstep": [{"path": "a.json", "JSON_POINTER": "/v"}]}`, "lockstep entry"},
		"mixed-case entry key": {`{"kind": "binary", "lockstep": [{"Path": "a.json"}]}`, "lockstep entry"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			art, err := ParseArtefact([]byte(c.body))
			if err == nil {
				t.Fatalf("accepted %s as %+v", c.body, art)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal does not name %q: %v", c.want, err)
			}
		})
	}
}
