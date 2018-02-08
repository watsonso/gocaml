package closure

import (
	"github.com/rhysd/gocaml/mir"
)

// In post process:
//   - CLOSURE_CALL flag is set to each 'app' instruction
type postProcess struct {
	closures           mir.Closures
	funcs              mir.Toplevel
	processingFuncName string
	processingFunc     *mir.Fun
}

// TODO:
// Reveal how each adhoc polymorphic types should be instantiated.
// e.g.
//   let o = None in o = (o = Some 10) || (o = Some true) in ...
// In expression `None`, it's type is 'a option and 'a should be instantiated as int and bool.
// In this process, collect the actual instantiated types for adhoc polymorphic type variables
// (in above example, they're int and bool).
// It also corrects captured values. In above case, when capturing `o`, the type of capture
// value is 'a option. However, it actually needs to capture two values typed as int option and
// bool option because we introduce code duplication to generate code for polymorphic type
// expressions.

// TODO:
// Rearrange basic blocks to represents actual DAG.
// All blocks should be flattened in a function.
//
// e.g.
//
// From:
//   block {
//       // entry block
//       if
//       then {
//           // then block
//       }
//       else {
//           // else block
//       }
//       insns...
//   }
//
// To:
//   block {
//       // entry block
//       if
//   }
//   then {
//       // then block
//   }
//   else {
//       // else block
//   }
//   precede {
//       // rest block
//       insns...
//   }

func (pp *postProcess) processInsn(insn *mir.Insn) {
	switch val := insn.Val.(type) {
	case *mir.App:
		if val.Callee == pp.processingFuncName && pp.processingFunc != nil {
			pp.processingFunc.IsRecursive = true
		}
		if val.Kind == mir.EXTERNAL_CALL {
			break
		}
		if _, ok := pp.closures[val.Callee]; ok {
			val.Kind = mir.CLOSURE_CALL
			break
		}
		if _, ok := pp.funcs[val.Callee]; ok {
			// Callee register name is a name of function, but not a closure.
			// So it must be known function.
			break
		}
		// It's not an external symbol, closure nor known function. So it must be a function
		// variable. All function variables are closures. So the callee must be a closure.
		val.Kind = mir.CLOSURE_CALL
	case *mir.If:
		pp.processBlock(val.Then)
		pp.processBlock(val.Else)
	case *mir.Fun:
		panic("unreachable")
	}
}

func (pp *postProcess) processBlock(block *mir.Block) {
	begin, end := block.WholeRange()
	for i := begin; i != end; i = i.Next {
		pp.processInsn(i)
	}
}

func (pp *postProcess) process(n string, f *mir.Fun, b *mir.Block) {
	pp.processingFuncName = n
	pp.processingFunc = f
	pp.processBlock(b)
}

func doPostProcess(prog *mir.Program) {
	pp := &postProcess{
		prog.Closures,
		prog.Toplevel,
		"",
		nil,
	}
	for n, f := range prog.Toplevel {
		pp.process(n, f.Val, f.Val.Body)
	}
	pp.process("", nil, prog.Entry)
}