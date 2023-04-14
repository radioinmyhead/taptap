package view

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"math"

	"taptap/biz/cell"

	"gocv.io/x/gocv"
)

// View 局部区域抽象成一个view
type View struct {
	list []*cell.Cell
	cols int
}

// NewView 给定一个cell的list和base，得到一个view
func NewView(list []*cell.Cell, base int) *View {
	return &View{
		list: list,
		cols: base,
	}
}

// Rows 返回有多少行
func (v *View) Rows() int {
	return len(v.list) / v.cols
}

// Cols 返回有多少列
func (v *View) Cols() int {
	return v.cols
}

// 根据x,y计算真实的index
func (v *View) GetIndex(x, y int) int {
	return cell.Index(x, y, v.cols)
}

// 根据x,y得到对应的cell
func (v *View) GetCell(x, y int) *cell.Cell {
	// todo 下标越界
	index := v.GetIndex(x, y)
	return v.list[index]
}

// GetSub 根据x,y得到这一个区域的9个格子
func (v *View) GetSub(c *cell.Cell) (sub []*cell.Cell) {
	for _, p := range c.Pt().GetSub() {
		sub = append(sub, v.GetCell(p.X, p.Y))
	}
	return
}

// 给我一个x,y得到可以与之关联的12个格子的中心
func (v *View) GetRel(c *cell.Cell) (sub []*cell.Cell) {
	for _, p := range c.Pt().GetRel() {
		sub = append(sub, v.GetCell(p.X, p.Y))
	}
	return
}

func (v *View) showCell(src *gocv.Mat, cell *cell.Cell, red bool) {
	c := color.RGBA{0, 255, 0, 0}
	if red {
		c = color.RGBA{255, 0, 0, 0}
	}
	gocv.Circle(src, cell.Point(), cell.Step(), c, 3)
}

func (v *View) Show3(src *gocv.Mat, boom, empty []*cell.Cell) {
	for _, cell := range boom {
		v.showCell(src, cell, true)
	}
	for _, cell := range empty {
		v.showCell(src, cell, false)
	}
}

func (v *View) Show() {
	if len(v.list)%v.cols != 0 {
		fmt.Println("col error")
		return
	}

	var buf bytes.Buffer

	buf.Write([]byte("   "))
	for i := 0; i < v.cols; i++ {
		d := fmt.Sprintf("%d ", i%10)
		buf.Write([]byte(d))
	}
	row := -1
	for i, cell := range v.list {
		x := i / v.cols
		if row != x {
			row = x
			d := fmt.Sprintf("\n%v: ", row%10)
			buf.Write([]byte(d))
		}

		buf.Write([]byte{cell.Byte()})
		buf.Write([]byte{' '})
	}
	fmt.Println(buf.String())

	return
}

func (v *View) Show2() {
	dic := map[byte]int{
		'0': 0,
		'1': 1,
		'2': 2,
		'3': 3,
		'4': 4,
		'5': 5,
		'6': 6,
		'7': 7,
		'8': 8,
		'f': 9,
		'?': 10,
		'_': 11,
	}
	list := [][]int{}
	tmp := []int{}
	for _, c := range v.list {
		b := c.Byte()
		i, ok := dic[b]
		if !ok {
			panic("show")
		}
		tmp = append(tmp, i)
		if len(tmp) == v.cols {
			list = append(list, tmp)
			tmp = []int{}
		}
	}
	j, _ := json.Marshal(list)
	fmt.Println(string(j))
}

func (v *View) FindBoom() (boom []*cell.Cell) {
	/*
		有N个没开的格子，有N个雷，那么所有的格子都是雷
		偏移量

			1,1
				-1,-1
	*/

	for i := 1; i < v.Rows()-1; i++ {
		for j := 1; j < v.Cols()-1; j++ {
			main := v.GetCell(i, j)             // 中心
			sub := v.GetSub(main)               // 9个格子
			list := append(sub[:4], sub[5:]...) // 边缘

			if main.IsUnTap() {
				continue // 没开的格子
			}
			if main.Byte() == '0' {
				continue // 铁定没雷
			}

			tmp := []*cell.Cell{}
			num := 0
			for _, cell := range list {
				if cell.IsUnknown() {
					tmp = append(tmp, cell)
				}
				if cell.IsUnTap() {
					num++
				}
			}
			if num == main.Int() {
				boom = append(boom, tmp...)
			}
		}
	}
	return boom
}

