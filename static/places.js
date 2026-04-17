(function () {
  "use strict";

  var placesSection = document.getElementById("places");
  var honeymoonSection = document.getElementById("honeymoon");
  if (!placesSection && !honeymoonSection) return;

  var modalEl = placesSection ? placesSection.querySelector("#places-modal") : null;
  var modalImage = modalEl ? modalEl.querySelector("#places-modal-image") : null;
  var modalLabel = modalEl ? modalEl.querySelector("#places-modal-label") : null;
  var modalDate = modalEl ? modalEl.querySelector("#places-modal-date") : null;
  var modalClose = modalEl ? modalEl.querySelector("#places-modal-close") : null;
  var activePin = null;

  if (placesSection) initSection(placesSection, "places");
  if (honeymoonSection) initSection(honeymoonSection, "honeymoon");
  bindModal();

  function initSection(sectionEl, mode) {
    var mapEl = sectionEl.querySelector(".timeline-map");
    var pinsEl = sectionEl.querySelector(".timeline-pins-layer");
    var dataEl = sectionEl.querySelector(".timeline-data");
    if (!mapEl || !pinsEl) return;

    var items = parseTimelineData(dataEl);
    if (!items.length) return;

    var map = null;
    var entries = [];

    if ("IntersectionObserver" in window) {
      var observer = new IntersectionObserver(function (observed) {
        observed.forEach(function (entry) {
          if (!entry.isIntersecting) return;
          init();
          observer.disconnect();
        });
      }, { rootMargin: "240px 0px" });
      observer.observe(sectionEl);
    } else {
      init();
    }

    function init() {
      if (map) return;
      if (typeof L === "undefined") return;

      map = L.map(mapEl, {
        scrollWheelZoom: false,
        zoomControl: false,
        attributionControl: true
      });

      L.tileLayer("https://{s}.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}{r}.png", {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; <a href="https://carto.com/">CARTO</a>',
        subdomains: "abcd",
        maxZoom: 19
      }).addTo(map);

      L.tileLayer("https://{s}.basemaps.cartocdn.com/light_only_labels/{z}/{x}/{y}{r}.png", {
        attribution: "",
        subdomains: "abcd",
        maxZoom: 19,
        pane: "overlayPane"
      }).addTo(map);

      entries = buildEntries(mode, items, pinsEl, map);

      if (mode === "honeymoon" && entries.length > 1) {
        L.polyline(createCurvedRoute(entries.map(function (e) { return e.latlng; })), {
          color: "#8a6a4d",
          weight: 3,
          opacity: 0.95,
          lineCap: "round",
          lineJoin: "round",
          dashArray: "2 12",
          interactive: false,
          className: "timeline-route timeline-route--active"
        }).addTo(map);
      }

      map.on("move zoom resize load", render);

      fitMap(true);
      window.setTimeout(function () {
        map.invalidateSize();
        fitMap(true);
        render();
      }, 120);
    }

    function fitMap(immediate) {
      if (!map || !entries.length) return;

      var minZoom = mode === "places" ? 6 : 4;
      var maxZoom = mode === "places" ? 14 : 8;

      if (entries.length === 1) {
        map.setView(entries[0].latlng, Math.max(minZoom, Math.min(maxZoom, 13)), { animate: !immediate });
        return;
      }

      var bounds = L.latLngBounds(entries.map(function (e) { return e.latlng; }));
      var pad = mode === "places"
        ? L.point(80, 80)
        : L.point(
            Math.max(36, Math.round(mapEl.clientWidth * 0.08)) * 2,
            Math.max(32, Math.round(mapEl.clientHeight * 0.08)) + Math.max(120, Math.round(mapEl.clientHeight * 0.18))
          );

      var fitZoom = Math.max(minZoom, Math.min(maxZoom, map.getBoundsZoom(bounds, false, pad)));
      map.setView(weightedCentroid(entries), fitZoom, { animate: !immediate });
    }

    function render() {
      if (!map) return;
      var visibleEntries = [];

      entries.forEach(function (entry, idx) {
        var point = map.latLngToContainerPoint(entry.latlng);
        var overflow = mode === "honeymoon" ? 240 : 80;
        var isVisible =
          point.x >= -overflow &&
          point.y >= -overflow &&
          point.x <= mapEl.clientWidth + overflow &&
          point.y <= mapEl.clientHeight + overflow;

        entry.element.style.display = isVisible ? "" : "none";
        setPinScale(entry, 1);
        setPinOffset(entry, 0, 0);

        if (!isVisible) return;

        entry.element.style.left = point.x + "px";
        entry.element.style.top = point.y + "px";
        entry.element.classList.toggle("places-pin--active", mode === "places" && idx === getActivePlaceIndex());
        visibleEntries.push(entry);
      });

      applyOverlapLayout(visibleEntries, mode);
    }
  }

  function buildEntries(mode, items, pinsEl, map) {
    var entries = [];

    items.forEach(function (place, idx) {
      if (typeof place.lat !== "number" || typeof place.lng !== "number" || (!place.lat && !place.lng)) return;

      var latlng = L.latLng(place.lat, place.lng);
      var pin = buildPin(place, idx, mode);
      if (!pin) return;

      if (mode === "places") {
        pin.addEventListener("click", function () {
          openPlace(place, pin);
        });
      }

      pinsEl.appendChild(pin);
      entries.push({ place: place, latlng: latlng, element: pin, mode: mode });
    });

    return entries;
  }

  function buildPin(place, idx, mode) {
    var pin;
    if (mode === "honeymoon") {
      pin = document.createElement("article");
      pin.className = "places-pin places-pin--honeymoon";
      pin.dataset.placeIndex = String(idx);
      pin.innerHTML = renderHoneymoonPin(place);
      return pin;
    }

    pin = document.createElement("button");
    pin.type = "button";
    pin.className = "places-pin";
    pin.dataset.placeIndex = String(idx);
    if (!place.image) {
      pin.classList.add("places-pin--placeholder");
      pin.innerHTML = '<span>' + escapeHtml(initials(place.label || place.name || "P")) + "</span>";
    } else {
      pin.innerHTML = '<img src="' + place.image + '" alt="' + escapeHtml(place.label || "Place") + '"/>';
    }
    return pin;
  }

  function renderHoneymoonPin(place) {
    var title = escapeHtml(place.label || place.name || "Stop");
    var transparentClass = supportsTransparency(place.image) ? " places-pin-media--transparent" : "";
    return place.image
      ? '<div class="places-pin-media' + transparentClass + '"><img src="' + place.image + '" alt="' + title + '"/><div class="places-pin-overlay"><h3>' + title + '</h3></div></div>'
      : '<div class="places-pin-media places-pin-media--placeholder"><span>' + escapeHtml(initials(place.label || place.name || "H")) + '</span><div class="places-pin-overlay"><h3>' + title + '</h3></div></div>';
  }

  function applyOverlapLayout(entries, mode) {
    if (entries.length < 2) return;

    var minScale = mode === "honeymoon" ? 0.58 : 0.72;
    var overlapTarget = 0.25;
    var maxOffset = mode === "honeymoon" ? 32 : 14;
    var layouts = entries.map(function () {
      return { scale: 1, offsetX: 0, offsetY: 0 };
    });

    for (var pass = 0; pass < 10; pass++) {
      var changed = false;

      for (var i = 0; i < entries.length; i++) {
        for (var j = i + 1; j < entries.length; j++) {
          var overlapRatio = getOverlapRatio(buildPinRect(entries[i], layouts[i]), buildPinRect(entries[j], layouts[j]));

          if (overlapRatio <= overlapTarget) continue;

          var nextScaleA = Math.max(minScale, layouts[i].scale - 0.05);
          var nextScaleB = Math.max(minScale, layouts[j].scale - 0.05);
          if (nextScaleA !== layouts[i].scale || nextScaleB !== layouts[j].scale) {
            layouts[i].scale = nextScaleA;
            layouts[j].scale = nextScaleB;
            changed = true;
          }

          overlapRatio = getOverlapRatio(buildPinRect(entries[i], layouts[i]), buildPinRect(entries[j], layouts[j]));
          if (overlapRatio <= overlapTarget) continue;

          var separation = getSeparationVector(entries[i], entries[j], layouts[i], layouts[j], i, j);
          var nudgeDistance = mode === "honeymoon" ? 8 : 5;
          if (nudgeEntries(layouts[i], layouts[j], separation, nudgeDistance, maxOffset)) {
            changed = true;
          }
        }
      }

      if (!changed) break;
    }

    entries.forEach(function (entry, index) {
      setPinScale(entry, layouts[index].scale);
      setPinOffset(entry, layouts[index].offsetX, layouts[index].offsetY);
    });
  }

  function buildPinRect(entry, layout) {
    var scale = getPinVisualScale(entry, layout.scale);
    var width = entry.element.offsetWidth * scale;
    var height = entry.element.offsetHeight * scale;
    var center = getPinCenter(entry, layout);

    return {
      left: center.x - width / 2,
      top: center.y - height / 2,
      right: center.x + width / 2,
      bottom: center.y + height / 2,
      area: width * height
    };
  }

  function getPinCenter(entry, layout) {
    return {
      x: (parseFloat(entry.element.style.left) || 0) + (layout.offsetX || 0),
      y: (parseFloat(entry.element.style.top) || 0) + (layout.offsetY || 0)
    };
  }

  function getPinVisualScale(entry, scale) {
    return scale * (entry.element.classList.contains("places-pin--active") ? 1.12 : 1);
  }

  function getOverlapRatio(rectA, rectB) {
    var overlapWidth = Math.min(rectA.right, rectB.right) - Math.max(rectA.left, rectB.left);
    var overlapHeight = Math.min(rectA.bottom, rectB.bottom) - Math.max(rectA.top, rectB.top);
    if (overlapWidth <= 0 || overlapHeight <= 0) return 0;

    return (overlapWidth * overlapHeight) / Math.min(rectA.area, rectB.area);
  }

  function setPinScale(entry, scale) {
    entry.element.style.setProperty("--places-pin-scale", String(scale));
  }

  function setPinOffset(entry, offsetX, offsetY) {
    entry.element.style.setProperty("--places-pin-offset-x", offsetX + "px");
    entry.element.style.setProperty("--places-pin-offset-y", offsetY + "px");
  }

  function getSeparationVector(entryA, entryB, layoutA, layoutB, indexA, indexB) {
    var centerA = getPinCenter(entryA, layoutA);
    var centerB = getPinCenter(entryB, layoutB);
    var dx = centerB.x - centerA.x;
    var dy = centerB.y - centerA.y;

    if (dx || dy) {
      return normalizeVector(dx, dy);
    }

    var angle = ((indexA + indexB + 1) * 137.5 * Math.PI) / 180;
    return {
      x: Math.cos(angle),
      y: Math.sin(angle)
    };
  }

  function weightedCentroid(entries) {
    var sumLat = 0, sumLng = 0;
    entries.forEach(function (entry) {
      sumLat += entry.latlng.lat;
      sumLng += entry.latlng.lng;
    });
    return L.latLng(sumLat / entries.length, sumLng / entries.length);
  }

  function normalizeVector(dx, dy) {
    var length = Math.sqrt(dx * dx + dy * dy) || 1;
    return {
      x: dx / length,
      y: dy / length
    };
  }

  function nudgeEntries(layoutA, layoutB, separation, distance, maxOffset) {
    var nextA = clampOffset(layoutA.offsetX - separation.x * distance, layoutA.offsetY - separation.y * distance, maxOffset);
    var nextB = clampOffset(layoutB.offsetX + separation.x * distance, layoutB.offsetY + separation.y * distance, maxOffset);
    var changed = nextA.x !== layoutA.offsetX || nextA.y !== layoutA.offsetY || nextB.x !== layoutB.offsetX || nextB.y !== layoutB.offsetY;

    layoutA.offsetX = nextA.x;
    layoutA.offsetY = nextA.y;
    layoutB.offsetX = nextB.x;
    layoutB.offsetY = nextB.y;

    return changed;
  }

  function clampOffset(x, y, maxDistance) {
    var distance = Math.sqrt(x * x + y * y);
    if (distance <= maxDistance) {
      return { x: x, y: y };
    }

    var ratio = maxDistance / distance;
    return {
      x: x * ratio,
      y: y * ratio
    };
  }

  function createCurvedRoute(latlngs) {
    if (latlngs.length < 2) return latlngs;

    var curve = [latlngs[0]];
    for (var i = 0; i < latlngs.length - 1; i++) {
      var start = latlngs[i];
      var end = latlngs[i + 1];
      var dx = end.lng - start.lng;
      var dy = end.lat - start.lat;
      var length = Math.sqrt(dx * dx + dy * dy) || 1;
      var normalLat = -dx / length;
      var normalLng = dy / length;
      var sign = i % 2 === 0 ? 1 : -1;
      var offset = length * 0.26;
      var control1 = L.latLng(
        start.lat + dy * 0.22 + normalLat * offset * sign,
        start.lng + dx * 0.22 + normalLng * offset * sign
      );
      var control2 = L.latLng(
        start.lat + dy * 0.78 + normalLat * offset * sign,
        start.lng + dx * 0.78 + normalLng * offset * sign
      );
      var sampled = sampleBezier(start, control1, control2, end, 14);
      for (var j = 1; j < sampled.length; j++) {
        curve.push(sampled[j]);
      }
    }
    return curve;
  }

  function sampleBezier(start, control1, control2, end, steps) {
    var points = [];
    for (var i = 0; i <= steps; i++) {
      var t = i / steps;
      var oneMinusT = 1 - t;
      var lat =
        oneMinusT * oneMinusT * oneMinusT * start.lat +
        3 * oneMinusT * oneMinusT * t * control1.lat +
        3 * oneMinusT * t * t * control2.lat +
        t * t * t * end.lat;
      var lng =
        oneMinusT * oneMinusT * oneMinusT * start.lng +
        3 * oneMinusT * oneMinusT * t * control1.lng +
        3 * oneMinusT * t * t * control2.lng +
        t * t * t * end.lng;
      points.push(L.latLng(lat, lng));
    }
    return points;
  }

  function openPlace(place, pin) {
    if (!modalEl) return;
    if (modalImage) {
      modalImage.src = place.image || "";
      modalImage.alt = place.label || "Place";
      modalImage.style.display = place.image ? "" : "none";
    }
    if (modalLabel) modalLabel.textContent = place.label || place.name || "";
    if (modalDate) modalDate.textContent = place.formatted_date || place.date || "";

    setActivePin(pin);
    modalEl.classList.remove("is-closing");
    modalEl.style.display = "";
    document.body.style.overflow = "hidden";
  }

  function bindModal() {
    if (modalClose) {
      modalClose.addEventListener("click", closeModal);
    }
    if (modalEl) {
      modalEl.addEventListener("click", function (event) {
        if (event.target === modalEl) closeModal();
      });
    }
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && modalEl && modalEl.style.display !== "none") {
        closeModal();
      }
    });
  }

  function closeModal() {
    if (!modalEl) return;
    if (prefersReducedMotion()) {
      finishModalClose();
      return;
    }

    modalEl.classList.add("is-closing");
    var closed = false;
    function onClose() {
      if (closed) return;
      closed = true;
      finishModalClose();
    }

    modalEl.addEventListener("animationend", onClose, { once: true });
    window.setTimeout(onClose, 260);
  }

  function finishModalClose() {
    if (!modalEl) return;
    modalEl.style.display = "none";
    modalEl.classList.remove("is-closing");
    document.body.style.overflow = "";
    setActivePin(null);
  }

  function setActivePin(pin) {
    if (activePin) {
      activePin.classList.remove("places-pin--active");
    }
    activePin = pin;
    if (activePin) {
      activePin.classList.add("places-pin--active");
    }
  }

  function getActivePlaceIndex() {
    if (!activePin) return -1;
    return parseInt(activePin.dataset.placeIndex || "-1", 10);
  }

  function parseTimelineData(el) {
    if (!el) return [];
    try {
      var parsed = JSON.parse(el.textContent);
      return Array.isArray(parsed) ? parsed : [];
    } catch (e) {
      return [];
    }
  }

  function prefersReducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function initials(value) {
    return String(value)
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map(function (part) { return part.charAt(0).toUpperCase(); })
      .join("");
  }

  function supportsTransparency(image) {
    if (!image) return false;
    return /^data:image\/png/i.test(image) || /\.png(?:$|[?#])/i.test(image);
  }
})();
