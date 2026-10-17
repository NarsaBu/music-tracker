// Package importer holds the artist import use cases (FR-2.3, US-2, US-3):
// CreateImport, ProcessItem, ResolveItem and the read models.
package importer

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// Subscriber subscribes to an artist card at hand (artist.Subscribe).
type Subscriber interface {
	SubscribeArtist(ctx context.Context, a domain.Artist, o domain.FilterOverride) (created bool, err error)
}

// CreateImport stores an uploaded file as a job with one pending item per line
// and queues the items; matching runs in the background.
type CreateImport struct {
	parser  usecase.ImportFileParser
	imports usecase.ImportRepository
	queue   usecase.JobQueue
	tx      usecase.TxManager
	clock   usecase.Clock
}

func NewCreateImport(
	parser usecase.ImportFileParser,
	imports usecase.ImportRepository,
	queue usecase.JobQueue,
	tx usecase.TxManager,
	clock usecase.Clock,
) *CreateImport {
	return &CreateImport{parser: parser, imports: imports, queue: queue, tx: tx, clock: clock}
}

func (uc *CreateImport) Execute(ctx context.Context, filename string, file io.Reader) (domain.ImportJob, error) {
	// Browsers send a bare name, but some clients send a path (on Windows, with backslashes).
	filename = strings.TrimSpace(path.Base(strings.ReplaceAll(filename, `\`, "/")))
	if filename == "" || filename == "." || filename == "/" {
		return domain.ImportJob{}, &domain.ValidationError{Fields: []domain.FieldError{{Field: "file", Message: "file name required"}}}
	}
	lines, err := uc.parser.Parse(filename, file)
	if err != nil {
		return domain.ImportJob{}, err
	}

	now := uc.clock.Now()
	var job domain.ImportJob
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		var items []domain.ImportItem
		job, items, err = uc.imports.Create(ctx, filename, lines, now)
		if err != nil {
			return err
		}
		jobs := make([]usecase.Job, len(items))
		for i, it := range items {
			jobs[i] = usecase.NewImportItemJob(it.ID, now)
		}
		return uc.queue.EnqueueMany(ctx, jobs)
	})
	if err != nil {
		return domain.ImportJob{}, fmt.Errorf("create import %q: %w", filename, err)
	}
	return job, nil
}
