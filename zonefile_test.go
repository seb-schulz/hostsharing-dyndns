package main

import (
	"net/netip"
	"strings"
	"testing"
)

func TestZonefileWrite(t *testing.T) {
	ipv4 := netip.MustParseAddr("192.168.178.2")
	ipv6 := netip.MustParseAddr("2001:db8::68")

	for _, testCase := range []struct {
		subdomain      subdomain
		expectedResult string
	}{
		{subdomain: subdomain{}, expectedResult: `{DEFAULT}`},
		{subdomain: subdomain{"foobar", 60, &ipv4, nil},
			expectedResult: `{DEFAULT}
foobar.{DOM_HOSTNAME}. 60 IN A 192.168.178.2`},
		{subdomain: subdomain{"foobar", 60, nil, &ipv6},
			expectedResult: `{DEFAULT}

foobar.{DOM_HOSTNAME}. 60 IN AAAA 2001:db8::68`},
		{subdomain: subdomain{"foobar", 60, &ipv4, &ipv6},
			expectedResult: `{DEFAULT}
foobar.{DOM_HOSTNAME}. 60 IN A 192.168.178.2
foobar.{DOM_HOSTNAME}. 60 IN AAAA 2001:db8::68`},
		{subdomain: subdomain{"foobar", 120, &ipv4, &ipv6},
			expectedResult: `{DEFAULT}
foobar.{DOM_HOSTNAME}. 120 IN A 192.168.178.2
foobar.{DOM_HOSTNAME}. 120 IN AAAA 2001:db8::68`},
	} {
		z, err := newZonefile()
		if err != nil {
			t.Errorf("failed to init zonefile struc: %s", err)
		}

		z.Set(testCase.subdomain)

		b := strings.Builder{}
		if err := z.Write(&b); err != nil {
			t.Errorf("failed to write zonefile: %s", err)
		}

		if strings.Trim(b.String(), " \n") != testCase.expectedResult {
			t.Errorf("zonefile does not look as expected: \"%v\" != \"%v\"", b.String(), testCase.expectedResult)
		}

	}
}
