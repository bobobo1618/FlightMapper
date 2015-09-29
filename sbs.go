package main

import (
    "net"
    "bufio"
    "fmt"
    "log"
    "time"
    "strings"
    "strconv"
)

// Connects to the FlightAware Firehose API and outputs positions to the given channel.
func sbsHose(output chan<- Point, host, port string) {
    connString := fmt.Sprintf("%s:%s", host, port)
    conn, err := net.Dial("tcp", connString)
    if err != nil {
        log.Fatal(err)
    } else {
        scanner := bufio.NewScanner(conn)
        for scanner.Scan() {
            splitLine := strings.Split(scanner.Text(), ",")
            if len(splitLine) < 16 || splitLine[1] != "3" {
                continue
            }
            
            lat, err := strconv.ParseFloat(splitLine[14], 64)
            if err != nil { log.Printf("Couldn't parse location %s, %s\n", splitLine[14], splitLine[15]); continue }
            lon, err := strconv.ParseFloat(splitLine[15], 64)
            if err != nil { log.Printf("Couldn't parse location %s, %s\n", splitLine[14], splitLine[15]); continue }
            
            output <- Point{lat, lon}
        }
    }
}

func sbsHoseClockFiller(input <-chan Point, clockArray []map[rune]*Tile, tileSize float64){
    lastHour := time.Now().Hour()
    for pos := range input {
        hour := time.Now().Hour()
        if clockArray[hour] == nil || hour != lastHour {
            clockArray[hour] = make(map[rune]*Tile)
            lastHour = hour
        }

        updateType := 'A'

        if clockArray[hour][updateType] == nil {
            newTile := &Tile{}
            newTile.Parent = nil
            newTile.Level = 0
            newTile.Position = Point{0, 0}
            newTile.Width = tileSize
            clockArray[hour][updateType] = newTile
        }

        x, y := project(pos.X, pos.Y, tileSize)
        // Skip tiles that were cut off at the poles.
        if y < 0 || y > tileSize {
            continue
        }
        clockArray[hour][updateType].AddPoint(Point{x, y})
    }
}