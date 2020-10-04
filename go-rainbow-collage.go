package gorainbowcollage

import (
	"image"
	"image/color"
	"math"
	"sort"
	"sync"

	gim "github.com/ozankasikci/go-image-merge"
)

type entryColor struct {
	Hue   float64
	Sat   float64
	Value float64
}

type collageEntry struct {
	Image *image.Image
	Color entryColor
}

// hsv converts a go color type to HSV format (Hue, Saturation, Value)
func hsv(c color.Color) (h, s, v float64) {
	fR, fG, fB := normalize(c)

	max := math.Max(math.Max(fR, fG), fB)
	min := math.Min(math.Min(fR, fG), fB)
	d := max - min
	s, v = 0, max
	if max > 0 {
		s = d / max
	}
	if max == min {
		// Achromatic.
		h = 0
	} else {
		// Chromatic.
		switch max {
		case fR:
			h = (fG - fB) / d
			if fG < fB {
				h += 6
			}
		case fG:
			h = (fB-fR)/d + 2
		case fB:
			h = (fR-fG)/d + 4
		}
		h /= 6
	}
	return
}

// Convert Go colors to RGB values in range [0,1).
func normalize(col color.Color) (r, g, b float64) {
	ri, gi, bi, _ := col.RGBA()
	r = float64(ri) / float64(0x10000)
	g = float64(gi) / float64(0x10000)
	b = float64(bi) / float64(0x10000)
	return
}

// averageImageColor gets the average RGB color from an image type by inspecting all pixels and determining the average
func averageImageColor(i image.Image) color.Color {
	var r, g, b uint32

	bounds := i.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pr, pg, pb, _ := i.At(x, y).RGBA()

			r += pr
			g += pg
			b += pb
		}
	}

	d := uint32(bounds.Dy() * bounds.Dx())

	r /= d
	g /= d
	b /= d

	return color.NRGBA{uint8(r / 0x101), uint8(g / 0x101), uint8(b / 0x101), 255}
}

func rearrangeImages(entries []collageEntry, x int, y int) {

	matrix := make([][]collageEntry, y)
	for i := 0; i < y; i++ {
		matrix[i] = make([]collageEntry, x)
	}

	row, col := 0, 0
	var dir = true
	l := 0

	for row < y && col < x {
		matrix[row][col] = entries[l]
		l++

		var newRow int
		var newCol int

		if dir {
			newRow = row + -1
			newCol = col + 1
		} else {
			newRow = row + 1
			newCol = col + -1
		}

		if newRow < 0 || newRow == y || newCol < 0 || newCol == x {
			if dir {
				if col == x-1 {
					row++
				}
				if col < x-1 {
					col++
				}
			} else {
				if row == y-1 {
					col++
				}
				if row < y-1 {
					row++
				}
			}
			dir = !dir
		} else {
			row = newRow
			col = newCol
		}

	}

	l = 0

	for i := 0; i < y; i++ {
		for j := 0; j < x; j++ {
			entries[l] = matrix[i][j]
			l++
		}
	}
}

func getImageColor(i *image.Image) entryColor {
	color := averageImageColor(*i)
	h, s, v := hsv(color)
	return entryColor{
		Hue:   h * 60,
		Sat:   s,
		Value: v,
	}
}

func sortImagesByHsv(entries []collageEntry) {

	var wg sync.WaitGroup

	for i := range entries {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			entries[i].Color = getImageColor(entries[i].Image)
		}(i, &wg)
	}

	wg.Wait()

	sort.Slice(entries, func(i int, j int) bool {
		if entries[i].Color.Hue == entries[j].Color.Hue {
			if entries[i].Color.Sat == entries[j].Color.Sat {
				return entries[i].Color.Value < entries[j].Color.Value
			}
			return entries[i].Color.Sat < entries[j].Color.Sat
		}
		return entries[i].Color.Hue < entries[j].Color.Hue
	})
}

// New creates a collage from the array of images supplied and with the x/y dimensions specified
func New(images []*image.Image, x int, y int) (*image.RGBA, error) {
	var entries = make([]collageEntry, len(images))

	for i := range images {
		entries[i].Image = images[i]
	}

	sortImagesByHsv(entries)

	rearrangeImages(entries, x, y)

	var grids = make([]*gim.Grid, len(entries))

	for i, e := range entries {
		if e.Image != nil {
			grids[i] = &gim.Grid{
				Image: e.Image,
			}
		}
	}

	collage, err := gim.New(grids, x, y).Merge()
	if err != nil {
		return nil, err
	}
	return collage, nil
}
