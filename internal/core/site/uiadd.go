package site

// Completing an older ui.json (the TG1 ruling, recorded in adr-2609301720596683).
//
// A repository's ui.json is the repository's own once it exists: `site setup`
// seeds it and never rewrites it. The one exception is a label the closed
// allowlist declares and the file does not carry at all, which is what a file
// written before that label existed looks like. `site setup` and `site build`
// add each such label with the word abcd's own ui.json gives it, and change
// nothing else: a label the file declares keeps its wording (a blank one is
// still refused by name), an unknown key is still refused by the closed
// allowlist and blocks the adding, and every byte already in the file stays
// where it is. The added members go at the end of their block, in the block's
// own indentation.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// jsonObject is one object of the file as written: where its members start and
// end, and the objects its members hold.
type jsonObject struct {
	open     int64 // offset just past '{'
	lastEnd  int64 // offset just past the last member's value; -1 when empty
	close    int64 // offset of '}'
	keys     map[string]bool
	children map[string]*jsonObject
}

// uiEdit replaces src[start:end] with text.
type uiEdit struct {
	start, end int64
	text       string
}

// addMissingLabels adds to the ui.json at rel, under repoRoot, every label the
// allowlist declares and the file does not carry, each with its default words,
// and returns the added labels by their path in the file (`status.target`).
// A file that carries every label, that does not decode against the allowlist
// (an unknown key, a syntax fault), or that is absent, is left untouched and
// nothing is returned: LoadUI reports it. A symlinked or non-regular file is
// refused as LoadUI refuses it.
func addMissingLabels(repoRoot, rel string) ([]string, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	src, err := fsutil.ReadGuardedInRoot(root, rel, maxUIBytes)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.DisallowUnknownFields()
	var probe UI
	if dec.Decode(&probe) != nil {
		return nil, nil
	}
	top, err := scanObject(src)
	if err != nil {
		return nil, nil
	}
	defaults, err := defaultUI()
	if err != nil {
		return nil, err
	}
	unit := memberIndent(src, top)
	if unit == "" {
		unit = "  "
	}
	var added []string
	var edits []uiEdit
	planLabels(src, top, reflect.TypeOf(UI{}), reflect.ValueOf(defaults), "", unit, true, &added, &edits)
	if len(edits) == 0 {
		return nil, nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := append([]byte(nil), src...)
	for _, e := range edits {
		out = append(out[:e.start:e.start], append([]byte(e.text), out[e.end:]...)...)
	}
	check := json.NewDecoder(bytes.NewReader(out))
	check.DisallowUnknownFields()
	if err := check.Decode(&probe); err != nil {
		return nil, fmt.Errorf("%w: %s: adding the missing labels did not produce a readable file: %v", ErrUIInvalid, rel, err)
	}
	if err := fsutil.WriteFileAtomicPreserveModeInRoot(root, rel, out); err != nil {
		return nil, err
	}
	return added, nil
}

// defaultUI is abcd's own ui.json, the source of every default word.
func defaultUI() (UI, error) {
	data, err := setupSources.ReadFile("setupsrc/ui.json")
	if err != nil {
		return UI{}, err
	}
	var ui UI
	if err := json.Unmarshal(data, &ui); err != nil {
		return UI{}, err
	}
	return ui, nil
}

// planLabels walks one object against its struct, recording an edit for the
// declared labels the object lacks. A declared block the file lacks is added
// whole; a map (forge_names) is never required, so never added; `_purpose` is
// never rendered, so never added.
func planLabels(src []byte, obj *jsonObject, t reflect.Type, def reflect.Value, prefix, unit string, parentMultiline bool, added *[]string, edits *[]uiEdit) {
	var members []string
	indent := memberIndent(src, obj)
	multiline := strings.Contains(string(src[obj.open:firstNonSpace(src, obj.open)]), "\n")
	if obj.lastEnd < 0 {
		// An empty block has no member to copy: it takes its indentation from
		// its closing brace, and breaks its lines when its parent does.
		indent = lineIndent(src, obj.close) + unit
		multiline = parentMultiline
	}
	nl := lineBreak(src)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		key := strings.Split(f.Tag.Get("json"), ",")[0]
		if key == "_purpose" || f.Type.Kind() == reflect.Map {
			continue
		}
		path := prefix + key
		if obj.keys[key] {
			if child := obj.children[key]; child != nil && f.Type.Kind() == reflect.Struct {
				planLabels(src, child, f.Type, def.Field(i), path+".", unit, multiline, added, edits)
			}
			continue
		}
		var val string
		switch f.Type.Kind() {
		case reflect.String:
			if strings.TrimSpace(def.Field(i).String()) == "" {
				continue
			}
			val = jsonString(def.Field(i).String())
			*added = append(*added, path)
		case reflect.Struct:
			val = renderBlock(f.Type, def.Field(i), path+".", indent, unit, multiline, nl, added)
			if val == "" {
				continue
			}
		default:
			continue
		}
		members = append(members, jsonString(key)+": "+val)
	}
	if len(members) == 0 {
		return
	}
	sep := ", "
	lead := ""
	if multiline {
		sep = "," + nl + indent
		lead = nl + indent
	}
	if obj.lastEnd >= 0 {
		*edits = append(*edits, uiEdit{start: obj.lastEnd, end: obj.lastEnd, text: "," + strings.TrimPrefix(sep, ",") + strings.Join(members, sep)})
		return
	}
	text := lead + strings.Join(members, sep)
	if multiline {
		text += nl + lineIndent(src, obj.close)
	}
	*edits = append(*edits, uiEdit{start: obj.open, end: obj.close, text: text})
}

