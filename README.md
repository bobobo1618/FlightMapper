# FlightMapper
Maps Flightaware or SBS feeds.

Uses https://github.com/lucasb-eyer/heatmap for rendering heatmaps.

## Building

`go build .`

## Running

```
Usage of ./transformer2:
  -bind string
    	The address to bind on. (default ":8080")
  -depth uint
    	Number of levels deep to allow rendering. More levels uses more memory. (default 10)
  -multi_type
    	Expect 3 columns, with the third being a map type.
  -size uint
    	Size of tile (default 256)
  -source string
    	The datasource to pull flightdata from. Can be a file name, "flightaware:username:password" or sbs:host:port
```

`./FlightMapper -depth 10 -multi_type=true -source=$HOME/fa_2015-09-20.log`
