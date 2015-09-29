function initMap() {
  var map = new google.maps.Map(document.getElementById('map'), {
    zoom: 4,
    center: {lat: 29.7604, lng: -95.3698},
    styles: mapStyle
  });

  function getNormalizedCoord(coord, zoom) {
    var y = coord.y;
    var x = coord.x;

    // tile range in one direction range is dependent on zoom level
    // 0 = 1 tile, 1 = 2 tiles, 2 = 4 tiles, 3 = 8 tiles, etc
    var tileRange = 1 << zoom;

    // don't repeat across y-axis (vertically)
    if (y < 0 || y >= tileRange) {
      return null;
    }

    // repeat across x-axis
    if (x < 0 || x >= tileRange) {
      x = (x % tileRange + tileRange) % tileRange;
    }

    return {x: x, y: y};
  }

  function buildHeatmapType(codes) {
    return new google.maps.ImageMapType({
      getTileUrl: function(coord, zoom) {
          var normalizedCoord = getNormalizedCoord(coord, zoom);
          if (!normalizedCoord) {
            return null;
          }
          var bound = Math.pow(2, zoom);
          //return '/tile' + '/' + zoom + '/' + (coord.x % bound) + '/' + (coord.y % bound);
          return '/tile' + '/' + zoom + '/' + normalizedCoord.x + '/' + normalizedCoord.y + '/' + codes
      },
      tileSize: new google.maps.Size(256, 256),
      maxZoom: 9,
      minZoom: 0,
      //radius: 1738000,
      //name: 'Moon'
    });
  }

  window.updateLayers = function() {
    var selectedBoxes = document.querySelectorAll('input.layer-toggle:checked');
    var codes = '';
    for(var i=0; i < selectedBoxes.length; i++) {
      codes += selectedBoxes[i].value;
    }
    map.overlayMapTypes.pop(0);
    map.overlayMapTypes.push(buildHeatmapType(codes));
  }

  updateLayers();
}