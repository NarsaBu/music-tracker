package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/importer"
)

type (
	importCreateFunc  func(context.Context, string, io.Reader) (domain.ImportJob, error)
	importListFunc    func(context.Context) ([]domain.ImportJob, error)
	importGetFunc     func(context.Context, string) (domain.ImportJob, error)
	importItemsFunc   func(context.Context, string, string) ([]domain.ImportItem, error)
	importResolveFunc func(context.Context, importer.ResolveInput) (domain.ImportItem, error)
)

func (f importCreateFunc) Execute(ctx context.Context, n string, r io.Reader) (domain.ImportJob, error) {
	return f(ctx, n, r)
}
func (f importListFunc) Execute(ctx context.Context) ([]domain.ImportJob, error) { return f(ctx) }
func (f importGetFunc) Execute(ctx context.Context, id string) (domain.ImportJob, error) {
	return f(ctx, id)
}
func (f importItemsFunc) Execute(ctx context.Context, id, st string) ([]domain.ImportItem, error) {
	return f(ctx, id, st)
}
func (f importResolveFunc) Execute(ctx context.Context, in importer.ResolveInput) (domain.ImportItem, error) {
	return f(ctx, in)
}

var sampleJob = domain.ImportJob{
	ID: "job-1", Filename: "artists.csv", Total: 3, CreatedAt: started,
	Counts: map[domain.ImportItemStatus]int{domain.ImportPending: 1, domain.ImportNeedsReview: 2},
}

func newImportRouter(uploaded *string, resolved *importer.ResolveInput) http.Handler {
	job := func(id string) (domain.ImportJob, error) {
		if id != sampleJob.ID {
			return domain.ImportJob{}, domain.ErrNotFound
		}
		return sampleJob, nil
	}
	return NewRouter(Deps{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Readiness: fakePinger{},
		Imports: ImportUseCases{
			Create: importCreateFunc(func(_ context.Context, name string, r io.Reader) (domain.ImportJob, error) {
				b, _ := io.ReadAll(r)
				*uploaded = name + ":" + string(b)
				return sampleJob, nil
			}),
			List: importListFunc(func(context.Context) ([]domain.ImportJob, error) {
				return []domain.ImportJob{sampleJob}, nil
			}),
			Get: importGetFunc(func(_ context.Context, id string) (domain.ImportJob, error) { return job(id) }),
			Items: importItemsFunc(func(_ context.Context, id, status string) ([]domain.ImportItem, error) {
				if _, err := job(id); err != nil {
					return nil, err
				}
				return []domain.ImportItem{{
					ID: "item-1", LineNo: 2, RawValue: "Nirvana", Status: domain.ImportItemStatus(status),
					Candidates: []domain.ArtistCandidate{{
						Artist: domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana", Country: "US",
							LifeSpan: domain.LifeSpan{Begin: "1987", End: "1994", Ended: true}},
						Score: 100,
					}},
				}}, nil
			}),
			Resolve: importResolveFunc(func(_ context.Context, in importer.ResolveInput) (domain.ImportItem, error) {
				*resolved = in
				if in.ItemID == "done" {
					return domain.ImportItem{}, domain.ErrConflict
				}
				return domain.ImportItem{
					ID: in.ItemID, LineNo: 2, RawValue: "Nirvana", Status: domain.ImportMatched,
					Artist: &domain.Artist{MBID: nirvana, Name: "Nirvana"},
				}, nil
			}),
		},
	})
}

func multipartBody(t *testing.T, field, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte(content))
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestCreateImport(t *testing.T) {
	var uploaded string
	var resolved importer.ResolveInput
	h := newImportRouter(&uploaded, &resolved)

	body, ct := multipartBody(t, "file", "artists.csv", "name\nNirvana\n")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if uploaded != "artists.csv:name\nNirvana\n" {
		t.Errorf("uploaded = %q", uploaded)
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	counts, _ := got["counts"].(map[string]any)
	if got["id"] != "job-1" || got["finished"] != false || len(counts) != 6 || counts["needs_review"] != 2.0 || counts["matched"] != 0.0 {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestCreateImport_BadUploads(t *testing.T) {
	var uploaded string
	var resolved importer.ResolveInput
	h := newImportRouter(&uploaded, &resolved)

	tests := []struct {
		name    string
		field   string
		content string
		want    string
	}{
		{"missing file", "other", "x", "required"},
		{"too large", "file", strings.Repeat("a", maxImportBytes+1), "at most 1 MB"},
	}
	for _, tt := range tests {
		body, ct := multipartBody(t, tt.field, "a.txt", tt.content)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/imports", body)
		req.Header.Set("Content-Type", ct)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), tt.want) {
			t.Errorf("%s: status = %d, body %s", tt.name, rec.Code, rec.Body)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports", strings.NewReader(`{"file":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("json body: status = %d", rec.Code)
	}
	if uploaded != "" {
		t.Error("a bad upload reached the use case")
	}
}

func TestImportReads(t *testing.T) {
	var uploaded string
	var resolved importer.ResolveInput
	h := newImportRouter(&uploaded, &resolved)

	tests := []struct {
		path   string
		status int
		want   string
	}{
		{"/api/v1/imports", http.StatusOK, `"filename":"artists.csv"`},
		{"/api/v1/imports/job-1", http.StatusOK, `"total":3`},
		{"/api/v1/imports/nope", http.StatusNotFound, "Not found"},
		{"/api/v1/imports/job-1/items?status=needs_review", http.StatusOK,
			`"status":"needs_review","candidates":[{"mbid":"` + nirvana + `","name":"Nirvana","sort_name":"Nirvana","disambiguation":"","type":null,"country":"US","life_span":{"begin":"1987","end":"1994","ended":true},"score":100}],"artist":null,"error":null`},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.want) {
			t.Errorf("GET %s: status = %d, body %s", tt.path, rec.Code, rec.Body)
		}
	}
}

func TestResolveImportItem(t *testing.T) {
	var uploaded string
	var resolved importer.ResolveInput
	h := newImportRouter(&uploaded, &resolved)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/imports/job-1/items/item-1/resolve",
		strings.NewReader(`{"mbid":"`+nirvana+`","include_featured":true}`)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"artist":{"mbid":"`+nirvana+`","name":"Nirvana"}`) {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if resolved.JobID != "job-1" || resolved.ItemID != "item-1" || resolved.MBID != nirvana ||
		resolved.IncludeFeatured == nil || !*resolved.IncludeFeatured || resolved.Skip {
		t.Errorf("input = %+v", resolved)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/imports/job-1/items/done/resolve",
		strings.NewReader(`{"skip":true}`)))
	if rec.Code != http.StatusConflict || !resolved.Skip {
		t.Errorf("resolved twice: status = %d, input %+v", rec.Code, resolved)
	}
}