// renderBlock renders a whole declared block the file lacks, naming each of
// its labels in added.
func renderBlock(t reflect.Type, def reflect.Value, prefix, indent, unit string, multiline bool, nl string, added *[]string) string {
	var members []string
	inner := indent + unit
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		key := strings.Split(f.Tag.Get("json"), ",")[0]
		var val string
		switch f.Type.Kind() {
		case reflect.String:
			if strings.TrimSpace(def.Field(i).String()) == "" {
				continue
			}
			val = jsonString(def.Field(i).String())
			*added = append(*added, prefix+key)
		case reflect.Struct:
			val = renderBlock(f.Type, def.Field(i), prefix+key+".", inner, unit, multiline, nl, added)
			if val == "" {
				continue
			}
		default:
			continue
		}
		members = append(members, jsonString(key)+": "+val)
	}
	if len(members) == 0 {
		return ""
	}
	if !multiline {
		return "{" + strings.Join(members, ", ") + "}"
	}
	return "{" + nl + inner + strings.Join(members, ","+nl+inner) + nl + indent + "}"
}

// jsonString encodes s as a JSON string without HTML escaping, as a person
// would write it.
func jsonString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimRight(b.String(), "\n")
}

// scanObject reads the file's top-level object and every object it holds,
// with the offsets an insertion needs.
func scanObject(src []byte) (*jsonObject, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("ui.json is not an object")
	}
	return readObject(dec)
}

// readObject reads the members of an object whose '{' was just read.
func readObject(dec *json.Decoder) (*jsonObject, error) {
	obj := &jsonObject{open: dec.InputOffset(), lastEnd: -1, keys: map[string]bool{}, children: map[string]*jsonObject{}}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if d, ok := tok.(json.Delim); ok && d == '}' {
			obj.close = dec.InputOffset() - 1
			return obj, nil
		}
		key, ok := tok.(string)
		if !ok {
			return nil, errors.New("ui.json: an object key is not a string")
		}
		obj.keys[key] = true
		child, err := readValue(dec)
		if err != nil {
			return nil, err
		}
		if child != nil {
			obj.children[key] = child
		}
		obj.lastEnd = dec.InputOffset()
	}
}

// readValue reads one value, returning it when it is an object.
func readValue(dec *json.Decoder) (*jsonObject, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return nil, nil
	}
	switch d {
	case '{':
		return readObject(dec)
	case '[':
		for dec.More() {
			if _, err := readValue(dec); err != nil {
				return nil, err
			}
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return nil, io.ErrUnexpectedEOF
}

// firstNonSpace is the offset of the first byte at or after i that is not
// JSON whitespace.
func firstNonSpace(src []byte, i int64) int64 {
	for i < int64(len(src)) && strings.IndexByte(" \t\r\n", src[i]) >= 0 {
		i++
	}
	return i
}

// memberIndent is the indentation of an object's first member: the spaces
// after the last line break before it, empty when it shares the brace's line.
func memberIndent(src []byte, obj *jsonObject) string {
	run := string(src[obj.open:firstNonSpace(src, obj.open)])
	nl := strings.LastIndexByte(run, '\n')
	if nl < 0 {
		return ""
	}
	return run[nl+1:]
}

// lineIndent is the leading whitespace of the line holding offset i.
func lineIndent(src []byte, i int64) string {
	start := bytes.LastIndexByte(src[:i], '\n') + 1
	end := start
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return string(src[start:end])
}

// lineBreak is the file's own line break: CRLF when it uses one, LF otherwise.
func lineBreak(src []byte) string {
	if bytes.Contains(src, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}
