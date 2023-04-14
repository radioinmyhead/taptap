package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"taptap/biz/cell"
	"taptap/biz/view"
	"taptap/img"

	"gocv.io/x/gocv"
)

/*
todo
先知道方块的大小，然后判断一下，这个线距离边缘够不远远，够远才是真边
*/
var (
	tarDir   = "./tar"
	filename = "./1.jpg"
	pi       = math.Pi
)

func showIM(title string, src gocv.Mat) {
	window := gocv.NewWindow(title)
	defer window.Close()
	window.IMShow(src)
	window.WaitKey(-1)
}

func saveIM(title string, src gocv.Mat) {
	gocv.IMWrite(title, src)
}

func saveIMindex(title string, index any, src gocv.Mat) {
	title = fmt.Sprintf("%v-%v.png", title, index)
	gocv.IMWrite(title, src)
}

func ColorQuantization(src gocv.Mat) (ret gocv.Mat, bgColor, mainColor img.Color) {
	ret, dic := img.ColorQuantization(src, 2)

	bgColor = dic[0].Color   // BGR
	mainColor = dic[1].Color // BGR
	return
}

func getTarOne(i int) (a, b, c, d gocv.Mat, bg, mainColor img.Color) {
	// 获取编号 i 的二值图, 顺手看下是不是数字
	title := fmt.Sprintf(tarDir+"/tar%v.png", i)
	small := gocv.IMRead(title, 1)
	if small.Empty() {
		log.Fatal("read tar")
	}
	small = img.TransformSize(small, 45, 3)

	from, bg, mainColor := ColorQuantization(small) // BGR
	gray := gocv.NewMat()
	gocv.CvtColor(from, &gray, gocv.ColorBGRToGray)
	f3 := adaptiveThreshold(gray)
	return small, from, gray, f3, bg, mainColor
}

//type tar struct {
//	isNum bool
//	color [3]float32 // BGR
//}

func gets(x, y int) image.Rectangle {
	k := 45
	return image.Rect(x, y, x+k, y+k)
}

func getS(x int) (list [5]image.Rectangle) {
	t := 5
	for i := 0; i < 5; i++ {
		list[i] = gets(x, t+50*i)
	}
	return
}

func getTar() (gocv.Mat, img.TargetList) {
	//pic := map[int]byte{
	//	-5: '#',
	//	-4: '_',
	//	-3: '_',
	//	-2: 'f',
	//	-1: 'f',
	//	0:  '0',
	//	1:  '1',
	//	2:  '2',
	//	3:  '3',
	//	4:  '4',
	//	5:  '5',
	//	6:  '6',
	//	7:  '7',
	//	8:  '8',
	//}
	var dic img.TargetList
	empty := gocv.NewMatWithSize(5+50*5, 5+50*13, gocv.MatTypeCV8UC3)
	x := 5
	for i := -4; i < 9; i++ {
		aa, bb, cc, dd, bg, mainColor := getTarOne(i)
		defer aa.Close()
		defer bb.Close()
		defer cc.Close()
		defer dd.Close()
		ee := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(
			float64(mainColor[0]),
			float64(mainColor[1]),
			float64(mainColor[2]),
			0,
		), 45, 45, gocv.MatTypeCV8UC3)
		defer ee.Close()

		gocv.CvtColor(cc, &cc, gocv.ColorGrayToBGR)
		gocv.CvtColor(dd, &dd, gocv.ColorGrayToBGR)

		r := getS(x)
		x += 50
		r0 := empty.Region(r[0])
		r1 := empty.Region(r[1])
		r2 := empty.Region(r[2])
		r3 := empty.Region(r[3])
		r4 := empty.Region(r[4])

		aa.CopyTo(&r0)
		bb.CopyTo(&r1)
		cc.CopyTo(&r2)
		dd.CopyTo(&r3)
		ee.CopyTo(&r4)
		//tar := &tar{
		//	isNum: num,
		//	color: mainColor,
		//}
		tar := img.NewTarget(bg, mainColor)
		tar.SetNum(i)
		dic = append(dic, tar)
		// fmt.Println("in tar", i, tar.isNum, tar.color)

		// dic[pic[i]] = tar
	}
	return empty, dic
}

