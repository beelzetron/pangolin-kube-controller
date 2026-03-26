package logging

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

const unexpectedErrFmt = "unexpected error: %v"
const primitivePassThroughFmt = "expected primitive to pass through unchanged: %q vs %q"

// Test cases cover multiple secrets, overlaps, empty input, already-redacted values,
// unicode/special characters, nested objects and arrays. Also ensure non-secret content is preserved.
const red = "***redacted***"

func assertJSONEqual(t *testing.T, expected string, actual []byte) {
	t.Helper()
	var exp, act interface{}
	if err := json.Unmarshal([]byte(expected), &exp); err != nil {
		t.Fatalf("invalid expected JSON: %v", err)
	}
	if err := json.Unmarshal(actual, &act); err != nil {
		t.Fatalf("invalid actual JSON: %v", err)
	}
	if !reflect.DeepEqual(exp, act) {
		t.Fatalf("redaction mismatch\nexpected: %s\nactual:   %s", expected, string(actual))
	}
}

func TestRedactSimpleSecrets(t *testing.T) {
	in := `{"password":"p@ss","apiKey":"abc","TOKEN":"t"}`
	want := `{"password":"` + red + `","apiKey":"` + red + `","TOKEN":"` + red + `"}`
	got, err := RedactJSONLike([]byte(in))
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	assertJSONEqual(t, want, got)
}

func TestRedactOverlappingKeyNames(t *testing.T) {
	in := `{"passphrase":"value","secretSauce":"yummy","monkey":"banana"}`
	want := `{"passphrase":"` + red + `","secretSauce":"` + red + `","monkey":"` + red + `"}`
	got, err := RedactJSONLike([]byte(in))
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	assertJSONEqual(t, want, got)
}

func TestRedactEmptyObject(t *testing.T) {
	in := `{}`
	want := `{}`
	got, err := RedactJSONLike([]byte(in))
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	assertJSONEqual(t, want, got)
}

func TestRedactAlreadyRedacted(t *testing.T) {
	in := `{"token":"` + red + `","user":"bob"}`
	want := `{"token":"` + red + `","user":"bob"}`
	got, err := RedactJSONLike([]byte(in))
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	assertJSONEqual(t, want, got)
}

func TestRedactUnicodeKeysNested(t *testing.T) {
	in := `{"🔑KeyName":"x","profile":{"authHeader":"y","nested":[{"secret":"s"},{"data":"ok"}]}}`
	want := `{"🔑KeyName":"` + red + `","profile":{"authHeader":"` + red + `","nested":[{"secret":"` + red + `"},{"data":"ok"}]}}`
	got, err := RedactJSONLike([]byte(in))
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	assertJSONEqual(t, want, got)
}

func TestRedactArraysMixed(t *testing.T) {
	in := `{"items":[{"key":"k"},{"token":123},{"details":{"pass":"p"}}],"note":"n"}`
	want := `{"items":[{"key":"` + red + `"},{"token":"` + red + `"},{"details":{"pass":"` + red + `"}}],"note":"n"}`
	got, err := RedactJSONLike([]byte(in))
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	assertJSONEqual(t, want, got)
}

func TestRedactJSONLikeInvalidJSONReturnsInputAndError(t *testing.T) {
	in := []byte("{not-json}")
	got, err := RedactJSONLike(in)
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("expected input to be returned unchanged on error: %q vs %q", string(in), string(got))
	}
}

// Merged from redact_more_test.go
// TestRedactJSONLikeSimpleValues covers primitive JSON values and a couple
// of simple structured values (for example, an empty array) to ensure they
// are passed through unchanged when appropriate.
func TestRedactJSONLikeSimpleValues(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "string", in: `"hello"`},
		{name: "empty string", in: `""`},
		{name: "backslash-n sequence", in: `"hello\\nworld"`},
		{name: "int", in: `12345`},
		{name: "large int", in: `9999999999999999`},
		{name: "float", in: `1.23`},
		{name: "zero int", in: `0`},
		{name: "zero float", in: `0.0`},
		{name: "negative", in: `-42`},
		{name: "scientific", in: `6.02e23`},
		{name: "boolean", in: `true`},
		{name: "false", in: `false`},
		{name: "unicode string", in: `"你好"`},
		{name: "null", in: `null`},
		{name: "empty array", in: `[]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkPrimitivePassThrough(t, tc.in) })
	}
}

// checkPrimitivePassThrough validates that RedactJSONLike returns the same
// primitive JSON value for a given input. Extracted to reduce cognitive
// complexity of the table-driven test and to provide a reusable helper.
func checkPrimitivePassThrough(t *testing.T, inStr string) {
	t.Helper()
	in := []byte(inStr)
	got, err := RedactJSONLike(in)
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	// Try to compare semantically by unmarshalling JSON values (this
	// tolerates equivalent numeric formatting like 6.02e23 vs 6.02e+23).
	var expVal, gotVal interface{}
	if json.Unmarshal(in, &expVal) == nil && json.Unmarshal(got, &gotVal) == nil {
		if !reflect.DeepEqual(expVal, gotVal) {
			t.Fatalf(primitivePassThroughFmt, string(in), string(got))
		}
		return
	}
	if !bytes.Equal(got, in) {
		t.Fatalf(primitivePassThroughFmt, string(in), string(got))
	}
}
