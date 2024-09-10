package main

import (
	"io"

	"github.com/thesoulless/watchmyback/services/email"
)

var (
	// emailSrvs = NewList[*Service[*email.Core]]()
	emailSrvs = NewList[*email.Core]()
)

type Service[T io.Closer] struct {
	Name string
	next *Service[T]
	val  T
}

func NewService[T io.Closer](val T) *Service[T] {
	// return &Service[T]{val: val}
	return &Service[T]{val: val}
}

// func (s *Service[T]) Close() error {
// 	return s.val.Close()
// }

type List[T io.Closer] struct {
	head *Service[T]
	l    int
	c    map[string]*Service[T]
}

func NewList[T io.Closer]() *List[T] {
	return &List[T]{c: make(map[string]*Service[T])}
}

func (l *List[T]) Len() int {
	return l.l
}

func (l *List[T]) Add(name string, val T) {
	if _, ok := l.c[name]; ok {
		return
	}
	node := NewService[T](val)
	node.Name = name
	node.val = val

	defer func() {
		l.l++
	}()

	var index int
	if l.head == nil {
		l.head = node
	} else {
		for n := l.head; n != nil; n = n.next {
			index++
			if n.next == nil {
				n.next = node
				break
			}
		}
	}
	l.c[name] = node
}

func (l *List[T]) Get(name string) (T, bool) {
	node, ok := l.c[name]
	if !ok {
		var t T
		return t, false
	}

	return node.val, true
}

// func (l *List[T]) Add(name string, val any) {
// 	if _, ok := l.c[name]; ok {
// 		return
// 	}
// 	node := NewService(val)
// 	node.Name = name
// 	node.val = val

// 	var index int
// 	if l.head == nil {
// 		l.head = node
// 	} else {
// 		for n := l.head; n != nil; n = n.next {
// 			index++
// 			if n.next == nil {
// 				n.next = node
// 				break
// 			}
// 		}
// 	}
// 	l.c[name] = index
// }
