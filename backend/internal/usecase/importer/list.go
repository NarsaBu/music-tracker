package importer

import (
	"context"
	"fmt"
	"strings"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// listLimit: the screen shows the recent imports only.
const listLimit = 20

type ListImports struct{ imports usecase.ImportRepository }

func NewListImports(imports usecase.ImportRepository) *ListImports {
	return &ListImports{imports: imports}
}

func (uc *ListImports) Execute(ctx context.Context) ([]domain.ImportJob, error) {
	jobs, err := uc.imports.List(ctx, listLimit)
	if err != nil {
		return nil, fmt.Errorf("list imports: %w", err)
	}
	return jobs, nil
}

type GetImport struct{ imports usecase.ImportRepository }

func NewGetImport(imports usecase.ImportRepository) *GetImport {
	return &GetImport{imports: imports}
}

func (uc *GetImport) Execute(ctx context.Context, id string) (domain.ImportJob, error) {
	job, err := uc.imports.Get(ctx, id)
	if err != nil {
		return domain.ImportJob{}, fmt.Errorf("import %s: %w", id, err)
	}
	return job, nil
}

type ListItems struct{ imports usecase.ImportRepository }

func NewListItems(imports usecase.ImportRepository) *ListItems {
	return &ListItems{imports: imports}
}

// Execute lists the job's lines; status is a comma-separated list, "" = all.
func (uc *ListItems) Execute(ctx context.Context, jobID, status string) ([]domain.ImportItem, error) {
	var sts []domain.ImportItemStatus
	for raw := range strings.SplitSeq(status, ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		st, err := domain.ParseImportItemStatus(raw)
		if err != nil {
			return nil, err
		}
		sts = append(sts, st)
	}
	if _, err := uc.imports.Get(ctx, jobID); err != nil {
		return nil, fmt.Errorf("import %s: %w", jobID, err)
	}
	items, err := uc.imports.ListItems(ctx, jobID, sts)
	if err != nil {
		return nil, fmt.Errorf("list import items of %s: %w", jobID, err)
	}
	return items, nil
}
