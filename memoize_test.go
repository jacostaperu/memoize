package memoize_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jacostaperu/memoize"
)

func TestPanic(t *testing.T) {
	count := 0
	f := func(i int) {
		count++
		if count%2 == 1 {
			panic(count)
		}
	}

	cache := memoize.NewMemoizer(90*time.Second, 10*time.Minute)

	f = cache.Memoize(f).(func(int))

	expect := func(p interface{}, i int) {
		defer func() {
			if r := recover(); p != r {
				t.Errorf("for input %d:\nexpected: %v\nactual: %v", i, p, r)
			}
		}()

		f(i)
	}

	expect(1, 1)
	expect(1, 1)
	expect(nil, 2)
	expect(nil, 2)
	expect(1, 1)
	expect(3, 100)
}

func TestVariadic(t *testing.T) {
	count := 0
	var concat func(string, ...string) string
	concat = func(s0 string, s1 ...string) string {
		count++

		if len(s1) == 0 {
			return s0
		}
		return concat(s0+s1[0], s1[1:]...)
	}
	cache := memoize.NewMemoizer(90*time.Second, 10*time.Minute)

	concat = cache.Memoize(concat).(func(string, ...string) string)

	expect := func(actual, expected string, n int) {
		if actual != expected || n != count {
			t.Errorf("expected: %q\nactual: %q\nexpected count: %d\nactual count: %d", expected, actual, n, count)
		}
	}

	expect("", "", 0)
	expect(concat("string"), "string", 1)
	expect(concat("string", "one"), "stringone", 3)
	expect(concat("string", "one"), "stringone", 3)
	expect(concat("string", "two"), "stringtwo", 5)
	expect(concat("string", "one"), "stringone", 5)
	expect(concat("stringone", "two"), "stringonetwo", 7)
	expect(concat("string", "one", "two"), "stringonetwo", 8)
}

func fib(n uint64) uint64 {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	} else {
		return fib(n-1) + fib(n-2)
	}
}

func TestFibNoMemoize(t *testing.T) {
	lastValue2 := fib(47)
	fmt.Println("lastValue1", lastValue2)
}

func TestSimple(t *testing.T) {
	fmt.Println("hola Mundo")

	cache := memoize.NewMemoizer(900*time.Second, 100*time.Minute)

	expensi := cache.Memoize(fib).(func(uint64) uint64)

	newFib := func(n uint64) uint64 {
		if n == 0 {
			return 0
		} else if n == 1 {
			return 1
		} else {
			return expensi(n-1) + expensi(n-2)
		}

	}

	newFibo := cache.Memoize(newFib).(func(uint64) uint64)

	for i := 0; i < 100; i++ {

		fibbonaci := newFibo(uint64(i))
		fmt.Printf("fib(%d)=%d\n", i, fibbonaci)
	}

	firstValue := expensi(10)
	fmt.Println("firstValue", firstValue)
	lastValue := expensi(10)
	fmt.Println("lastValue", lastValue)
	lastValue1 := expensi(11)
	fmt.Println("lastValue1", lastValue1)

	if firstValue != lastValue {
		t.Errorf(`memoize not working`)
	}
	if firstValue == lastValue1 {
		t.Errorf(`memoize not working`)
	}
}
