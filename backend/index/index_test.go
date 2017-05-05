package index

import (
	"strconv"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

type testThing int64

func (t testThing) Key() string {
	return strconv.FormatInt(int64(t), 10)
}

func (t testThing) Prefix() string {
	return "integers"
}

func (t testThing) New() store.KeySaver {
	return t
}

func (t testThing) Backend() store.SimpleStore {
	return nil
}

func (t testThing) Indexes() map[string]Maker {
	return map[string]Maker{
		"Base": Make(
			true,
			func(i, j store.KeySaver) bool {
				return i.(testThing) < j.(testThing)
			},
			func(ref store.KeySaver) (gte, gt Test) {
				return func(s store.KeySaver) bool {
						return s.(testThing) >= ref.(testThing)
					},
					func(s store.KeySaver) bool {
						return s.(testThing) > ref.(testThing)
					}
			},
			func(s string) (store.KeySaver, error) {
				res, err := strconv.ParseInt(s, 10, 64)
				return testThing(res), err
			}),
	}
}

func matchIdx(t *testing.T, i *Index, ints ...int64) {
	if len(i.objs) != len(ints) {
		t.Errorf("Expected %d items, got %d", len(ints), len(i.objs))
	}
	for j := range ints {
		if int64(i.objs[j].(testThing)) != ints[j] {
			t.Errorf("At position %d, expected %d, got %d", j, ints[j], i.objs[j])
		}
	}
}

func TestIndexes(t *testing.T) {
	objs := make([]store.KeySaver, 100)
	for i := range objs {
		objs[i] = testThing(len(objs) - i)
	}
	idx := New(objs)
	lim, err := Limit(10)(idx)
	if err != nil {
		t.Errorf("Limit returned an unexpected error: %v", err)
	}
	if len(lim.objs) != 10 {
		t.Errorf("Limit failed to limit returned 10 items, returned %d", len(lim.objs))
	} else {
		t.Logf("Limit returned 10 items")
	}
	offs, err := Offset(5)(lim)
	if err != nil {
		t.Errorf("Offset returned an unexpected error: %v", err)
	}
	if len(offs.objs) != 5 {
		t.Errorf("Offset failed to return 5 items, returned %d", len(offs.objs))
	} else {
		t.Logf("Offset returned 5 items")
	}
	indexes := testThing(0).Indexes()
	offs, err = Sort(indexes["Base"])(offs)
	if err != nil {
		t.Errorf("Sort returned an unexpected error: %v", err)
	}
	matchIdx(t, offs, 91, 92, 93, 94, 95)
	idx, err = All(Reverse(), Reverse())(idx)
	if err != nil {
		t.Errorf("Reverse returned an unexpected error: %v", err)
	}
	lim, err = Limit(10)(idx)
	if err != nil {
		t.Errorf("Unexpected error taking the limit: %v", err)
	}
	matchIdx(t, lim, 100, 99, 98, 97, 96, 95, 94, 93, 92, 91)
	idx, err = Sort(indexes["Base"])(idx)
	if err != nil {
		t.Errorf("Unexpected error sorting base: %v", err)
	}
	lim, err = Limit(10)(idx)
	if err != nil {
		t.Errorf("Unexpected error taking the limit: %v", err)
	}
	matchIdx(t, lim, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	tween, err := Between("5", "10")(idx)
	if err != nil {
		t.Errorf("Unexpected error processing Between: %v", err)
	}
	matchIdx(t, tween, 5, 6, 7, 8, 9, 10)
	_, err = Limit(-1)(tween)
	if err == nil {
		t.Errorf("Limit should have thrown an error with a negative limit")
	} else {
		t.Logf("Got expected error taking negative limit: %v", err)
	}
	matchIdx(t, tween, 5, 6, 7, 8, 9, 10)
	_, err = Offset(-1)(tween)
	if err == nil {
		t.Errorf("Offset should have thrown an error with a negative offset")
	} else {
		t.Logf("Got expected error taking negative offset: %v", err)
	}
	matchIdx(t, tween, 5, 6, 7, 8, 9, 10)
	lim, err = Limit(10)(tween)
	if err != nil {
		t.Errorf("Got unexpected error taking limit: %v", err)
	}
	matchIdx(t, lim, 5, 6, 7, 8, 9, 10)
	offs, err = Offset(10)(tween)
	if err != nil {
		t.Errorf("Got unexpected error taking limit: %v", err)
	}
	matchIdx(t, offs)
	tween, err = Except("6", "9")(tween)
	if err != nil {
		t.Errorf("Got unexpected error taking except: %v", err)
	}
	matchIdx(t, tween, 5, 10)
	lt, err := Lt("6")(idx)
	if err != nil {
		t.Errorf("Got unexpected error running Lt: %v", err)
	}
	matchIdx(t, lt, 1, 2, 3, 4, 5)
	lt, err = Lte("6")(idx)
	if err != nil {
		t.Errorf("Got unexpected error running Lte: %v", err)
	}
	matchIdx(t, lt, 1, 2, 3, 4, 5, 6)
	lt, err = Eq("6")(idx)
	if err != nil {
		t.Errorf("Got unexpected error running Eq: %v", err)
	}
	matchIdx(t, lt, 6)
	lt, err = Gte("95")(idx)
	if err != nil {
		t.Errorf("Got unexpected error running Gte: %v", err)
	}
	matchIdx(t, lt, 95, 96, 97, 98, 99, 100)
	lt, err = Gt("95")(idx)
	if err != nil {
		t.Errorf("Got unexpected error running Gt: %v", err)
	}
	matchIdx(t, lt, 96, 97, 98, 99, 100)
	lt, err = Ne("98")(lt)
	if err != nil {
		t.Errorf("Got unexpected error running Ne: %v", err)
	}
	matchIdx(t, lt, 96, 97, 99, 100)
	lt, err = Select(func(s store.KeySaver) bool { return s.(testThing)%2 == 0 })(lt)
	if err != nil {
		t.Errorf("Got unexpected error running Select: %v", err)
	}
	matchIdx(t, lt, 96, 100)
	ref, err := idx.Fill("6")
	if err != nil {
		t.Errorf("Unexpected error creating reference testThing from `6`")
	}
	lower, upper := idx.Tests(ref)
	sub, err := Subset(lower, upper)(idx)
	if err != nil {
		t.Errorf("Got unexpected error running Subset: %v", err)
	}
	matchIdx(t, sub, 6)
}
