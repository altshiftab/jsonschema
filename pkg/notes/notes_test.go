// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notes

import (
	"reflect"
	"testing"
)

func checkGet(t *testing.T, n *Notes, key string, want any) {
	t.Helper()
	if got, ok := n.Get(key); !ok {
		if want != nil {
			t.Errorf("n.Get(%q) = %q, %t, want %v, true", key, got, ok, want)
		}
	} else if want == nil {
		t.Errorf("n.Get(%q) = %q, %t, want false", key, got, ok)
	} else if !reflect.DeepEqual(got, want) {
		t.Errorf("n.Get(%q) = %q, %t, want %v, true", key, got, ok, want)
	}
}

func TestNotes(t *testing.T) {
	var n Notes
	checkGet(t, &n, "key1", nil)
	if !n.IsEmpty() {
		t.Error("n.IsEmpty() == true, want false")
	}
	n.Set("key1", "val1")
	checkGet(t, &n, "key1", "val1")
	if n.IsEmpty() {
		t.Error("n.IsEmpty() == false, want true")
	}

	AppendNote(&n, "key2", "a")
	AppendNote(&n, "key2", "b", "c")
	checkGet(t, &n, "key2", []string{"a", "b", "c"})

	want := "map[key1:val1 key2:[a b c]]"
	if got := n.String(); got != want {
		t.Errorf("n.String() == %q, want %q", got, want)
	}
}

func TestAddNotes(t *testing.T) {
	var n Notes
	n.Set("key1", "val1")
	n.Set("key2", "val2")
	AppendNote(&n, "key3", 1)

	var n2 Notes
	n.Set("key1", "val11")
	n.Set("key4", "val4")
	AppendNote(&n2, "key3", 2, 10)

	var n3 Notes
	n.Set("key5", "val5")
	AppendNote(&n3, "key3", 3)

	n.AddNotes(n2, n3)

	checkGet(t, &n, "key0", nil)
	checkGet(t, &n, "key1", "val11")
	checkGet(t, &n, "key2", "val2")
	checkGet(t, &n, "key3", []int{1, 2, 10, 3})
	checkGet(t, &n, "key4", "val4")
	checkGet(t, &n, "key5", "val5")
}

func TestClear(t *testing.T) {
	var n Notes
	n.Set("key1", "val1")
	n.Clear()
	checkGet(t, &n, "key1", nil)
	if !n.IsEmpty() {
		t.Error("n.IsEmpty() == true, want false")
	}
}
