package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/importer"
)

// maxImportBytes is the import file limit (docs/06-api.md §5).
const maxImportBytes = 1 << 20

type (
	ImportCreator interface {
		Execute(ctx context.Context, filename string, file io.Reader) (domain.ImportJob, error)
	}
	ImportLister interface {
		Execute(ctx context.Context) ([]domain.ImportJob, error)
	}
	ImportGetter interface {
		Execute(ctx context.Context, id string) (domain.ImportJob, error)
	}
	ImportItemLister interface {
		Execute(ctx context.Context, jobID, status string) ([]domain.ImportItem, error)
	}
	ImportItemResolver interface {
		Execute(ctx context.Context, in importer.ResolveInput) (domain.ImportItem, error)
	}
)

// ImportUseCases groups the artist import operations (US-2, US-3).
type ImportUseCases struct {
	Create  ImportCreator
	List    ImportLister
	Get     ImportGetter
	Items   ImportItemLister
	Resolve ImportItemResolver
}

type importHandler struct {
	uc  ImportUseCases
	log *slog.Logger
}

// POST /imports (multipart, field "file") → 202: lines are matched in the background.
func (h *importHandler) create(w http.ResponseWriter, r *http.Request) {
	// Headroom for the multipart envelope; the file itself is checked below.
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes+64<<10)
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(h.log, w, r, uploadError(err))
		return
	}
	defer func() {
		_ = file.Close()
		_ = r.MultipartForm.RemoveAll()
	}()
	if header.Size > maxImportBytes {
		respondError(h.log, w, r, fileFieldError("at most 1 MB"))
		return
	}

	job, err := h.uc.Create.Execute(r.Context(), header.Filename, file)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toImportDTO(job))
}

func uploadError(err error) error {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		return fileFieldError("at most 1 MB")
	case errors.Is(err, http.ErrMissingFile):
		return fileFieldError("required")
	default:
		return fileFieldError("expected a multipart/form-data upload: " + err.Error())
	}
}

func fileFieldError(msg string) error {
	return &domain.ValidationError{Fields: []domain.FieldError{{Field: "file", Message: msg}}}
}

// GET /imports → the recent imports, newest first.
func (h *importHandler) list(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.uc.List.Execute(r.Context())
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, itemsDTO[importDTO]{Items: mapSlice(jobs, toImportDTO)})
}

// GET /imports/{id}
func (h *importHandler) get(w http.ResponseWriter, r *http.Request) {
	job, err := h.uc.Get.Execute(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toImportDTO(job))
}

// GET /imports/{id}/items?status=needs_review,not_found
func (h *importHandler) items(w http.ResponseWriter, r *http.Request) {
	items, err := h.uc.Items.Execute(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("status"))
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, itemsDTO[importItemDTO]{Items: mapSlice(items, toImportItemDTO)})
}

type resolveRequest struct {
	MBID            string `json:"mbid"`
	IncludeFeatured *bool  `json:"include_featured"`
	Skip            bool   `json:"skip"`
}

// POST /imports/{id}/items/{itemId}/resolve: {mbid, include_featured?} or {skip: true}.
func (h *importHandler) resolve(w http.ResponseWriter, r *http.Request) {
	var req resolveRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(h.log, w, r, err)
		return
	}
	item, err := h.uc.Resolve.Execute(r.Context(), importer.ResolveInput{
		JobID: chi.URLParam(r, "id"), ItemID: chi.URLParam(r, "itemId"),
		MBID: req.MBID, IncludeFeatured: req.IncludeFeatured, Skip: req.Skip,
	})
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toImportItemDTO(item))
}

type importDTO struct {
	ID       string         `json:"id"`
	Filename string         `json:"filename"`
	Total    int            `json:"total"`
	Counts   map[string]int `json:"counts"`
	// Finished: no line is pending; some may still need review.
	Finished   bool       `json:"finished"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func toImportDTO(j domain.ImportJob) importDTO {
	counts := make(map[string]int, len(domain.ImportItemStatuses))
	for _, st := range domain.ImportItemStatuses {
		counts[string(st)] = j.Counts[st]
	}
	return importDTO{
		ID: j.ID, Filename: j.Filename, Total: j.Total, Counts: counts,
		Finished: j.Finished(), CreatedAt: j.CreatedAt, FinishedAt: j.FinishedAt,
	}
}

type importCandidateDTO struct {
	artistDTO
	LifeSpan lifeSpanDTO `json:"life_span"`
	Score    int         `json:"score"`
}

type importArtistDTO struct {
	MBID string `json:"mbid"`
	Name string `json:"name"`
}

type importItemDTO struct {
	ID              string               `json:"id"`
	LineNo          int                  `json:"line_no"`
	RawValue        string               `json:"raw_value"`
	InputMBID       *string              `json:"input_mbid"`
	IncludeFeatured *bool                `json:"include_featured"`
	Status          string               `json:"status"`
	Candidates      []importCandidateDTO `json:"candidates"`
	Artist          *importArtistDTO     `json:"artist"`
	Error           *string              `json:"error"`
}

func toImportItemDTO(it domain.ImportItem) importItemDTO {
	dto := importItemDTO{
		ID: it.ID, LineNo: it.LineNo, RawValue: it.RawValue, InputMBID: nullIfEmpty(string(it.InputMBID)),
		IncludeFeatured: it.IncludeFeatured, Status: string(it.Status), Error: nullIfEmpty(it.Error),
		Candidates: make([]importCandidateDTO, len(it.Candidates)),
	}
	for i, c := range it.Candidates {
		dto.Candidates[i] = importCandidateDTO{
			artistDTO: toArtistDTO(c.Artist),
			LifeSpan: lifeSpanDTO{
				Begin: nullIfEmpty(c.LifeSpan.Begin), End: nullIfEmpty(c.LifeSpan.End), Ended: c.LifeSpan.Ended,
			},
			Score: c.Score,
		}
	}
	if it.Artist != nil {
		dto.Artist = &importArtistDTO{MBID: string(it.Artist.MBID), Name: it.Artist.Name}
	}
	return dto
}
