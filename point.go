package main

import (
	"math"
)

func project(lat, lon, tile_size float64) (x, y float64) {
	// Project X by reducing it to -0.5,0.5, then adding 0.5 to move it to 0,1,
	// then multiplying to get to 0,size
	x = float64(tile_size) * (0.5 + (lon / 360.0))

	// Project to 0,1 by converting latitude to radians and taking its Sin
	siny := math.Sin(lat * math.Pi / 180.0)
	// Cut off the poles
	siny = math.Min(math.Max(siny, -0.9999), 0.9999)

	// Math magic...
	y = float64(tile_size) * (0.5 - math.Log((1.0+siny)/(1.0-siny))/(4.0*math.Pi))
	return
}

// Convert 0..1 coordinates to 0..worldSize co-ordinates at the given level.
func tileToWorld(inp Point, level uint, tileSize float64) (p Point){
	factor := tileSize / math.Pow(2, float64(level))
	return Point{inp.X * factor, inp.Y * factor}
}

type Point struct {
	X float64
	Y float64
}
