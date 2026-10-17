// Package importfile parses artist lists for import (FR-2.3): .txt with one
// name, MBID or MusicBrainz link per line, and .csv with a header row naming
// the columns name, mbid and include_featured.
package importfile

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"music-tracker/backend/internal/domain"
)

// maxValueLength matches the limit of an interactive artist search.
const maxValueLength = 200

// Parser implements usecase.ImportFileParser.
type Parser struct{}

func (Parser) Parse(filename string, r io.Reader) ([]domain.ImportLine, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read import file: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // UTF-8 BOM
	if !utf8.Valid(data) {
		return nil, fileError("must be UTF-8 text")
	}

	var lines []domain.ImportLine
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt":
		lines, err = parseText(data)
	case ".csv":
		lines, err = parseCSV(data)
	default:
		return nil, fileError("must be a .txt or .csv file")
	}
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fileError("contains no artists")
	}
	return lines, nil
}

// parseText: blank lines and lines starting with # are skipped.
func parseText(data []byte) ([]domain.ImportLine, error) {
	var lines []domain.ImportLine
	sc := bufio.NewScanner(bytes.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		v := strings.TrimSpace(sc.Text())
		if v == "" || strings.HasPrefix(v, "#") {
			continue
		}
		line, err := newLine(n, v, "", nil)
		if err != nil {
			return nil, err
		}
		if lines, err = appendLine(lines, line); err != nil {
			return nil, err
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fileError(err.Error())
	}
	return lines, nil
}

func parseCSV(data []byte) ([]domain.ImportLine, error) {
	cr := csv.NewReader(bytes.NewReader(data))
	cr.Comma = delimiterOf(data)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil
	}
	if err != nil {
		return nil, csvError(err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	_, hasName := col["name"]
	_, hasMBID := col["mbid"]
	if !hasName && !hasMBID {
		return nil, fileError(`the first CSV row must name the columns: "name" and/or "mbid", optionally "include_featured"`)
	}
	field := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}

	var lines []domain.ImportLine
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return lines, nil
		}
		if err != nil {
			return nil, csvError(err)
		}
		n, _ := cr.FieldPos(0)
		name, mbid := field(rec, "name"), field(rec, "mbid")
		if name == "" && mbid == "" {
			continue
		}
		featured, err := parseBool(field(rec, "include_featured"))
		if err != nil {
			return nil, lineError(n, "include_featured: "+err.Error())
		}
		line, err := newLine(n, name, mbid, featured)
		if err != nil {
			return nil, err
		}
		if lines, err = appendLine(lines, line); err != nil {
			return nil, err
		}
	}
}

// newLine builds a line from a name and/or an MBID (or link); a bare MBID in the
// name column also counts as one.
func newLine(n int, name, ref string, featured *bool) (domain.ImportLine, error) {
	value := name
	if ref == "" {
		ref = name
	} else if value == "" {
		value = ref
	}
	if utf8.RuneCountInString(value) > maxValueLength {
		return domain.ImportLine{}, lineError(n, fmt.Sprintf("at most %d characters", maxValueLength))
	}
	mbid, ok := domain.ParseArtistRef(ref)
	if !ok && ref != name {
		return domain.ImportLine{}, lineError(n, fmt.Sprintf("mbid %q is neither an MBID nor a MusicBrainz artist link", ref))
	}
	return domain.ImportLine{LineNo: n, Value: value, MBID: mbid, IncludeFeatured: featured}, nil
}

func appendLine(lines []domain.ImportLine, l domain.ImportLine) ([]domain.ImportLine, error) {
	if len(lines) == domain.MaxImportLines {
		return nil, fileError(fmt.Sprintf("at most %d artists per file", domain.MaxImportLines))
	}
	return append(lines, l), nil
}

// delimiterOf guesses the CSV delimiter from the header row: spreadsheet apps
// in many locales export with ";" instead of ",".
func delimiterOf(data []byte) rune {
	header, _, _ := bytes.Cut(data, []byte("\n"))
	best, count := ',', bytes.Count(header, []byte(","))
	for _, d := range []rune{';', '\t'} {
		if c := bytes.Count(header, []byte(string(d))); c > count {
			best, count = d, c
		}
	}
	return best
}

func parseBool(s string) (*bool, error) {
	var v bool
	switch strings.ToLower(s) {
	case "":
		return nil, nil
	case "true", "yes", "y", "1", "да":
		v = true
	case "false", "no", "n", "0", "нет":
		v = false
	default:
		return nil, fmt.Errorf("%q is not true/false", s)
	}
	return &v, nil
}

func csvError(err error) error {
	var pe *csv.ParseError
	if errors.As(err, &pe) {
		return lineError(pe.Line, pe.Err.Error())
	}
	return fileError(err.Error())
}

func lineError(n int, msg string) error {
	return fileError(fmt.Sprintf("line %d: %s", n, msg))
}

func fileError(msg string) error {
	return &domain.ValidationError{Fields: []domain.FieldError{{Field: "file", Message: msg}}}
}
