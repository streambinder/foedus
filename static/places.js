(function () {
  "use strict";

  var dataEl = document.getElementById("places-data");
  var mapEl = document.getElementById("places-map");
  var pinsEl = document.getElementById("places-pins");
  var modalEl = document.getElementById("places-modal");
  if (!dataEl || !mapEl || !pinsEl || !modalEl) return;

  var modalImage = document.getElementById("places-modal-image");
  var modalLabel = document.getElementById("places-modal-label");
  var modalDate = document.getElementById("places-modal-date");
  var modalClose = document.getElementById("places-modal-close");

  var places;
  try {
    places = JSON.parse(dataEl.textContent);
  } catch (e) {
    return;
  }
  if (!Array.isArray(places) || places.length === 0) return;

  var map = null;
  var activePin = null;
  var pinEntries = [];
  var mapInitialized = false;

  function initMap() {
    if (mapInitialized || typeof L === "undefined") return;
    mapInitialized = true;

    map = L.map("places-map", {
      scrollWheelZoom: false,
      zoomControl: false,
      attributionControl: true,
    });

    L.tileLayer("https://{s}.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}{r}.png", {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; <a href="https://carto.com/">CARTO</a>',
      subdomains: "abcd",
      maxZoom: 19,
    }).addTo(map);

    L.tileLayer("https://{s}.basemaps.cartocdn.com/light_only_labels/{z}/{x}/{y}{r}.png", {
      attribution: "",
      subdomains: "abcd",
      maxZoom: 19,
      pane: "overlayPane",
    }).addTo(map);

    var latlngs = [];

    places.forEach(function (place, idx) {
      if (typeof place.lat !== "number" || typeof place.lng !== "number" || (!place.lat && !place.lng)) {
        return;
      }

      var latlng = L.latLng(place.lat, place.lng);
      latlngs.push(latlng);

      var pin = document.createElement("button");
      pin.type = "button";
      pin.className = "places-pin";
      pin.dataset.placeIndex = String(idx);
      if (!place.image) {
        pin.classList.add("places-pin--placeholder");
        pin.innerHTML = '<span>' + escapeHtml(initials(place.label || place.name || "P")) + "</span>";
      } else {
        pin.innerHTML = '<img src="' + place.image + '" alt="' + escapeHtml(place.label || "Place") + '"/>';
      }
      pin.addEventListener("click", function () {
        openPlace(place, pin);
      });

      pinsEl.appendChild(pin);
      pinEntries.push({ place: place, latlng: latlng, element: pin });
    });

    if (latlngs.length > 1) {
      map.fitBounds(latlngs, { padding: [80, 80], maxZoom: 14 });
    } else if (latlngs.length === 1) {
      map.setView(latlngs[0], 13);
    }

    map.on("move zoom resize load", renderPins);

    window.setTimeout(function () {
      map.invalidateSize();
      renderPins();
    }, 120);
  }

  function renderPins() {
    if (!map) return;

    pinEntries.forEach(function (entry) {
      var point = map.latLngToContainerPoint(entry.latlng);
      var isVisible =
        point.x >= -80 &&
        point.y >= -80 &&
        point.x <= mapEl.clientWidth + 80 &&
        point.y <= mapEl.clientHeight + 80;

      entry.element.style.display = isVisible ? "" : "none";
      entry.element.style.left = point.x + "px";
      entry.element.style.top = point.y + "px";
    });
  }

  function openPlace(place, pin) {
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

  function closeModal() {
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

  function setActivePin(pin) {
    if (activePin) {
      activePin.classList.remove("places-pin--active");
    }
    activePin = pin;
    if (activePin) {
      activePin.classList.add("places-pin--active");
    }
  }

  function finishModalClose() {
    modalEl.style.display = "none";
    modalEl.classList.remove("is-closing");
    document.body.style.overflow = "";
    setActivePin(null);
  }

  function prefersReducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  if (modalClose) {
    modalClose.addEventListener("click", closeModal);
  }
  modalEl.addEventListener("click", function (event) {
    if (event.target === modalEl) closeModal();
  });
  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && modalEl.style.display !== "none") {
      closeModal();
    }
  });

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
})();
