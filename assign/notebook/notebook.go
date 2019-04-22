// Package notebook provides utility functions for working with Jupyter/IPython
// notebooks, i.e. JSON files following some conventions.
package notebook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
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
