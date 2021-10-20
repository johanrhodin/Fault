package smt

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/llir/llvm/ir"
)

func (g *Generator) runParallel(perm [][]string, vars map[string]string) {
	var waitGroup sync.WaitGroup
	r := make(chan [][]rule, len(perm))
	waitGroup.Add(len(perm))

	go func() {
		waitGroup.Wait()
		close(r)
	}()
	for _, p := range perm {
		go func(calls []string) {
			defer waitGroup.Done()
			var opts [][]rule
			startVars := make(map[string]string)
			for k, v := range vars {
				startVars[k] = v
			}
			for _, c := range calls {
				v := g.functions[c]
				raw := g.parseFunction(v, startVars)
				opts = append(opts, raw)
			}
			r <- opts
		}(p)
	}
	for opts := range r {
		raw := g.parallelRules(opts)
		for _, v := range raw {
			g.rules = append(g.rules, g.writeRule(v))
		}
	}
	g.rules = append(g.rules, g.capParallel()...)
}

func (g *Generator) parallelRules(r [][]rule) []rule {
	var rules []rule
	s := g.paraStateChanges(r)
	for k, v := range s {
		if len(v) > 1 {
			g.parallelEnds[k] = append(g.parallelEnds[k], g.getEnds(v))
		}
	}
	for _, op := range r {
		rules = append(rules, op...) // Flatten
	}
	return rules
}

func (g *Generator) capParallel() []string {
	// writes OR nodes to end each parallel run
	var rules []string
	for k, v := range g.parallelEnds {
		id := g.advanceSSA(k)
		g.declareVar(id, "Real")
		ends := g.formatEnds(k, v, id)
		//(assert (or (= bathtub_drawn_water_level_10 bathtub_drawn_water_level_7) (= bathtub_drawn_water_level_10  bathtub_drawn_water_level_9)))
		rule := g.writeAssert("or", ends)
		rules = append(rules, rule)
	}
	g.parallelEnds = map[string][]int{}
	return rules
}

func (g *Generator) capCond(state map[string]map[int][]int) []string {
	var ends []string
	for i, num := range state {
		end := g.getEnds(num)
		id := g.advanceSSA(i)
		g.declareVar(id, "Real")
		ends = append(ends, g.formatEnds(i, []int{end}, id))
	}
	return ends
}

func (g *Generator) capCondSync(tstate map[string]map[int][]int, fstate map[string]map[int][]int) ([]string, []string) {
	var tends []string
	var fends []string
	for i, num := range fstate {
		if tstate[i] == nil {
			start := g.getStarts(num)
			id := g.getSSA(i)
			tends = append(tends, g.formatEnds(i, []int{start}, id))
		}
	}

	for i, num := range tstate {
		if fstate[i] == nil {
			start := g.getStarts(num)
			id := g.getSSA(i)
			fends = append(fends, g.formatEnds(i, []int{start}, id))
		}
	}
	return tends, fends
}

func (g *Generator) setStartVar(id string, startVars map[string]string) map[string]string {
	if _, ok := startVars[id]; !ok {
		startVars[id] = fmt.Sprint(id, "_", g.ssa[id])
	}
	return startVars
}

func (g *Generator) gatherStarts(p []string, startVars map[string]string) map[string]string {
	for _, fname := range p {
		f := g.functions[fname]
		for _, block := range f.Blocks {
			for _, inst := range block.Insts {
				switch inst := inst.(type) {
				case *ir.InstLoad:
					id := g.formatIdent(inst.Src.Ident())
					startVars = g.setStartVar(id, startVars)
				}
			}
		}
	}
	return startVars
}

func (g *Generator) getStarts(n map[int][]int) int {
	start := 0
	for _, v := range n {
		if v[0] < start {
			start = v[0]
		}
	}
	return start
}

func (g *Generator) getEnds(n map[int][]int) int {
	end := 0
	for _, v := range n {
		if v[len(v)-1] > end {
			end = v[len(v)-1]
		}
	}
	return end
}

func (g *Generator) formatEnds(k string, nums []int, id string) string {
	var e []string
	for _, v := range nums {
		v := fmt.Sprint(k, "_", strconv.Itoa(v))
		r := g.writeInfix(id, v, "=")
		e = append(e, r)
	}
	return strings.Join(e, " ")
}

