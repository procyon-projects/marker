package visitor

import "strings"

type Slice struct {
	elem Type
}

func (s *Slice) Elem() Type {
	return s.elem
}

func (s *Slice) Underlying() Type {
	return s
}

func (s *Slice) String() string {
	var builder strings.Builder
	builder.WriteString("[]")
	builder.WriteString(s.elem.String())
	return builder.String()
}