package llvm

import (
	"fault/ast"
	"fault/llvm/variables"
	"fmt"
	"unicode"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

// Representation of a spec
type spec struct {
	name  string
	types map[string]ast.Type
	vars  *variables.LookupTable
}

func NewCompiledSpec(name string) *spec {
	return &spec{
		name:  name,
		types: make(map[string]ast.Type),
		vars:  variables.NewTable(),
	}
}

func (s *spec) DefineSpecVar(id []string, val value.Value) {
	if s.GetSpecVar(id) != nil {
		s.vars.Update(id, val)
	} else {
		s.vars.Add(id, val)
	}
}

func (s *spec) GetSpecVar(id []string) value.Value {
	return s.vars.Get(id)
}

func (s *spec) GetSpecVarState(id []string) int16 {
	return s.vars.GetState(id)
}

func (s *spec) GetSpecVarPointer(name string) *ir.InstAlloca {
	return s.vars.GetPointer(name)
}

func (s *spec) GetParams(id []string) []value.Value {
	return s.vars.GetParams(id)
}

func (s *spec) AddParam(id []string, p value.Value) {
	s.vars.AddParam(id, p)
}

func (s *spec) DefineSpecType(name string, ty ast.Type) {
	s.types[name] = ty
}

func (s *spec) GetSpecType(name string, inSamePackage bool) (ast.Type, bool) {
	if unicode.IsLower([]rune(name)[0]) && !inSamePackage {
		panic(fmt.Sprintf("Can't use %s from outside of %s", name, s.name))
	}

	v, ok := s.types[name]
	return v, ok
}