func (v *View) SetFlag(x, y int) {
	cell := v.GetCell(x, y)
	cell.SetFlag()
}

func (v *View) FindNum() (empty []*cell.Cell) {
	/*
		如果周围的雷和数字一致，剩余空间都不是雷
		偏移量:

			1,1
				-1,-1
	*/
	for i := 1; i < v.Rows()-1; i++ {
		for j := 1; j < v.Cols()-1; j++ {
			main := v.GetCell(i, j)
			sub := v.GetSub(main)
			list := append(sub[:4], sub[5:]...) // 边缘

			if main.IsUnTap() {
				continue
			}
			if main.Byte() == '0' {
				continue
			}
			num := 0
			tmp := []*cell.Cell{}
			for _, cell := range list {
				if cell.IsFlag() {
					num++
				}
				if cell.IsUnknown() {
					tmp = append(tmp, cell)
				}
			}
			if num == main.Int() {
				empty = append(empty, tmp...)
			}
		}
	}
	return empty
}

func (v *View) FindDiff() (boom, empty []*cell.Cell) {
	/*
		假设 n<=m, n分布在AB中，m分布在BC中，那么，日常满足：
		m-n<=C && B<=n && 0<=A<=n
		当m-n=C时, 此时C全是雷，A全不是

		因为mn要有交集, 所以偏移量会大一点

			1,3
			-3,-3
	*/
	for i := 1; i < v.Rows()-3; i++ {
		for j := 3; j < v.Cols()-3; j++ {
			main := v.GetCell(i, j)
			if main.IsUnTap() || main.Int() == 0 {
				continue
			}
			list := v.GetRel(main)
			for k := 1; k < len(list); k++ {
				cell := list[k]
				if cell.IsUnTap() || cell.Int() == 0 {
					continue
				}
				tmpB, tmpE := v.diff(main, cell)
				boom = append(boom, tmpB...)
				empty = append(empty, tmpE...)
			}
		}
	}
	return
}

func (v *View) diff(cell1, cell2 *cell.Cell) (boom, empty []*cell.Cell) {
	if cell1.Gt(cell2) {
		cell1, cell2 = cell2, cell1
	}
	sub1 := v.GetSub(cell1)
	sub2 := v.GetSub(cell2)
	n := sub1[4].Int()
	m := sub2[4].Int()
	list1 := v.filterAround(sub1)
	list2 := v.filterAround(sub2)
	A := v.Sub(list1, list2)
	C := v.Sub(list2, list1)
	if m-n == len(C) {
		for _, cell := range C {
			if cell.IsUnknown() {
				boom = append(boom, cell)
			}
		}
		empty = append(empty, A...)
	}
	return
}

func (v *View) Sub(list1, list2 []*cell.Cell) (sub []*cell.Cell) {
	dic := make(map[int]bool)
	for _, cell := range list1 {
		if cell.IsUnTap() {
			index := cell.Index(v.cols)
			dic[index] = true
		}
	}
	for _, cell := range list2 {
		if cell.IsUnTap() {
			index := cell.Index(v.cols)
			dic[index] = false
		}
	}
	for k, exit := range dic {
		if exit {
			sub = append(sub, v.list[k])
		}
	}
	return
}

func (v *View) Reset(x, y int) {
	cell := v.GetCell(x, y)
	cell.SetUnknown()
}

