package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
	"sort"
)

func help() {
	arguments := []string{}
	arguments = append(arguments, fmt.Sprintf("%s [INPUT] [OUTPUT] [WINDOW]", os.Args[0]))
	arguments = append(arguments, "  INPUT - a path to png/jpg image")
	arguments = append(arguments, "  OUTPUT - a name for a png image")
	arguments = append(arguments, "  WINDOW - indow to use for the filter, odd number")
	for _, arg := range arguments {
		fmt.Println(arg)
	}
}

func main() {
	if len(os.Args) < 4 {
		help()
		os.Exit(1)
	}

	img, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("ERROR: could not open the image: ", err)
		os.Exit(1)
	}
	defer img.Close()

	image_data, _, err := image.Decode(img)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}

	output_file, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println("ERROR: could create the output file: ", err)
		os.Exit(1)
	}

	png.Encode(output_file, kuwahara(image_data, 11))
}

func kuwahara(img image.Image, window int) image.Image {
	bounds := img.Bounds()
	new_image := image.NewRGBA(bounds)

	if window%2 == 0 {
		fmt.Println("ERROR: the window needs to be odd")
	}

	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			r, g, b, a := kuwahara_window(img, window, image.Point{x, y}).RGBA()
			pixel := x*4 + y*bounds.Max.X*4
			new_image.Pix[pixel] = uint8(r)
			new_image.Pix[pixel+1] = uint8(g)
			new_image.Pix[pixel+2] = uint8(b)
			new_image.Pix[pixel+3] = uint8(a)
		}
	}

	return new_image
}

type KuwQuarter struct {
	mean  color.Color
	stdev uint32
}

func sum_colors(r, g, b *uint32, col1 color.Color) {
	r1, g1, b1, _ := col1.RGBA()

	*r += r1 >> 8
	*g += g1 >> 8
	*b += b1 >> 8
}

func calc_stdev(img image.Image, q_size uint32, pos image.Point, means color.Color) uint32 {
	var sum_r, sum_g, sum_b uint32 = 0, 0, 0
	mean_r, mean_g, mean_b, _ := means.RGBA()

	for y := pos.Y; y < int(q_size)+pos.Y; y++ {
		for x := pos.X; x < int(q_size)+pos.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum_r += (r - mean_r) * (r - mean_r)
			sum_g += (g - mean_g) * (g - mean_g)
			sum_b += (b - mean_b) * (b - mean_b)
		}
	}

	return (sum_r/(q_size*q_size) + sum_g/(q_size*q_size) + sum_b/(q_size*q_size)) / 3
}

func calc_rect(img image.Image, q_size uint32, pos image.Point) KuwQuarter {
	var r, g, b uint32 = 0, 0, 0

	for wy := pos.Y; wy < int(q_size)+pos.Y; wy++ {
		for wx := pos.X; wx < int(q_size)+pos.X; wx++ {
			sum_colors(&r, &g, &b, img.At(wx, wy))
		}
	}
	mean := color.RGBA{
		uint8(r / (q_size * q_size)),
		uint8(g / (q_size * q_size)),
		uint8(b / (q_size * q_size)),
		255,
	}

	stdev := calc_stdev(img, q_size, pos, mean)
	return KuwQuarter{mean, stdev}
}

func kuwahara_window(img image.Image, window int, pos image.Point) color.Color {
	var frect, srect, trect, prect KuwQuarter
	var q_size uint32 = uint32(window)/2 + 1

	min_stdev := []KuwQuarter{}

	min_x := pos.X - int(q_size) + 1
	min_y := pos.Y - int(q_size) + 1
	max_x := pos.X + int(q_size) - 1
	max_y := pos.Y + int(q_size) - 1

	if min_x >= img.Bounds().Min.X {
		if min_y >= img.Bounds().Min.Y {
			frect = calc_rect(img, q_size, image.Point{min_x, min_y})
			min_stdev = append(min_stdev, frect)
		}
		if max_y <= img.Bounds().Max.Y {
			trect = calc_rect(img, q_size, image.Point{min_x, pos.Y})
			min_stdev = append(min_stdev, trect)
		}
	}

	if max_x <= img.Bounds().Max.X {
		if min_y >= img.Bounds().Min.Y {
			srect = calc_rect(img, q_size, image.Point{pos.X, min_y})
			min_stdev = append(min_stdev, srect)
		}
		if max_y <= img.Bounds().Max.Y {
			prect = calc_rect(img, q_size, image.Point{pos.X, pos.Y})
			min_stdev = append(min_stdev, prect)
		}
	}

	sort.Slice(min_stdev, func(i, j int) bool {
		return min_stdev[i].stdev < min_stdev[j].stdev
	})

	// fmt.Println(min_stdev)

	return min_stdev[0].mean
}
