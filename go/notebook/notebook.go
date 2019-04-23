// Package notebook provides utility functions for working with Jupyter/IPython
// notebooks, i.e. JSON files following some conventions.
package notebook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

// Notebook represents a parsed Jupyter notebook.
type Notebook struct {
	// NBFormat is the nbformat field.
	NBFormat int `json:"nbformat"`
	// NBFormatMinor is the nbformat_minor field.
	NBFormatMinor int `json:"nbformat_minor"`
	// Data is the raw parsed JSON data. It is not written back on serialization.
	Data map[string]interface{} `json:"-"`
	// Metadat is the map of metadata.
	Metadata map[string]interface{} `json:"metadata"`
	// Cells is the list of cells.
	Cells []*Cell `json:"cells"`
}

// Cell represents one cell of a Jupyter notebook. It is limited in
// the kind of cells it can represent.
type Cell struct {
	Type string
	// Data is the raw parsed JSON contents of the cell.
	// When serializing cell back to JSON, Data is ignored.
	Data     map[string]interface{}
	Metadata map[string]interface{}
	Outputs  map[string]string
	Source   string
}

func Parse(filename string) (*Notebook, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %q: %s", filename, err)
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON from %q: %s", filename, err)
	}
	ret := &Notebook{
		Data: data,
	}
	if v, ok := data["nbformat"]; ok {
		val, _ := v.(float64)
		ret.NBFormat = int(val)
	}
	if v, ok := data["nbformat_minor"]; ok {
		val, _ := v.(float64)
		ret.NBFormatMinor = int(val)
	}
	ret.Metadata, _ = data["metadata"].(map[string]interface{})
	cells, ok := data["cells"]
	if ok {
		cellsList, ok := cells.([]interface{})
		if !ok {
			return nil, fmt.Errorf(".cells is not a list but %s", reflect.TypeOf(cells))
		}
		for _, x := range cellsList {
			celldata, ok := x.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("cell is not a map but %s", reflect.TypeOf(x))
			}
			cell := &Cell{}
			if v, ok := celldata["cell_type"]; ok {
				cell.Type, _ = v.(string)
			}
			if v, ok := celldata["metadata"]; ok {
				cell.Metadata, ok = v.(map[string]interface{})
			}
			if v, ok := celldata["source"]; ok {
				ss, ok := v.([]interface{})
				if !ok {
					return nil, fmt.Errorf("cell.source is not a list but %s",
						reflect.TypeOf(v))
				}
				var lines []string
				for _, s := range ss {
					str, ok := s.(string)
					if !ok {
						return nil, fmt.Errorf("cell.source has not a string but %s",
							reflect.TypeOf(s))
					}
					lines = append(lines, str)
				}
				cell.Source = strings.Join(lines, "")
			}
			if v, ok := celldata["outputs"]; ok {
				ss, ok := v.([]interface{})
				if !ok {
					return nil, fmt.Errorf("cell.outputs is not a list but %s",
						reflect.TypeOf(v))
				}
				outputs := make(map[string]string)
				for _, s := range ss {
					m, ok := s.(map[string]interface{})
					if !ok {
						continue
					}
					nameVal, ok := m["name"]
					if !ok {
						continue
					}
					name, ok := nameVal.(string)
					if !ok {
						return nil, fmt.Errorf("output name is not a string but %s",
							reflect.TypeOf(nameVal))
					}
					textVal, ok := m["text"]
					if !ok {
						// Skip any non-text outputs.
						continue
					}
					ss, ok := textVal.([]interface{})
					if !ok {
						return nil, fmt.Errorf("cell.output.text is not a list but %s",
							reflect.TypeOf(textVal))
					}
					var lines []string
					for _, s := range ss {
						str, ok := s.(string)
						if !ok {
							return nil, fmt.Errorf("cell.output.text item is not a string but %s",
								reflect.TypeOf(s))
						}
						lines = append(lines, str)
					}
					outputs[name] = strings.Join(lines, "")
				}
				cell.Outputs = outputs
			}
			ret.Cells = append(ret.Cells, cell)
		}
	}
	return ret, nil
}

func marshalText(text string) []interface{} {
	var ret []interface{}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == len(lines)-1 {
			ret = append(ret, line)
			break
		}
		ret = append(ret, line+"\n")
	}
	return ret
}

