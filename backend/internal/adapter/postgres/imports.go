package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
)

// ImportRepository implements usecase.ImportRepository.
type ImportRepository struct {
	pool *pgxpool.Pool
}

func NewImportRepository(pool *pgxpool.Pool) *ImportRepository { return &ImportRepository{pool: pool} }

func (r *ImportRepository) Create(ctx context.Context, filename string, lines []domain.ImportLine, at time.Time) (domain.ImportJob, []domain.ImportItem, error) {
	qs := queries(ctx, r.pool)
	row, err := qs.CreateImportJob(ctx, sqlc.CreateImportJobParams{
		Filename: filename, Total: int32(len(lines)), CreatedAt: at, //nolint:gosec // bounded by domain.MaxImportLines
	})
	if err != nil {
		return domain.ImportJob{}, nil, fmt.Errorf("create import job: %w", err)
	}

	params := make([]sqlc.InsertImportItemParams, len(lines))
	for i, l := range lines {
		mbid, err := nullMBID(l.MBID)
		if err != nil {
			return domain.ImportJob{}, nil, err
		}
		params[i] = sqlc.InsertImportItemParams{
			JobID: row.ID, LineNo: int32(l.LineNo), RawValue: l.Value, //nolint:gosec // line numbers of a 1 MB file
			InputMbid: mbid, IncludeFeatured: optionalBool(l.IncludeFeatured),
		}
	}
	job := importJobFromRow(row, nil)
	items := make([]domain.ImportItem, len(lines))
	var errs []error
	b := qs.InsertImportItem(ctx, params)
	b.QueryRow(func(i int, id uuid.UUID, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("insert import item %d: %w", i, err))
			return
		}
		l := lines[i]
		items[i] = domain.ImportItem{
			ID: id.String(), JobID: job.ID, LineNo: l.LineNo, RawValue: l.Value,
			InputMBID: l.MBID, IncludeFeatured: l.IncludeFeatured, Status: domain.ImportPending,
		}
	})
	if err := errors.Join(append(errs, b.Close())...); err != nil {
		return domain.ImportJob{}, nil, err
	}
	job.Counts = map[domain.ImportItemStatus]int{domain.ImportPending: len(lines)}
	return job, items, nil
}

func (r *ImportRepository) Get(ctx context.Context, id string) (domain.ImportJob, error) {
	jobID, err := parseID("import", id)
	if err != nil {
		return domain.ImportJob{}, err
	}
	qs := queries(ctx, r.pool)
	row, err := qs.GetImportJob(ctx, jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ImportJob{}, fmt.Errorf("import %s: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return domain.ImportJob{}, fmt.Errorf("get import %s: %w", id, err)
	}
	jobs, err := r.withCounts(ctx, []sqlc.ImportJob{row})
	if err != nil {
		return domain.ImportJob{}, err
	}
	return jobs[0], nil
}

func (r *ImportRepository) List(ctx context.Context, limit int) ([]domain.ImportJob, error) {
	rows, err := queries(ctx, r.pool).ListImportJobs(ctx, int32(limit)) //nolint:gosec // bounded by the use case
	if err != nil {
		return nil, fmt.Errorf("list imports: %w", err)
	}
	return r.withCounts(ctx, rows)
}

func (r *ImportRepository) withCounts(ctx context.Context, rows []sqlc.ImportJob) ([]domain.ImportJob, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	counts, err := queries(ctx, r.pool).CountImportItems(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("count import items: %w", err)
	}
	byJob := map[uuid.UUID]map[domain.ImportItemStatus]int{}
	for _, c := range counts {
		if byJob[c.JobID] == nil {
			byJob[c.JobID] = map[domain.ImportItemStatus]int{}
		}
		byJob[c.JobID][domain.ImportItemStatus(c.Status)] = int(c.N)
	}
	out := make([]domain.ImportJob, len(rows))
	for i, row := range rows {
		out[i] = importJobFromRow(row, byJob[row.ID])
	}
	return out, nil
}

func (r *ImportRepository) GetItem(ctx context.Context, jobID, itemID string) (domain.ImportItem, error) {
	id, err := parseID("import item", itemID)
	if err != nil {
		return domain.ImportItem{}, err
	}
	var job uuid.NullUUID
	if jobID != "" {
		if job.UUID, err = parseID("import", jobID); err != nil {
			return domain.ImportItem{}, err
		}
		job.Valid = true
	}
	row, err := queries(ctx, r.pool).GetImportItem(ctx, sqlc.GetImportItemParams{ID: id, JobID: job})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ImportItem{}, fmt.Errorf("import item %s: %w", itemID, domain.ErrNotFound)
	}
	if err != nil {
		return domain.ImportItem{}, fmt.Errorf("get import item %s: %w", itemID, err)
	}
	return importItemFromRow(row.ImportItem, row.ArtistName)
}

