package sourcemap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Sourcemap struct {
	// File holds the name of the generated code that this sourcemap
	// is associated with.
	File string `json:"file"`

	// Mappings holds the mappings from generated to original source
	// locations, along with symbol and source names. Mappings will
	// be sorted by generated source location (line, then column).
	Mappings []Mapping `json:"mappings"`

	SourceContent map[string]string `json:"source_content"`
}

type Mapping struct {
	// Name holds a symbol name for the mapping.
	Name string `json:"name,omitempty"`

	// Source holds the source path.
	Source string `json:"source,omitempty"`

	// SourceLine holds the original source line.
	SourceLine int `json:"source_line"`

	// SourceColumn holds the original source column.
	SourceColumn int `json:"source_column"`

	// GenLine holds the generated source line.
	GenLine int `json:"gen_line"`

	// GenColumn holds the generated source column.
	GenColumn int `json:"gen_column"`
}

func Parse(r io.Reader) (*Sourcemap, error) {
	var sourcemap struct {
		Version        int      `json:"version"`
		File           string   `json:"file"`
		SourceRoot     string   `json:"sourceRoot"`
		Sources        []string `json:"sources"`
		SourcesContent []string `json:"sourcesContent"`
		Names          []string `json:"names"`
		Mappings       string   `json:"mappings"`
	}

	if err := json.NewDecoder(r).Decode(&sourcemap); err != nil {
		return nil, err
	}

	if sourcemap.Version != 0 && sourcemap.Version != 3 {
		// If version is unspecified, we assume version 3.
		return nil, fmt.Errorf("sourcemap version %d unsupported", sourcemap.Version)
	}

	if sourcemap.SourceRoot != "" {
		for i, source := range sourcemap.Sources {
			sourcemap.Sources[i] = sourcemap.SourceRoot + source
		}
	}

	mappings, err := parseMappings(sourcemap.Mappings, sourcemap.Names, sourcemap.Sources)
	if err != nil {
		return nil, err
	}

	sourceContent := make(map[string]string)
	for i, source := range sourcemap.Sources {
		if i >= len(sourcemap.SourcesContent) {
			break
		}
		content := sourcemap.SourcesContent[i]
		if content != "" {
			sourceContent[source] = content
		}
	}

	return &Sourcemap{
		File:          sourcemap.File,
		SourceContent: sourceContent,
		Mappings:      mappings,
	}, nil
}

func parseMappings(s string, names, sources []string) ([]Mapping, error) {
	if s == "" {
		return nil, errors.New("sourcemap: mappings are empty")
	}

	rd := strings.NewReader(s)
	m := &mappings{
		rd:      rd,
		dec:     newVLQDecoder(rd),
		names:   names,
		sources: sources,
	}
	m.value.GenLine = 1
	m.value.SourceLine = 1

	if err := m.parse(); err != nil {
		return nil, err
	}
	sort.Slice(m.values, func(i, j int) bool {
		lhs := &m.values[i]
		rhs := &m.values[j]
		if lhs.GenLine < rhs.GenLine {
			return true
		}
		return lhs.GenLine == rhs.GenLine && lhs.GenColumn < rhs.GenColumn
	})
	return m.values, nil
}

type mappings struct {
	rd      *strings.Reader
	dec     vlqDecoder
	names   []string
	sources []string

	namesIndex   int32
	sourcesIndex int32
	value        Mapping

	values []Mapping
}

func (m *mappings) parse() error {
	next := parseGenCol
	for {
		c, err := m.rd.ReadByte()
		if err == io.EOF {
			m.pushValue()
			return nil
		}
		if err != nil {
			return err
		}

		switch c {
		case ',':
			m.pushValue()
			next = parseGenCol
		case ';':
			m.pushValue()

			m.value.GenLine++
			m.value.GenColumn = 0

			next = parseGenCol
		default:
			err := m.rd.UnreadByte()
			if err != nil {
				return err
			}

			next, err = next(m)
			if err != nil {
				return err
			}
		}
	}
}

type fn func(m *mappings) (fn, error)

func parseGenCol(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.value.GenColumn += int(n)
	return parseSourcesInd, nil
}

func parseSourcesInd(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourcesIndex += n
	m.value.Source = m.sources[m.sourcesIndex]
	return parseSourceLine, nil
}

func parseSourceLine(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.value.SourceLine += int(n)
	return parseSourceCol, nil
}

func parseSourceCol(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.value.SourceColumn += int(n)
	return parseNamesInd, nil
}

func parseNamesInd(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.namesIndex += n
	m.value.Name = m.names[m.namesIndex]
	return parseGenCol, nil
}

func (m *mappings) pushValue() {
	if m.value.SourceLine == 1 && m.value.SourceColumn == 0 {
		return
	}
	m.values = append(m.values, m.value)
	m.value.Name = ""
	m.value.Source = ""
}
