package surface

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type shapeLeaf struct {
	Deep string `json:"deep"`
}

type shapeL3 struct {
	Leaf shapeLeaf `json:"leaf"`
}

type shapeL2 struct {
	L3 shapeL3 `json:"l3"`
}

type shapeEmbedded struct {
	Shared string `json:"shared" doc:"from the embedded struct"`
}

type shapeVerdict string

type shapeDoc struct {
	shapeEmbedded
	SchemaVersion int               `json:"schema_version"`
	Name          string            `json:"name" doc:"what the input is called"`
	Optional      string            `json:"optional,omitempty"`
	Skipped       string            `json:"-"`
	unexported    string            //nolint:unused // proves an unexported field is skipped
	Count         *int              `json:"count"`
	Ratio         float64           `json:"ratio"`
	On            bool              `json:"on"`
	Verdict       shapeVerdict      `json:"verdict"`
	At            time.Time         `json:"at"`
	Raw           json.RawMessage   `json:"raw"`
	Any           any               `json:"any"`
	Tags          []string          `json:"tags"`
	Items         []shapeL2         `json:"items"`
	Meta          map[string]string `json:"meta"`
	NoTag         string
}

func TestFieldsOfFollowsEncodingJSON(t *testing.T) {
	fields := fieldsOf(reflect.TypeOf(shapeDoc{}), 1)
	byName := map[string]Field{}
	var names []string
	for _, f := range fields {
		byName[f.Name] = f
		names = append(names, f.Name)
	}
	want := []string{"shared", "schema_version", "name", "optional", "count", "ratio", "on", "verdict", "at", "raw", "any", "tags", "items", "meta", "NoTag"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("members %v, want %v", names, want)
	}
	types := map[string]string{
		"shared": "string", "schema_version": "integer", "count": "integer", "ratio": "number", "on": "boolean",
		"verdict": "string", "at": "string", "raw": "any", "any": "any", "tags": "list of string",
		"items": "list of object", "meta": "map of string",
	}
	for name, typ := range types {
		if byName[name].Type != typ {
			t.Errorf("%s is %q, want %q", name, byName[name].Type, typ)
		}
	}
	if byName["optional"].Required || !byName["name"].Required || !byName["count"].Required {
		t.Errorf("requiredness is not read from omitempty: %+v", fields)
	}
	if byName["name"].Doc != "what the input is called" || byName["shared"].Doc != "from the embedded struct" {
		t.Errorf("doc tags not read: %+v", fields)
	}
}

func TestFieldsOfStopsAtTheDepthLimit(t *testing.T) {
	fields := fieldsOf(reflect.TypeOf(shapeDoc{}), 1)
	var items Field
	for _, f := range fields {
		if f.Name == "items" {
			items = f
		}
	}
	// depth 1: items; 2: l3; 3: leaf; 4: deep. Four levels are listed.
	if len(items.Fields) != 1 || items.Fields[0].Name != "l3" {
		t.Fatalf("items' members %+v", items.Fields)
	}
	l3 := items.Fields[0]
	if len(l3.Fields) != 1 || l3.Fields[0].Name != "leaf" {
		t.Fatalf("l3's members %+v", l3.Fields)
	}
	leaf := l3.Fields[0]
	if len(leaf.Fields) != 1 || leaf.Fields[0].Name != "deep" {
		t.Fatalf("leaf's members %+v, want its one member at depth four", leaf.Fields)
	}

	type d5 struct {
		A struct {
			B struct {
				C struct {
					D struct {
						E string `json:"e"`
					} `json:"d"`
				} `json:"c"`
			} `json:"b"`
		} `json:"a"`
	}
	f := fieldsOf(reflect.TypeOf(d5{}), 1)
	d := f[0].Fields[0].Fields[0].Fields[0]
	if d.Name != "d" || d.Type != "object" || d.Fields != nil {
		t.Errorf("the object at depth four lists its members: %+v", d)
	}
}

func TestNewInputShapeDerivesAnObjectAndAnArray(t *testing.T) {
	obj, err := NewInputShape("abcd x", "doc-json", "a document", reflect.TypeOf(shapeDoc{}), RootObject, 2, VersionField, `{"schema_version": 2}`)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Fact() != (InputFact{Flag: "doc-json", Kind: InputJSON, Name: "a document", SchemaVersion: 2, VersionIn: VersionField}) {
		t.Errorf("fact %+v", obj.Fact())
	}
	arr, err := NewInputShape("abcd x", "docs-json", "documents", reflect.TypeOf([]*shapeLeaf{}), RootArray, 1, VersionNone, `[{"deep": "x"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(arr.Fields) != 1 || arr.Fields[0].Name != "deep" {
		t.Errorf("an array root lists its element's members: %+v", arr.Fields)
	}
}

func TestNewInputShapeRefusesWhatCannotBeADeclaration(t *testing.T) {
	doc, leaves := reflect.TypeOf(shapeDoc{}), reflect.TypeOf([]shapeLeaf{})
	cases := []struct {
		name      string
		t         reflect.Type
		root      string
		version   int
		versionIn string
		example   string
		want      string
	}{
		{"no type", nil, RootObject, 1, VersionNone, "{}", "no Go type"},
		{"array root on a struct", doc, RootArray, 1, VersionNone, "[]", "not a slice"},
		{"object root on a slice", leaves, RootObject, 1, VersionNone, "{}", "not a struct"},
		{"unknown root", doc, "string", 1, VersionNone, "{}", "root"},
		{"version zero", doc, RootObject, 0, VersionField, "{}", "below 1"},
		{"no example", doc, RootObject, 1, VersionField, " ", "no example"},
		{"version in a member the type lacks", reflect.TypeOf(shapeLeaf{}), RootObject, 1, VersionField, "{}", "does not declare"},
		{"version in the _type tag of an array", leaves, RootArray, 1, VersionType, "[]", "array root"},
		{"unknown carrier", doc, RootObject, 1, "header", "{}", "carrier"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewInputShape("abcd x", "doc-json", "a document", c.t, c.root, c.version, c.versionIn, c.example)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one naming %q", err, c.want)
			}
		})
	}
}
