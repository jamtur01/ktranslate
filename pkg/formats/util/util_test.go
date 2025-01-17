package util

import (
	"regexp"
	"strings"
	"testing"

	"github.com/kentik/ktranslate/pkg/kt"

	"github.com/stretchr/testify/assert"
)

func TestDropOnFilter(t *testing.T) {
	tests := []struct {
		attr    map[string]interface{}
		in      *kt.JCHF
		metrics map[string]kt.MetricInfo
		lm      kt.LastMetadata
		drop    bool
	}{
		{
			map[string]interface{}{},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{},
			false,
		},
		{
			map[string]interface{}{
				"foo": "bar",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"foo": regexp.MustCompile("bar"),
				},
			},
			false,
		},
		{
			map[string]interface{}{
				"foo": "ba11",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"foo": regexp.MustCompile("bar"),
				},
			},
			true,
		},
		{
			map[string]interface{}{
				"foo": "ba11",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"foo": regexp.MustCompile("^ba"),
				},
			},
			false,
		},
		{
			map[string]interface{}{
				kt.AdminStatus: "down",
				"fooXX":        "bar",
			},
			kt.NewJCHF().SetIFPorts(10),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"fooXX":        regexp.MustCompile("bar"),
					kt.AdminStatus: regexp.MustCompile("up"),
				},
				InterfaceInfo: map[kt.IfaceID]map[string]interface{}{
					10: map[string]interface{}{
						"Description": "myIfDesc",
					},
				},
			},
			true,
		},
		{ // 5
			map[string]interface{}{
				kt.AdminStatus: "up",
				"foo":          "bar",
				"aaa":          "aaa",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"foo":          regexp.MustCompile("abar"),
					"aaa":          regexp.MustCompile("aaa"),
					kt.AdminStatus: regexp.MustCompile("up"),
				},
			},
			false,
		},
		{ // 6
			map[string]interface{}{
				kt.AdminStatus: "up",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"fooAAA":       regexp.MustCompile("abar"),
					kt.AdminStatus: regexp.MustCompile("up"),
				},
			},
			false, // Let through because status is up and fooAAA doesn't exist in the attribute list.
		},
		{
			map[string]interface{}{
				kt.AdminStatus: "up",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					kt.AdminStatus: regexp.MustCompile("up"),
				},
			},
			false,
		},
		{
			map[string]interface{}{
				kt.AdminStatus: "up",
				"foo":          "bar",
				"aaa":          "aaa",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"foo":          regexp.MustCompile("no"),
					"aaa":          regexp.MustCompile("no"),
					kt.AdminStatus: regexp.MustCompile("up"),
				},
			},
			true, // Drop because neither foo or aaa match even though admin is up.
		},
		{
			map[string]interface{}{
				kt.AdminStatus: "up",
				"foo":          "bar",
				"aaa":          "aaa",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"foo":          regexp.MustCompile("no"),
					"aaa":          regexp.MustCompile("aa"),
					kt.AdminStatus: regexp.MustCompile("up"),
				},
			},
			false, // Keep because aaa matches and admin is up.
		},
		{
			map[string]interface{}{
				kt.AdminStatus: "up",
			},
			kt.NewJCHF().SetIFPorts(20),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"if_Description": regexp.MustCompile("igb3"),
					kt.AdminStatus:   regexp.MustCompile("up"),
					"device_name":    regexp.MustCompile("bart"),
				},
				InterfaceInfo: map[kt.IfaceID]map[string]interface{}{
					20: map[string]interface{}{
						"Description": "igb2",
					},
				},
			},
			true, // Drop because desc doesn't match.
		},
		{
			map[string]interface{}{
				"mib-name": "UDP-MIB",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"if_Description": regexp.MustCompile("igb3"),
					kt.AdminStatus:   regexp.MustCompile("up"),
					"mib-name":       regexp.MustCompile("UDP"),
				},
			},
			false, // keep because mib-name matches and no admin status.
		},
		{
			map[string]interface{}{
				kt.AdminStatus: "up",
				"if_Alias":     "foo",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"!if_Description": regexp.MustCompile("igb3"),
					kt.AdminStatus:    regexp.MustCompile("up"),
				},
			},
			true, // drop because missing desciption.
		},
		{
			map[string]interface{}{
				kt.AdminStatus:   "up",
				"if_Alias":       "foo",
				"if_Description": "igb3",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"!if_Description": regexp.MustCompile("igb3"),
					kt.AdminStatus:    regexp.MustCompile("up"),
				},
			},
			false, // keep because matching desciption.
		},
		{
			map[string]interface{}{
				kt.AdminStatus:   "up",
				"if_Description": "igb4",
			},
			kt.NewJCHF(),
			map[string]kt.MetricInfo{},
			kt.LastMetadata{
				MatchAttr: map[string]*regexp.Regexp{
					"!if_Alias":      regexp.MustCompile("igb3"),
					"if_Description": regexp.MustCompile("igb4"),
					kt.AdminStatus:   regexp.MustCompile("up"),
				},
			},
			true, // drop because alias is missing.
		},
	}

	for i, test := range tests {
		SetAttr(test.attr, test.in, test.metrics, &test.lm)
		isIf := false
		for k, _ := range test.attr {
			if strings.HasPrefix(k, "if_") {
				isIf = true
			}
		}
		drop := DropOnFilter(test.attr, &test.lm, isIf)
		assert.Equal(t, test.drop, drop, "Test %d", i)
	}
}
