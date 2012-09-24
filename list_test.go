package main

import (
	"reflect"
	"testing"
)

var nilEnv Environment = newEnv(0)

func TestEvalNil(t *testing.T) {
	n := Nil.Eval(nilEnv)
	switch v := n.(type) {
	case lispNil:
		t.Log("(eval ()) -> ()")
	default:
		t.Errorf("Expected (eval()) -> (), got %v instead", v)
	}
}

func TestEvalFixnum(t *testing.T) {
	n := fixnum(10).Eval(nilEnv)
	switch v := n.(type) {
	case fixnum:
		{
			if v != 10 {
				t.Errorf("Expected (eval 10) -> 10, got %v instead", v)
			}
			t.Log("(eval 10) -> 10")
		}
	default:
		t.Errorf("Expected (eval 10) -> 10, got %v instead", v)
	}
}

func TestEvalSymbol(t *testing.T) {
	testEnv := nilEnv.FromParent([]string{"x"}, []LispObject{fixnum(123)})
	n := symbol("x").Eval(testEnv)
	switch v := n.(type) {
	case fixnum:
		{
			if v != 123 {
				t.Errorf("Expected (eval x) -> 123, got %v instead", v)
			}
			t.Log("(eval x) -> 123")
		}
	default:
		t.Errorf("Expected (eval x) -> 123, got %v instead", v)
	}
}

func TestEvalLambda(t *testing.T) {
	testLambda := lambda{
		fn:      symbol("x"),
		arglist: []string{"x"}}

	n := testLambda.Eval(nilEnv)
	switch v := n.(type) {
	case lambda:
		t.Log("(eval (lambda x x)) -> (lambda x x)")
	default:
		t.Errorf("Expected (eval (lambda x x)) -> (lambda x x), got %v instead", v)
	}
}
func TestEvalList(t *testing.T) {
	testLambda := lambda{
		fn:      symbol("x"),
		arglist: []string{"x"}}
	testList := list{testLambda, fixnum(5)}

	n := testList.Eval(nilEnv)
	switch v := n.(type) {
	case fixnum:
		if v != 5 {
			t.Errorf("Expected (eval ((lambda x x) 5) -> 5, got %v instead", v)
		}
		t.Log("(eval ((lambda x x) 5) -> 5")
	default:
		t.Errorf("Expected (eval ((lambda x x) 5) -> 5, got %v instead", v)
	}
}

func TestParseList(t *testing.T) {
	input := [][]string{
		[]string{")"},
		[]string{"(", ")", ")"},
		[]string{"+", "1", "2", "3", ")"}}
	expected := []LispObject{
		Nil,
		list{Nil},
		list{symbol("+"), fixnum(1), fixnum(2), fixnum(3)}}
	for i := range input {
		obj, _ := ParseList(input[i])
		if !reflect.DeepEqual(obj, expected[i]) {
			t.Logf("expected %v to parse to %v, got %v", input[i], expected[i], obj)
			t.Fail()
		}
	}
}
