package surface

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// inputs.go — the shape of a filled-in input, derived from the Go type its
// owning package decodes it into (spc-2610301431075964 step 1, approach 2).
//
// The shape had no home in code: it was a Go type in the owning core package,
// its version a constant there, and its example was written only in a command
// page's prose, which nothing checked. The input registry in the front door
// (internal/surface/cli/inputs.go) now names, for each input flag, that type,
// that constant, an example held beside the type, and the owning package's own
// parse-and-validate function. This file turns the type into a field list by
// reflection, so the list an agent is shown is the one the decoder holds a
// document to and can never be written separately from it.

// The roots an input's document may have.
const (
	// RootObject is a JSON object decoded into a struct.
	RootObject = "object"
	// RootArray is a bare JSON array, decoded into a slice of a struct; its
	// field list is the element's.
	RootArray = "array"
)

// Where an input carries its schema version.
const (
	// VersionField is a top-level "schema_version" member.
	VersionField = "schema_version"
	// VersionType is the version suffix of the document's "_type" tag
	// ("abcd.reading.output/1").
	VersionType = "_type"
	// VersionNone is an input that carries no version: the version is the
	// registry's alone, which `explain` says ("the input carries none").
	VersionNone = "none"
)

// MaxFieldDepth is how deep a field list goes: the top-level fields are
// depth one, and a nested object's fields are listed down to depth four. A
// deeper object is named with its type and no fields.
const MaxFieldDepth = 4

// Field is one member of an input's document.
type Field struct {
	// Name is the JSON member name.
	Name string `json:"name"`
	// Type is "string", "integer", "number", "boolean", "object", "any",
	// "list of <type>" or "map of <type>".
	Type string `json:"type"`
	// Required is a member the type declares without omitempty. It is the
	// declaration's claim, which is what the decoder reads: a member left out
	// decodes to its zero value, and the owning package's validator is what
	// refuses a zero it cannot take.
	Required bool `json:"required"`
	// Doc is the member's `doc:"…"` tag, where the owning package writes one.
	Doc string `json:"doc,omitempty"`
	// Fields are a nested object's members (an object, or the elements of a
	// list of objects), down to MaxFieldDepth.
	Fields []Field `json:"fields,omitempty"`
}

// InputShape is one filled-in input in full: what the facts name, plus the
// field list and the example. It is what `explain --input` prints.
type InputShape struct {
	Path          string  `json:"path"`
	Flag          string  `json:"flag"`
	Name          string  `json:"name"`
	SchemaVersion int     `json:"schema_version"`
	VersionIn     string  `json:"version_in"`
	Root          string  `json:"root"`
	Fields        []Field `json:"fields"`
	Example       string  `json:"example"`
}

// Fact is the shape as the facts name it.
func (s InputShape) Fact() InputFact {
	return InputFact{Flag: s.Flag, Kind: InputJSON, Name: s.Name, SchemaVersion: s.SchemaVersion, VersionIn: s.VersionIn}
}

// NewInputShape builds the shape of the input a command reads through flag,
// decoded into t from a document with the given root. It refuses a
// declaration that cannot be one: an unknown root or carrier, a root t is not
// (an object root needs a struct, an array root a slice of structs), a
// version below one, a version carried in a member t does not declare, and an
// empty example.
func NewInputShape(path, flag, name string, t reflect.Type, root string, version int, versionIn, example string) (InputShape, error) {
	s := InputShape{Path: path, Flag: flag, Name: name, SchemaVersion: version, VersionIn: versionIn, Root: root, Example: example}
	fail := func(format string, a ...any) (InputShape, error) {
		return InputShape{}, fmt.Errorf("input %s --%s: %s", path, flag, fmt.Sprintf(format, a...))
	}
	if t == nil {
		return fail("names no Go type")
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var elem reflect.Type
	switch root {
	case RootObject:
		elem = t
	case RootArray:
		if t.Kind() != reflect.Slice {
			return fail("declares an array root, but %s is not a slice", t)
		}
		elem = t.Elem()
		for elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}
	default:
		return fail("root %q is not %s or %s", root, RootObject, RootArray)
	}
	if elem.Kind() != reflect.Struct {
		return fail("decodes into %s, which is not a struct, so it has no field list", elem)
	}
	if version < 1 {
		return fail("schema version %d is below 1", version)
	}
	if strings.TrimSpace(name) == "" {
		return fail("names no input")
	}
	if strings.TrimSpace(example) == "" {
		return fail("carries no example")
	}
	s.Fields = fieldsOf(elem, 1)
	switch versionIn {
	case VersionField, VersionType:
		if root != RootObject {
			return fail("carries its version in %q, which an array root has nowhere to hold", versionIn)
		}
		if !hasField(s.Fields, versionIn) {
			return fail("carries its version in %q, which %s does not declare", versionIn, elem)
		}
	case VersionNone:
	default:
		return fail("version carrier %q is not %s, %s or %s", versionIn, VersionField, VersionType, VersionNone)
	}
	return s, nil
}

func hasField(fields []Field, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

var (
	rawMessageType      = reflect.TypeOf(json.RawMessage(nil))
	timeType            = reflect.TypeOf(time.Time{})
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// fieldsOf lists the members encoding/json decodes t's fields from, in
// declaration order. It follows encoding/json's own rules for the parts that
// decide a member name: an unexported field is skipped, `json:"-"` is
// skipped, and an embedded struct with no name of its own contributes its
// members as if they were declared in place.
func fieldsOf(t reflect.Type, depth int) []Field {
	var out []Field
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if sf.Anonymous && name == "" {
			et := sf.Type
			for et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				out = append(out, fieldsOf(et, depth)...)
				continue
			}
		}
		if !sf.IsExported() {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		f := Field{
			Name:     name,
			Required: !hasOption(opts, "omitempty") && !hasOption(opts, "omitzero"),
			Doc:      sf.Tag.Get("doc"),
		}
		f.Type, f.Fields = describeType(sf.Type, depth)
		out = append(out, f)
	}
	return out
}

func hasOption(opts, want string) bool {
	for _, o := range strings.Split(opts, ",") {
		if o == want {
			return true
		}
	}
	return false
}

// describeType names t as a JSON type, and lists its members when it is an
// object (or a list of objects) above the depth limit.
func describeType(t reflect.Type, depth int) (string, []Field) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == rawMessageType:
		return "any", nil
	case t == timeType:
		return "string", nil
	case reflect.PointerTo(t).Implements(textUnmarshalerType) && t.Kind() != reflect.Struct:
		return "string", nil
	}
	switch t.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer", nil
	case reflect.Float32, reflect.Float64:
		return "number", nil
	case reflect.Interface:
		return "any", nil
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 && t.Kind() == reflect.Slice {
			return "string", nil // encoding/json reads a []byte as base64 text
		}
		name, fields := describeType(t.Elem(), depth)
		return "list of " + name, fields
	case reflect.Map:
		name, fields := describeType(t.Elem(), depth)
		return "map of " + name, fields
	case reflect.Struct:
		if depth >= MaxFieldDepth {
			return "object", nil
		}
		return "object", fieldsOf(t, depth+1)
	}
	return t.Kind().String(), nil
}
