package set

import "sync"

type Set struct {
	set  map[string]struct{}
	lock sync.RWMutex
}

func New() *Set {
	return &Set{
		set:  make(map[string]struct{}),
		lock: sync.RWMutex{},
	}
}

func (s *Set) Has(value string) bool {
	s.lock.RLock()
	_, ok := s.set[value]
	s.lock.RUnlock()

	return ok
}

func (s *Set) Add(value string) {
	s.lock.Lock()
	s.set[value] = struct{}{}
	s.lock.Unlock()
}
