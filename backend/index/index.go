package index

import (
	"sort"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

// Tester is a function that tests to see if an item matches a condition
type Test func(store.KeySaver) bool

// Less is a function that tests to see if the first item is less than the second item.
type Less func(store.KeySaver, store.KeySaver) bool

// TestMaker is a function that takes a reference object and spits out
// appropriate Tests for gte and gt, in that order.
type TestMaker func(store.KeySaver) (Test, Test)

// Index declares a struct field that can be indexed for a given
// Model, along with the function that should be used to sort things
// in key order.  Unless otherwise indicated, any methods that act on an Index
// return a new Index with its own reference to indexed items.
type Index struct {
	less          Less
	gtRef, gteRef Test
	sorted        bool
	objs          []store.KeySaver
}

// Maker is used to hold reference functions for a specific index on a
// specific struct.  The functions are:
//
// Less, which compares one item in the index with another, and
// returns true if it is.  It is used to sort values in the index.
//
// Tests, which takes an example item of the type indexed with the
// appropriate reference field filled in.  It returns a pair of
// functions that satisfy the constraints of the SetComparators filter
// and the Subset filter.
type Maker struct {
	Less  Less
	Tests func(store.KeySaver) (Test, Test)
}

// Make takes a Less function and a TestMaker function and returns a
// Maker.
func Make(less Less, maker TestMaker) Maker {
	return Maker{Less: less, Tests: maker}
}

// New returns a new Index that is populated with a copy of the
// passed-in objs.
func New(objs []store.KeySaver) *Index {
	res := &Index{}
	res.objs = make([]store.KeySaver, len(objs))
	copy(res.objs, objs)
	return res
}

// Items returns the current items the index has filtered.
func (i *Index) Items() []store.KeySaver {
	res := make([]store.KeySaver, len(i.objs))
	copy(res, i.objs)
	return res
}

// SetItems returns a new *Index with its items set to s.
// The new Index is not sorted.
func (i *Index) SetItems(s []store.KeySaver) *Index {
	res := i.cp(s)
	res.sorted = false
	return res
}

// cp makes a copy of the current Index with a copy of the passed-in
// slice of objects that the index should reference.
func (i *Index) cp(newObjs []store.KeySaver) *Index {
	objs := make([]store.KeySaver, len(newObjs))
	copy(objs, newObjs)
	return &Index{
		less:   i.less,
		sorted: i.sorted,
		gtRef:  i.gtRef,
		gteRef: i.gteRef,
		objs:   objs,
	}
}

// Subset causes the index to discard all elements that fall outside
// the first index for which lower returns true and the first index
// for which upper returns true.  The index must be sorted first, or
// Subset will panic.
//
// lower and upper should be thunks that examine the object passed to
// determine where the subset should start and end at, and must choose
// items based on what the index is currently sorted by.
func (i *Index) subset(lower, upper Test) *Index {
	if !i.sorted {
		panic("Cannot take subset of unsorted index")
	}
	totalCount := len(i.objs)
	start := sort.Search(totalCount, func(j int) bool { return lower(i.objs[j]) })
	var objs []store.KeySaver
	if start == totalCount {
		objs = []store.KeySaver{}
	} else {
		objs = i.objs[start:sort.Search(totalCount, func(j int) bool { return upper(i.objs[j]) })]
	}
	return i.cp(objs)
}

// Filter is a function that takes an index, does stuff with it, and
// returns another index
type Filter func(*Index) *Index

// All returns a filter that returns an index tha contains items that
// pass all the filters.
func All(filters ...Filter) Filter {
	return func(i *Index) *Index {
		for _, filter := range filters {
			i = filter(i)
		}
		return i
	}
}

// Any returns a filter that returns an index that contains items that
// pass any of the filters.  There may be duplicate items.
func Any(filters ...Filter) Filter {
	return func(i *Index) *Index {
		items := []store.KeySaver{}
		for j := range filters {
			items = append(items, filters[j](i).objs...)
		}
		res := i.cp(items)
		res.sorted = false
		return res
	}
}

func nativeLess(a, b store.KeySaver) bool {
	return a.Key() < b.Key()
}

// Sort returns a filter that sorts an index references in a stable
// fashion based on the passed-in Less function.
func Sort(l Less) Filter {
	return func(i *Index) *Index {
		res := i.cp(i.objs)
		res.less = l
		less := func(j, k int) bool { return l(res.objs[j], res.objs[k]) }
		sort.SliceStable(res.objs, less)
		res.sorted = true
		return res
	}
}

// Native returns a filter that will sort an Index based on key order.
func Native() Filter {
	return Sort(nativeLess)
}

// Resort sorts the index with the same function passed in to the most
// recent Sort filter.
func Resort() Filter {
	return func(i *Index) *Index {
		return Sort(i.less)(i)
	}
}

// A couple of helper functions for using Subset to implement the
// usual comparison operators
func alwaysFalse(store.KeySaver) bool { return false }
func alwaysTrue(store.KeySaver) bool  { return true }

// SetComparators allows you to add static comparators that test for
// greater-than and greater-than-or-equal against values stored in the
// Index.  The comaprators must test the field that the index is
// sorted by, or you will get nonsensical results for the comparison
// operators.
//
// Once comparators are set for an index, the usual comparison methods
// (Lt, Eq, Gt, etc.)  will work without panicing, and will be
// inherited by any Indexes created from this one.
func SetComparators(gt, gte Test) Filter {
	return func(i *Index) *Index {
		res := i.cp(i.objs)
		res.gtRef = gt
		res.gteRef = gte
		return res
	}
}

// Subset returns a filter that will sort an index based on the lower
// bound test and the upper bound tests, which correspond to the gte
// and gt functions that SetComparators takes.
func Subset(lower, upper Test) Filter {
	return func(i *Index) *Index {
		return i.subset(lower, upper)
	}
}

// Lt returns a filter that will keep all items that are less than the
// current comparators
func Lt() Filter {
	return func(i *Index) *Index {
		return i.subset(alwaysFalse, i.gteRef)
	}
}

// Lte returns a filter that will keep all the items that are less
// than or equal to the current comparators.
func Lte() Filter {
	return func(i *Index) *Index {
		return i.subset(alwaysFalse, i.gtRef)
	}
}

// Eq returns a filter that will keep all the items that are equal to
// the current comparators.
func Eq() Filter {
	return func(i *Index) *Index {
		return i.subset(i.gteRef, i.gtRef)
	}
}

// Gte returns a filter that will keep all the items that are greater
// than or equal to the current comparators
func Gte() Filter {
	return func(i *Index) *Index {
		return i.subset(i.gteRef, alwaysTrue)
	}
}

// Gt returns a filter that will keep all the items that are
// greater-than the current comparators
func Gt() Filter {
	return func(i *Index) *Index {
		return i.subset(i.gtRef, alwaysTrue)
	}
}

// Ne returns a filter that will keep all the items that are not equal
// to the current comparators.
func Ne() Filter {
	return func(i *Index) *Index {
		lt := Lt()(i)
		gt := Gt()(i)
		objs := lt.objs
		objs = append(objs, gt.objs...)
		return i.cp(objs)
	}
}

// Select returns a filter that picks all items that match the passed
// Test.  It does not rely on the Index being sorted in any particular
// order.
func Select(t Test) Filter {
	return func(i *Index) *Index {
		objs := []store.KeySaver{}
		for _, obj := range i.objs {
			if t(obj) {
				objs = append(objs, obj)
			}
		}
		return i.cp(objs)
	}
}

// Offset returns a filter that returns all but the first n items
func Offset(num int) Filter {
	return func(i *Index) *Index {
		return i.cp(i.objs[num:])
	}
}

// Limit returns a filter that returns the first n items
func Limit(num int) Filter {
	return func(i *Index) *Index {
		return i.cp(i.objs[:num])
	}
}

// Reverse returns a filter that will reverse an index.
func Reverse() Filter {
	return func(i *Index) *Index {
		res := i.cp(i.objs)
		res.sorted = false
		for lower, upper := 0, len(res.objs)-1; lower < upper; lower, upper = lower+1, upper-1 {
			res.objs[lower], res.objs[upper] = res.objs[upper], res.objs[lower]
		}
		return res
	}
}
