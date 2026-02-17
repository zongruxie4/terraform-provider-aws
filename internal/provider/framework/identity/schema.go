// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package identity

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	inttypes "github.com/hashicorp/terraform-provider-aws/internal/types"
)

func NewIdentitySchema(identitySpec inttypes.Identity) identityschema.Schema {
	schemaAttrs := make(map[string]identityschema.Attribute, len(identitySpec.Attributes))
	for _, attr := range identitySpec.Attributes {
		schemaAttrs[attr.Name()] = newIdentityAttribute(attr)
	}
	return identityschema.Schema{
		Attributes: schemaAttrs,
	}
}

func newIdentityAttribute(attribute inttypes.IdentityAttribute) identityschema.Attribute {
	required := attribute.Required()

	switch attribute.IdentityType() {
	case inttypes.BoolIdentityType:
		attr := identityschema.BoolAttribute{}
		if required {
			attr.RequiredForImport = true
		} else {
			attr.OptionalForImport = true
		}
		return attr
	case inttypes.FloatIdentityType:
		attr := identityschema.Float32Attribute{}
		if required {
			attr.RequiredForImport = true
		} else {
			attr.OptionalForImport = true
		}
		return attr
	case inttypes.IntIdentityType:
		attr := identityschema.Int32Attribute{}
		if required {
			attr.RequiredForImport = true
		} else {
			attr.OptionalForImport = true
		}
		return attr
	default:
		attr := identityschema.StringAttribute{}
		if required {
			attr.RequiredForImport = true
		} else {
			attr.OptionalForImport = true
		}
		return attr
	}
}
