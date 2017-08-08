package index

import (
	"errors"
	"fmt"
	s "sort"

	"github.com/digitalrebar/store"
)

type Indexer interface {
	Indexes() map[string]Maker
}

// Tester is a function that tests to see if an item matches a condition
type Test func(store.KeySaver) bool

// Less is a function that tests to see if the first item is less than
// the second item.
type Less func(store.KeySaver, store.KeySaver) bool

// TestMaker is a function that takes a reference object and spits out
// appropriate Tests for gte and gt, in that order.
type TestMaker func(store.KeySaver) (Test, Test)

// Filler takes a value from a query parameter and plugs it into the
// appropriate slot in the proper model for an index.  It is
// responsible for doing whatever conversion is needed to translate
// from a string to the appropriate type for the index.
type Filler func(string) (store.KeySaver, error)

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
//
// Fill, which takes a string from a query parameter and turns it into
// a store.KeySaver that has the appropriate slot filled.
type Maker struct {
	keyOrder bool
	Unique   bool
	Type     string
	Less     Less
	Tests    TestMaker
	Fill     Filler
}

// Index declares a struct field that can be indexed for a given
// Model, along with the function that should be used to sort things
// in key order.  Unless otherwise indicated, any methods that act on an Index
// return a new Index with its own reference to indexed items.
type Index struct {
	Maker
	sorted bool
	base   bool
	objs   []store.KeySaver
}

// Make takes a Less function, a TestMaker function, and a Filler
// function and returns a Maker.  t is a textual type identifier for docs/helps
func Make(unique bool, t string, less Less, maker TestMaker, filler Filler) Maker {
	return Maker{Unique: unique, Type: t, Less: less, Tests: maker, Fill: filler}
}

func Create(objs []store.KeySaver) *Index {
	res := &Index{Maker: MakeKey(), sorted: true, base: true, objs: objs}
	s.Slice(res.objs, func(j, k int) bool { return res.Less(res.objs[j], res.objs[k]) })
	return res
}

type Fake string

func (f Fake) Prefix() string       { return "fake" }
func (f Fake) Key() string          { return string(f) }
func (f Fake) New() store.KeySaver  { return f }
func (f Fake) Backend() store.Store { return nil }

func MakeKey() Maker {
	return Maker{
		keyOrder: true,
		Unique:   true,
		Less: func(i, j store.KeySaver) bool {
			return i.Key() < j.Key()
		},
		Tests: func(ref store.KeySaver) (gte, gt Test) {
			key := ref.Key()
			return func(s store.KeySaver) bool { return s.Key() >= key },
				func(s store.KeySaver) bool { return s.Key() > key }
		},
		Fill: func(s string) (store.KeySaver, error) {
			return Fake(s), nil
		},
	}
}