func (g *Generator) paraStateChanges(r [][]rule) map[string]map[int][]int {
	// When running functions in parallel,
	//	checks for conflicting state change
	state := make(map[string]map[int][]int)
	for i, v := range r {
		for _, ru := range v {
			switch r := ru.(type) {
			case *infix:
				if r.ty != "" { //Variable assignment
					id, num := g.getVarBase(r.x.String())
					if _, ok := state[id]; ok {
						state[id][i] = append(state[id][i], num)
					} else {
						state[id] = make(map[int][]int)
						state[id][i] = append(state[id][i], num)
					}
				}
			}
		}

	}
	return state
}

func (g *Generator) writeInfix(x string, y string, op string) string {
	return fmt.Sprintf("(%s %s %s)", op, x, y)
}

func (g *Generator) writeBranch(ty string, cond string, t string, f string) string {
	return fmt.Sprintf("(%s %s %s %s)", ty, cond, t, f)
}

func (g *Generator) declareVar(id string, t string) {
	def := fmt.Sprintf("(declare-fun %s () %s)", id, t)
	g.inits = append(g.inits, def)
}
func (g *Generator) writeAssert(op string, stmt string) string {
	if op == "" {
		return fmt.Sprintf("(assert %s)", stmt)
	}
	return fmt.Sprintf("(assert (%s %s))", op, stmt)
}

func (g *Generator) writeRule(ru rule) string {
	switch r := ru.(type) {
	case *infix:
		y := g.unpackRule(r.y)
		x := g.unpackRule(r.x)
		if r.op != "" {
			return g.writeInfix(x, y, r.op)
		}
		return g.writeInitRule(x, r.ty, y)
	case *ite:
		cond := g.writeRule(r.cond)
		tstate := g.paraStateChanges([][]rule{r.t})
		fstate := g.paraStateChanges([][]rule{r.f})
		var tRule, fRule string
		tEnds := g.capCond(tstate)
		fEnds := g.capCond(fstate)

		// Keep variable names in sync across branches
		tSync, fSync := g.capCondSync(tstate, fstate)
		tEnds = append(tEnds, tSync...)
		fEnds = append(fEnds, fSync...)

		if len(tEnds) > 1 {
			tRule = fmt.Sprintf("(and %s)", strings.Join(tEnds, " "))
		} else if len(tEnds) == 1 {
			tRule = strings.Join(tEnds, " ")
		}

		if len(fEnds) > 1 {
			fRule = fmt.Sprintf("(and %s)", strings.Join(fEnds, " "))
		} else if len(fEnds) == 1 {
			fRule = strings.Join(fEnds, " ")
		}

		br := g.writeBranch("ite", cond, tRule, fRule)
		return g.writeAssert("", br)
	case *wrap:

	default:
		panic(fmt.Sprintf("%T is not a valid rule type", r))
	}
	return ""
}

func (g *Generator) unpackRule(x rule) string {
	switch r := x.(type) {
	case *wrap:
		return r.value
	case *infix:
		return g.writeRule(r)
	default:
		panic(fmt.Sprintf("%T is not a valid rule type", r))
	}
}

func (g *Generator) writeInitRule(id string, t string, val string) string {
	// Initialize: x = Int("x")
	g.declareVar(id, t)
	// Set rule: s.add(x == 2)
	return fmt.Sprintf("(assert (= %s %s))", id, val)
}

func (g *Generator) generateRules(raw []rule) []string {
	var rules []string
	for _, v := range raw {
		rules = append(rules, g.writeRule(v))
	}
	return rules
}

func (g *Generator) parallelPermutations(p []string) (permuts [][]string) {
	var rc func([]string, int)
	rc = func(a []string, k int) {
		if k == len(a) {
			permuts = append(permuts, append([]string{}, a...))
		} else {
			for i := k; i < len(p); i++ {
				a[k], a[i] = a[i], a[k]
				rc(a, k+1)
				a[k], a[i] = a[i], a[k]
			}
		}
	}
	rc(p, 0)

	return permuts
}