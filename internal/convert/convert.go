// Package convert renders the JSON a model returns into a chosen output
// format. The model always emits JSON; everything else is produced here with
// standard data libraries.
package convert

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/clbanning/mxj/v2"
	"gopkg.in/yaml.v3"
)

// Result is a converted payload plus the metadata needed to display or download
// it.
type Result struct {
	Data        []byte
	ContentType string
	Ext         string
}

// Formats lists the supported output formats, in UI display order.
func Formats() []string { return []string{"json", "yaml", "xml", "csv"} }

// Convert transforms model JSON into the requested format.
func Convert(jsonData []byte, format string) (Result, error) {
	switch format {
	case "json":
		var buf bytes.Buffer
		if err := json.Indent(&buf, jsonData, "", "  "); err != nil {
			return Result{}, fmt.Errorf("invalid JSON from model: %w", err)
		}
		return Result{Data: buf.Bytes(), ContentType: "application/json", Ext: "json"}, nil
	case "yaml":
		return toYAML(jsonData)
	case "xml":
		return toXML(jsonData)
	case "csv":
		return toCSV(jsonData)
	default:
		return Result{}, fmt.Errorf("unsupported format: %q", format)
	}
}

// toYAML round-trips the JSON through a yaml.Node. Because JSON is a subset of
// YAML, this preserves key order and number types without a lossy float decode.
func toYAML(jsonData []byte) (Result, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(jsonData, &node); err != nil {
		return Result{}, fmt.Errorf("parse JSON for YAML: %w", err)
	}
	// JSON parses as flow-style nodes; clear styles to emit readable block YAML.
	clearStyle(&node)
	out, err := yaml.Marshal(&node)
	if err != nil {
		return Result{}, fmt.Errorf("encode YAML: %w", err)
	}
	return Result{Data: out, ContentType: "application/x-yaml", Ext: "yaml"}, nil
}

func clearStyle(n *yaml.Node) {
	n.Style = 0
	for _, c := range n.Content {
		clearStyle(c)
	}
}

// toXML uses mxj to encode arbitrary JSON. UseNumber keeps integers exact
// (avoids float64 scientific notation). A list becomes <items><item>...</item>.
func toXML(jsonData []byte) (Result, error) {
	dec := json.NewDecoder(bytes.NewReader(jsonData))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return Result{}, fmt.Errorf("parse JSON for XML: %w", err)
	}
	body, err := mxj.AnyXmlIndent(v, "", "  ", "items", "item")
	if err != nil {
		return Result{}, fmt.Errorf("encode XML: %w", err)
	}
	out := append([]byte(xml.Header), body...)
	return Result{Data: out, ContentType: "application/xml", Ext: "xml"}, nil
}

// toCSV flattens a list of objects to rows. Columns are the union of object
// keys in first-appearance order; nested values are compact-JSON encoded.
func toCSV(jsonData []byte) (Result, error) {
	v, err := decodeOrdered(jsonData)
	if err != nil {
		return Result{}, fmt.Errorf("parse JSON for CSV: %w", err)
	}

	var rows []any
	switch t := v.(type) {
	case []any:
		rows = t
	case *omap:
		rows = []any{t}
	default:
		return Result{}, fmt.Errorf("CSV requires a list or object, got a scalar value")
	}

	var cols []string
	seen := map[string]bool{}
	for _, r := range rows {
		o, ok := r.(*omap)
		if !ok {
			continue
		}
		for _, k := range o.keys {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	if len(cols) == 0 {
		return Result{}, fmt.Errorf("CSV requires a list of objects with fields")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(cols); err != nil {
		return Result{}, err
	}
	for _, r := range rows {
		o, _ := r.(*omap)
		rec := make([]string, len(cols))
		if o != nil {
			for i, c := range cols {
				if val, ok := o.m[c]; ok {
					rec[i] = cell(val)
				}
			}
		}
		if err := w.Write(rec); err != nil {
			return Result{}, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return Result{}, err
	}
	return Result{Data: buf.Bytes(), ContentType: "text/csv", Ext: "csv"}, nil
}

func cell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	default: // nested object or array
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}
