package domain

import (
	"errors"
	"slices"
	"testing"
)

func TestFilterOverride_Validate(t *testing.T) {
	ok := FilterOverride{PrimaryTypes: []PrimaryType{PrimaryAlbum}, ExcludedSecondaryTypes: []string{"Live"}}
	if err := ok.Validate(); err != nil {
		t.Errorf("valid override: unexpected error %v", err)
	}

	bad := FilterOverride{PrimaryTypes: []PrimaryType{PrimaryAlbum, "lp"}, ExcludedSecondaryTypes: []string{"live"}}
	err := bad.Validate()
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("invalid override: error = %v, want *ValidationError", err)
	}
	gotFields := []string{verr.Fields[0].Field, verr.Fields[1].Field}
	if !slices.Equal(gotFields, []string{"primary_types[1]", "excluded_secondary_types[0]"}) {
		t.Errorf("fields = %v", gotFields)
	}
}

func TestFilterOverride_Resolve(t *testing.T) {
	defaults := ReleaseFilter{
		PrimaryTypes:           []PrimaryType{PrimaryAlbum, PrimaryEP},
		ExcludedSecondaryTypes: []string{"Compilation"},
		IncludeFeatured:        false,
	}

	t.Run("empty override keeps defaults", func(t *testing.T) {
		r := FilterOverride{}.Resolve(defaults)
		if r.PrimaryTypes.Overridden || r.ExcludedSecondaryTypes.Overridden || r.IncludeFeatured.Overridden {
			t.Errorf("nothing should be overridden: %+v", r)
		}
		if !slices.Equal(r.PrimaryTypes.Value, defaults.PrimaryTypes) || r.IncludeFeatured.Value {
			t.Errorf("values should come from defaults: %+v", r)
		}
	})

	t.Run("set fields win, including empty lists and false", func(t *testing.T) {
		on := true
		r := FilterOverride{ExcludedSecondaryTypes: []string{}, IncludeFeatured: &on}.Resolve(defaults)
		if r.PrimaryTypes.Overridden {
			t.Error("primary types must not be overridden")
		}
		if !r.ExcludedSecondaryTypes.Overridden || len(r.ExcludedSecondaryTypes.Value) != 0 {
			t.Errorf("empty exclusion list is an explicit override: %+v", r.ExcludedSecondaryTypes)
		}
		if !r.IncludeFeatured.Overridden || !r.IncludeFeatured.Value {
			t.Errorf("include_featured = %+v, want overridden true", r.IncludeFeatured)
		}
	})
}
