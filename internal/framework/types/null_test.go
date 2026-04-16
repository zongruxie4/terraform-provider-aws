// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package types_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
)

type enumString string

const (
	enumStringValue1 enumString = "value1"
)

func (enumString) Values() []enumString {
	return []enumString{
		enumStringValue1,
	}
}

func TestNullValueOf_primitiveTypes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    any
		expected attr.Value
	}{
		"bool": {
			input:    types.BoolValue(true),
			expected: types.BoolNull(),
		},
		"float32": {
			input:    types.Float32Value(1.0),
			expected: types.Float32Null(),
		},
		"float64": {
			input:    types.Float64Value(1.0),
			expected: types.Float64Null(),
		},
		"int32": {
			input:    types.Int32Value(1),
			expected: types.Int32Null(),
		},
		"int64": {
			input:    types.Int64Value(1),
			expected: types.Int64Null(),
		},
		"string": {
			input:    types.StringValue("test"),
			expected: types.StringNull(),
		},

		"enum": {
			input:    fwtypes.StringEnumValue(enumStringValue1),
			expected: fwtypes.StringEnumNull[enumString](),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fwtypes.NullValueOf(context.Background(), testCase.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if e, a := testCase.expected, got; !e.Equal(a) {
				t.Errorf("Did not get Null value")
			}
		})
	}
}

func TestNullValueOf_listTypes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    any
		expected attr.Value
	}{
		"typed": {
			input:    fwtypes.NewListValueOfMust[types.String](context.Background(), []attr.Value{}),
			expected: fwtypes.NewListValueOfNull[types.String](context.Background()),
		},
		"raw": {
			input:    types.ListValueMust(types.StringType, []attr.Value{}),
			expected: types.ListNull(types.StringType),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fwtypes.NullValueOf(context.Background(), testCase.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if e, a := testCase.expected, got; !e.Equal(a) {
				t.Errorf("Did not get Null value")
			}
		})
	}
}

func TestNullValueOf_setTypes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    any
		expected attr.Value
	}{
		"typed": {
			input:    fwtypes.NewSetValueOfMust[types.String](context.Background(), []attr.Value{}),
			expected: fwtypes.NewSetValueOfNull[types.String](context.Background()),
		},
		"raw": {
			input:    types.SetValueMust(types.StringType, []attr.Value{}),
			expected: types.SetNull(types.StringType),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fwtypes.NullValueOf(context.Background(), testCase.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if e, a := testCase.expected, got; !e.Equal(a) {
				t.Errorf("Did not get Null value")
			}
		})
	}
}

func TestNullValueOf_mapTypes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    any
		expected attr.Value
	}{
		"typed": {
			input:    fwtypes.NewMapValueOfMust[types.String](context.Background(), map[string]attr.Value{}),
			expected: fwtypes.NewMapValueOfNull[types.String](context.Background()),
		},
		"raw": {
			input:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			expected: types.MapNull(types.StringType),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fwtypes.NullValueOf(context.Background(), testCase.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if e, a := testCase.expected, got; !e.Equal(a) {
				t.Errorf("Did not get Null value")
			}
		})
	}
}

func TestNullValueOf_objectTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type object struct {
		Name types.String `tfsdk:"name"`
	}

	testCases := map[string]struct {
		input    any
		expected attr.Value
	}{
		"typed": {
			input: fwtypes.NewObjectValueOfMust(ctx, &object{
				Name: types.StringValue("test"),
			}),
			expected: fwtypes.NewObjectValueOfNull[object](ctx),
		},
		"raw": {
			input: types.ObjectValueMust(fwtypes.AttributeTypesMust[object](ctx), map[string]attr.Value{
				"name": types.StringValue("test"),
			}),
			expected: types.ObjectNull(fwtypes.AttributeTypesMust[object](ctx)),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fwtypes.NullValueOf(ctx, testCase.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if e, a := testCase.expected, got; !e.Equal(a) {
				t.Errorf("Did not get Null value")
			}
		})
	}
}
