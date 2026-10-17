package importfile

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"music-tracker/backend/internal/domain"
)

const nirvana = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"

func ptr(b bool) *bool { return &b }

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		want     []domain.ImportLine
	}{
		{
			name:     "text: names, MBIDs, links, comments and blank lines",
			filename: "artists.TXT",
			content: "\xef\xbb\xbf# my list\nRadiohead\n\n  Sigur Rós  \r\n" + strings.ToUpper(nirvana) +
				"\nhttps://musicbrainz.org/artist/" + nirvana + "/releases\n",
			want: []domain.ImportLine{
				{LineNo: 2, Value: "Radiohead"},
				{LineNo: 4, Value: "Sigur Rós"},
				{LineNo: 5, Value: strings.ToUpper(nirvana), MBID: nirvana},
				{LineNo: 6, Value: "https://musicbrainz.org/artist/" + nirvana + "/releases", MBID: nirvana},
			},
		},
		{
			name:     "csv: columns in any order, featuring flags, extra columns",
			filename: "artists.csv",
			content: "Include_Featured,genre,Name,MBID\n" +
				"yes,rock,Nirvana," + nirvana + "\n" +
				",electronic,Björk,\n" +
				"0,,,https://musicbrainz.org/artist/" + nirvana + "\n" +
				",,,\n" +
				"нет,pop,\"Crosby, Stills & Nash\"\n",
			want: []domain.ImportLine{
				{LineNo: 2, Value: "Nirvana", MBID: nirvana, IncludeFeatured: ptr(true)},
				{LineNo: 3, Value: "Björk"},
				{LineNo: 4, Value: "https://musicbrainz.org/artist/" + nirvana, MBID: nirvana, IncludeFeatured: ptr(false)},
				{LineNo: 6, Value: "Crosby, Stills & Nash", IncludeFeatured: ptr(false)},
			},
		},
		{
			name:     "csv: semicolon delimiter from a spreadsheet",
			filename: "export.csv",
			content:  "name;include_featured\r\nAC/DC;true\r\nПесняры;\r\n",
			want: []domain.ImportLine{
				{LineNo: 2, Value: "AC/DC", IncludeFeatured: ptr(true)},
				{LineNo: 3, Value: "Песняры"},
			},
		},
		{
			name:     "csv: only the mbid column",
			filename: "ids.csv",
			content:  "mbid\n" + nirvana + "\n",
			want:     []domain.ImportLine{{LineNo: 2, Value: nirvana, MBID: nirvana}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parser{}.Parse(tt.filename, strings.NewReader(tt.content))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		wantMsg  string
	}{
		{"unsupported extension", "artists.xlsx", "Radiohead", "must be a .txt or .csv file"},
		{"not UTF-8", "artists.txt", "\xcf\xe5\xf1\xed\xff\xf0\xfb", "must be UTF-8 text"},
		{"empty text", "artists.txt", "\n# nothing\n  \n", "contains no artists"},
		{"csv without header columns", "artists.csv", "artist,year\nRadiohead,1985\n", `"name" and/or "mbid"`},
		{"csv with header only", "artists.csv", "name\n", "contains no artists"},
		{"bad mbid", "artists.csv", "name,mbid\nNirvana,not-an-id\n", "line 2: mbid"},
		{"bad flag", "artists.csv", "name,include_featured\nNirvana,maybe\n", "line 2: include_featured"},
		{"broken quotes", "artists.csv", "name\n\"Nirvana\n", "line 2"},
		{"name too long", "artists.txt", strings.Repeat("a", 201), "line 1: at most 200 characters"},
		{"too many lines", "artists.txt", strings.Repeat("x\n", domain.MaxImportLines+1), fmt.Sprintf("at most %d artists", domain.MaxImportLines)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parser{}.Parse(tt.filename, strings.NewReader(tt.content))
			var verr *domain.ValidationError
			if !errors.As(err, &verr) || verr.Fields[0].Field != "file" {
				t.Fatalf("err = %v, want a validation error of the file field", err)
			}
			if !strings.Contains(verr.Fields[0].Message, tt.wantMsg) {
				t.Errorf("message = %q, want it to contain %q", verr.Fields[0].Message, tt.wantMsg)
			}
		})
	}
}
