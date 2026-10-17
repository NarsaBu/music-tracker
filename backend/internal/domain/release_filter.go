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

// Matches decides whether a release group passes the filter (docs/08-notifications.md §1):
// featured releases only with IncludeFeatured; the primary type must be allowed (a
// release group without one passes only when "other" is); no excluded secondary type.
func (f ReleaseFilter) Matches(rg ReleaseGroup, role CreditRole) bool {
	if role == RoleFeatured && !f.IncludeFeatured {
		return false
	}
	pt := rg.PrimaryType
	if pt == "" {
		pt = PrimaryOther
	}
	if !slices.Contains(f.PrimaryTypes, pt) {
		return false
	}
	for _, t := range rg.SecondaryTypes {
		if slices.Contains(f.ExcludedSecondaryTypes, t) {
			return false
		}
	}
	return true
}

// Validate reports unknown types in a filter, with field names under prefix.
func (f ReleaseFilter) Validate(prefix string) []FieldError {
	return FilterOverride{PrimaryTypes: f.PrimaryTypes, ExcludedSecondaryTypes: f.ExcludedSecondaryTypes}.fieldErrors(prefix)
}

// FilterOverride is a per-subscription override; nil fields fall back to the global filter.
type FilterOverride struct {
	PrimaryTypes           []PrimaryType
	ExcludedSecondaryTypes []string
	IncludeFeatured        *bool
}

// Validate reports unknown types. Field names match the API (docs/06-api.md).
func (o FilterOverride) Validate() error {
	if fields := o.fieldErrors(""); len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func (o FilterOverride) fieldErrors(prefix string) []FieldError {
	var fields []FieldError
	for i, t := range o.PrimaryTypes {
		if !slices.Contains(PrimaryTypes, t) {
			fields = append(fields, FieldError{
				Field: fmt.Sprintf("%sprimary_types[%d]", prefix, i), Message: fmt.Sprintf("unknown value %q", t),
			})
		}
	}
	for i, t := range o.ExcludedSecondaryTypes {
		if !slices.Contains(SecondaryTypes, t) {
			fields = append(fields, FieldError{
				Field: fmt.Sprintf("%sexcluded_secondary_types[%d]", prefix, i), Message: fmt.Sprintf("unknown value %q", t),
			})
		}
	}
	return fields
}

// Effective flattens the resolved filter to the filter that is actually applied.
func (r ResolvedFilter) Effective() ReleaseFilter {
	return ReleaseFilter{
		PrimaryTypes:           r.PrimaryTypes.Value,
		ExcludedSecondaryTypes: r.ExcludedSecondaryTypes.Value,
		IncludeFeatured:        r.IncludeFeatured.Value,
	}
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
