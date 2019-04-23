package notebook

import (
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type toStudentTest struct {
	name string
	// input is the list of input cells (source code).
	input []string
	// want is the list of expected output cells (source code).
	want []string
}

// createNotebook is a helper function to create
// a notebook from a list of strings following a few mnemonics.
// - A cell is 'code' by default.
// - If the source starts with "## ", it is changed to 'markdown'.
func createNotebook(src []string) *Notebook {
	var cells []*Cell
	for _, cellSource := range src {
		ty := "code"
		if strings.HasPrefix(cellSource, "## ") {
			ty = "markdown"
		}
		cells = append(cells, &Cell{
			Type:   ty,
			Source: cellSource,
		})
	}
	return &Notebook{
		Cells: cells,
	}
}

func TestToStudent(t *testing.T) {
	tests := []toStudentTest{
		{
			name:  "Unchanged1",
			input: []string{"# unchanged"},
			want:  []string{"# unchanged"},
		},
		{
			name:  "Unchanged2",
			input: []string{"# unchanged\nmore", "aaa\nbbb"},
			want:  []string{"# unchanged\nmore", "aaa\nbbb"},
		},
		{
			name:  "Solution1",
			input: []string{"# BEGIN SOLUTION\nx = 1\n# END SOLUTION"},
			want:  []string{"..."},
		},
		{
			name:  "Unittest1",
			input: []string{"# BEGIN UNITTEST\nx = 1\n# END UNITTEST"},
			want:  []string{},
		},
		{
			name:  "Autotest1",
			input: []string{"# BEGIN AUTOTEST\nx = 1\n# END AUTOTEST"},
			want:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := createNotebook(tt.input)
			got, err := n.ToStudent()
			if err != nil {
				t.Errorf("ToStudent([%s]) returned error %s, want success",
					strings.Join(tt.input, "]["), err)
				return
			}
			if len(got.Cells) != len(tt.want) {
				t.Errorf("got %d output cells, want %d", len(got.Cells), len(tt.want))
			}
			var gotSources []string
			for _, cell := range got.Cells {
				gotSources = append(gotSources, cell.Source)
			}
			wantText := strings.Join(tt.want, "\n")
			gotText := strings.Join(gotSources, "\n")
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(wantText, gotText, true)
			different := false
			for _, d := range diffs {
				if d.Type != diffmatchpatch.DiffEqual {
					different = true
					break
				}
			}
			if different {
				t.Logf("Got:\n%q\n--\nWant:\n%q\n--", gotText, wantText)
				t.Errorf("Diffs:\n%s", dmp.DiffPrettyText(diffs))
			}
		})
	}
}
