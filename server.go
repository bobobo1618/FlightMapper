package main

// #include "heatmap.h"
// #cgo CFLAGS: -fPIC -I. -O3 -g -DNDEBUG -fopenmp -pedantic
// #cgo LDFLAGS: -O3 -lm -L/usr/local/Cellar/libiomp/20150401/lib -liomp5
import "C"

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "strconv"
)


func fillHeatmapFromTypeMap(heat *C.struct___0, worldPoint Point, tileSize float64, level int, tileMap map[rune]*Tile, mapTypeString string) (newHeat *C.struct___0, tile *Tile) {
    // Check to make sure there's at least one tile with usable data.
    numBad := 0
    numTotal := 0
    for _, mapType := range mapTypeString {
        numTotal++
        curTile := tileMap[mapType].GetTile(worldPoint, uint(level))
        if curTile == nil {
            numBad++
        }
    }

    if numBad == numTotal {
        return
    }

    newHeat = heat

    // Add points to the heatmap.
    for _, mapType := range mapTypeString {
        curTile := tileMap[mapType].GetTile(worldPoint, uint(level))
        if curTile != nil {
            newHeat = curTile.AddPointsToHeatmap(heat, tileSize)
            // Keep a tile around for the rendering.
            tile = curTile
        }
    }

    return
}

func startServer(tileMap map[rune]*Tile, rootTile *Tile, clockArray []map[rune]*Tile, tileSize float64, host string) {
    if (tileMap == nil) && (rootTile == nil) && (clockArray == nil) {
        log.Print("Failing to start server, no map or tiles were given.")
        return
    }

    multiType := tileMap != nil
    clockType := clockArray != nil

    http.HandleFunc("/tile/", func(res http.ResponseWriter, req *http.Request){
        path := req.URL.Path
        log.Printf("%s.\n", path)
        pathComponents := strings.Split(path, "/")
        if ((multiType || clockType) && len(pathComponents) < 4) || (!multiType && len(pathComponents) < 3) {
            res.WriteHeader(400)
            fmt.Fprintf(res, "Not enough URL components.\n")
            return
        } else {
            level, err := strconv.Atoi(pathComponents[2])
            if err != nil {res.WriteHeader(400); return}

            x, err := strconv.ParseFloat(pathComponents[3], 64)
            if err != nil {res.WriteHeader(400); return}
            y, err := strconv.ParseFloat(pathComponents[4], 64)
            if err != nil {res.WriteHeader(400); return}

            worldPoint := tileToWorld(Point{x, y}, uint(level), tileSize)
            if worldPoint.X < 0 || worldPoint.Y < 0 || worldPoint.X > tileSize || worldPoint.Y > tileSize {
                res.WriteHeader(404)
                return
            }

            var tile *Tile

            if clockType {
                mapTypeString := pathComponents[5]
                
                var heat *C.struct___0
                var tile *Tile

                for _, tileMap := range clockArray {
                    newHeat, newTile := fillHeatmapFromTypeMap(heat, worldPoint, tileSize, level, tileMap, mapTypeString)
                    if newHeat != nil {
                        heat = newHeat
                        tile = newTile
                        log.Print("Heat wasn't nil!")
                    }
                }

                if heat == nil {
                    log.Printf("Failing %s because all tiles were missing.\n", mapTypeString)
                    res.WriteHeader(404)
                    return
                }
                
                // Render the header straight to the response.
                res.Header().Set("Content-Type", "image/png")
                res.WriteHeader(200)
                tile.RenderHeatmapToFile(heat, tileSize, res)
            } else if multiType {
                mapTypeString := pathComponents[5]
                
                var heat *C.struct___0
                var tile *Tile

                heat, tile = fillHeatmapFromTypeMap(heat, worldPoint, tileSize, level, tileMap, mapTypeString)

                if heat == nil {
                    log.Printf("Failing %s because all tiles were missing.\n", mapTypeString)
                    res.WriteHeader(404)
                    return
                }
                
                // Render the header straight to the response.
                res.Header().Set("Content-Type", "image/png")
                res.WriteHeader(200)
                tile.RenderHeatmapToFile(heat, tileSize, res)
            } else {
                // Much simpler, just get the right tile from the root and render it.
                tile = rootTile
                requestedTile := tile.GetTile(worldPoint, uint(level))
                if requestedTile == nil {
                    res.WriteHeader(404)
                } else {
                    res.Header().Set("Content-Type", "image/png")
                    res.WriteHeader(200)
                    requestedTile.HeatmapToFile(tileSize, res)
                }
            }
        }
    })
    http.Handle("/", http.FileServer(http.Dir("static")))
    http.ListenAndServe(host, nil)
}