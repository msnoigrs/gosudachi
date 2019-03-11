package dartsclone

import (
	"bytes"
	"testing"
)

func TestAsUInt32Array(t *testing.T) {
	ba := []byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}
	ia := asUInt32Array(ba)
	if len(ia) != 2 {
		t.Errorf("length is %d", len(ia))
	}
	if ia[0] != 1 {
		t.Errorf("unexpected error %v", ia[0])
	}
	if ia[1] != 2 {
		t.Errorf("unexpected error %v", ia[1])
	}
}

func TestAsByteArray(t *testing.T) {
	ia := []uint32{1, 2}
	ba := asByteArray(ia)
	if len(ba) != 8 {
		t.Errorf("length is %d", len(ba))
	}
	if !bytes.Equal(ba, []byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}) {
		t.Errorf("unexpected error %v", ba)
	}
}

func TestBuild(t *testing.T) {
	keys := [][]byte{
		[]byte("電気"),
		[]byte("電気通信"),
		[]byte("電気通信大学"),
		[]byte("電気通信大学大学院"),
		[]byte("電気通信大学大学院大学"),
	}
	values := []int{
		0,
		1,
		2,
		3,
		4,
	}
	t.Run("Build", func(t *testing.T) {
		trie := NewDoubleArray()
		err := trie.Build(keys, values, func(state int, max int) {
			return
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		t.Run("CommonPrefixSearch", func(t *testing.T) {
			ret := trie.CommonPrefixSearch([]byte("電気通信大学大学院大学"), 0, 5)
			for i := 0; i < len(ret); i++ {
				if got, expected := ret[i][0], i; got != expected {
					t.Errorf("got %v, expected %v", got, expected)
				}
				if got, expected := []byte("電気通信大学大学院大学")[0:ret[i][1]], keys[i]; string(got) != string(expected) {
					t.Errorf("got %v, expected %v", string(got), string(expected))
				}
			}
		})
		t.Run("CommonPrefixSearchItr", func(t *testing.T) {
			it := trie.CommonPrefixSearchItr([]byte("電気通信大学大学院大学"), 0)
			i := 0
			for it.Next() {
				if it.Err() != nil {
					t.Errorf("unexpected error: %v", err)
				}
				got1, got2 := it.Get()
				if got1 != i {
					t.Errorf("got %v, expected %v", got1, i)
				}
				if string([]byte("電気通信大学大学院大学")[0:got2]) != string(keys[i]) {
					t.Errorf("got %v, expected %v", string([]byte("電気通信大学大学院大学")[0:got2]), string(keys[i]))
				}
				i++
			}
			if it.Err() != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if i != 5 {
				t.Errorf("no match")
			}
		})
		t.Run("CommonPrefixSearchItr offset", func(t *testing.T) {
			it := trie.CommonPrefixSearchItr([]byte("あ電気通信大学大学院大学"), 3)
			i := 0
			for it.Next() {
				if it.Err() != nil {
					t.Errorf("unexpected error: %v", err)
				}
				got1, got2 := it.Get()
				if got1 != i {
					t.Errorf("got %v, expected %v", got1, i)
				}
				if string([]byte("あ電気通信大学大学院大学")[3:got2]) != string(keys[i]) {
					t.Errorf("got %v, expected %v", string([]byte("あ電気通信大学大学院大学")[3:got2]), string(keys[i]))
				}
				i++
			}
			if it.Err() != nil && i != 5 {
				t.Errorf("unexpected error: %v", err)
			}
			if i != 5 {
				t.Errorf("no match")
			}
		})
	})
}
