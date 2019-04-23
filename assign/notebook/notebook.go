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

	"gopkg.in/yaml.v2"
)

// Notebook represents a parsed Jupyter notebook.
type Notebook struct {
	NBFormat      int
	NBFormatMinor int
	Data          map[string]interface{}
	Metadata      map[string]interface{}
	Cells         []*Cell
}

// Cell represents one cell of a Jupyter notebook. It is limited in
// the kind of cells it can represent.
type Cell struct {
	Type string
	// Data is the raw contents of the parsed cell.
	// When serializing cell back to JSON, Metadata, Outputs and Source
	// take precedence of ver the contents in this map.
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

var (
	assignmentMetadataRegex = regexp.MustCompile("(?m)^# ASSIGNMENT METADATA")
	exerciseMetadataRegex   = regexp.MustCompile("(?m)^# EXERCISE METADATA")
	tripleBacktickedRegex   = regexp.MustCompile("(?ms)^```.*^```")
	solutionBeginRegex      = regexp.MustCompile("(?m)^# BEGIN SOLUTION")
	solutionEndRegex        = regexp.MustCompile("(?m)^# END SOLUTION")
	promptBeginRegex        = regexp.MustCompile("(?m)^# BEGIN PROMPT")
	promptEndRegex          = regexp.MustCompile("(?m)^# END PROMPT")
	testRegex               = regexp.MustCompile("(?m)^# TEST")
	unittestBeginRegex      = regexp.MustCompile("(?m)^# BEGIN UNITTEST")
	unittestEndRegex        = regexp.MustCompile("(?m)^# END UNITTEST")
	autotestBeginRegex      = regexp.MustCompile("(?m)^# BEGIN AUTOTEST")
	autotestEndRegex        = regexp.MustCompile("(?m)^# END AUTOTEST")
)

// ToStudent converts a master notebook into the student notebook.
func (n *Notebook) ToStudent() (*Notebook, error) {
	assignmentMetadata := make(map[string]interface{})
	exerciseMetadata := make(map[string]interface{})
	transformed, err := n.MapCells(func(cell *Cell) (*Cell, error) {
		if cell.Type == "markdown" {
			var outputs []string
			mm := tripleBacktickedRegex.FindAllStringIndex(cell.Source, -1)
			replace := false
			for i, m := range mm {
				if len(outputs) == 0 {
					outputs = append(outputs, cell.Source[0:m[0]])
				}
				text := cell.Source[m[0]+3 : m[1]-3]
				if assignmentMetadataRegex.MatchString(text) {
					fmt.Printf("%q", text)
					err := yaml.Unmarshal([]byte(text), &assignmentMetadata)
					if err != nil {
						return nil, fmt.Errorf("error parsing ASSIGNMENT METADATA: %s", err)
					}
					replace = true
				}
				if exerciseMetadataRegex.MatchString(text) {
					exerciseMetadata = make(map[string]interface{})
					fmt.Printf("%q", text)
					err := yaml.Unmarshal([]byte(text), &exerciseMetadata)
					if err != nil {
						return nil, fmt.Errorf("error parsing EXERCISE METADATA: %s", err)
					}
					replace = true
				}
				if i < len(mm)-1 {
					outputs = append(outputs, cell.Source[m[1]:mm[i+1][0]])
				} else {
					outputs = append(outputs, cell.Source[m[1]:])
				}
			}
			if replace {
				cell.Source = strings.Join(outputs, "")
			}
		}
		if cell.Type != "code" {
			return cell, nil
		}
		if mbeg := solutionBeginRegex.FindAllStringIndex(cell.Source, -1); mbeg != nil {
			mend := solutionEndRegex.FindAllStringIndex(cell.Source, -1)
			if len(mbeg) != len(mend) {
				return nil, fmt.Errorf("cell has mismatched number of BEGIN SOLUTION and END SOLUTION, %d != %d", len(mbeg), len(mend))
			}
			var outputs []string
			for i, m := range mbeg {
				if i == 0 {
					outputs = append(outputs, cell.Source[0:m[0]])
				}
				// TODO(salikh): Fix indentation and add more heuristics.
				outputs = append(outputs, "...")
				if i < len(mbeg)-1 {
					outputs = append(outputs, cell.Source[mend[i][1]:mbeg[i+1][0]])
				} else {
					outputs = append(outputs, cell.Source[mend[i][1]:])
				}
			}
			return &Cell{
				Type:     "code",
				Metadata: exerciseMetadata,
				Source:   strings.Join(outputs, ""),
			}, nil
		}
		// Skip any test cells.
		if unittestBeginRegex.MatchString(cell.Source) ||
			autotestBeginRegex.MatchString(cell.Source) {
			return nil, nil
		}
		return cell, nil
	})
	if err != nil {
		return nil, err
	}
	return transformed, nil
}