package cell

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

type Point struct {
	X, Y int
}

func Pt(x, y int) Point {
	return Point{x, y}
}

func (p Point) GetRel() (list []Point) {
	/*
		12个位置如下：

					X	1	2
			3	4	5	6	7
			8	9	10	11	12
	*/
	for offset := 0; offset < 13; offset++ {
		rowOffset := (offset + 2) / 5
		colOffset := (offset+2)%5 - 2
		row := p.X + rowOffset
		col := p.Y + colOffset
		list = append(list, Pt(row, col))
	}
	return
}

func (p Point) GetSub() (list []image.Point) {
	/*
		9个offset如下

			-1,-1 -1,+0 -1,+1
			+0,-1 +0,+0 +0,+1
			+1,-1 +1,+0 +1,+1
	*/
	for offset := 0; offset < 9; offset++ {
		rowOffset := offset/3 - 1
		colOffset := offset%3 - 1
		row := p.X + rowOffset
		col := p.Y + colOffset
		list = append(list, image.Pt(row, col))
	}
	return
}

type Cell struct {
	row      int       // 相对坐标 row
	col      int       // 相对坐标 col
	mat      *gocv.Mat // 这个坐标的小图
	centerX  int       // 这个小图的中心点x
	centerY  int       // 小图中心点y
	ret      byte      // 小图的识别内容
	retIndex int
}

func (c *Cell) Pt() Point {
	return Pt(c.row, c.col)
}

func (c *Cell) Step() int {
	s := c.mat.Size()
	r := s[0] / 3
	return r
}

func (c *Cell) Point() image.Point {
	return image.Point{c.centerX, c.centerY}
}

func New(row, col, x, y int, mat *gocv.Mat, ret byte, retIndex int) *Cell {
	return &Cell{
		row:      row,
		col:      col,
		mat:      mat,
		centerX:  x,
		centerY:  y,
		ret:      ret,
		retIndex: retIndex,
	}
}

func (c *Cell) Byte() byte {
	return c.ret
}

func (c *Cell) S() string {
	return string([]byte{c.ret})
}

func (c *Cell) String() string {
	s := fmt.Sprintf("%v,%v:", c.row, c.col)
	return "(" + s + string([]byte{c.ret}) + ")"
}

func (c *Cell) Int() int {
	return c.retIndex
}

func (c *Cell) IsUnTap() bool {
	return c.IsFlag() || c.IsUnknown()
}

func (c *Cell) IsUnknown() bool {
	return c.ret == '_'
}

func (c *Cell) IsFlag() bool {
	return c.ret == 'f'
}

func (c *Cell) SetFlag() {
	c.ret = 'f'
	c.retIndex = -1
	fmt.Println("find boom at", c.row, c.col)
}

func (c *Cell) SetUnknown() {
	c.ret = '_'
	c.retIndex = -3
	fmt.Println("find unk  at", c.row, c.col)
}

func (c *Cell) Tap() {
	fmt.Println("find num  at", c.row, c.col)
}

func (a *Cell) Gt(b *Cell) bool {
	// a>b
	return a.retIndex > b.retIndex
}

func Index(x, y, base int) int {
	return x*base + y
}

func (c *Cell) Index(base int) int {
	return Index(c.row, c.col, base)
}
