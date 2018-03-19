package policy

import (
	"reflect"
	"testing"
)

func TestLex(t *testing.T) {

	var cases = []struct {
		Input string
		Type  itemType
		Value string
	}{
		//{"-1223", itemNumber, "-1223"},
		{"-1223", itemConstString, "-1223"},
		{`"a string with \"spaces"`, itemConstString, `a string with \"spaces`},
		{`%(blafasel)s`, itemVariable, `blafasel`},
		//{`True`, itemBool, `True`},
		{`True`, itemConstString, `True`},
		//{`False`, itemBool, `False`},
		{`False`, itemConstString, `False`},
		{`not`, itemNot, `not`},
	}

	for i, c := range cases {
		l := newLexer(c.Input)
		r := <-l.items
		if r.typ != c.Type {
			t.Errorf("Expected type %d, got %d for case %d (%s)", c.Type, r.typ, i, c.Input)
		}
		if r.val != c.Value {
			t.Errorf("Expected %#q, got %#q for case %d (%s)", c.Value, r.val, i, c.Input)
		}
	}

	l := newLexer(`@ and not (user_id:%(target.user_id)s and rule:blafasel) or (user_is:'blafa"sel') and True:%(target.what)s or is_admin:1`)

	expected := []itemType{itemTrueCheck, itemAnd, itemNot, itemLeftParen, itemString, itemColon, itemVariable, itemAnd, itemString, itemColon, itemString, itemRightParen, itemOr, itemLeftParen, itemString, itemColon, itemConstString, itemRightParen, itemAnd, itemConstString, itemColon, itemVariable, itemOr, itemString, itemColon, itemConstString, itemEOF}

	result := make([]itemType, 0, len(expected))
	for i := range l.items {
		result = append(result, i.typ)
		//fmt.Print(i)
		//if i.typ != itemEOF {
		//  fmt.Print(", ")
		//}
	}
	//fmt.Println()
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("\nExpected %#v\nGot      %#v", expected, result)
	}
}
