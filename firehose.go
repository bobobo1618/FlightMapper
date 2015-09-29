package main

import (
    "crypto/tls"
    "bufio"
    "fmt"
    "encoding/json"
    "unicode/utf8"
    "log"
    "time"
)

type FAPosition struct {
    MessageType string `json:"type"`
    FlightIdentifier string `json:"ident"`
    Latitude float64 `json:"lat,string"`
    Longitude float64 `json:"lon,string"`
    ReportTime uint32 `json:"clock,string"`
    FlightId string `json:"id"`
    UpdateType string `json:"updateType"`
    ReportFacilityHash string `json:"facility_hash"`
    ReportFacilityName string `json:"facility_name"`
}

// Connects to the FlightAware Firehose API and outputs positions to the given channel.
func fireHose(output chan<- FAPosition, username, password string) {
    conn, err := tls.Dial("tcp", "firehose.flightaware.com:1501", &tls.Config{})
    if err != nil {
        panic(err)
    } else {
        connString := fmt.Sprintf("live username %s password %s events \"position\"\n", username, password)
        conn.Write([]byte(connString))
        scanner := bufio.NewScanner(conn)
        for scanner.Scan() {
            pos := FAPosition{}
            err := json.Unmarshal(scanner.Bytes(), &pos)
            if err != nil {
                log.Printf("Couldn't unmarshall %s\n.", scanner.Text())
                continue
            }
            output <- pos
        }
    }
}

func fireHoseClockFiller(input <-chan FAPosition, clockArray []map[rune]*Tile, tileSize float64){
    lastHour := time.Now().Hour()
    for pos := range input {
        hour := time.Now().Hour()
        if clockArray[hour] == nil || hour != lastHour {
            clockArray[hour] = make(map[rune]*Tile)
            lastHour = hour
        }

        updateType, _ := utf8.DecodeRuneInString(pos.UpdateType)

        if clockArray[hour][updateType] == nil {
            newTile := &Tile{}
            newTile.Parent = nil
            newTile.Level = 0
            newTile.Position = Point{0, 0}
            newTile.Width = tileSize
            clockArray[hour][updateType] = newTile
        }

        x, y := project(pos.Latitude, pos.Longitude, tileSize)
        // Skip tiles that were cut off at the poles.
        if y < 0 || y > tileSize {
            continue
        }
        clockArray[hour][updateType].AddPoint(Point{x, y})
    }
}