func (r *ImportRepository) ListItems(ctx context.Context, jobID string, statuses []domain.ImportItemStatus) ([]domain.ImportItem, error) {
	id, err := parseID("import", jobID)
	if err != nil {
		return nil, err
	}
	sts := make([]sqlc.ImportItemStatus, len(statuses))
	for i, st := range statuses {
		sts[i] = sqlc.ImportItemStatus(st)
	}
	rows, err := queries(ctx, r.pool).ListImportItems(ctx, sqlc.ListImportItemsParams{JobID: id, Statuses: sts})
	if err != nil {
		return nil, fmt.Errorf("list import items of %s: %w", jobID, err)
	}
	out := make([]domain.ImportItem, len(rows))
	for i, row := range rows {
		if out[i], err = importItemFromRow(row.ImportItem, row.ArtistName); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *ImportRepository) UpdateItem(ctx context.Context, item domain.ImportItem) error {
	id, err := parseID("import item", item.ID)
	if err != nil {
		return err
	}
	var candidates []byte
	if item.Candidates != nil {
		if candidates, err = json.Marshal(toCandidatesJSON(item.Candidates)); err != nil {
			return fmt.Errorf("marshal candidates: %w", err)
		}
	}
	var artist uuid.NullUUID
	if item.Artist != nil {
		if artist, err = nullMBID(item.Artist.MBID); err != nil {
			return err
		}
	}
	err = queries(ctx, r.pool).UpdateImportItem(ctx, sqlc.UpdateImportItemParams{
		ID: id, Status: sqlc.ImportItemStatus(item.Status), Candidates: candidates,
		ArtistMbid: artist, Error: optionalText(item.Error),
	})
	if err != nil {
		return fmt.Errorf("update import item %s: %w", item.ID, err)
	}
	return nil
}

func (r *ImportRepository) FinishIfDone(ctx context.Context, jobID string, at time.Time) error {
	id, err := parseID("import", jobID)
	if err != nil {
		return err
	}
	if err := queries(ctx, r.pool).FinishImportJobIfDone(ctx, sqlc.FinishImportJobIfDoneParams{ID: id, Now: &at}); err != nil {
		return fmt.Errorf("finish import %s: %w", jobID, err)
	}
	return nil
}

// candidateJSON is the stored form of a search candidate: the whole artist card,
// so that a chosen candidate is subscribed without another catalog request.
type candidateJSON struct {
	MBID           string `json:"mbid"`
	Name           string `json:"name"`
	SortName       string `json:"sort_name"`
	Disambiguation string `json:"disambiguation"`
	Type           string `json:"type,omitempty"`
	Country        string `json:"country,omitempty"`
	LifeSpan       struct {
		Begin string `json:"begin,omitempty"`
		End   string `json:"end,omitempty"`
		Ended bool   `json:"ended"`
	} `json:"life_span"`
	Score int `json:"score"`
}

func toCandidatesJSON(cs []domain.ArtistCandidate) []candidateJSON {
	out := make([]candidateJSON, len(cs))
	for i, c := range cs {
		out[i] = candidateJSON{
			MBID: string(c.MBID), Name: c.Name, SortName: c.SortName, Disambiguation: c.Disambiguation,
			Type: c.Type, Country: c.Country, Score: c.Score,
		}
		out[i].LifeSpan.Begin, out[i].LifeSpan.End, out[i].LifeSpan.Ended = c.LifeSpan.Begin, c.LifeSpan.End, c.LifeSpan.Ended
	}
	return out
}

func importJobFromRow(row sqlc.ImportJob, counts map[domain.ImportItemStatus]int) domain.ImportJob {
	if counts == nil {
		counts = map[domain.ImportItemStatus]int{}
	}
	return domain.ImportJob{
		ID: row.ID.String(), Filename: row.Filename, Total: int(row.Total), Counts: counts,
		CreatedAt: row.CreatedAt.UTC(), FinishedAt: utcPtr(row.FinishedAt),
	}
}

func importItemFromRow(row sqlc.ImportItem, artistName pgtype.Text) (domain.ImportItem, error) {
	item := domain.ImportItem{
		ID: row.ID.String(), JobID: row.JobID.String(), LineNo: int(row.LineNo), RawValue: row.RawValue,
		IncludeFeatured: nullableBool(row.IncludeFeatured), Status: domain.ImportItemStatus(row.Status),
		Error: row.Error.String,
	}
	if row.InputMbid.Valid {
		item.InputMBID = domain.MBID(row.InputMbid.UUID.String())
	}
	if row.ArtistMbid.Valid {
		item.Artist = &domain.Artist{MBID: domain.MBID(row.ArtistMbid.UUID.String()), Name: artistName.String}
	}
	if row.Candidates != nil {
		var cs []candidateJSON
		if err := json.Unmarshal(row.Candidates, &cs); err != nil {
			return domain.ImportItem{}, fmt.Errorf("import item %s: decode candidates: %w", item.ID, err)
		}
		item.Candidates = make([]domain.ArtistCandidate, len(cs))
		for i, c := range cs {
			item.Candidates[i] = domain.ArtistCandidate{
				Artist: domain.Artist{
					MBID: domain.MBID(c.MBID), Name: c.Name, SortName: c.SortName, Disambiguation: c.Disambiguation,
					Type: c.Type, Country: c.Country,
					LifeSpan: domain.LifeSpan{Begin: c.LifeSpan.Begin, End: c.LifeSpan.End, Ended: c.LifeSpan.Ended},
				},
				Score: c.Score,
			}
		}
	}
	return item, nil
}

// parseID maps a malformed ID to domain.ErrNotFound: such a resource cannot exist.
func parseID(what, id string) (uuid.UUID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s %q: %w", what, id, domain.ErrNotFound)
	}
	return u, nil
}

func nullMBID(mbid domain.MBID) (uuid.NullUUID, error) {
	if mbid == "" {
		return uuid.NullUUID{}, nil
	}
	id, err := toUUID(mbid)
	return uuid.NullUUID{UUID: id, Valid: true}, err
}
