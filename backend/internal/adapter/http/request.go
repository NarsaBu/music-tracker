package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"music-tracker/backend/internal/domain"
)

const maxBodyBytes = 1 << 20

// decodeJSON reads a JSON body strictly: unknown fields and trailing data are rejected.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return bodyError(err)
	}
	if dec.More() {
		return bodyError(errors.New("unexpected data after the JSON object"))
	}
	return nil
}

func bodyError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return &domain.ValidationError{Fields: []domain.FieldError{
			{Field: typeErr.Field, Message: "must be " + typeErr.Type.String()},
		}}
	}
	return &domain.ValidationError{Fields: []domain.FieldError{{Field: "body", Message: err.Error()}}}
}

// queryInt parses an optional integer query parameter; absent means 0.
func queryInt(r *http.Request, name string) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &domain.ValidationError{Fields: []domain.FieldError{
			{Field: name, Message: fmt.Sprintf("must be an integer, got %q", raw)},
		}}
	}
	return v, nil
}
