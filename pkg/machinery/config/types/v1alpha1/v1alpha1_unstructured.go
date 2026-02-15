// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"fmt"
	"reflect"
	"slices"
)

// Unstructured allows wrapping any map[string]interface{} into a config object.
//
// docgen: nodoc
type Unstructured struct {
	Object map[string]any `yaml:",inline"`
}

// DeepCopy performs copying of the Object contents.
func (in *Unstructured) DeepCopy() *Unstructured {
	if in == nil {
		return nil
	}

	out := new(Unstructured)

	out.Object = deepCopyUnstructured(in.Object).(map[string]any) //nolint:forcetypeassert

	return out
}

func deepCopyUnstructured(x any) any {
	switch x := x.(type) {
	case map[string]any:
		if x == nil {
			return x
		}

		clone := make(map[string]any, len(x))

		for k, v := range x {
			clone[k] = deepCopyUnstructured(v)
		}

		return clone
	case []any:
		if x == nil {
			return x
		}

		clone := make([]any, len(x))

		for i, v := range x {
			clone[i] = deepCopyUnstructured(v)
		}

		return clone
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, complex64, complex128, nil:
		return x
	case []byte:
		return slices.Clone(x)
	default:
		// Some parts of the config tree build strongly-typed slices/maps (e.g. []map[string]any),
		// while YAML/JSON decoding often yields []any/map[string]any. Handle both.
		v := reflect.ValueOf(x)

		switch v.Kind() {
		case reflect.Slice:
			if v.IsNil() {
				return x
			}

			clone := reflect.MakeSlice(v.Type(), v.Len(), v.Len())

			for i := range v.Len() {
				orig := v.Index(i).Interface()
				copied := deepCopyUnstructured(orig)

				copiedV := reflect.ValueOf(copied)
				elemT := v.Type().Elem()

				if copiedV.IsValid() && copiedV.Type().AssignableTo(elemT) {
					clone.Index(i).Set(copiedV)
					continue
				}

				if copiedV.IsValid() && copiedV.Type().ConvertibleTo(elemT) {
					clone.Index(i).Set(copiedV.Convert(elemT))
					continue
				}

				// Fall back to the original element when we can't assign the copied value.
				clone.Index(i).Set(v.Index(i))
			}

			return clone.Interface()
		case reflect.Map:
			if v.IsNil() {
				return x
			}

			// Unstructured config objects should be string-keyed. If we can't guarantee that,
			// fail loudly instead of silently producing a malformed config tree.
			if v.Type().Key().Kind() != reflect.String {
				panic(fmt.Errorf("cannot deep copy %T (map key type %s)", x, v.Type().Key()))
			}

			clone := reflect.MakeMapWithSize(v.Type(), v.Len())

			iter := v.MapRange()
			for iter.Next() {
				k := iter.Key()
				orig := iter.Value().Interface()
				copied := deepCopyUnstructured(orig)
				copiedV := reflect.ValueOf(copied)

				elemT := v.Type().Elem()
				switch {
				case copiedV.IsValid() && copiedV.Type().AssignableTo(elemT):
					clone.SetMapIndex(k, copiedV)
				case copiedV.IsValid() && copiedV.Type().ConvertibleTo(elemT):
					clone.SetMapIndex(k, copiedV.Convert(elemT))
				default:
					clone.SetMapIndex(k, iter.Value())
				}
			}

			return clone.Interface()
		}

		panic(fmt.Errorf("cannot deep copy %T", x))
	}
}
