package main

import (
	"bufio"
	"flag"
	"image/png"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var level_depth uint
var pngEncoder *png.Encoder

func main() {
	flag.UintVar(&level_depth, "depth", 10, "Number of levels deep to allow rendering. More levels uses more memory.")
	int_comp_tile_size := flag.Uint("size", 256, "Size of tile")
	comp_tile_size := float64(*int_comp_tile_size)
	multi_type := flag.Bool("multi_type", false, "Expect 3 columns, with the third being a map type.")
	source := flag.String("source", "", "The datasource to pull flightdata from. Can be a file name, \"flightaware:username:password\" or sbs:host:port")
	host := flag.String("bind", ":8080", "The address to bind on.")

	flag.Parse()

	pngEncoder = &png.Encoder{0}

	var rootTile *Tile
	var tileMap map[rune]*Tile
	var clockArray []map[rune]*Tile

	if strings.HasPrefix(*source, "flightaware:") {
		log.Print("Connecting to Flightaware.")
		splitSource := strings.Split(*source, ":")
		if len(splitSource) < 3 {
			log.Fatal("Source appeared to be flightaware but lacked the necessary number of components (username, password).")
		}

		positionChan := make(chan FAPosition, 100)
		clockArray = make([]map[rune]*Tile, 24, 24)
		go fireHose(positionChan, splitSource[1], splitSource[2])
		go fireHoseClockFiller(positionChan, clockArray, comp_tile_size)

	} else if strings.HasPrefix(*source, "sbs:") {
		log.Print("Connecting to SBS.")
		splitSource := strings.Split(*source, ":")
		if len(splitSource) < 3 {
			log.Fatal("Source appeared to be SBS but lacked the necessary number of components (host, port).")
		}

		positionChan := make(chan Point, 100)
		clockArray = make([]map[rune]*Tile, 24, 24)
		go sbsHose(positionChan, splitSource[1], splitSource[2])
		go sbsHoseClockFiller(positionChan, clockArray, comp_tile_size)
	} else {
		log.Printf("Reading %s.\n", *source)
		input, err := os.Open(*source)
		if err != nil {
			log.Fatalf("Couldn't open %s.\n", *source)
		}
		defer input.Close()

		scanner := bufio.NewScanner(input)

		// If multi_type is turned on, build a map of char->rootTile, otherwise make a single tile for everything.
		if !*multi_type {
			rootTile := &Tile{}
			rootTile.Parent = nil
			rootTile.Level = 0
			rootTile.Position = Point{0, 0}
			rootTile.Width = comp_tile_size
		} else {
			tileMap = make(map[rune]*Tile)
		}

		start := time.Now()
		parseTime := 0.0
		processTime := 0.0
		lines := 0

		for scanner.Scan() {
			parseStart := time.Now()
			split := strings.Split(scanner.Text(), ",")
			lines++

			if *multi_type {
				if len(split) < 3 {
					log.Print("Skipping row due to insufficient items.")
					continue
				}

				pointType, _ := utf8.DecodeRuneInString(split[2])

				if tileMap[pointType] == nil {
					tileMap[pointType] = &Tile{}
					rootTile = tileMap[pointType]
					rootTile.Parent = nil
					rootTile.Level = 0
					rootTile.Position = Point{0, 0}
					rootTile.Width = comp_tile_size
				}

				rootTile = tileMap[pointType]

			} else if len(split) < 2 {
				log.Print("Skipping row due to insufficient items.")
				continue
			}

			lat, err := strconv.ParseFloat(split[0], 64)
			if err != nil {
				log.Printf("Skipping row due to parse error on line %d\n", lines)
				continue
			}
			lon, err := strconv.ParseFloat(split[1], 64)
			if err != nil {
				log.Printf("Skipping row due to parse error on line %d\n", lines)
				continue
			}
			parseTime += time.Now().Sub(parseStart).Seconds()
			processStart := time.Now()
			x, y := project(lat, lon, comp_tile_size)
			// Skip tiles that were cut off at the poles.
			if y < 0 || y > comp_tile_size {
				continue
			}
			rootTile.AddPoint(Point{x, y})
			processTime += time.Now().Sub(processStart).Seconds()
		}

		duration := time.Now().Sub(start)
		log.Printf("Loading %d lines of data took %f seconds: %f parsing, %f processing, %f I/O\nPer line: %f, %f, %f, %f\n",
			lines,
			duration.Seconds(), parseTime, processTime,
			(duration.Seconds() - parseTime - processTime),
			duration.Seconds()/float64(lines), parseTime/float64(lines), processTime/float64(lines),
			(duration.Seconds()-parseTime-processTime)/float64(lines))
	}

	startServer(tileMap, rootTile, clockArray, comp_tile_size, *host)
}
