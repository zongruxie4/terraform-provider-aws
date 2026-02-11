// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package cty

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
)

// PathSafeApply is equivalent to cty.Path.Apply but does not return an error when one of the steps has a null or unkown value.
// Instead, it returns the value at the point of failure and a boolean indicating whether the path was fully applied or not.
// Other conditions, such as invalid types or non-existent indexes, will still return an error.
func PathSafeApply(p cty.Path, val cty.Value) (cty.Value, bool, error) {
	var err error

	l := len(p)
	for i, step := range p {
		val, err = step.Apply(val)
		if err != nil {
			return cty.NilVal, false, fmt.Errorf("at step %d: %s", i, err)
		}
		if !HasValue(val) && i < l-1 {
			return val, false, nil
		}
	}
	return val, true, nil
}
