package domain

import (
	"fmt"
	"slices"
)

// ReleaseFilter decides which release groups produce notifications.
// The global one lives in settings; a subscription may override any field.
type ReleaseFilter struct {
	PrimaryTypes           []PrimaryType
	ExcludedSecondaryTypes []string
	IncludeFeatured        bool
}

// FilterOverride is a per-subscription override; nil fields fall back to the global filter.
type FilterOverride struct {
	PrimaryTypes           []PrimaryType
	ExcludedSecondaryTypes []string
	IncludeFeatured        *bool
}

// Validate reports unknown types. Field names match the API (docs/06-api.md).
func (o FilterOverride) Validate() error {
	var fields []FieldError
	for i, t := range o.PrimaryTypes {
		if !slices.Contains(PrimaryTypes, t) {
			fields = append(fields, FieldError{
				Field: fmt.Sprintf("primary_types[%d]", i), Message: fmt.Sprintf("unknown value %q", t),
			})
		}
	}
	for i, t := range o.ExcludedSecondaryTypes {
		if !slices.Contains(SecondaryTypes, t) {
			fields = append(fields, FieldError{
				Field: fmt.Sprintf("excluded_secondary_types[%d]", i), Message: fmt.Sprintf("unknown value %q", t),
			})
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// Overridable is a filter field value together with where it came from.
type Overridable[T any] struct {
	Value      T
	Overridden bool
}

// ResolvedFilter is the effective filter of a subscription, field by field.
type ResolvedFilter struct {
	PrimaryTypes           Overridable[[]PrimaryType]
	ExcludedSecondaryTypes Overridable[[]string]
	IncludeFeatured        Overridable[bool]
}

// Resolve applies the override on top of the global defaults.
func (o FilterOverride) Resolve(defaults ReleaseFilter) ResolvedFilter {
	r := ResolvedFilter{
		PrimaryTypes:           Overridable[[]PrimaryType]{Value: defaults.PrimaryTypes},
		ExcludedSecondaryTypes: Overridable[[]string]{Value: defaults.ExcludedSecondaryTypes},
		IncludeFeatured:        Overridable[bool]{Value: defaults.IncludeFeatured},
	}
	if o.PrimaryTypes != nil {
		r.PrimaryTypes = Overridable[[]PrimaryType]{Value: o.PrimaryTypes, Overridden: true}
	}
	if o.ExcludedSecondaryTypes != nil {
		r.ExcludedSecondaryTypes = Overridable[[]string]{Value: o.ExcludedSecondaryTypes, Overridden: true}
	}
	if o.IncludeFeatured != nil {
		r.IncludeFeatured = Overridable[bool]{Value: *o.IncludeFeatured, Overridden: true}
	}
	return r
}
