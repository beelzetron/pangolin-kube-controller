package protocol

import "testing"

func TestEqualsTargets(t *testing.T) {
	tests := []struct {
		name     string
		left     kubeServiceTarget
		right    kubeServiceTarget
		expected bool
	}{
		{
			name:     "identical",
			left:     kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "http"},
			right:    kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "http"},
			expected: true,
		},
		{
			name:     "name mismatch",
			left:     kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "http"},
			right:    kubeServiceTarget{name: "n2", namespace: "ns", port: 80, scheme: "http"},
			expected: false,
		},
		{
			name:     "namespace mismatch",
			left:     kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "http"},
			right:    kubeServiceTarget{name: "n", namespace: "other", port: 80, scheme: "http"},
			expected: false,
		},
		{
			name:     "port mismatch",
			left:     kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "http"},
			right:    kubeServiceTarget{name: "n", namespace: "ns", port: 8080, scheme: "http"},
			expected: false,
		},
		{
			name:     "scheme mismatch",
			left:     kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "http"},
			right:    kubeServiceTarget{name: "n", namespace: "ns", port: 80, scheme: "https"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.left.equals(&tt.right)
			if got != tt.expected {
				t.Errorf("left.equals(&right) = %v, want %v", got, tt.expected)
			}
		})
	}
}
