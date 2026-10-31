package auth

import "testing"

func TestSMTPUsesImplicitTLSOnSMTPSPorts(t *testing.T) {
	for _, test := range []struct {
		port string
		want bool
	}{{"465", true}, {"2465", true}, {"587", false}, {"2587", false}, {"25", false}} {
		if got := usesImplicitTLS(test.port); got != test.want {
			t.Errorf("usesImplicitTLS(%q) = %v, want %v", test.port, got, test.want)
		}
	}
}