// json returns a JSON-like map representing a cell.
func (cell *Cell) json() map[string]interface{} {
	emptyMap := make(map[string]interface{})
	ret := make(map[string]interface{})
	var outputs []interface{}
	// TODO(salikh): Do we need to handle any other kind of output?
	for name, output := range cell.Outputs {
		o := make(map[string]interface{})
		o["name"] = name
		o["output_type"] = "stream"
		o["text"] = marshalText(output)
		outputs = append(outputs, o)
	}
	if cell.Metadata != nil {
		ret["metadata"] = cell.Metadata
	} else {
		ret["metadata"] = emptyMap
	}
	ret["cell_type"] = cell.Type
	if cell.Type == "code" {
		ret["execution_count"] = nil
		if len(outputs) > 0 {
			ret["outputs"] = outputs
		} else {
			// Empty slice.
			ret["outputs"] = []interface{}{}
		}
	}
	ret["source"] = marshalText(cell.Source)
	return ret
}

// Marshal produces a JSON content suitable for writing to .ipynb file.
func (n *Notebook) Marshal() ([]byte, error) {
	output := make(map[string]interface{})
	var cells []interface{}
	for _, cell := range n.Cells {
		cells = append(cells, cell.json())
	}
	output["nbformat"] = n.NBFormat
	output["nbformat_minor"] = n.NBFormatMinor
	output["metadata"] = n.Metadata
	output["cells"] = cells
	return json.Marshal(output)
}

func (n *Notebook) MapCells(mapFunc func(c *Cell) (*Cell, error)) (*Notebook, error) {
	var out []*Cell
	for _, cell := range n.Cells {
		ncell, err := mapFunc(cell)
		if err != nil {
			return nil, err
		}
		if ncell != nil {
			out = append(out, ncell)
		}
	}
	return &Notebook{
		NBFormat:      n.NBFormat,
		NBFormatMinor: n.NBFormatMinor,
		Metadata:      n.Metadata,
		Cells:         out,
	}, nil
}

// TODO(salikh): Implement smarter replacement strategies similar to jassign, e.g.
// x = 1 # SOLUTION   ===>   x = ...
var (
	assignmentMetadataRegex = regexp.MustCompile("(?m)^[ \t]*# ASSIGNMENT METADATA")
	exerciseMetadataRegex   = regexp.MustCompile("(?m)^[ \t]*# EXERCISE METADATA")
	tripleBacktickedRegex   = regexp.MustCompile("(?ms)^```.*^```")
	solutionBeginRegex      = regexp.MustCompile("(?m)^([ \t]*)# BEGIN SOLUTION *\n")
	solutionEndRegex        = regexp.MustCompile("(?m)^[ \t]*# END SOLUTION *")
	promptBeginRegex        = regexp.MustCompile("(?m)^[ \t]*\"\"\" # BEGIN PROMPT *\n")
	promptEndRegex          = regexp.MustCompile("\n[ \t]*\"\"\" # END PROMPT *\n")
	testRegex               = regexp.MustCompile("(?m)^[ \t]*# TEST *")
	unittestBeginRegex      = regexp.MustCompile("(?m)^[ \t]*# BEGIN UNITTEST *\n")
	unittestEndRegex        = regexp.MustCompile("(?m)^[ \t]*# END UNITTEST *")
	autotestBeginRegex      = regexp.MustCompile("(?m)^[ \t]*# BEGIN AUTOTEST *")
	autotestEndRegex        = regexp.MustCompile("(?m)^[ \t]*# END AUTOTEST *")
)