func getImage() (src, gray gocv.Mat) {
	src = gocv.IMRead(filename, gocv.IMReadUnchanged)
	src = img.DeleteColor(src)
	gray = gocv.NewMat()
	gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	return
}

func adaptiveThreshold(gray gocv.Mat) gocv.Mat {
	dst := gocv.NewMat()
	gocv.AdaptiveThreshold(gray, &dst, 255, gocv.AdaptiveThresholdGaussian, gocv.ThresholdBinary, 5, 0)
	return dst
}

func getLine(dst gocv.Mat) (h, v, mask gocv.Mat) {
	lineh := gocv.NewMat()
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(40, 1)) // 40
	gocv.Erode(dst, &lineh, kernel)
	gocv.Dilate(lineh, &lineh, kernel)

	linev := gocv.NewMat()
	kernel = gocv.GetStructuringElement(gocv.MorphRect, image.Pt(1, 33))
	gocv.Erode(dst, &linev, kernel)
	gocv.Dilate(linev, &linev, kernel)

	line := gocv.NewMat()
	gocv.Add(lineh, linev, &line)
	return lineh, linev, line
}

func cropImage(xList, yList []int, src gocv.Mat, dic img.TargetList) (list []*cell.Cell) {
	xStep := getStep(xList)
	yStep := getStep(yList)
	for i := 1; i < len(xList); i++ {
		for j := 1; j < len(yList); j++ {
			// i, j = 1, 5
			// i = 5
			// fmt.Println("===")
			a := xList[i-1]
			b := xList[i]
			c := yList[j-1]
			d := yList[j]
			r := image.Rect(c, a, d, b)
			s := src.Region(r)
			t := s.Clone()
			ret, retIndex := checkImage(t, dic)
			cc := cell.New(
				i-1,
				j-1,
				c+xStep,
				a+yStep,
				&t,
				ret,
				retIndex,
			)
			// fmt.Println(i-1, j-1, string([]byte{ret}), retIndex)

			list = append(list, cc)
			// fmt.Println(cc)
			// showIM("haha", *(cc.Mat))
		}
		// return
	}
	return
}

func getStep(list []int) (step int) {
	l := len(list)
	step = (list[l-1] - list[0]) / l / 2
	return
}

func get_x_list(lineh gocv.Mat) (list []int, img gocv.Mat) {
	// 相连的数字，分成很多簇，每个簇，计算出一个中间的一个数据
	// 每个簇的间隔必须大于10，否则不认为是一个簇
	// 如果一根线小于宽度的20%，直接省略
	tmp := []int{}
	sep := 0
	img = gocv.NewMatWithSize(lineh.Rows(), lineh.Cols(), gocv.MatTypeCV8U)
	for i := 0; i < lineh.Rows(); i++ {
		if i > lineh.Rows()-120 || i < 200 {
			continue
		}
		var sum int
		for j := 0; j < lineh.Cols(); j++ {
			f := lineh.GetUCharAt(i, j)
			if f > 0 {
				sum++
			}
		}
		if sum < lineh.Cols()*20/100 {
			sep++
			continue
		}
		if sep > 10 {
			if len(tmp) > 0 {
				x := tmp[0] + tmp[len(tmp)-1]
				x = x / 2
				list = append(list, x)
			}
			tmp = []int{}
			sep = 0
			gocv.Line(&img, image.Point{0, i}, image.Point{sum, i}, color.RGBA{255, 255, 255, 1}, 1)
		}
		tmp = append(tmp, i)
	}
	if len(tmp) > 0 {
		x := tmp[0] + tmp[len(tmp)-1]
		x = x / 2
		list = append(list, x)
	}
	return list, img
}

