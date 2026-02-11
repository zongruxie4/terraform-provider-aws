// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package cty_test

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	tfcty "github.com/hashicorp/terraform-provider-aws/internal/cty"
)

func TestHasValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    cty.Value
		expected bool
	}{
		"NilVal": {
			value:    cty.NilVal,
			expected: false,
		},
		"EmptyList": {
			value:    cty.ListValEmpty(cty.String),
			expected: false,
		},
		"EmptyMap": {
			value:    cty.MapValEmpty(cty.String),
			expected: false,
		},
		"String": {
			value:    cty.StringVal("test"),
			expected: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := tfcty.HasValue(tt.value)
			if actual != tt.expected {
				t.Errorf("HasValue() = %t, expected %t", actual, tt.expected)
			}
		})
	}
}
