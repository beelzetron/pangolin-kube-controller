package protocol

import "testing"

func TestChooseAddressType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"ipv4 single", []string{"10.0.0.1"}, "IPv4"},               //NOSONAR - suppress [warning-code] for test data
		{"ipv6 single", []string{"2001:db8::1"}, "IPv6"},            //NOSONAR - suppress [warning-code] for test data
		{"fqdn single", []string{"example.com"}, "FQDN"},            //NOSONAR - suppress [warning-code] for test data
		{"multi host", []string{"example.com", "10.0.0.2"}, "FQDN"}, //NOSONAR - suppress [warning-code] for test data
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := chooseAddressType(tc.in)
			if err != nil {
				t.Fatalf("unexpected error from chooseAddressType: %v", err)
			}
			if string(got) != tc.want { // chooseAddressType returns discoveryv1.AddressType
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
