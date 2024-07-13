package main

import (
	"net"
	"strings"
	"testing"
)

func TestZonefileWrite(t *testing.T) {
	ipv4 := net.ParseIP("192.168.178.2")
	ipv6 := net.ParseIP("2001:db8::68")

	for _, testCase := range []struct {
		subdomains     []subdomain
		expectedResult string
	}{
		{subdomains: []subdomain{}, expectedResult: `{DEFAULT}`},
		{subdomains: []subdomain{{"foobar", 60, &ipv4, nil}},
			expectedResult: `{DEFAULT}
foobar.{DOM_HOSTNAME}. 60 IN A 192.168.178.2
`},
		{subdomains: []subdomain{{"foobar", 60, nil, &ipv6}},
			expectedResult: `{DEFAULT}

foobar.{DOM_HOSTNAME}. 60 IN AAAA 2001:db8::68`},
		{subdomains: []subdomain{{"foobar", 60, &ipv4, &ipv6}},
			expectedResult: `{DEFAULT}
foobar.{DOM_HOSTNAME}. 60 IN A 192.168.178.2
foobar.{DOM_HOSTNAME}. 60 IN AAAA 2001:db8::68`},
		{subdomains: []subdomain{{"foobar", 120, &ipv4, &ipv6}},
			expectedResult: `{DEFAULT}
foobar.{DOM_HOSTNAME}. 120 IN A 192.168.178.2
foobar.{DOM_HOSTNAME}. 120 IN AAAA 2001:db8::68`},
	} {
		z, err := New(testCase.subdomains)
		if err != nil {
			t.Errorf("failed to init zonefile struc: %s", err)
		}

		b := strings.Builder{}
		if err := z.Write(&b); err != nil {
			t.Errorf("failed to write zonefile: %s", err)
		}

		if b.String() != testCase.expectedResult {
			t.Errorf("zonefile does not look as expected: \"%v\" != \"%v\"", b.String(), testCase.expectedResult)
		}

	}
}
