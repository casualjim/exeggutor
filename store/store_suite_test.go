package store

import (
	"sort"
	"strconv"

	. "github.com/smartystreets/goconvey/convey"
)

// func TestState(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	pth := fmt.Sprintf("../test-reports/junit_exeggutor_store_%d.xml", config.GinkgoConfig.ParallelNode)
// 	junitReporter := reporters.NewJUnitReporter(pth)
// 	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor Store Test Suite", []Reporter{junitReporter})
// }

type StoreExampleContext struct {
	Backing map[string][]byte
	Store   KVStore
	Keys    []string
	Values  [][]byte
}

func DefaultExampleContext() StoreExampleContext {
	backing := map[string][]byte{
		"key1":  []byte("value1"),
		"key2":  []byte("value2"),
		"key3":  []byte("value3"),
		"key4":  []byte("value4"),
		"key5":  []byte("value5"),
		"key6":  []byte("value6"),
		"key7":  []byte("value7"),
		"key8":  []byte("value8"),
		"key9":  []byte("value9"),
		"key10": []byte("value10"),
		"key11": []byte("value11"),
		"key12": []byte("value12"),
		"key13": []byte("value13"),
		"key14": []byte("value14"),
		"key15": []byte("value15"),
	}
	var values [][]byte
	for i := 1; i < 16; i++ {
		values = append(values, []byte("value"+strconv.Itoa(i)))
	}
	return StoreExampleContext{
		Backing: backing,
		Keys: []string{
			"key1", "key2", "key3", "key4", "key5", "key6", "key7",
			"key8", "key9", "key10", "key11", "key12", "key13", "key14", "key15",
		},
		Values: values,
	}
}

func SharedStoreBehavior(context *StoreExampleContext) {
	Convey("should get a value in the store", func() {
		store := context.Store
		actual, _ := store.Get("key1")
		So(actual, ShouldResemble, []byte("value1"))
	})

	Convey("should set a value in the store", func() {
		store := context.Store
		expected := []byte("new value")
		store.Set("key1", expected)
		actual, _ := store.Get("key1")
		So(actual, ShouldResemble, expected)
	})

	Convey("should delete a key from the store", func() {
		store := context.Store
		store.Delete("key1")
		actual, _ := store.Get("key1")
		So(actual, ShouldBeEmpty)
	})

	Convey("should get the size of the store", func() {
		ln, _ := context.Store.Size()
		So(ln, ShouldEqual, len(context.Backing))
	})

	Convey("should get the keys from the store", func() {
		expected := context.Keys
		actual, _ := context.Store.Keys()
		sort.Strings(actual)
		sort.Strings(expected)
		So(actual, ShouldResemble, expected)
	})

	Convey("should iterate over the keys", func() {
		store := context.Store
		expected := context.Keys
		store.ForEachKey(func(key string) {
			for i := 0; i < len(expected); i++ {
				if expected[i] == key {
					expected = append(expected[:i], expected[i+1:]...)
				}
			}
		})
		So(expected, ShouldBeEmpty)
	})

	Convey("should iterate over the values", func() {
		store := context.Store
		expected := context.Values
		store.ForEachValue(func(value []byte) {
			for i := 0; i < len(expected); i++ {
				v := expected[i]
				if string(v) == string(value) {
					expected = append(expected[:i], expected[i+1:]...)
				}
			}
		})
		So(expected, ShouldBeEmpty)
	})

	Convey("should iterate over the key value pairs", func() {
		backing, store := context.Backing, context.Store
		expected := make(map[string][]byte, len(backing))
		for k, v := range backing {
			expected[k] = v
		}
		store.ForEach(func(kv *KVData) {
			if string(backing[kv.Key]) == string(kv.Value) {
				delete(expected, kv.Key)
			}
		})
		So(expected, ShouldBeEmpty)
	})

	Convey("should find the first item that matches", func() {
		_, store := context.Backing, context.Store
		expected := &KVData{Key: "key10", Value: []byte("value10")}
		actual, err := store.Find(func(item *KVData) bool { return string(item.Value) == "value10" })
		So(err, ShouldBeNil)
		So(actual.Key, ShouldEqual, expected.Key)
		So(actual.Value, ShouldResemble, expected.Value)
	})

	Convey("should say an item exists if it does", func() {
		res, err := context.Store.Contains("key1")
		So(err, ShouldBeNil)
		So(res, ShouldBeTrue)
	})

	Convey("should say an item does not exist if it doesnt", func() {
		res, err := context.Store.Contains("key")
		So(err, ShouldBeNil)
		So(res, ShouldBeFalse)
	})
}
