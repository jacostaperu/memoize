// Package memoize caches return values of functions.
package memoize

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

var interfaceType = reflect.TypeOf(new(interface{})).Elem()
var valueType = reflect.TypeOf(new(call))

type call struct {
	wait     <-chan struct{}
	results  []reflect.Value
	panicked reflect.Value
}

// Memoizer allows you to memoize function calls. Memoizer is safe for concurrent use by multiple goroutines.
type Memoizer struct {

	// Storage exposes the underlying cache of memoized results to manipulate as desired - for example, to Flush().
	Storage *cache.Cache

	group singleflight.Group
}

// Memoize takes a function and returns a function of the same type. The
// returned function remembers the return value(s) of the function call.
// Any pointer values will be used as an address, so functions that modify
// their arguments or programs that modify returned values will not work.
//
// The returned function is safe to call from multiple goroutines if the
// original function is. Panics are handled, so calling panic from a function
// will call panic with the same value on future invocations with the same
// arguments.
//
// The arguments to the function must be of comparable types. Slices, maps,
// functions, and structs or arrays that contain slices, maps, or functions
// cause a runtime panic if they are arguments to a memoized function.
// See also: https://golang.org/ref/spec#Comparison_operators
//

// NewMemoizer creates a new Memoizer with the configured expiry and cleanup policies.
// If desired, use cache.NoExpiration to cache values forever.
func NewMemoizer(defaultExpiration, cleanupInterval time.Duration) *Memoizer {
	return &Memoizer{
		Storage: cache.New(defaultExpiration, cleanupInterval),
		group:   singleflight.Group{},
	}
}

//AsSha256 hash a function
func AsSha256(o interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", o)))

	return fmt.Sprintf("%x", h.Sum(nil))
}

//Memoize   As a special case, variadic functions (func(x, y, ...z)) are allowed.
func (m *Memoizer) Memoize(fn interface{}) interface{} {
	v := reflect.ValueOf(fn)

	t := v.Type()

	keyType := reflect.ArrayOf(t.NumIn(), interfaceType)
	//cache := reflect.MakeMap(reflect.MapOf(keyType, valueType))
	//will be replaces by cache := m.Storage

	var mtx sync.Mutex

	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		key := reflect.New(keyType).Elem()
		for i, v := range args {

			if i == len(args)-1 && t.IsVariadic() {
				a := reflect.New(reflect.ArrayOf(v.Len(), v.Type().Elem())).Elem()
				for j, l := 0, v.Len(); j < l; j++ {
					a.Index(j).Set(v.Index(j))
				}
				v = a
			}
			vi := v.Interface()
			key.Index(i).Set(reflect.ValueOf(&vi).Elem())
		}
		mtx.Lock()

		mykey := AsSha256(key)

		//fmt.Println("mykey -->", mykey, "<--")

		//val := cache.MapIndex(key)
		mval, found := m.Storage.Get(mykey)

		if found {
			mtx.Unlock()

			//fmt.Println("val", mval)
			//fmt.Println("return cached value.")
			//c := val.(*call)
			c := mval.(reflect.Value).Interface().(*call)
			//fmt.Println("val", c)

			//c := val.Interface().(*call)

			<-c.wait
			if c.panicked.IsValid() {
				panic(c.panicked.Interface())
			}
			return c.results
		}
		//fmt.Println("value not cached")

		w := make(chan struct{})
		c := &call{wait: w}
		//cache.SetMapIndex(key, reflect.ValueOf(c))
		m.Storage.Set(mykey, reflect.ValueOf(c), cache.DefaultExpiration)

		//mvall, found := m.Storage.Get(mykey)
		//myyval, _ := json.Marshal(mvall)
		//fmt.Println("Value a Guardar            :", reflect.ValueOf(c))
		//fmt.Println("Value a Guardado recuperado:", mvall)

		mtx.Unlock()

		panicked := true
		defer func() {
			if panicked {
				p := recover()
				c.panicked = reflect.ValueOf(p)
				close(w)
				panic(p)
			}
		}()

		if t.IsVariadic() {
			results = v.CallSlice(args)
		} else {
			results = v.Call(args)
		}
		panicked = false
		c.results = results
		close(w)

		return
	}).Interface()
}
