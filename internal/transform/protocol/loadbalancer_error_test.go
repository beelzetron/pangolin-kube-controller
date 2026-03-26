package protocol

import (
	"encoding/json"
	"testing"
)

// helper to force JSON into RawMessage
func raw(s string) json.RawMessage { return json.RawMessage(s) }

// TestExtractLBAddressesPortMalformedAddress ensures malformed server addresses error.
func TestExtractLBAddressesPortMalformedAddress(t *testing.T) {
	svc := raw(`{"loadBalancer":{"servers":[{"address":"badstuff"}]}}`)
	_, _, err := extractLBAddressesPort(svc)
	if err == nil {
		t.Fatalf("expected error for malformed address")
	}
}

// TestExtractLBAddressesPortNoServers ensures missing servers list errors.
func TestExtractLBAddressesPortNoServers(t *testing.T) {
	svc := raw(`{"loadBalancer":{}}`)
	_, _, err := extractLBAddressesPort(svc)
	if err == nil {
		t.Fatalf("expected error for missing servers")
	}
}
