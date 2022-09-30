package list_test

import (
	"testing"

	"golang.org/x/exp/container/list"
)

func TestStringList(t *testing.T) {
	input := []string{"1", "2", "3"}
	list := list.New[string]()

	for _, v := range input {
		list.PushBack(v)
	}

	if list.Len() != 3 {
		t.Fail()
	}

	e := list.Front()
	if e == nil {
		t.Fail()
	}

	for i := 0; i < list.Len(); i++ {
		if e.Value != input[i] {
			t.Fail()
		}
		e = e.Next()
	}
}

func TestIntList(t *testing.T) {
	input := []int{123, 456, 789, 1442}
	list := list.New[int]()

	for _, v := range input {
		list.PushBack(v)
	}

	if list.Len() != 4 {
		t.Fail()
	}

	e := list.Front()
	if e == nil {
		t.Fail()
	}

	for i := 0; i < list.Len(); i++ {
		if e.Value != input[i] {
			t.Fail()
		}
		e = e.Next()
	}
}
