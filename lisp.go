package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// returns interface{} because that's the only way to ensure that no circular
// dependencies appear in the types
type Environment struct {
	Fields map[string]interface{}
	Parent *Environment
}

func newEnv(length int) (e Environment) {
	e.Fields = make(map[string]interface{}, length)
	return e
}

type LispObject interface {
	Eval(env Environment) LispObject
	Print() string
}

// this is placed here because you can't have circular types in go, but I want
// to always work with LispObjects
func (e *Environment) Get(s string) LispObject {
	if val, ok := e.Fields[s]; ok {
		return val.(LispObject)
	}
	if e.Parent == nil {
		return Nil
	}
	return e.Parent.Get(s)
}

func (e *Environment) Put(s string, l LispObject) {
	e.Fields[s] = l
}

// creates a new env with args -> context
func (e *Environment) FromParent(args []string, context []LispObject) Environment {
	env := newEnv(len(args))
	env.Parent = e
	for i := range args {
		env.Put(args[i], context[i])
	}
	return env
}

type lispNil int

var Nil lispNil = lispNil(0)

// (eval ()) -> ()
func (n lispNil) Eval(env Environment) LispObject {
	return n
}
func (n lispNil) Print() string {
	return "()"
}

type fixnum int

// (eval 1) -> 1
func (num fixnum) Eval(env Environment) LispObject {
	return num
}
func (num fixnum) Print() string {
	return strconv.Itoa(int(num))
}

type symbol string

// (eval (x)) where x = 1 -> 1
func (s symbol) Eval(env Environment) LispObject {
	return env.Get(string(s))
}
func (s symbol) Print() string {
	return string(s)
}

type lambda struct {
	fn      LispObject
	arglist []string
}

// (eval (lambda (x) ()))
func (l lambda) Eval(env Environment) LispObject {
	return l
}
func (l lambda) Print() string {
	return "<lambda>"
}

type Intrinsic struct {
	op func([]LispObject, Environment) LispObject
}

func (i Intrinsic) Eval(env Environment) LispObject {
	return i
}
func (i Intrinsic) Print() string {
	return "<intrinsic>"
}

type list []LispObject

// (eval (* 1 2)) -> 2
func (l list) Eval(env Environment) LispObject {
	first := l[0].Eval(env)
	context := l[1:]
	var retVal LispObject = Nil
	switch f := first.(type) {
	case lambda:
		e := env.FromParent(f.arglist, context)
		retVal = f.fn.Eval(e)
	case Intrinsic:
		retVal = f.op(l, env)
	default:
		panic("tried to apply a non-lambda value")
	}
	return retVal
}

func (l list) Print() string {
	buf := "("
	for _, val := range l {
		buf += val.Print() + " "
	}
	return buf + ")"
}

func mathOp(operation func(fixnum, fixnum) fixnum) Intrinsic {
	return Intrinsic{op: func(rawlist []LispObject, env Environment) LispObject {
		total := rawlist[1].Eval(env).(fixnum)
		for _, obj := range rawlist[2:] {
			num := obj.Eval(env).(fixnum)
			total = operation(total, num)
		}
		return total
	}}
}

func car(rawlist []LispObject, env Environment) LispObject {
	theList := rawlist[1].(list)
	return theList[0]
}

func cdr(rawlist []LispObject, env Environment) LispObject {
	theList := rawlist[1].(list)
	return theList[1:]
}

func mklambda(rawlist []LispObject, env Environment) LispObject {
	rawargs := rawlist[1].(list)
	strargs := []string{}

	for i := range rawargs {
		strargs = append(strargs, string(rawargs[i].(symbol)))
	}

	return lambda{
		arglist: strargs,
		fn:      rawlist[2]}
}

func def(rawlist []LispObject, env Environment) LispObject {
	lam := mklambda(rawlist[1:], env)
	env.Put(string(rawlist[1].(symbol)), lam)
	return lam
}
func lispToBool(l LispObject) bool {
	switch l.(type) {
	case lispNil:
		return false
	default:
		return true
	}
	return true
}

func If(rawlist []LispObject, env Environment) LispObject {
	if lispToBool(rawlist[1]) {
		return rawlist[2].Eval(env)
	} else {
		return rawlist[3].Eval(env)
	}
	return Nil
}
func boolOp(fn func(bool, bool) bool) Intrinsic {
	return Intrinsic{
		op: func(rawlist []LispObject, env Environment) LispObject {
			a := lispToBool(rawlist[1].Eval(env))
			b := lispToBool(rawlist[2].Eval(env))
			if fn(a, b) {
				return Nil
			} else {
				return fixnum(1)
			}
			return fixnum(1)
		}}
}

