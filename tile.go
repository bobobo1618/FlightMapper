package main

// #include "heatmap.h"
import "C"

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"os"
)

type Tile struct {
	Parent    *Tile
	SubTiles  [4]*Tile
	Points    []Point
	Level     uint
	Position  Point
	Width     float64
	NumPoints uint32
}

// Get position relative to tile origin
func (t *Tile) RelativePos(p Point) (np Point) {
	np = Point{p.X - t.Position.X, p.Y - t.Position.Y}
	return
}

// Get position normalised to 0,1 relative to tile origin
func (t *Tile) ScaledRelativePos(p Point) (np Point) {
	relPos := t.RelativePos(p)
	np = Point{relPos.X / t.Width, relPos.Y / t.Width}
	return
}

// Multiply the normalised position by the tile size to get a pixel position.
func (t *Tile) PixelPos(p Point, size float64) (np Point) {
	scalePos := t.ScaledRelativePos(p)
	np = Point{
		scalePos.X * size,
		scalePos.Y * size,
	}
	return
}

// Get a the tile for the given level and point. Returns nil if it couldn't be found.
func (t *Tile) GetTile(p Point, level uint) *Tile {
	if t == nil || t.Level == level {
		return t
	} else {
		scaledRelativePos := t.ScaledRelativePos(p)
		subTileIndex := 0

		// If it's on the right column, add 1 to the index to make it odd
		if scaledRelativePos.X >= 0.5 {
			subTileIndex++
		}

		// If it's on the bottom, add 2 so it's either 2 or 3
		if scaledRelativePos.Y >= 0.5 {
			subTileIndex += 2
		}

		return t.SubTiles[subTileIndex].GetTile(p, level)
	}
}

// Add a point to a tile, recursively searching downward until it reaches the level limit.
func (t *Tile) AddPoint(p Point) {
	scaledRelativePos := t.ScaledRelativePos(p)
	if scaledRelativePos.X > 1 || scaledRelativePos.X < 0 {
		log.Printf("%f, %f is an invalid point for this tile.\n",
			scaledRelativePos.X, scaledRelativePos.Y)
		return
	}

	if scaledRelativePos.Y > 1 || scaledRelativePos.Y < 0 {
		log.Printf("%f, %f is an invalid point for this tile.\n",
			scaledRelativePos.X, scaledRelativePos.Y)
		return
	}

	// Terminate at the level goal.
	if t.Level == level_depth {
		t.Points = append(t.Points, p)
	} else {
		subTileIndex := 0

		// If it's on the right column, add 1 to the index to make it odd
		if scaledRelativePos.X >= 0.5 {
			subTileIndex++
		}

		// If it's on the bottom, add 2 so it's either 2 or 3
		if scaledRelativePos.Y >= 0.5 {
			subTileIndex += 2
		}

		// If the tile doesn't exist, create it.
		if t.SubTiles[subTileIndex] == nil {
			newPosition := t.Position
			if (subTileIndex % 2) == 1 {
				newPosition.X += t.Width / 2.0
			}

			if subTileIndex >= 2 {
				newPosition.Y += t.Width / 2.0
			}

			t.SubTiles[subTileIndex] = &Tile{
				Parent:   t,
				SubTiles: [4]*Tile{},
				Points:   []Point{},
				Level:    t.Level + 1,
				Position: newPosition,
				Width:    t.Width / 2.0,
			}
		}

		// Add the point.
		t.SubTiles[subTileIndex].AddPoint(p)
		t.NumPoints++
	}
}

// Outputs its points into the given channel. Recurses into children.
func (t *Tile) GetPoints(output chan<- Point) {
	if t.Level == level_depth {
		for _, point := range t.Points {
			output <- point
		}
	} else {
		for _, tile := range t.SubTiles {
			if tile != nil {
				tile.GetPoints(output)
			}
		}
	}

	return
}

