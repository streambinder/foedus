(function () {
  "use strict";

  var dataEl = document.getElementById("places-data");
  var mapEl = document.getElementById("places-map");
  if (!dataEl || !mapEl) return;

  var places;
  try {
    places = JSON.parse(dataEl.textContent);
  } catch (e) {
    return;
  }
  if (!places || places.length === 0) return;

  var mapInitialized = false;

  function initMap() {
    if (mapInitialized || typeof L === "undefined") return;
    mapInitialized = true;

    var map = L.map("places-map", {
      scrollWheelZoom: false,
      zoomControl: true,
      attributionControl: true,
    });

    L.tileLayer("https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png", {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; <a href="https://carto.com/">CARTO</a>',
      subdomains: "abcd",
      maxZoom: 19,
    }).addTo(map);

    var markers = [];
    var latlngs = [];

    places.forEach(function (place, idx) {
      if (!place.lat && !place.lng) return;
      var ll = [place.lat, place.lng];
      latlngs.push(ll);

      var marker = L.marker(ll, {
        icon: L.divIcon({
          className: "places-marker",
          html: '<span class="places-marker-number">' + (idx + 1) + "</span>",
          iconSize: [32, 32],
          iconAnchor: [16, 32],
          popupAnchor: [0, -32],
        }),
      }).addTo(map);

      var popupContent = "<strong>" + escapeHtml(place.label) + "</strong>";
      if (place.name) popupContent += "<br>" + escapeHtml(place.name);
      marker.bindPopup(popupContent);
      markers.push(marker);
    });

    if (latlngs.length > 1) {
      L.polyline(latlngs, {
        color: "#D4A59A",
        weight: 2,
        dashArray: "6 8",
        opacity: 0.7,
      }).addTo(map);
    }

    if (latlngs.length > 0) {
      map.fitBounds(latlngs, { padding: [40, 40], maxZoom: 14 });
    }

    document.querySelectorAll(".timeline-item").forEach(function (item) {
      item.style.cursor = "pointer";
      item.addEventListener("click", function () {
        var idx = parseInt(item.dataset.index, 10);
        if (markers[idx]) {
          map.flyTo(markers[idx].getLatLng(), 15, { duration: 0.8 });
          markers[idx].openPopup();
        }

        document.querySelectorAll(".timeline-item").forEach(function (ti) {
          ti.classList.remove("timeline-item--active");
        });
        item.classList.add("timeline-item--active");
      });
    });
  }

  if ("IntersectionObserver" in window) {
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        initMap();
        observer.disconnect();
      });
    }, { rootMargin: "240px 0px" });

    observer.observe(mapEl);
  } else {
    initMap();
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }
})();