func compOp(fn func(fixnum, fixnum) bool) Intrinsic {
	return Intrinsic{
		op: func(rawlist []LispObject, env Environment) LispObject {
			a := rawlist[1].Eval(env).(fixnum)
			b := rawlist[2].Eval(env).(fixnum)
			if fn(a, b) {
				return Nil
			} else {
				return fixnum(1)
			}
			return fixnum(1)
		}}
}

func set(rawlist []LispObject, env Environment) LispObject {
	sym := rawlist[1].(symbol)
	env.Put(string(sym), rawlist[2].Eval(env))
	return Nil
}
func quote(rawlist []LispObject, env Environment) LispObject {
	return rawlist[1]
}
func toList(rawlist []LispObject, env Environment) LispObject {
	return list(rawlist[1:])
}
func appendList(rawlist []LispObject, env Environment) LispObject {
	l := rawlist[1].Eval(env).(list)
	return append(l, rawlist[2])
}
func let(rawlist []LispObject, env Environment) LispObject {
	args := []string{}
	context := []LispObject{}
	arglist := rawlist[1].(list)

	for _, argcons := range arglist {
		cons := argcons.(list)
		name := cons[0].(symbol)
		val := cons[1].Eval(env)
		args = append(args, string(name))
		context = append(context, val)
	}
	e := env.FromParent(args, context)
	return rawlist[2].Eval(e)
}
func length(rawlist []LispObject, env Environment) LispObject {
	switch v := rawlist[1].Eval(env).(type) {
	case list:
		return fixnum(len(v))
	case lispNil:
		return fixnum(0)
	default:
		return fixnum(1)
	}
	return fixnum(0)
}

func print(rawlist []LispObject, env Environment) LispObject {
	for _, val := range rawlist {
		fmt.Print(val.Print())
	}
	return Nil
}

func eq(rawlist []LispObject, env Environment) LispObject {
	a := rawlist[1].Eval(env)
	b := rawlist[2].Eval(env)

	switch v1 := a.(type) {
	case list:
		if v2, ok := b.(list); ok {
			if &v1 == &v2 {
				return fixnum(1)
			}
			return Nil
		}
		return Nil
	case fixnum:
		if v2, ok := b.(fixnum); ok {
			if v1 == v2 {
				return fixnum(1)
			}
			return Nil
		}
		return Nil
	case symbol:
		if v2, ok := b.(symbol); ok {
			if v1 == v2 {
				return fixnum(1)
			}
			return Nil
		}
		return Nil
	case lispNil:
		if _, ok := b.(lispNil); ok {
			return fixnum(1)
		}
		return Nil
	}

	return Nil
}

func equalHelper(a, b LispObject) bool {
	switch v1 := a.(type) {
	case list:
		if v2, ok := b.(list); ok {
			if len(v1) != len(v2) {
				return false
			}
			for i := range v1 {
				if !equalHelper(v1[i], v2[i]) {
					return false
				}
			}
			return true
		}
		return false
	case fixnum:
		if v2, ok := b.(fixnum); ok {
			if v1 == v2 {
				return true
			}
			return false
		}
		return false
	case symbol:
		if v2, ok := b.(symbol); ok {
			if v1 == v2 {
				return true
			}
			return false
		}
		return false
	case lispNil:
		if _, ok := b.(lispNil); ok {
			return true
		}
		return false
	}
	return false
}

func equal(rawlist []LispObject, env Environment) LispObject {
	a := rawlist[1].Eval(env)
	b := rawlist[2].Eval(env)
	if equalHelper(a, b) {
		return fixnum(1)
	}
	return Nil
}

