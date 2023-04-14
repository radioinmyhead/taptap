package view

import (
	"testing"
)

func TestGet(t *testing.T) {
	v := NewView(nil, 0)
	v.GetRelBig(0, 0)
}

func TestCmn(t *testing.T) {
	v := NewView(nil, 0)
	v.Cmn(3, 2)
	//for i := 1; i <= 7; i++ {
	//c := v.Count(i)
	//fmt.Println(i, c)
	//}
}