// hasMetadata detects whether the markdown block has a triple backtick-fenced block
// with a metadata marker given as a Regexp.
func hasMetadata(re *regexp.Regexp, source string) bool {
	mm := tripleBacktickedRegex.FindAllStringIndex(source, -1)
	for _, m := range mm {
		text := source[m[0]+3 : m[1]-3]
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// If text matches begin and end regexes in sequences, returns
// the text that is enclosed by matches. If the text does not match,
// return empty string. Returns error in pathological cases.
func cutText(begin, end *regexp.Regexp, text string) (string, error) {
	mbeg := begin.FindStringIndex(text)
	if mbeg == nil {
		return "", nil
	}
	mend := end.FindStringIndex(text)
	if mend == nil {
		return "", fmt.Errorf("missing %s", end)
	}
	if mend[1] < mbeg[0] {
		return "", fmt.Errorf("%s before %s", end, begin)
	}
	return text[mbeg[1]:mend[0]], nil
}

// extractMetadata extracts the metadata from the markdown cell, using the provided
// regexp to detect metadata fenced blocks. It returns nil if the source does not
// have any metadata fenced block. The second return argument is the source code
// with metadata block cut out, or the input source string if there were no metadata.
func extractMetadata(re *regexp.Regexp, source string) (metadata map[string]interface{}, newSource string, err error) {
	var outputs []string
	mm := tripleBacktickedRegex.FindAllStringIndex(source, -1)
	for i, m := range mm {
		if len(outputs) == 0 {
			outputs = append(outputs, source[0:m[0]])
		}
		text := source[m[0]+3 : m[1]-3]
		if re.MatchString(text) {
			metadata = make(map[string]interface{})
			err = yaml.Unmarshal([]byte(text), &metadata)
			if err != nil {
				err = fmt.Errorf("error parsing metadata: %s", err)
				return
			}
		} else {
			outputs = append(outputs, source[m[0]:m[1]])
		}
		if i < len(mm)-1 {
			outputs = append(outputs, source[m[1]:mm[i+1][0]])
		} else {
			outputs = append(outputs, source[m[1]:])
		}
	}
	newSource = strings.Join(outputs, "")
	return
}

// ToStudent converts a master notebook into the student notebook.
func (n *Notebook) ToStudent() (*Notebook, error) {
	// Assignment metadata is global for the notebook.
	assignmentMetadata := make(map[string]interface{})
	// Exercise metadata only applies to the next code block,
	// and is nil otherwise.
	var exerciseMetadata map[string]interface{}
	transformed, err := n.MapCells(func(cell *Cell) (*Cell, error) {
		source := cell.Source
		if cell.Type == "markdown" {
			var err error
			if hasMetadata(assignmentMetadataRegex, cell.Source) {
				var metadata map[string]interface{}
				metadata, source, err = extractMetadata(assignmentMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
				// Merge assignment metadata to global table.
				for k, v := range metadata {
					assignmentMetadata[k] = v
				}
			}
			if hasMetadata(exerciseMetadataRegex, cell.Source) {
				// Replace exercise metadata.
				exerciseMetadata, source, err = extractMetadata(exerciseMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
			}
		}
		if cell.Type != "code" {
			return &Cell{Type: cell.Type, Source: source}, nil
		}
		prompt := ""
		if promptBeginRegex.MatchString(source) {
		}
		if mbeg := promptBeginRegex.FindStringIndex(source); mbeg != nil {
			mend := promptEndRegex.FindStringIndex(source)
			if mend == nil {
				return nil, fmt.Errorf("BEGIN PROMPT has no matching END PROMPT")
			}
			if mend[1] < mbeg[0] {
				return nil, fmt.Errorf("END PROMPT is before BEGIN  PROMPT")
			}
			prompt = source[mbeg[1]:mend[0]]
			glog.V(3).Infof("prompt = %q", prompt)
			source = strings.Join([]string{source[:mbeg[0]], source[mend[1]:]}, "")
			glog.V(3).Infof("stripped source = %q", source)
		}
		if mbeg := solutionBeginRegex.FindAllStringSubmatchIndex(source, -1); mbeg != nil {
			mend := solutionEndRegex.FindAllStringIndex(source, -1)
			if len(mbeg) != len(mend) {
				return nil, fmt.Errorf("cell has mismatched number of BEGIN SOLUTION and END SOLUTION, %d != %d", len(mbeg), len(mend))
			}
			var outputs []string
			for i, m := range mbeg {
				if i == 0 {
					outputs = append(outputs, source[0:m[0]])
				}
				// TODO(salikh): Fix indentation and add more heuristics.
				if prompt == "" {
					indent := source[m[2]:m[3]]
					prompt = indent + "..."
				}
				outputs = append(outputs, prompt)
				glog.V(3).Infof("prompt: %q", prompt)
				if i < len(mbeg)-1 {
					outputs = append(outputs, source[mend[i][1]:mbeg[i+1][0]])
				} else {
					outputs = append(outputs, source[mend[i][1]:])
					glog.V(3).Infof("last part: %q", source[mend[i][1]:])
				}
			}
			return &Cell{
				Type:     "code",
				Metadata: exerciseMetadata,
				Source:   strings.Join(outputs, ""),
			}, nil
		} else {
			glog.V(3).Infof("BEGIN SOLUTION did not match")
		}
		// Skip any test cells.
		if unittestBeginRegex.MatchString(source) ||
			autotestBeginRegex.MatchString(source) {
			return nil, nil
		}
		return cell, nil
	})
	if err != nil {
		return nil, err
	}
	return transformed, nil
}

func cloneMetadata(metadata map[string]interface{}, extras ...interface{}) map[string]interface{} {
	ret := make(map[string]interface{})
	// Copy the metadata.
	for k, v := range metadata {
		ret[k] = v
	}
	// Add the extra values.
	for i := 0; i < len(extras); i += 2 {
		ret[extras[i].(string)] = extras[i+1]
	}
	return ret
}

var (
	testClassRegex = regexp.MustCompile(`(?m)^[ \t]*class ([a-zA-Z_0-9]*)\(unittest\.TestCase\):`)
)

// ToAutograder converts a master notebook into the intermediate format called "autograder notebook".
// The autograder notebook is a format where each cell corresponds to one file,
// and the file name is stored in metadata["filename"]. It is later written into the autograder directory.
func (n *Notebook) ToAutograder() (*Notebook, error) {
	// Assignment metadata is global for the notebook.
	assignmentMetadata := make(map[string]interface{})
	var assignmentID string
	// Exercise ID is state that applies to subsequent unittest cells.
	var exerciseID string
	var exerciseMetadata map[string]interface{}
	transformed, err := n.MapCells(func(cell *Cell) (*Cell, error) {
		source := cell.Source
		if cell.Type == "markdown" {
			var err error
			if hasMetadata(assignmentMetadataRegex, cell.Source) {
				var metadata map[string]interface{}
				metadata, source, err = extractMetadata(assignmentMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
				if v, ok := metadata["assignment_id"]; ok {
					id, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("assignment_id is not a string, but %s", reflect.TypeOf(v))
					}
					assignmentID = id
				}
				// Merge assignment metadata to global table.
				for k, v := range metadata {
					assignmentMetadata[k] = v
				}
			}
			if hasMetadata(exerciseMetadataRegex, cell.Source) {
				// Replace exercise metadata.
				exerciseMetadata, source, err = extractMetadata(exerciseMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
				if v, ok := exerciseMetadata["exercise_id"]; ok {
					id, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("exercise_id is not a string, but %s", reflect.TypeOf(v))
					}
					exerciseID = id
					_ = exerciseID
				}
				glog.V(3).Infof("parsed metadata: %s", exerciseMetadata)
			}
		}
		if cell.Type != "code" {
			// We do not need to emit non-code cells.
			return nil, nil
		}
		if unittestBeginRegex.MatchString(source) {
			text, err := cutText(unittestBeginRegex, unittestEndRegex, source)
			if err != nil {
				return nil, err
			}
			filename := ""
			if m := testClassRegex.FindStringSubmatch(source); m != nil {
				basename := m[1]
				if strings.HasSuffix(basename, "Test") {
					basename = basename[:len(basename)-4]
				} else if strings.HasPrefix(basename, "Test") {
					basename = basename[4:]
				}
				filename = basename + "_test.py"
			}
			if filename == "" {
				return nil, fmt.Errorf("could not detect the test name for unittest: %s", source)
			}
			// TODO(salikh): Implement syntax tests too based on metadata.
			text = "import submission;\n" + text
			glog.V(3).Infof("metadata: %v, exercise_id: %q", exerciseMetadata, exerciseID)
			glog.V(3).Infof("parsed unit test: %s\n", text)
			return &Cell{
				Type:     "code",
				Metadata: cloneMetadata(exerciseMetadata, "filename", filename, "assignment_id", assignmentID),
				Source:   text,
			}, nil
		}
		// Do not emit other code cells.
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	transformed.Metadata = assignmentMetadata
	return transformed, nil
}