package apply

func NumberToFloat(v interface{}) (float64, bool) {
	const maxExact = 1 << 53
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	}
	switch n := v.(type) {
	case int:
		n64 := int64(n)
		if n64 >= -maxExact && n64 <= maxExact {
			return float64(n64), true
		}
		return 0, false
	case int8, int16, int32:
		return float64(reflectInt64(n)), true
	case int64:
		if n >= -maxExact && n <= maxExact {
			return float64(n), true
		}
		return 0, false
	}
	switch n := v.(type) {
	case uint:
		n64 := uint64(n)
		if n64 <= uint64(maxExact) {
			return float64(n64), true
		}
		return 0, false
	case uint8, uint16, uint32:
		return float64(reflectUint64(n)), true
	case uint64:
		if n <= uint64(maxExact) {
			return float64(n), true
		}
		return 0, false
	}
	return 0, false
}

func reflectInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	default:
		return 0
	}
}

func reflectUint64(v interface{}) uint64 {
	switch n := v.(type) {
	case uint8:
		return uint64(n)
	case uint16:
		return uint64(n)
	case uint32:
		return uint64(n)
	default:
		return 0
	}
}
