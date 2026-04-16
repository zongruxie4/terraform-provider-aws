// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
)

// NullValueOf returns a null attr.Value for the specified `v`.
func NullValueOf(ctx context.Context, v any) (attr.Value, error) {
	var attrType attr.Type
	var tfType tftypes.Type

	switch v := v.(type) {
	case basetypes.BoolValuable:
		attrType = v.Type(ctx)
		toType := attrType.(basetypes.BoolTypable)
		return fwdiag.Must(toType.ValueFromBool(ctx, types.BoolNull())), nil

	case basetypes.Float32Valuable:
		attrType = v.Type(ctx)
		toType := attrType.(basetypes.Float32Typable)
		return fwdiag.Must(toType.ValueFromFloat32(ctx, types.Float32Null())), nil

	case basetypes.Float64Valuable:
		attrType = v.Type(ctx)
		toType := attrType.(basetypes.Float64Typable)
		return fwdiag.Must(toType.ValueFromFloat64(ctx, types.Float64Null())), nil

	case basetypes.Int32Valuable:
		attrType = v.Type(ctx)
		toType := attrType.(basetypes.Int32Typable)
		return fwdiag.Must(toType.ValueFromInt32(ctx, types.Int32Null())), nil

	case basetypes.Int64Valuable:
		attrType = v.Type(ctx)
		toType := attrType.(basetypes.Int64Typable)
		return fwdiag.Must(toType.ValueFromInt64(ctx, types.Int64Null())), nil

	case basetypes.StringValuable:
		attrType = v.Type(ctx)
		toType := attrType.(basetypes.StringTypable)
		return fwdiag.Must(toType.ValueFromString(ctx, types.StringNull())), nil

	case basetypes.ListValuable:
		attrType = v.Type(ctx)
		if v, ok := attrType.(attr.TypeWithElementType); ok {
			toType := attrType.(basetypes.ListTypable)
			return fwdiag.Must(toType.ValueFromList(ctx, types.ListNull(v.ElementType()))), nil
		} else {
			tfType = tftypes.List{}
		}

	case basetypes.SetValuable:
		attrType = v.Type(ctx)
		if v, ok := attrType.(attr.TypeWithElementType); ok {
			toType := attrType.(basetypes.SetTypable)
			return fwdiag.Must(toType.ValueFromSet(ctx, types.SetNull(v.ElementType()))), nil
		} else {
			tfType = tftypes.Set{}
		}

	case basetypes.MapValuable:
		attrType = v.Type(ctx)
		if v, ok := attrType.(attr.TypeWithElementType); ok {
			toType := attrType.(basetypes.MapTypable)
			return fwdiag.Must(toType.ValueFromMap(ctx, types.MapNull(v.ElementType()))), nil
		} else {
			tfType = tftypes.Map{}
		}

	case basetypes.ObjectValuable:
		attrType = v.Type(ctx)
		if v, ok := attrType.(attr.TypeWithAttributeTypes); ok {
			toType := attrType.(basetypes.ObjectTypable)
			return fwdiag.Must(toType.ValueFromObject(ctx, types.ObjectNull(v.AttributeTypes()))), nil
		} else {
			tfType = tftypes.Object{}
		}

	default:
		return nil, nil
	}

	return attrType.ValueFromTerraform(ctx, tftypes.NewValue(tfType, nil))
}