func get_y_list(linev gocv.Mat) (list []int, img gocv.Mat) {
	img = gocv.NewMatWithSize(linev.Rows(), linev.Cols(), gocv.MatTypeCV8U)
	sep := 0
	tmp := []int{}
	for j := 0; j < linev.Cols(); j++ {
		if j > 700 {
			continue
		}
		var sum int
		for i := 0; i < linev.Rows(); i++ {
			f := linev.GetUCharAt(i, j)
			if f > 0 {
				sum++
			}
		}
		if sum < linev.Cols()*55/100 {
			sep++
			continue
		}
		if sep > 10 {
			if len(tmp) > 0 {
				y := tmp[0] + tmp[len(tmp)-1]
				y = y / 2
				list = append(list, y)
			}
			sep = 0
			tmp = []int{}
		}
		tmp = append(tmp, j)
		gocv.Line(&img, image.Point{j, 0}, image.Point{j, sum}, color.RGBA{255, 255, 255, 1}, 1)
	}
	if len(tmp) > 0 {
		y := tmp[0] + tmp[len(tmp)-1]
		y = y / 2
		list = append(list, y)
	}
	return list, img
}

func drawGrid(x_list, y_list []int) gocv.Mat {
	grid := gocv.NewMatWithSize(1600, 720, gocv.MatTypeCV8U)
	for _, i := range x_list {
		pt1, pt2 := image.Point{0, i}, image.Point{720, i}
		gocv.Line(&grid, pt1, pt2, color.RGBA{255, 255, 255, 1}, 1)
	}
	for _, j := range y_list {
		pt1, pt2 := image.Point{j, 0}, image.Point{j, 1600}
		gocv.Line(&grid, pt1, pt2, color.RGBA{255, 255, 255, 1}, 1)
	}
	return grid
}

func showContours(src gocv.Mat, contours gocv.PointsVector, index int) {
	f11 := src.Clone()
	defer f11.Close()
	gocv.CvtColor(f11, &f11, gocv.ColorGrayToBGR)
	gocv.DrawContours(&f11, contours, index, color.RGBA{255, 0, 0, 1}, 1)
	showIM("sc", f11)
}

//func min(dic map[byte]*tar, isNum bool, mainColor [3]float32) (byte, int) {
//	// fmt.Println("in min", mainColor)
//	minF := float64(500)
//	minB := byte('?')
//	var index int
//	for b, tar := range dic {
//		if tar.isNum != isNum {
//			continue
//		}
//		f := far(tar.color, mainColor)
//		// fmt.Println("juli", string([]byte{b}), f)
//		if f < minF {
//			minF = f
//			minB = b
//		}
//	}
//	tmp := map[byte]int{
//		'_': -3,
//		'f': -1,
//		'0': 0,
//		'1': 1,
//		'2': 2,
//		'3': 3,
//		'4': 4,
//		'5': 5,
//		'6': 6,
//		'7': 7,
//		'8': 8,
//	}
//	index, ok := tmp[minB]
//	if !ok {
//		fmt.Println(string([]byte{minB}))
//		panic("aaa")
//	}
//	return minB, index
//}

func checkImage(src gocv.Mat, dic img.TargetList) (byte, int) {
	// 要检查的图需要先整理成45*45大小
	// showIM("check", img)
	ha := img.TransformSize(src, 45, 3)
	defer ha.Close()
	f1, bg, mainColor := ColorQuantization(ha)
	// fmt.Println(mainColor, isnum)
	// showIM("img", img)
	// showIM("img2", img2)
	showIM("img3", f1)
	// defer f1.Close()
	// return min(dic, isnum, mainColor)
	tar := img.NewTarget(bg, mainColor, 0)
	index := dic.Check(tar)
	tar.SetNum(index)
	return ' ', 0
}

func imgSaver(src gocv.Mat) {
	window := gocv.NewWindow("crop")
	defer window.Close()
	// haha := gocv.IMRead("/Users/bytedance/Desktop/taptap/1.jpg", gocv.IMReadColor)
	// haha = haha.Region(image.Rect(458, 1041, 689, 1270))
	// saveIM("/Users/bytedance/Desktop/taptap/tar8.png", haha)
	// window.IMShow(haha)
	// window.WaitKey(-1)
	r := window.SelectROI(src)
	fmt.Println(r)
	return
}

