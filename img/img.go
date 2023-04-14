package img

import (
	"fmt"
	"image"
	"log"
	"math"

	"gocv.io/x/gocv"
	"golang.org/x/exp/slices"
)

// Color 记录颜色，RGB
type Color [3]uint8

func (c *Color) String() string {
	return fmt.Sprintf("(%v,%v,%v)", c[0], c[1], c[2])
}

func (c *Color) far(a Color) float64 {
	sum := float64(0)
	for k := range c {
		i := float64(c[k])
		j := float64(a[k])
		sum += math.Pow(i-j, 2)
	}
	return math.Sqrt(sum)
}

func (c *Color) IsDark() bool {
	sum := 0
	for _, k := range c {
		sum += int(k)
	}
	return sum/3 < 125
}

type ColorCount struct {
	Color Color
	Count int
}

type ColorRegion struct {
	x1, x2, y1, y2, z1, z2 uint8
	c                      *Color
}

func (cr ColorRegion) String() string {
	return fmt.Sprintf("%v-%v,%v-%v,%v-%v", cr.x1, cr.x2, cr.y1, cr.y2, cr.z1, cr.z2)
}

func (cr *ColorRegion) Match(x, y, z uint8) (v *Color, ok bool) {
	ok = cr.x1 <= x && x <= cr.x2 &&
		cr.y1 <= y && y <= cr.y2 &&
		cr.z1 <= z && z <= cr.z2
	if ok {
		v = cr.c
	}
	return
}

func NewColorRegion(x1, x2, y1, y2, z1, z2 uint8) *ColorRegion {
	return &ColorRegion{
		x1: x1,
		x2: x2,
		y1: y1,
		y2: y2,
		z1: z1,
		z2: z2,
		c:  &Color{0, 0, 0},
	}
}

type ColorRegionList []*ColorRegion

func (cl ColorRegionList) get(x, y, z uint8) (*Color, bool) {
	for _, one := range cl {
		v, ok := one.Match(x, y, z)
		if ok {
			return v, ok
		}
	}
	return nil, false
}

func DeleteColor(src gocv.Mat) (ret gocv.Mat) {
	var list ColorRegionList
	list = append(list, NewColorRegion(203, 203, 172, 174, 166, 166))
	list = append(list, NewColorRegion(99, 127, 76, 98, 75, 98))
	list = append(list, NewColorRegion(141, 141, 75, 76, 58, 58))
	bgr := gocv.Split(src)
	for i := 0; i < src.Rows(); i++ {
		for j := 0; j < src.Cols(); j++ {
			x := bgr[0].GetUCharAt(i, j)
			y := bgr[1].GetUCharAt(i, j)
			z := bgr[2].GetUCharAt(i, j)
			c := &Color{x, y, z}
			if v, ok := list.get(x, y, z); ok {
				c = v
			}
			bgr[0].SetUCharAt(i, j, uint8(c[0]))
			bgr[1].SetUCharAt(i, j, uint8(c[1]))
			bgr[2].SetUCharAt(i, j, uint8(c[2]))
		}
	}
	ret = gocv.NewMat()
	gocv.Merge(bgr, &ret)
	return ret
}

//func CutSize(){
// 上切200
// 下切120
// 右边切20
// 原图是1600*720，但是他上边，下边什么的都不能用，必须要切除。
//}

// TransformColor 如果图是4通道的，就转成3通道
func TransformColor(from gocv.Mat) (to gocv.Mat) {
	if from.Type() == gocv.MatTypeCV8UC4 {
		to = gocv.NewMat()
		gocv.CvtColor(from, &to, gocv.ColorBGRAToBGR)
		return
	}
	if from.Type() != gocv.MatTypeCV8UC3 {
		err := fmt.Errorf("need %v, get %v", gocv.MatTypeCV8UC3, from.Type())
		log.Fatal(err)
	}
	to = from.Clone()
	return to
}

func TransformSize(from gocv.Mat, size, offset int) (to gocv.Mat) {
	// 给我一张图，我处理成一个小矩形，并裁边
	// size, offset := 45, 3
	l := size + offset
	big := size + offset*2
	roiFrom := image.Rect(offset, offset, l, l)

	to = gocv.NewMatWithSize(l, l, gocv.MatTypeCV8UC3)
	from = TransformColor(from)
	gocv.Resize(from, &to, image.Point{big, big}, 0, 0, gocv.InterpolationArea)
	to = to.Region(roiFrom)
	return to
}

// ColorQuantization 用K种颜色重新画图,返回色板
func ColorQuantization(src gocv.Mat, K int) (img gocv.Mat, count []ColorCount) {
	// count = make(map[Color]int)
	img = TransformColor(src)
	img.ConvertTo(&img, gocv.MatTypeCV32F)
	img = img.Reshape(1, img.Total())

	bestLabels := gocv.NewMat()
	defer bestLabels.Close()
	criteria := gocv.NewTermCriteria(gocv.EPS+gocv.MaxIter, 100, 0.001)
	attempts := 5
	flags := gocv.KMeansRandomCenters
	centers := gocv.NewMat()
	defer centers.Close()
	gocv.KMeans(img, K, &bestLabels, criteria, attempts, flags, &centers)

	bestLabels.ConvertTo(&bestLabels, gocv.MatTypeCV8U) // 转一下，才能用后续的get方法
	dic := make([]int, K)                               // 统计一下那个颜色多。我要找到第二多的颜色
	for i := 0; i < bestLabels.Rows(); i++ {
		ci := bestLabels.GetUCharAt(0, i)
		dic[ci]++
		for j := 0; j < centers.Cols(); j++ {
			bgr := centers.GetFloatAt(int(ci), j)
			img.SetFloatAt(i, j, bgr)
		}
	}

	for i := 0; i < K; i++ {
		c := Color{}
		for j := 0; j < centers.Cols(); j++ {
			c[j] = uint8(centers.GetFloatAt(i, j))
		}
		count = append(count, ColorCount{
			Color: c,
			Count: dic[i],
		})
	}
	// 结果从大到小排列
	slices.SortFunc(count, func(a, b ColorCount) bool {
		return a.Count > b.Count
	})

	img = img.Reshape(3, src.Rows())
	img.ConvertTo(&img, gocv.MatTypeCV8UC3)
	return
}

type Target struct {
	img     gocv.Mat
	imgList []gocv.Mat
	bg      Color
	color   Color
	isNum   bool
	num     int
}

func NewTarget(img gocv.Mat) *Target {
	tar := &Target{
		img: img,
	}
	return tar
}

//func (t *Target) SetNum(num int) {
//	t.num = num
//}
//
//func NewTarget(bgColor, mainColor Color) *Target {
//	tar := &Target{
//		bg:    bgColor,
//		color: mainColor,
//	}
//	tar.isNum = bgColor.IsDark()
//	return tar
//}
//
//type TargetList []*Target
//
//func (tl TargetList) Check(tar *Target) (index int) {
//	min := float64(500)
//	for i, current := range tl {
//		if tar.isNum != current.isNum {
//			continue
//		}
//		f := tar.color.far(current.color)
//		if f < min {
//			min = f
//			index = i
//		}
//	}
//	return
//}
//
