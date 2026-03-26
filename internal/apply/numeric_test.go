package apply

import (
	"math"
	"testing"
)

func TestNumberToFloat(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   float64
		wantOk bool
	}{
		{"float64", float64(1.5), 1.5, true},
		{"float32", float32(2.5), 2.5, true},
		{"int", int(10), 10.0, true},
		{"int8", int8(-20), -20.0, true},
		{"int16", int16(300), 300.0, true},
		{"int32", int32(-400), -400.0, true},
		{"int64 large within range", int64(1 << 50), float64(1 << 50), true},
		{"int64 beyond safe range", int64(1 << 60), 0, false},
		{"uint", uint(15), 15.0, true},
		{"uint8", uint8(250), 250.0, true},
		{"uint16", uint16(60000), 60000.0, true},
		{"uint32", uint32(4000000000), 4000000000.0, true},
		{"uint64 beyond safe range", uint64(1 << 60), 0, false},
		{"string", "invalid", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NumberToFloat(tt.input)
			if ok != tt.wantOk {
				t.Errorf("NumberToFloat(%v) ok = %v, want %v", tt.input, ok, tt.wantOk)
				return
			}
			if ok && got != tt.want {
				t.Errorf("NumberToFloat(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNumberToFloatEdgeCases(t *testing.T) {
	maxExact := int64(1 << 53)
	if got, ok := NumberToFloat(maxExact); !ok || got != float64(maxExact) {
		t.Errorf("expected int64(%v) to convert successfully to %v", maxExact, float64(maxExact))
	}

	overMax := maxExact + 1
	if _, ok := NumberToFloat(overMax); ok {
		t.Errorf("expected int64(%v) to fail conversion (beyond maxExact)", overMax)
	}
}

func TestIntUintOverflowOn64Bit(t *testing.T) {
	// Skip on 32-bit architectures. This matches the requested detection pattern.
	if ^uint(0)>>63 == 0 {
		t.Skip("skipping on 32-bit architectures")
	}

	large := uint64(1) << 54
	largeInt := int(large)
	largeUint := uint(large)

	if _, ok := NumberToFloat(largeInt); ok {
		t.Errorf("expected NumberToFloat(%v) to reject large int beyond 2^53 safe range", largeInt)
	}
	if _, ok := NumberToFloat(largeUint); ok {
		t.Errorf("expected NumberToFloat(%v) to reject large uint beyond 2^53 safe range", largeUint)
	}
}

func TestReflectInt64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
	}{
		{"int8", int8(-8), -8},
		{"int16", int16(1000), 1000},
		{"int32", int32(-50000), -50000},
		{"invalid string", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectInt64(tt.input); got != tt.want {
				t.Errorf("reflectInt64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestReflectUint64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  uint64
	}{
		{"uint8", uint8(8), 8},
		{"uint16", uint16(1000), 1000},
		{"uint32", uint32(50000), 50000},
		{"invalid string", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectUint64(tt.input); got != tt.want {
				t.Errorf("reflectUint64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLargeInt64Precision(t *testing.T) {
	large := int64(1 << 61)
	if _, ok := NumberToFloat(large); ok {
		t.Error("expected large int64 to fail conversion")
	}
}

func TestMaxExactBoundary(t *testing.T) {
	maxExact := int64(1 << 53)
	if got, ok := NumberToFloat(maxExact); !ok || got != float64(maxExact) {
		t.Errorf("maxExact int64 should convert successfully, got %v, ok %v", got, ok)
	}

	negMaxExact := int64(-(1 << 53))
	if got, ok := NumberToFloat(negMaxExact); !ok || got != float64(negMaxExact) {
		t.Errorf("negative maxExact int64 should convert successfully, got %v, ok %v", got, ok)
	}
}

func TestNaNInf(t *testing.T) {
	if got, ok := NumberToFloat(math.NaN()); !ok || !math.IsNaN(got) {
		t.Errorf("NaN should return ok=true and NaN value, got %v, ok %v", got, ok)
	}
	if got, ok := NumberToFloat(math.Inf(1)); !ok || got != math.Inf(1) {
		t.Errorf("+Inf should return ok=true, got %v, ok %v", got, ok)
	}
	if got, ok := NumberToFloat(math.Inf(-1)); !ok || got != math.Inf(-1) {
		t.Errorf("-Inf should return ok=true, got %v, ok %v", got, ok)
	}
}