func x(src gocv.Mat) {
	// showIM("t", src)
	imgSaver(src)
	src = src.Region(image.Rect(428, 575, 456, 615))
	// 看一个图里一共有多少颜色，画3个图
	// gocv.Split(src)
	empty1 := gocv.NewMatWithSize(255, 255, gocv.MatTypeCV8UC3)
	empty2 := gocv.NewMatWithSize(255, 255, gocv.MatTypeCV8UC3)
	empty3 := gocv.NewMatWithSize(255, 255, gocv.MatTypeCV8UC3)
	src = img.TransformColor(src)
	fmt.Println(src.Type())
	bgr := gocv.Split(src)
	for j := 0; j < src.Cols(); j++ {
		for i := 0; i < src.Rows(); i++ {
			x := bgr[0].GetUCharAt(i, j)
			y := bgr[1].GetUCharAt(i, j)
			z := bgr[2].GetUCharAt(i, j)
			fmt.Println(x, y, z)
			p1 := image.Point{int(x), int(y)}
			p2 := image.Point{int(x), int(z)}
			p3 := image.Point{int(y), int(z)}
			gocv.Circle(&empty1, p1, 1, color.RGBA{z, y, x, 0}, 1)
			gocv.Circle(&empty2, p2, 1, color.RGBA{z, y, x, 0}, 1)
			gocv.Circle(&empty3, p3, 1, color.RGBA{z, y, x, 0}, 1)
		}
		return
	}
	showIM("1", empty1)
	showIM("2", empty2)
	showIM("3", empty3)
}

func main() {
	empty, dic := getTar()
	defer empty.Close()
	// showIM("tar", empty)
	src, gray := getImage()
	defer src.Close()
	defer gray.Close()

	showIM("src", src)
	// imgSaver(src)
	// x(src)
	// return
	// showIM("gray", gray)

	// img2, _ := img.ColorQuantization(src, 17)
	// showIM("gray", img2)
	// return

	dst := adaptiveThreshold(gray)
	defer dst.Close()
	showIM("dst", dst)

	lineh, linev, line := getLine(dst)
	defer lineh.Close()
	defer linev.Close()
	defer line.Close()
	x_list, _ := get_x_list(lineh)
	x_list, xstep := solveStep(x_list)
	// 上下到边了。

	y_list, _ := get_y_list(linev)
	y_list, ystep := solveStep(y_list)
	fmt.Println(xstep, ystep)
	// 左右到边了。
	// g := drawGrid(x_list, y_list)
	// showIM("g", g)

	cellList := cropImage(x_list, y_list, src, dic)

	v := view.NewView(cellList, len(y_list)-1)
	v.Show2()
	v.Show()
	boom1, empty1 := finder(v)
	v.Show3(&src, boom1, empty1)
	showIM("ret", src)
	return
}

func solveStep(list []int) (tmp []int, step int) {
	steps := []int{}
	for i := 1; i < len(list); i++ {
		steps = append(steps, list[i]-list[i-1])
	}
	dic := make(map[int]int)
	for _, k := range steps {
		dic[k] += 1
	}
	max := 0
	// step := 0
	for k, count := range dic {
		if count > max {
			step = k
			max = count
		}
	}
	// tmp := []int{}
	tmp = append(tmp, list[0])
	for i, k := range steps {
		if k < step/2 {
			return
		}
		if float64(k) < float64(step)*0.9 || float64(k) > float64(step)*1.1 {
			get := k / step
			a := math.Abs(float64(k - (get+0)*step))
			b := math.Abs(float64(k - (get+1)*step))
			if b < a {
				get += 1
			}
			for j := 1; j < get; j++ {
				// fmt.Println("=====", i, k, get, list[i+1], list[i+1]-step*j)
				tmp = append(tmp, list[i+1]-step*j)
			}
		}
		tmp = append(tmp, list[i+1])
	}
	return
}

func finder(view *view.View) (boom, empty []*cell.Cell) {
	boom1 := view.FindBoom()
	boom = append(boom, boom1...)

	empty1 := view.FindNum()
	empty = append(empty, empty1...)

	boom2, empty2 := view.FindDiff()
	boom = append(boom, boom2...)
	empty = append(empty, empty2...)

	boom3, empty3 := view.FindWa()
	boom = append(boom, boom3...)
	empty = append(empty, empty3...)

	if len(boom) > 0 {
		for _, cell := range boom {
			cell.SetFlag()
		}
		b, e := finder(view)
		boom = append(boom, b...)
		empty = append(empty, e...)
		return
	}
	if len(empty) > 0 {
		for _, cell := range empty {
			// todo 挖开这个格子
			cell.Tap()
		}
	}
	return
}
