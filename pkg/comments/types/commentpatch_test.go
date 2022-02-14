package types

import "testing"

func TestFieldMaskContains(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mask   FieldMask
		field  Field
		wanted bool
	}{
		{
			name:   "exact match",
			mask:   FieldID.Mask(),
			field:  FieldID,
			wanted: true,
		},
		{
			name:   "mismatch",
			mask:   0,
			field:  FieldID,
			wanted: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			found := testCase.mask.Contains(testCase.field)
			if found != testCase.wanted {
				t.Logf("mask:       %b", testCase.mask)
				t.Logf("field:      %b", testCase.field)
				t.Logf("mask&field: %b", Field(testCase.mask)&testCase.field)
				if testCase.wanted {
					t.Fatal("wanted `true`; found `false`")
				}
				t.Fatal("wanted `false`; found `true`")
			}
		})
	}
}
