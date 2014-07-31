/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package labels

import (
	"testing"
)

func TestSelectorParse(t *testing.T) {
	testGoodStrings := []string{
		"x=a,y=b,z=c",
		"",
		"x!=a,y=b",
	}
	testBadStrings := []string{
		"x=a||y=b",
		"x==a==b",
	}
	for _, test := range testGoodStrings {
		lq, err := ParseSelector(test)
		if err != nil {
			t.Errorf("%v: error %v (%#v)\n", test, err, err)
		}
		if test != lq.String() {
			t.Errorf("%v restring gave: %v\n", test, lq.String())
		}
	}
	for _, test := range testBadStrings {
		_, err := ParseSelector(test)
		if err == nil {
			t.Errorf("%v: did not get expected error\n", test)
		}
	}
}

func TestDeterministicParse(t *testing.T) {
	s1, err := ParseSelector("x=a,a=x")
	s2, err2 := ParseSelector("a=x,x=a")
	if err != nil || err2 != nil {
		t.Errorf("Unexpected parse error")
	}
	if s1.String() != s2.String() {
		t.Errorf("Non-deterministic parse")
	}
}

func expectMatch(t *testing.T, selector string, ls Set) {
	lq, err := ParseSelector(selector)
	if err != nil {
		t.Errorf("Unable to parse %v as a selector\n", selector)
		return
	}
	if !lq.Matches(ls) {
		t.Errorf("Wanted %s to match '%s', but it did not.\n", selector, ls)
	}
}

func expectNoMatch(t *testing.T, selector string, ls Set) {
	lq, err := ParseSelector(selector)
	if err != nil {
		t.Errorf("Unable to parse %v as a selector\n", selector)
		return
	}
	if lq.Matches(ls) {
		t.Errorf("Wanted '%s' to not match '%s', but it did.", selector, ls)
	}
}

func TestEverything(t *testing.T) {
	if !Everything().Matches(Set{"x": "y"}) {
		t.Errorf("Nil selector didn't match")
	}
	if !Everything().Empty() {
		t.Errorf("Everything was not empty")
	}
}

func TestSelectorMatches(t *testing.T) {
	expectMatch(t, "", Set{"x": "y"})
	expectMatch(t, "x=y", Set{"x": "y"})
	expectMatch(t, "x=y,z=w", Set{"x": "y", "z": "w"})
	expectMatch(t, "x!=y,z!=w", Set{"x": "z", "z": "a"})
	expectNoMatch(t, "x=y", Set{"x": "z"})
	expectNoMatch(t, "x=y,z=w", Set{"x": "w", "z": "w"})
	expectNoMatch(t, "x!=y,z!=w", Set{"x": "z", "z": "w"})

	labelset := Set{
		"foo": "bar",
		"baz": "blah",
	}
	expectMatch(t, "foo=bar", labelset)
	expectMatch(t, "baz=blah", labelset)
	expectMatch(t, "foo=bar,baz=blah", labelset)
	expectNoMatch(t, "foo=blah", labelset)
	expectNoMatch(t, "baz=bar", labelset)
	expectNoMatch(t, "foo=bar,foobar=bar,baz=blah", labelset)
}

func expectMatchDirect(t *testing.T, selector, ls Set) {
	if !SelectorFromSet(selector).Matches(ls) {
		t.Errorf("Wanted %s to match '%s', but it did not.\n", selector, ls)
	}
}

func expectNoMatchDirect(t *testing.T, selector, ls Set) {
	if SelectorFromSet(selector).Matches(ls) {
		t.Errorf("Wanted '%s' to not match '%s', but it did.", selector, ls)
	}
}

func TestSetMatches(t *testing.T) {
	labelset := Set{
		"foo": "bar",
		"baz": "blah",
	}
	expectMatchDirect(t, Set{}, labelset)
	expectMatchDirect(t, Set{"foo": "bar"}, labelset)
	expectMatchDirect(t, Set{"baz": "blah"}, labelset)
	expectMatchDirect(t, Set{"foo": "bar", "baz": "blah"}, labelset)
	expectNoMatchDirect(t, Set{"foo": "=blah"}, labelset)
	expectNoMatchDirect(t, Set{"baz": "=bar"}, labelset)
	expectNoMatchDirect(t, Set{"foo": "=bar", "foobar": "bar", "baz": "blah"}, labelset)
}

func TestNilMapIsValid(t *testing.T) {
	selector := Set(nil).AsSelector()
	if selector == nil {
		t.Errorf("Selector for nil set should be Everything")
	}
	if !selector.Empty() {
		t.Errorf("Selector for nil set should be Empty")
	}
}

func TestSetIsEmpty(t *testing.T) {
	if !(Set{}).AsSelector().Empty() {
		t.Errorf("Empty set should be empty")
	}
	if !(andTerm(nil)).Empty() {
		t.Errorf("Nil andTerm should be empty")
	}
	if (&hasTerm{}).Empty() {
		t.Errorf("hasTerm should not be empty")
	}
	if !(andTerm{andTerm{}}).Empty() {
		t.Errorf("Nested andTerm should be empty")
	}
	if (andTerm{&hasTerm{"a", "b"}}).Empty() {
		t.Errorf("Nested andTerm should not be empty")
	}
}

func expectMatchRequirement(t *testing.T, req Requirement, ls Set) {
	if !req.Matches(ls) {
		t.Errorf("Wanted '%+v' to match '%s', but it did not.\n", req, ls)
	}
}

func expectNoMatchRequirement(t *testing.T, req Requirement, ls Set) {
	if req.Matches(ls) {
		t.Errorf("Wanted '%+v' to not match '%s', but it did.", req, ls)
	}
}

func TestRequirementMatches(t *testing.T) {
	s := Set{"x": "foo", "y": "baz"}
	a := Requirement{key: "x", comparator: IN, strValues: []string{"foo"}}
	b := Requirement{key: "x", comparator: NOT_IN, strValues: []string{"beta"}}
	c := Requirement{key: "y", comparator: IN, strValues: nil}
	d := Requirement{key: "y", strValues: []string{"foo"}}
	expectMatchRequirement(t, a, s)
	expectMatchRequirement(t, b, s)
	expectNoMatchRequirement(t, c, s)
	expectNoMatchRequirement(t, d, s)
}

func expectMatchLabSelector(t *testing.T, lsel LabelSelector, s Set) {
	if !lsel.Matches(s) {
		t.Errorf("Wanted '%+v' to match '%s', but it did not.\n", lsel, s)
	}
}

func expectNoMatchLabSelector(t *testing.T, lsel LabelSelector, s Set) {
	if lsel.Matches(s) {
		t.Errorf("Wanted '%+v' to not match '%s', but it did.\n", lsel, s)
	}
}

func TestLabelSelectorMatches(t *testing.T) {
	s := Set{"x": "foo", "y": "baz"}
	allMatch := LabelSelector{
		Requirements: []Requirement{
			{key: "x", comparator: IN, strValues: []string{"foo"}},
			{key: "y", comparator: NOT_IN, strValues: []string{"alpha"}},
		},
	}
	singleNonMatch := LabelSelector{
		Requirements: []Requirement{
			{key: "x", comparator: IN, strValues: []string{"foo"}},
			{key: "y", comparator: IN, strValues: []string{"alpha"}},
		},
	}
	expectMatchLabSelector(t, allMatch, s)
	expectNoMatchLabSelector(t, singleNonMatch, s)
}
