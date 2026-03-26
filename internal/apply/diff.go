package apply

import (
	"crypto/sha256"
	"encoding/hex"
	"math"

	"k8s.io/apimachinery/pkg/api/equality"
)

func ComputeHash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func DiffKeys(oldM map[string]interface{}, newM map[string]interface{}) []string {
	keys := map[string]struct{}{}
	for k := range oldM {
		keys[k] = struct{}{}
	}
	for k := range newM {
		keys[k] = struct{}{}
	}
	var changed []string
	for k := range keys {
		if !ValuesSemanticallyEqual(oldM[k], newM[k]) {
			changed = append(changed, k)
		}
	}
	return changed
}

func ValuesSemanticallyEqual(a, b interface{}) bool {
	if equality.Semantic.DeepEqual(a, b) {
		return true
	}
	if equal, ok := semanticallyEqualNumbers(a, b); ok {
		return equal
	}
	if equal, ok := semanticallyEqualMaps(a, b); ok {
		return equal
	}
	if equal, ok := semanticallyEqualIntUint(a, b); ok {
		return equal
	}
	return false
}

func semanticallyEqualNumbers(a, b interface{}) (equal bool, ok bool) {
	av, okA := NumberToFloat(a)
	bv, okB := NumberToFloat(b)
	if okA && okB {
		if math.IsNaN(av) && math.IsNaN(bv) {
			return true, true
		}
		return av == bv, true
	}
	if okA || okB {
		return false, true
	}
	return false, false
}

func semanticallyEqualMaps(a, b interface{}) (equal bool, ok bool) {
	oldMap, oldIsMap := a.(map[string]interface{})
	newMap, newIsMap := b.(map[string]interface{})
	if !oldIsMap && !newIsMap {
		return false, false
	}
	if oldIsMap != newIsMap {
		return false, true
	}
	for k := range oldMap {
		if !ValuesSemanticallyEqual(oldMap[k], newMap[k]) {
			return false, true
		}
	}
	for k := range newMap {
		if !ValuesSemanticallyEqual(oldMap[k], newMap[k]) {
			return false, true
		}
	}
	return true, true
}

func semanticallyEqualIntUint(a, b interface{}) (equal bool, ok bool) {
	switch ax := a.(type) {
	case int64:
		switch bx := b.(type) {
		case int64:
			return ax == bx, true
		case uint64:
			if ax < 0 {
				return false, true
			}
			return uint64(ax) == bx, true
		}
	case uint64:
		switch bx := b.(type) {
		case uint64:
			return ax == bx, true
		case int64:
			if bx < 0 {
				return false, true
			}
			return ax == uint64(bx), true
		}
	}
	return false, false
}
