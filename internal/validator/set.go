// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package validator

import (
	"fmt"
	"reflect"
)

// setDefault sets a location in the instance to its default value.
func setDefault(to reflect.Value, from any) error {
	fromV := reflect.ValueOf(from)
	toType := to.Type()
	if !fromV.CanConvert(toType) {
		return fmt.Errorf("when setting default can't convert value %v of type %T to type %s", fromV, fromV, toType)
	}
	fromV = fromV.Convert(toType)
	to.Set(fromV)
	return nil
}