func (v *View) And(list1, list2 []*cell.Cell) (and []*cell.Cell) {
	dic := make(map[int]bool)
	for _, cell := range list1 {
		index := cell.Index(v.cols)
		dic[index] = false
	}
	for _, cell := range list2 {
		index := cell.Index(v.cols)
		if _, ok := dic[index]; ok {
			dic[index] = true
		}
	}
	for index, exit := range dic {
		if exit {
			and = append(and, v.list[index])
		}
	}
	return and
}

func (v *View) filterAround(a []*cell.Cell) (and []*cell.Cell) {
	and = append(and, a[:4]...)
	return append(and, a[5:]...)
}

func (v *View) filterFlag(sub []*cell.Cell) (and []*cell.Cell) {
	for _, cell := range sub {
		if cell.IsFlag() {
			and = append(and, cell)
		}
	}
	return
}

func (v *View) filterUnKnown(sub []*cell.Cell) (and []*cell.Cell) {
	for _, cell := range sub {
		// fmt.Println(cell.Row, cell.Col)
		if cell.IsUnknown() {
			and = append(and, cell)
		}
	}
	return
}

func (v *View) FindWa() (boom, empty []*cell.Cell) {
	/*
		偏移量

			3,3
			-3,-3
	*/
	//fmt.Println("s")
	for i := 3; i < v.Rows()-3; i++ {
		for j := 3; j < v.Cols()-3; j++ {
			// i, j = 6, 5
			sub := v.GetRelBig(i, j)
			if len(sub) == 0 {
				// 这个格子，不适合挖
				continue
			}
			//r := ""
			//for _, cell := range sub {
			//	r += fmt.Sprintf("(%v,%v:%s)", cell.Row, cell.Col, cell.S())
			//}
			//fmt.Println(i, j, len(sub), r)

			// 这个格子能挖，可以挖的新格子，放在sub里。
			// 需要分别尝试挖一下
			max := 3
			if len(sub) < max {
				max = len(sub)
			}
			for w := 2; w <= max; w++ {
				// fmt.Println("尝试挖", w, "个")
				// 挖w个，先尝试挖2个，最大是都挖了。
				e, b := v.Wa(i, j, w, sub)
				if len(e) > 0 {
					// 挖到了
					empty = append(empty, e...)
				}
				if len(b) > 0 {
					// 挖到了
					boom = append(boom, b...)
				}
			}
		}
	}
	return
}

func (v *View) Wa(i, j, w int, sub []*cell.Cell) (ret, boom []*cell.Cell) {
	/*
		给我一个格子i,j
		挖w个位置，sub是这些位置的array
	*/

	// 有这么多能挖的方案
	main := v.GetCell(i, j)
	list := v.Cmn(len(sub), w)
	// fmt.Printf("有%d个方案\n", len(list))
	for _, one := range list {
		tmp := []*cell.Cell{}
		tmp = append(tmp, main)
		sum := 0
		for index, b := range one {
			if b {
				c := sub[index]
				s := v.filterFlag(v.filterAround(v.GetSub(c)))
				sum += c.Int() - len(s)
				tmp = append(tmp, c)
			}
		}
		mainCount := main.Int() - len(v.filterFlag(v.filterAround(v.GetSub(main))))
		if mainCount < sum {
			// 溢出了
			continue
		}

		empty, err := v.wa2(tmp)
		if err != nil {
			continue
		}
		if len(empty) == mainCount-sum {
			boom = append(boom, empty...)
		}
		if mainCount == sum && len(empty) > 0 {
			ret = append(ret, empty...)
		}
	}
	return
}

func (v *View) wa3(list, sub []*cell.Cell) (ret []*cell.Cell, err error) {
	// 严格减法，如果有找不到的，要报错
	dic := make(map[int]bool)
	for _, one := range list {
		index := one.Index(v.cols)
		dic[index] = true
	}
	for _, one := range sub {
		index := one.Index(v.cols)
		_, ok := dic[index]
		if !ok {
			err = errors.New("fail")
			return
		}
		dic[index] = false
	}
	for index, exit := range dic {
		if exit {
			ret = append(ret, v.list[index])
		}
	}
	return
}

