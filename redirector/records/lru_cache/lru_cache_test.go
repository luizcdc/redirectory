package lru_cache

import (
	"fmt"
	"testing"
)

func TestNewCache(t *testing.T) {
	const size  uint = 1000
	c := NewCache(size)
	if c == nil {
		t.Fatalf("NewCache(%v) returned nil", size)
	}
	if c.Len() != 0 {
		t.Error("NewCache(size) returned cache with non-zero length")
	}
	if c.cap != size {
		t.Error("NewCache(size) returned cache with capacity not equal to 10")
	}
	if c.lru_head != nil {
		t.Error("NewCache(size) returned cache with non-nil lru_head")
	}
	if c.hashmap == nil {
		t.Error("NewCache(size) returned cache with nil hashmap")
	}
	if len(c.hashmap) != 0 {
		t.Error("NewCache(size) returned cache with non-zero hashmap")
	}
}

func TestInsert(t *testing.T) {
	const size  uint = 10
	const key = "key"
	const val = "val"

	c := NewCache(size)
	lenbefore := c.Len()
	if c.Insert(key, val) != val {
		t.Error("Insert returned wrong value")
	}
	if c.Len() <= lenbefore {
		t.Error("Insert did not increment length")
	}
	if c.Len() != 1 {
		t.Error("One Insert did not set length to 1")
	}
	const amountExtraInserts = 4
	for i := 0; i < amountExtraInserts; i++ {
		c.Insert(key+fmt.Sprint(i), val+fmt.Sprint(i))
	}
	if c.Len() != amountExtraInserts + 1 {
		t.Errorf("%v inserts did not set length to 5", amountExtraInserts+1)
	}
	const amountExtraInserts2 = 20
	for i := amountExtraInserts + 1; i < amountExtraInserts2; i++ {
		c.Insert(key+fmt.Sprint(i), val+fmt.Sprint(i))
	}
	if c.Len() != int(size) {
		t.Errorf("Capacity %v did not limit length to %v when doing %v inserts total.", size, size, amountExtraInserts2+amountExtraInserts+1)
	}
}

func TestContains(t *testing.T) {
	const size  uint = 20
	const key = "key"
	const val = "val"

	c := NewCache(size)
	if c.Contains(key) {
		t.Error("Contains returned true for empty cache")
	}
	c.Insert(key, val)
	if !c.Contains(key) {
		t.Error("Contains returned false for key in cache")
	}

	for i := 0; i < 5; i++ {
		c.Insert(key+fmt.Sprint(i), val+fmt.Sprint(i))
	}
	const anotherKey = "another key"
	const anotherVal = "another val"
	c.Insert(anotherKey, anotherVal)
	for i := 5; i < 10; i++ {
		c.Insert(key+fmt.Sprint(i), val+fmt.Sprint(i))
	}
	if !c.Contains("another key") {
		t.Errorf("Contains returned false for key '%v' in cache", anotherKey)
	}
}

func TestHit(t *testing.T) {
	const size  uint = 20
	const key = "key"
	const val = "val"
	c := NewCache(size)
	c.Insert(key, val)
	for i := 0; i < 5; i++ {
		c.Insert(key+fmt.Sprint(i), val+fmt.Sprint(i))
	}
	if c.peekMRU().key == key {
		t.Error("First key to be inserted is in the wrong position after inserts.")
	}
	if c.peekLRU().key == key+fmt.Sprint(4) {
		t.Error("Last key to be inserted is in the wrong position after inserts.")
	}
	c.Hit(key)
	if c.peekMRU().key != key {
		t.Error("Hit did not move key to MRU.")
	} else if c.peekLRU().key == key {
		t.Error("Hit did not move key from LRU.")
	}
}

func TestChangeCap(t *testing.T) {
	const size  uint = 20

	c := NewCache(size)
	for i := 0; i < 5; i++ {
		c.Insert(fmt.Sprint(i), fmt.Sprint(i))
	}
	const newcap = 3
	c.ChangeCap(newcap)
	if c.cap != newcap {
		t.Errorf("ChangeCap(%v) did not change capacity to %v", newcap, newcap)
	}
	if c.Len() != int(newcap) {
		t.Errorf("ChangeCap(%v) did not reduce length to %v", newcap, newcap)
	}
	if c.peekLRU().key != "2" {
		t.Errorf("ChangeCap(%v) did not remove the correct LRU entries. Got: %v, want: %v.", newcap, c.peekLRU().key, "2")
	}
	if c.peekMRU().key != "4" {
		t.Errorf("ChangeCap(%v) did not keep the correct MRU entries. Got: %v, want: %v.", newcap, c.peekMRU().key, "4")
	}
}

func TestRemove(t *testing.T) {
	const size  uint = 20
	const key = "key"
	const val = "val"
	c := NewCache(size)

	c.Insert(key, val)
	c.Remove(key)
	if c.Contains(key) {
		t.Error("Remove did not remove key from cache")
	}
	if c.Len() != 0 {
		t.Error("Remove did not decrement length")
	}
	for i := 0; i < 5; i++ {
		c.Insert(key+fmt.Sprint(i), val+fmt.Sprint(i))
	}
	c.Remove(key+fmt.Sprint(2))
	if c.Contains(key+fmt.Sprint(2)) {
		t.Error("Remove did not remove key from cache")
	}
	if c.Len() != 4 {
		t.Error("Remove did not decrement length")
	}
 }

 func TestDropLRU(t *testing.T) {
	const size  uint = 20
	c := NewCache(size)
	for i := 0; i < 20; i++ {
		c.Insert(fmt.Sprint(i), fmt.Sprint(i))
	}
	c.DropLRU()
	if c.Len() != 19 {
		t.Error("DropLRU did not decrement length")
	}
	if c.peekLRU().key == "0" {
		t.Error("DropLRU kept the LRU entry in the cache")
	}
	if c.peekLRU().key != "1" {
		t.Error("DropLRU did not remove the correct LRU entry")
	}
	c.DropLRU()
	if c.Len() != 18 {
		t.Error("DropLRU did not decrement length")
	}
	if c.peekLRU().key == "1" {
		t.Error("DropLRU kept the LRU entry in the cache")
	}
	if c.peekLRU().key != "2" {
		t.Error("DropLRU did not remove the correct LRU entry")
	}
 }