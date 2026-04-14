// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package dynamodb

import (
	"testing"

	awstypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
)

func TestGlobalSecondaryIndexKeySchemaListValidator(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		value       fwtypes.ListNestedObjectValueOf[keySchemaModel]
		expectError bool
	}{
		"unknown": {
			value:       fwtypes.NewListNestedObjectValueOfUnknown[keySchemaModel](t.Context()),
			expectError: false,
		},
		"null": {
			value:       fwtypes.NewListNestedObjectValueOfNull[keySchemaModel](t.Context()),
			expectError: true,
		},
		"fully known": {
			value: fwtypes.NewListNestedObjectValueOfValueSliceMust(t.Context(), []keySchemaModel{
				{
					AttributeName: types.StringValue("attribute_name"),
					AttributeType: fwtypes.StringEnumValue(awstypes.ScalarAttributeTypeS),
					KeyType:       fwtypes.StringEnumValue(awstypes.KeyTypeHash),
				},
			}),
			expectError: false,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request := validator.ListRequest{
				Path:           path.Root("test"),
				PathExpression: path.MatchRoot("test"),
				ConfigValue:    testCase.value.ListValue,
			}
			response := validator.ListResponse{}

			globalSecondaryIndexKeySchemaListValidator{}.ValidateList(t.Context(), request, &response)

			if !response.Diagnostics.HasError() && testCase.expectError {
				t.Fatal("expected error, got no error")
			}

			if response.Diagnostics.HasError() && !testCase.expectError {
				t.Fatalf("got unexpected error: %s", response.Diagnostics)
			}
		})
	}
}