func CheckUnique(s store.KeySaver, objs []store.KeySaver) error {
	testObj, ok := s.(Indexer)
	if !ok {
		return nil
	}
	for idxName, maker := range testObj.Indexes() {
		if !maker.Unique {
			continue
		}
		idx, err := All(Sort(maker), Subset(maker.Tests(s)))(New(objs))
		if err != nil {
			return err
		}
		switch len(idx.objs) {
		case 0:
			continue
		case 1:
			if idx.objs[0].Key() == s.Key() {
				continue
			}
		}
		return fmt.Errorf("%s:%s violates unique index %s", s.Prefix(), s.Key(), idxName)
	}
	return nil
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

func (i *Index) find(ref store.KeySaver) (int, bool) {
	gte, gt := i.Tests(ref)
	idx := s.Search(len(i.objs), func(j int) bool { return gte(i.objs[j]) })
	if idx == len(i.objs) {
		return idx, false
	}
	return idx, !gt(i.objs[idx])
}

func (i *Index) Find(key string) store.KeySaver {
	ref, err := i.Fill(key)
	if err == nil {
		if idx, found := i.find(ref); found {
			return i.objs[idx]
		}
	}
	return nil
}

func (i *Index) Add(items ...store.KeySaver) error {
	if !i.base {
		return fmt.Errorf("Can only add to a base index")
	}
	if !i.sorted {
		return fmt.Errorf("Cannot add items to a non-sorted Index")
	}
	growers, appenders := []store.KeySaver{}, []store.KeySaver{}
	growIndexes := []int{}
	for _, obj := range items {
		idx, found := i.find(obj)
		if found {
			i.objs[idx] = obj
			continue
		}
		if idx == len(i.objs) {
			appenders = append(appenders, obj)
			continue
		}
		growers = append(growers, obj)
		growIndexes = append(growIndexes, idx)
	}
	// Append new objects.  We do this before growing the slice to minimize the amount of
	// memory that potentially has to be copied.
	if len(appenders) > 0 {
		s.Slice(appenders, func(j, k int) bool { return i.Less(appenders[j], appenders[k]) })
		i.objs = append(i.objs, appenders...)
	}
	// Insert new items in sorted order with minimal data copying
	if len(growers) > 0 {
		s.Ints(growIndexes)
		s.Slice(growers, func(j, k int) bool {
			return i.Less(growers[j], growers[k])
		})
		oldLen := len(i.objs)
		i.objs = append(i.objs, make([]store.KeySaver, len(growers))...)
		for j := len(growIndexes) - 1; j >= 0; j-- {
			idx := growIndexes[j]
			dest, src := i.objs[idx+j+1:], i.objs[idx:oldLen]
			copy(dest, src)
			i.objs[idx+j] = growers[j]
			oldLen = idx
		}
	}
	return nil
}

func (i *Index) Remove(items ...store.KeySaver) error {
	if !i.base {
		return fmt.Errorf("Cannot remove from a non-base Indes")
	}
	if !i.sorted {
		return fmt.Errorf("Cannot remove items from a non-sorted Index")
	}
	idxs := []int{}
	for j := range items {
		idx, found := i.find(items[j])
		if !found {
			continue
		}
		idxs = append(idxs, idx)
	}
	if len(idxs) == 0 {
		return nil
	}
	s.Ints(idxs)
	lastDT := len(i.objs)
	lastIdx := len(idxs) - 1
	// Progressively copy over slices to overwrite entries we are
	// deleting
	for j, idx := range idxs {
		if idx == lastDT-1 {
			continue
		}
		var srcend int
		if j != lastIdx {
			srcend = idxs[j+1]
		} else {
			srcend = lastDT
		}
		// copy(dest, src)
		copy(i.objs[idx-j:srcend], i.objs[idx+1:srcend])
	}
	// Nil out entries that we should garbage collect.  We do this
	// so that we don't wind up leaking items based on the
	// underlying arrays still pointing at things we no longer
	// care about.
	for j := range idxs {
		i.objs[lastDT-j-1] = nil
	}
	// Resize dt.d to forget about the last elements.  This does
	// not always resize the underlying array, hence the above
	// manual GC enablement.
	//
	// At some point we may want to manually switch to a smaller
	// underlying array based on len() vs. cap() for dt.d, but
	// probably only when we can potentially free a significant
	// amount of memory by doing so.
	i.objs = i.objs[:len(i.objs)-len(idxs)]
	return nil
}

func (i *Index) Count() int {
	return len(i.objs)
}

func (i *Index) Empty() bool {
	return i.objs == nil || i.Count() == 0
}

// cp makes a copy of the current Index with a copy of the passed-in
// slice of objects that the index should reference.
func (i *Index) cp(newObjs []store.KeySaver) *Index {
	objs := make([]store.KeySaver, len(newObjs))
	copy(objs, newObjs)
	return &Index{
		Maker:  i.Maker,
		sorted: i.sorted,
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
	start := s.Search(totalCount, func(j int) bool { return lower(i.objs[j]) })
	end := s.Search(totalCount, func(j int) bool { return upper(i.objs[j]) })
	return i.cp(i.objs[start:end])
}

// Filter is a function that takes an index, does stuff with it, and
// returns another index
type Filter func(*Index) (*Index, error)

// All returns a filter that returns an index tha contains items that
// pass all the filters.
func All(filters ...Filter) Filter {
	return func(i *Index) (*Index, error) {
		var err error
		for _, filter := range filters {
			i, err = filter(i)
			if err != nil {
				break
			}
		}
		return i, err
	}
}

// Any returns a filter that returns an index that contains items that
// pass any of the filters.  There may be duplicate items.
func Any(filters ...Filter) Filter {
	return func(i *Index) (*Index, error) {
		items := []store.KeySaver{}
		for j := range filters {
			q, err := filters[j](i)
			if err != nil {
				return i, err
			}
			items = append(items, q.objs...)
		}
		res := i.cp(items)
		res.sorted = false
		return res, nil
	}
}

func nativeLess(a, b store.KeySaver) bool {
	return a.Key() < b.Key()
}

// Sort returns a filter that sorts an index references in a stable
// fashion based on the passed-in Less function.
func sort(l Less) Filter {
	return func(i *Index) (*Index, error) {
		res := i.cp(i.objs)
		less := func(j, k int) bool { return l(res.objs[j], res.objs[k]) }
		s.SliceStable(res.objs, less)
		res.sorted = true
		return res, nil
	}
}

// Native returns a filter that will sort an Index based on key order.
func Native() Filter {
	return sort(nativeLess)
}

// Resort sorts the index with the same function passed in to the most
// recent Sort filter.
func Resort() Filter {
	return func(i *Index) (*Index, error) {
		return sort(i.Less)(i)
	}
}

func Sort(m Maker) Filter {
	return func(i *Index) (*Index, error) {
		j := *i
		j.Maker = m
		return sort(j.Less)(&j)
	}
}

// A couple of helper functions for using Subset to implement the
// usual comparison operators
func alwaysFalse(store.KeySaver) bool { return false }
func alwaysTrue(store.KeySaver) bool  { return true }

// Subset returns a filter that will sort an index based on the lower
// bound test and the upper bound tests, which correspond to the gte
// and gt functions that SetComparators takes.
func Subset(lower, upper Test) Filter {
	return func(i *Index) (*Index, error) {
		return i.subset(lower, upper), nil
	}
}

func Between(lower, upper string) Filter {
	return func(i *Index) (*Index, error) {
		lRef, err := i.Fill(lower)
		if err != nil {
			return i, err
		}
		uRef, err := i.Fill(upper)
		if err != nil {
			return i, err
		}
		lTest, _ := i.Tests(lRef)
		_, uTest := i.Tests(uRef)
		return i.subset(lTest, uTest), nil
	}
}

func Except(lower, upper string) Filter {
	return func(i *Index) (*Index, error) {
		lRef, err := i.Fill(lower)
		if err != nil {
			return i, err
		}
		uRef, err := i.Fill(upper)
		if err != nil {
			return i, err
		}
		lTest, _ := i.Tests(lRef)
		_, uTest := i.Tests(uRef)
		lowerParts := i.subset(alwaysTrue, lTest)
		upperParts := i.subset(uTest, alwaysFalse)
		lowerParts.objs = append(lowerParts.objs, upperParts.objs...)
		return lowerParts, nil
	}
}

// Lt returns a filter that will keep all items that are less than the
// current comparators
func Lt(ref string) Filter {
	return func(i *Index) (*Index, error) {
		refTest, err := i.Fill(ref)
		if err != nil {
			return i, err
		}
		upper, _ := i.Tests(refTest)
		return i.subset(alwaysTrue, upper), nil
	}
}

// Lte returns a filter that will keep all the items that are less
// than or equal to the current comparators.
func Lte(ref string) Filter {
	return func(i *Index) (*Index, error) {
		refTest, err := i.Fill(ref)
		if err != nil {
			return i, err
		}
		_, upper := i.Tests(refTest)
		return i.subset(alwaysTrue, upper), nil
	}
}

// Eq returns a filter that will keep all the items that are equal to
// the current comparators.
func Eq(ref string) Filter {
	return func(i *Index) (*Index, error) {
		refTest, err := i.Fill(ref)
		if err != nil {
			return i, err
		}
		lower, upper := i.Tests(refTest)
		return i.subset(lower, upper), nil
	}
}

// Gte returns a filter that will keep all the items that are greater
// than or equal to the current comparators
func Gte(ref string) Filter {
	return func(i *Index) (*Index, error) {
		refTest, err := i.Fill(ref)
		if err != nil {
			return i, err
		}
		lower, _ := i.Tests(refTest)
		return i.subset(lower, alwaysFalse), nil
	}
}

// Gt returns a filter that will keep all the items that are
// greater-than the current comparators
func Gt(ref string) Filter {
	return func(i *Index) (*Index, error) {
		refTest, err := i.Fill(ref)
		if err != nil {
			return i, err
		}
		_, lower := i.Tests(refTest)
		return i.subset(lower, alwaysFalse), nil
	}
}

// Ne returns a filter that will keep all the items that are not equal
// to the current comparators.
func Ne(ref string) Filter {
	return func(i *Index) (*Index, error) {
		refTest, err := i.Fill(ref)
		if err != nil {
			return i, err
		}
		lower, upper := i.Tests(refTest)
		lowerParts := i.subset(alwaysTrue, lower)
		upperParts := i.subset(upper, alwaysFalse)
		lowerParts.objs = append(lowerParts.objs, upperParts.objs...)
		return lowerParts, nil
	}
}

// Select returns a filter that picks all items that match the passed
// Test.  It does not rely on the Index being sorted in any particular
// order.
func Select(t Test) Filter {
	return func(i *Index) (*Index, error) {
		objs := []store.KeySaver{}
		for _, obj := range i.objs {
			if t(obj) {
				objs = append(objs, obj)
			}
		}
		return i.cp(objs), nil
	}
}

// Offset returns a filter that returns all but the first n items
func Offset(num int) Filter {
	return func(i *Index) (*Index, error) {
		if num < 0 {
			return i, errors.New("Offset cannot be negative")
		}
		if num >= len(i.objs) {
			return i.cp([]store.KeySaver{}), nil
		}
		return i.cp(i.objs[num:]), nil
	}
}

// Limit returns a filter that returns the first n items
func Limit(num int) Filter {
	return func(i *Index) (*Index, error) {
		if num < 0 {
			return i, errors.New("Limit cannot be negative")
		}
		if num > len(i.objs) {
			return i, nil
		}
		return i.cp(i.objs[:num]), nil
	}
}

// Reverse returns a filter that will reverse an index.
func Reverse() Filter {
	return func(i *Index) (*Index, error) {
		res := i.cp(i.objs)
		res.sorted = false
		for lower, upper := 0, len(res.objs)-1; lower < upper; lower, upper = lower+1, upper-1 {
			res.objs[lower], res.objs[upper] = res.objs[upper], res.objs[lower]
		}
		return res, nil
	}
}
