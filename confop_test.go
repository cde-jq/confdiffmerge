package main

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDiff(t *testing.T) {
	type Case struct {
		l    string
		r    string
		diff map[string]any
	}
	cases := []Case{{
		l: `a: 3
b: false
onlyl: sdf
d:
  d1: 43
  d2: fwe
c: 4
sl: [3,1]
`,
		r: `a: 3
b: true
d:
  d1: 43
  d2: fwe3
  d3:    
    sd: sdf
new: 3
c: 445
sl: [3,1,1]
`,
		diff: map[string]any{
			"b":   true,
			"c":   445,
			"new": 3,
			"d": map[string]any{
				"d2": "fwe3",
				"d3": map[string]any{
					"sd": "sdf",
				},
			},
			"sl": []any{3, 1, 1},
		},
	},
	}
	for _, c := range cases {
		var lm, rm map[string]any
		err := yaml.Unmarshal([]byte(c.l), &lm)
		if err != nil {
			t.Error(err)
		}
		err = yaml.Unmarshal([]byte(c.r), &rm)
		if err != nil {
			t.Error(err)
		}
		res := diff(lm, rm)

		if !reflect.DeepEqual(res, c.diff) {
			t.Error(res, "not equal", c.diff)
		}
	}

}

func TestMerge(t *testing.T) {
	type Case struct {
		l     string
		r     string
		merge map[string]any
	}
	cases := []Case{{
		l: `a: 3
b: false
onlyl: sdf
d:
  d1: 43
  d2: fwe
c: 4
sl: [3,1]
`,
		r: `a: 3
b: true
d:
  d1: 43
  d2: fwe3
  d3:    
    sd: sdf
new: 3
c: 445
sl: [3,1]
`,
		merge: map[string]any{
			"a":     3,
			"b":     true,
			"c":     445,
			"onlyl": "sdf",
			"new":   3,
			"d": map[string]any{
				"d1": 43,
				"d2": "fwe3",
				"d3": map[string]any{
					"sd": "sdf",
				},
			},
			"sl": []any{3, 1}, // yaml.Unmarshal will produce []any slice not []int
		},
	},
	}
	for _, c := range cases {
		var lm, rm map[string]any
		err := yaml.Unmarshal([]byte(c.l), &lm)
		if err != nil {
			t.Error(err)
		}
		err = yaml.Unmarshal([]byte(c.r), &rm)
		if err != nil {
			t.Error(err)
		}
		res := merge(lm, rm)

		if !reflect.DeepEqual(res, c.merge) {
			t.Error(res, "not equal", c.merge)
		}
	}

}