func (v *View) wa2(list []*cell.Cell) (ret []*cell.Cell, err error) {
	// 0是被挖的这一个，省下的是要挖的
	// 这里是挖挖看
	main := list[0]
	mainlist := v.GetSub(main)
	mainlist = v.filterAround(mainlist)
	mainlist = v.filterUnKnown(mainlist)
	for i := 1; i < len(list); i++ {
		cell := list[i]
		sublist := v.GetSub(cell)
		sublist = v.filterAround(sublist)
		sublist = v.filterUnKnown(sublist)
		mainlist, err = v.wa3(mainlist, sublist)
		if err != nil {
			return
		}
	}
	return mainlist, nil
}

func (v *View) Count(x int) (count int, list []bool) {
	for x > 0 {
		var b bool
		if x&0x1 == 1 {
			count++
			b = true
		}
		list = append(list, b)
		x = (x >> 1)
	}
	return count, list
}

func (v *View) Cmn(m, n int) (ret [][]bool) {
	// 从m中取出n个数
	// 我需要一个m位的二进制，这个数字将是
	for i := 1; i < int(math.Pow(2, float64(m))); i++ {
		c, l := v.Count(i)
		if c == n {
			ret = append(ret, l)
		}
	}
	return
}

func (v *View) GetRelBig(x, y int) []*cell.Cell {
	/*
		受影响的格子，还有一种挖的情况，思路是挖出m个格子，里边包括了n个雷，此时m>n>0
		此时n最小是1，m最小是2
		此时有20个位置可能有影响
		上半截是10个，下半截是10个，如图：

			- 1 2 3 -
			4 5 6 7 8
			9 a x 1 2
			3 4 5 6 7
			- 8 9 a -

		但是实际中，有非常多的剪枝
		- 如果两个没有空白格子的交集，不会挖
		- 如果两个的交集只有1个，其实不会发生挖的情况
		- 如果这个位置有不重合的格子，也不会挖。

		因为一个数字周围只有8个格子，m最小是2，所以，最差的情况，只能挖4次就挖没了。
		所以挖的最大数是3
		所以，剪枝后，符合要求的格子，要做排列组合，分别取出2个，3个

		如果剪枝后，只有一个格子，直接放弃
		剪枝后的格子，只有两个，仅仅做取个就行。
	*/
	main := v.GetCell(x, y)
	if main.IsUnTap() || main.Byte() == '0' {
		// 如果没点开，或者是0，那就跳过
		return nil
	}
	mainList := v.GetSub(main)
	mainList = v.filterUnKnown(v.filterAround(mainList))
	if len(mainList) < 5 {
		// 最少要有5个空格子，才能这样分析
		return nil
	}
	sub := []*cell.Cell{}
	for offset := 0; offset < 25; offset++ {
		rowOffset := offset/5 - 2
		colOffset := offset%5 - 2
		// 剪枝0,
		c := rowOffset * colOffset
		if c == 4 || c == -4 || (rowOffset == 0 && colOffset == 0) {
			// 排除角落的4个, 排除自己
			continue
		}
		row := x + rowOffset
		col := y + colOffset
		cell := v.GetCell(row, col)
		if cell.IsUnTap() || cell.Byte() == '0' {
			// 如果没点开，或者是0，那就跳过
			continue
		}
		subList := v.GetSub(cell)
		subList = v.filterUnKnown(v.filterAround(subList)) // sub的空白格子
		sub1 := v.And(mainList, subList)
		// 剪枝1， 如果两个格子没交集，就continue
		if len(sub1) == 0 {
			continue
		}
		// 剪枝2,如果两个格子只有一个交集，continue
		if len(sub1) == 1 {
			continue
		}
		sub2 := v.Sub(subList, mainList)
		// 剪枝3,如果第二个格子不是被完全包含，continue
		if len(sub2) != 0 {
			continue
		}
		sub = append(sub, cell)
	}
	// 剪枝，如果没有，或者就一个，剪枝
	if len(sub) < 2 {
		return nil
	}
	return sub // 找到的关联格子
}
