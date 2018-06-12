// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authz

import (
	"reflect"
	"strings"
	"testing"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/types"
)

func TestStringMatch(t *testing.T) {
	testCases := []struct {
		Name   string
		S      string
		List   []string
		Expect bool
	}{
		{
			Name: "exact match", S: "product page", List: []string{"review page", "product page"},
			Expect: true,
		},
		{
			Name: "wild character match", S: "product page", List: []string{"review page", "*"},
			Expect: true,
		},
		{
			Name: "prefix match", S: "product page", List: []string{"review page", "product*"},
			Expect: true,
		},
		{
			Name: "suffix match", S: "product page", List: []string{"review page", "*page"},
			Expect: true,
		},
		{
			Name: "not matched", S: "product page", List: []string{"review page", "xyz product page"},
			Expect: false,
		},
	}

	for _, tc := range testCases {
		if actual := stringMatch(tc.S, tc.List); actual != tc.Expect {
			t.Errorf("%s: expecting: %v, but got: %v", tc.Name, tc.Expect, actual)
		}
	}
}

func TestConvertToCidr(t *testing.T) {
	testCases := []struct {
		Name   string
		V      string
		Expect *core.CidrRange
		Err    string
	}{
		{
			Name: "cidr with two /",
			V:    "192.168.0.0//16",
			Err:  "invalid cidr range",
		},
		{
			Name: "cidr with invalid prefix length",
			V:    "192.168.0.0/ab",
			Err:  "invalid cidr range",
		},
		{
			Name: "cidr with negative prefix length",
			V:    "192.168.0.0/-16",
			Err:  "invalid cidr range",
		},
		{
			Name: "valid cidr range",
			V:    "192.168.0.0/16",
			Expect: &core.CidrRange{
				AddressPrefix: "192.168.0.0",
				PrefixLen:     &types.UInt32Value{Value: 16},
			},
		},
		{
			Name: "invalid ip address",
			V:    "19216800",
			Err:  "invalid ip address",
		},
		{
			Name: "valid ipv4 address",
			V:    "192.168.0.0",
			Expect: &core.CidrRange{
				AddressPrefix: "192.168.0.0",
				PrefixLen:     &types.UInt32Value{Value: 32},
			},
		},
		{
			Name: "valid ipv6 address",
			V:    "2001:abcd:85a3::8a2e:370:1234",
			Expect: &core.CidrRange{
				AddressPrefix: "2001:abcd:85a3::8a2e:370:1234",
				PrefixLen:     &types.UInt32Value{Value: 128},
			},
		},
	}

	for _, tc := range testCases {
		actual, err := convertToCidr(tc.V)
		if tc.Err != "" {
			if err == nil {
				t.Errorf("%s: expecting error: %s but found no error", tc.Name, tc.Err)
			} else if !strings.HasPrefix(err.Error(), tc.Err) {
				t.Errorf("%s: expecting error: %s, but got: %s", tc.Name, tc.Err, err.Error())
			}
		} else if !reflect.DeepEqual(*tc.Expect, *actual) {
			t.Errorf("%s: expecting %v, but got %v", tc.Name, *tc.Expect, *actual)
		}
	}
}

func TestConvertToPort(t *testing.T) {
	testCases := []struct {
		Name   string
		V      string
		Expect uint32
		Err    string
	}{
		{
			Name: "negative port",
			V:    "-80",
			Err:  "invalid port -80:",
		},
		{
			Name: "invalid port",
			V:    "xyz",
			Err:  "invalid port xyz:",
		},
		{
			Name: "port too large",
			V:    "91234",
			Err:  "invalid port 91234:",
		},
		{
			Name:   "valid port",
			V:      "443",
			Expect: 443,
		},
	}

	for _, tc := range testCases {
		actual, err := convertToPort(tc.V)
		if tc.Err != "" {
			if err == nil {
				t.Errorf("%s: expecting error %s but found no error", tc.Name, tc.Err)
			} else if !strings.HasPrefix(err.Error(), tc.Err) {
				t.Errorf("%s: expecting error %s, but got: %s", tc.Name, tc.Err, err.Error())
			}
		} else if tc.Expect != actual {
			t.Errorf("%s: expecting %d, but got %d", tc.Name, tc.Expect, actual)
		}
	}
}

func TestConvertToHeaderMatcher(t *testing.T) {
	testCases := []struct {
		Name   string
		K      string
		V      string
		Expect *route.HeaderMatcher
	}{
		{
			Name: "exact match",
			K:    ":path",
			V:    "/productpage",
			Expect: &route.HeaderMatcher{
				Name: ":path",
				HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
					ExactMatch: "/productpage",
				},
			},
		},
		{
			Name: "regex match",
			K:    ":path",
			V:    "*/productpage*",
			Expect: &route.HeaderMatcher{
				Name: ":path",
				HeaderMatchSpecifier: &route.HeaderMatcher_RegexMatch{
					RegexMatch: "^.*/productpage.*$",
				},
			},
		},
	}

	for _, tc := range testCases {
		actual := convertToHeaderMatcher(tc.K, tc.V)
		if !reflect.DeepEqual(*tc.Expect, *actual) {
			t.Errorf("%s: expecting %v, but got %v", tc.Name, *tc.Expect, *actual)
		}
	}
}