// Renders the given heatmap to the given file output.
func (t *Tile) RenderHeatmapToFile(heat *C.struct___0, size float64, output io.Writer) {
	// Number of colours to generate for the colour scheme.
	numColors := 256

	colorMap := make([]uint8, numColors*4, numColors*4)
	// FlightAware colours
	startColor := color.RGBA{0, 0x2F, 0x5D, 64}
	endColor := color.RGBA{0, 0xA0, 0xE2, 192}

	// Figure out how much to increment each channel by on each iteration.
	rIncr := (float64(endColor.R) - float64(startColor.R)) / float64(numColors)
	gIncr := (float64(endColor.G) - float64(startColor.G)) / float64(numColors)
	bIncr := (float64(endColor.B) - float64(startColor.B)) / float64(numColors)
	aIncr := (float64(endColor.A) - float64(startColor.A)) / float64(numColors)

	// Generate the map
	for i := 1; i < int(numColors); i++ {
		colorMap[4*i+0] = uint8(float64(startColor.R) + rIncr*float64(i))
		colorMap[4*i+1] = uint8(float64(startColor.G) + gIncr*float64(i))
		colorMap[4*i+2] = uint8(float64(startColor.B) + bIncr*float64(i))
		colorMap[4*i+3] = uint8(float64(startColor.A) + aIncr*float64(i))
	}

	// Build a C colorscheme out of it.
	colorScheme := C.heatmap_colorscheme_load((*C.uchar)(&colorMap[0]), C.size_t(numColors))

	// Allocate an image buffer.
	heatImg := image.NewNRGBA(image.Rect(0, 0, int(size), int(size)))

	// Change the saturation based on the zoom, for aesthetics.
	saturationZoomFactor := (1.0 - 1/(math.Pow(2, float64(t.Level))))
	if t.Level == 0 {
		saturationZoomFactor = 0.9
	}
	saturation := C.float(5.0 * saturationZoomFactor)

	// Render the heatmap to the image buffer.
	C.heatmap_render_saturated_to(heat, colorScheme, saturation, (*C.uchar)(&heatImg.Pix[0]))
	// Deallocate the heatmap and color scheme.
	C.heatmap_free(heat)
	C.heatmap_colorscheme_free(colorScheme)
	/*start := time.Now()
	heatDuration := time.Now().Sub(start).Seconds()*/
	// PNG encode to the output.
	pngEncoder.Encode(output, heatImg)
	/*pngDuration := time.Now().Sub(start).Seconds() - heatDuration
	log.Printf("Mapped %d points in %f seconds mapping, %f seconds encoding (%fpps).\n", numRendered, heatDuration, pngDuration, float64(numRendered)/heatDuration)*/
	return
}

// Adds the tile's points to the given heatmap. If the given heatmap pointer is nil, one is created and returned.
func (t *Tile) AddPointsToHeatmap(heat *C.struct___0, size float64) *C.struct___0 {
	if t == nil {
		return nil
	}

	if heat == nil {
		heat, _ = C.heatmap_new(C.uint(size), C.uint(size))
	}

	// Build a stamp with a radius of 10 units.
	stamp := C.heatmap_stamp_gen(C.uint(10))

	// Count the points that have been rendered in case they're used for stats none day.
	numRendered := 0

	if t.Level == level_depth && len(t.Points) > 0 {
		// Grab points from this tile.
		for _, point := range t.Points {
			pixelPoint := t.PixelPos(point, size)
			C.heatmap_add_point_with_stamp(heat, C.uint(pixelPoint.X), C.uint(pixelPoint.Y), stamp)
			numRendered++
		}
	} else {
		// Get the points and have them sent to a buffered channel
		pointChan := make(chan Point, 100)
		// Fetch them in a goroutine so we can fetch them and add them at the same time.
		go func(tile *Tile, output chan<- Point) {
			tile.GetPoints(pointChan)
			close(pointChan)
		}(t, pointChan)
		// Add them.
		for point := range pointChan {
			pixelPoint := t.PixelPos(point, size)
			C.heatmap_add_point(heat, C.uint(pixelPoint.X), C.uint(pixelPoint.Y))
			numRendered++
		}
	}

	// Deallocate the stamp memory.
	C.heatmap_stamp_free(stamp)
	return heat
}

// Writes a heatmap to the given output.
func (t *Tile) HeatmapToFile(size float64, output io.Writer) {
	if t == nil {
		return
	}
	heat := t.AddPointsToHeatmap(nil, size)
	t.RenderHeatmapToFile(heat, size, output)
}

// Generates a static heatmap.
func (t *Tile) Heatmap(size float64) {
	name := fmt.Sprintf("tile.%02d.%05f.%05f.png", t.Level, t.Position.Y,
		t.Position.X)
	output, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	t.HeatmapToFile(size, output)
}
