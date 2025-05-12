package run

import "math/rand/v2"




type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) MarkDependsOn(group *Group, other ID) {
	if group.group.deps == nil {
		group.group.deps = make(map[ID][]ID)
	}
	group.group.deps[id] = append(group.group.deps[id], other)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

func RandStringBytes(n int) string {
    b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.UintN(uint(len(letterBytes)))]
	}
	return string(b)
}

func NewID() ID {
	return ID(RandStringBytes(10))
}