func isNil(rawlist []LispObject, env Environment) LispObject {
	if _, ok := rawlist[1].Eval(env).(lispNil); ok {
		return fixnum(1)
	}
	return Nil
}
func isSymbol(rawlist []LispObject, env Environment) LispObject {
	if _, ok := rawlist[1].Eval(env).(symbol); ok {
		return fixnum(1)
	}
	return Nil
}
func isFixnum(rawlist []LispObject, env Environment) LispObject {
	if _, ok := rawlist[1].Eval(env).(fixnum); ok {
		return fixnum(1)
	}
	return Nil
}
func isList(rawlist []LispObject, env Environment) LispObject {
	if _, ok := rawlist[1].Eval(env).(list); ok {
		return fixnum(1)
	}
	return Nil
}
func isLambda(rawlist []LispObject, env Environment) LispObject {
	if _, ok := rawlist[1].Eval(env).(lambda); ok {
		return fixnum(1)
	}
	return Nil
}
func isIntrinsic(rawlist []LispObject, env Environment) LispObject {
	if _, ok := rawlist[1].Eval(env).(Intrinsic); ok {
		return fixnum(1)
	}
	return Nil
}

var IntrinsicList map[string]Intrinsic = map[string]Intrinsic{
	"+":          mathOp(func(a fixnum, b fixnum) fixnum { return a + b }),
	"-":          mathOp(func(a fixnum, b fixnum) fixnum { return a - b }),
	"*":          mathOp(func(a fixnum, b fixnum) fixnum { return a * b }),
	"/":          mathOp(func(a fixnum, b fixnum) fixnum { return a / b }),
	"car":        Intrinsic{op: car},
	"cdr":        Intrinsic{op: cdr},
	"lambda":     Intrinsic{op: mklambda},
	"def":        Intrinsic{op: def},
	"if":         Intrinsic{op: If},
	"and":        boolOp(func(a bool, b bool) bool { return a && b }),
	"or":         boolOp(func(a bool, b bool) bool { return a || b }),
	">":          compOp(func(a fixnum, b fixnum) bool { return a > b }),
	">=":         compOp(func(a fixnum, b fixnum) bool { return a >= b }),
	"<":          compOp(func(a fixnum, b fixnum) bool { return a < b }),
	"<=":         compOp(func(a fixnum, b fixnum) bool { return a <= b }),
	"set!":       Intrinsic{op: set},
	"quote":      Intrinsic{op: quote},
	"list":       Intrinsic{op: toList},
	"append":     Intrinsic{op: appendList},
	"let":        Intrinsic{op: let},
	"length":     Intrinsic{op: length},
	"print":      Intrinsic{op: print},
	"eq?":        Intrinsic{op: eq},
	"equal?":     Intrinsic{op: equal},
	"nil?":       Intrinsic{op: isNil},
	"symbol?":    Intrinsic{op: isSymbol},
	"num?":       Intrinsic{op: isFixnum},
	"list?":      Intrinsic{op: isList},
	"lambda?":    Intrinsic{op: isLambda},
	"intrinsic?": Intrinsic{op: isIntrinsic}}

func ParseAtom(s string) LispObject {
	if num, err := strconv.ParseInt(s, 10, 0); err == nil {
		return fixnum(num)
	}
	return symbol(s)

}
func ParseList(tokens []string) (LispObject, []string) {
	if tokens[0] == ")" {
		return Nil, tokens[1:]
	}
	retList := list{}
	for {
		switch tokens[0] {
		case ")":
			return retList, tokens[1:]
		case "(":
			obj, t := ParseList(tokens[1:])
			tokens = t
			retList = append(retList, obj)
		default:
			retList = append(retList, ParseAtom(tokens[0]))
			tokens = tokens[1:]
		}
	}
	return Nil, nil
}
func ParseTree(tokens []string) (obj LispObject) {
	switch tok := tokens[0]; tok {
	case "(":
		obj, _ := ParseList(tokens[1:])
		return obj
	default:
		return ParseAtom(tok)
	}
	return Nil
}

func Read(input string) (obj LispObject) {
	newstr := strings.Replace(input, "(", " ( ", -1)
	newstr = strings.Replace(newstr, ")", " ) ", -1)
	tokens := strings.Fields(newstr)

	if len(tokens) == 0 {
		panic("expected data")
	}
	return ParseTree(tokens)
}

func main() {
	buffer := bufio.NewReader(os.Stdin)
	globalEnv := newEnv(50)
	for name, op := range IntrinsicList {
		globalEnv.Put(name, op)
	}
	for {
		fmt.Print("lisp.go>")
		line, _ := buffer.ReadString(byte('\n'))
		for strings.Count(line, "(") != strings.Count(line, ")") {
			tmpline, _ := buffer.ReadString(byte('\n'))
			line += tmpline
		}
		tree := Read(line)
		fmt.Printf("got %v\n", tree.Print())
		fmt.Printf("-> %v\n", tree.Eval(globalEnv).Print())
	}
}
