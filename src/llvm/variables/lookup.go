package variables

import (
	"fmt"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type LookupTable struct {
	state    map[string]State //ssa position
	pointers *Pointers
	values   map[string]Entry
}

func NewTable() *LookupTable {
	l := &LookupTable{
		state:    make(map[string]State),
		pointers: NewPointers(),
		values:   make(map[string]Entry),
	}
	return l
}

func (l *LookupTable) Add(id []string, val value.Value) {
	switch len(id) {
	case 2: // a global variable
		l.values[id[1]] = NewGlobalEntry(val)
		l.state[id[1]] = NewGlobalSEntry(0)
	case 3: // a struct param
		switch st := l.values[id[1]].(type) {
		case *structEntry:
			st.update(id, val)
			l.state[id[1]].(*structSEntry).update(id, 0)
		case *instanceEntry: // In the run block
			st.update(id, val)
			l.state[id[1]].(*instanceSEntry).update(id, 0)
		case nil:
			l.values[id[1]] = NewStructEntry(id, val)
			l.state[id[1]] = NewStructSEntry(id, 0)
		}
	case 4: // an instance of a struct
		if l.values[id[1]] != nil {
			l.values[id[1]].(*instanceEntry).update(id, val)
			l.state[id[1]].(*instanceSEntry).update(id, 0)
		} else {
			l.values[id[1]] = NewInstanceEntry(id, val)
			l.state[id[1]] = NewInstanceSEntry(id, 0)
		}
	default:
		panic(fmt.Sprintf("cannot add %s to lookup table, missing full namespace. got=%d",
			strings.Join(id, "_"), len(id)))
	}
}

func (l *LookupTable) Store(id []string, name string, point *ir.InstAlloca) {
	switch l.values[id[1]].(type) {
	case *globalEntry:
		l.pointers.store(name, point)
	case *structEntry:
		l.pointers.store(name, point)
	case *instanceEntry:
		l.pointers.store(name, point)
	default:
		panic(fmt.Sprintf("variable %s not in the lookup table", strings.Join(id, "_")))
	}
}

func (l *LookupTable) Get(id []string) value.Value {
	switch v := l.values[id[1]].(type) {
	case *globalEntry:
		return v.get(id)
	case *structEntry:
		return v.get(id)
	case *instanceEntry:
		return v.get(id)
	}
	return nil
}

func (l *LookupTable) GetState(id []string) int16 {
	switch v := l.state[id[1]].(type) {
	case *globalSEntry:
		return v.get(id)
	case *structSEntry:
		return v.get(id)
	case *instanceSEntry:
		return v.get(id)
	}
	fid := strings.Join(id, "_")
	panic(fmt.Sprintf("no state found for variable %s", fid))
}

func (l *LookupTable) GetPointer(name string) *ir.InstAlloca {
	p := l.pointers.get(name)
	if p == nil {
		panic(fmt.Sprintf("no pointer found for variable %s", name))
	}
	return p
}

func (l *LookupTable) Update(id []string, val value.Value) {
	switch v := l.values[id[1]].(type) {
	case *structEntry:
		v.update(id, val)
		l.state[id[1]].(*structSEntry).increment(id)
	case *instanceEntry:
		v.update(id, val)
		l.state[id[1]].(*instanceSEntry).increment(id)
	}
}