package term

import (
	"os"
	"testing"
)

// TestSizeFallsBackToColumnsThen80 pins the width a question is drawn at when
// the stream answers no window size (spc-2610030911534855, "The drawing at 80
// and at 160 columns"): COLUMNS when it is a positive whole number, else 80;
// the rows likewise from LINES, else 24.
func TestSizeFallsBackToColumnsThen80(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	cases := []struct {
		name       string
		env        map[string]string
		cols, rows int
	}{
		{"COLUMNS and LINES", map[string]string{"COLUMNS": "132", "LINES": "50"}, 132, 50},
		{"nothing set", map[string]string{}, 80, 24},
		{"COLUMNS not a number", map[string]string{"COLUMNS": "wide", "LINES": "tall"}, 80, 24},
		{"COLUMNS zero", map[string]string{"COLUMNS": "0", "LINES": "0"}, 80, 24},
		{"COLUMNS negative", map[string]string{"COLUMNS": "-120", "LINES": "-3"}, 80, 24},
		{"COLUMNS padded", map[string]string{"COLUMNS": " 100 "}, 100, 24},
	}
	for _, f := range map[string]*os.File{"a pipe": w, "no file": nil} {
		for _, c := range cases {
			cols, rows := Size(f, env(c.env))
			if cols != c.cols || rows != c.rows {
				t.Errorf("%s: Size = %d×%d, want %d×%d", c.name, cols, rows, c.cols, c.rows)
			}
		}
	}
}
