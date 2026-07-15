(() => {
  const modalEl = document.getElementById("parking-modal");
  const dataEl = document.querySelector(".parking-data");
  const trigger = document.querySelector("[data-open-parking]");
  if (!modalEl || !dataEl || !trigger) return;

  const mapEl = modalEl.querySelector(".parking-map");
  const pinsEl = modalEl.querySelector(".parking-pins-layer");
  const closeBtn = modalEl.querySelector("#parking-modal-close");
  if (!mapEl || !pinsEl) return;

  const spots = parseSpots(dataEl);
  if (!spots.length) return;

  // ceremony coords come from the dialog data attributes; a 0,0 pair means the
  // ceremony was saved before coordinates existed — omit its pin rather than
  // dropping it in the ocean.
  const ceremonyLat = parseFloat(modalEl.dataset.ceremonyLat || "0");
  const ceremonyLng = parseFloat(modalEl.dataset.ceremonyLng || "0");
  const hasCeremony =
    Number.isFinite(ceremonyLat) &&
    Number.isFinite(ceremonyLng) &&
    (ceremonyLat !== 0 || ceremonyLng !== 0);

  const LEAFLET_CSS_URL = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css";
  const LEAFLET_JS_URL = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
  let leafletAssetsPromise = null;

  let map = null;
  let entries = [];

  bindModal();

  function openModal(event) {
    // the trigger sits inside the ceremony <a>; stop the click from also
    // navigating to the venue's google maps link.
    if (event) {
      event.preventDefault();
      event.stopPropagation();
    }
    modalEl.classList.remove("is-closing");
    if (typeof modalEl.showModal === "function" && !modalEl.open) {
      modalEl.showModal();
    } else {
      modalEl.style.display = "";
    }
    document.body.style.overflow = "hidden";
    // the dialog has no layout box until shown, so leaflet must init/fit here.
    ensureLeaflet()
      .then(startMap)
      .catch(() => {});
  }

  function startMap() {
    if (typeof L === "undefined") return;
    if (!map) {
      map = L.map(mapEl, {
        boxZoom: true,
        doubleClickZoom: true,
        dragging: true,
        keyboard: true,
        scrollWheelZoom: false,
        tap: true,
        touchZoom: true,
        zoomControl: false,
        attributionControl: true,
      });

      L.tileLayer(
        "https://{s}.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}{r}.png",
        {
          attribution:
            '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; <a href="https://carto.com/">CARTO</a>',
          subdomains: "abcd",
          maxZoom: 19,
        },
      ).addTo(map);

      L.tileLayer(
        "https://{s}.basemaps.cartocdn.com/light_only_labels/{z}/{x}/{y}{r}.png",
        {
          attribution: "",
          subdomains: "abcd",
          maxZoom: 19,
          pane: "overlayPane",
        },
      ).addTo(map);

      entries = buildEntries();
      map.on("move zoom resize", render);
    }

    // re-run after the dialog paints so leaflet picks up real dimensions
    fitMap(true);
    window.setTimeout(() => {
      if (!map) return;
      map.invalidateSize();
      fitMap(true);
      render();
    }, 120);
  }

  function buildEntries() {
    const built = [];

    if (hasCeremony) {
      const ceremonyPin = document.createElement("div");
      ceremonyPin.className = "parking-pin parking-pin-ceremony";
      ceremonyPin.innerHTML =
        '<span class="venue-overlay-icon" aria-hidden="true"></span>';
      pinsEl.appendChild(ceremonyPin);
      built.push({
        latlng: L.latLng(ceremonyLat, ceremonyLng),
        element: ceremonyPin,
      });
    }

    spots.forEach((spot) => {
      if (
        typeof spot.lat !== "number" ||
        typeof spot.lng !== "number" ||
        (!spot.lat && !spot.lng)
      )
        return;

      const pin = document.createElement("button");
      pin.type = "button";
      pin.className = "parking-pin parking-pin-spot";
      pin.innerHTML =
        '<span class="venue-overlay-icon" aria-hidden="true"></span>';
      // each spot links to google maps driving directions to its coords
      pin.addEventListener("click", () => {
        window.open(directionsURL(spot.lat, spot.lng), "_blank", "noopener");
      });
      pinsEl.appendChild(pin);
      built.push({ latlng: L.latLng(spot.lat, spot.lng), element: pin });
    });

    return built;
  }

  function fitMap(immediate) {
    if (!map || !entries.length) return;

    const minZoom = 10;
    const maxZoom = 17;

    if (entries.length === 1) {
      map.setView(entries[0].latlng, Math.max(minZoom, Math.min(maxZoom, 15)), {
        animate: !immediate,
      });
      return;
    }

    const bounds = L.latLngBounds(entries.map((e) => e.latlng));
    const pad = L.point(60, 60);
    const fitZoom = Math.max(
      minZoom,
      Math.min(maxZoom, map.getBoundsZoom(bounds, false, pad)),
    );
    map.setView(bounds.getCenter(), fitZoom, { animate: !immediate });
  }

  function render() {
    if (!map) return;
    const width = mapEl.clientWidth;
    const height = mapEl.clientHeight;
    entries.forEach((entry) => {
      const point = map.latLngToContainerPoint(entry.latlng);
      const inside =
        point.x >= 0 && point.y >= 0 && point.x <= width && point.y <= height;
      entry.element.style.display = inside ? "" : "none";
      if (inside) {
        entry.element.style.left = `${point.x}px`;
        entry.element.style.top = `${point.y}px`;
      }
    });
  }

  function directionsURL(lat, lng) {
    return (
      "https://www.google.com/maps/dir/?api=1&destination=" +
      encodeURIComponent(lat + "," + lng)
    );
  }

  function bindModal() {
    trigger.addEventListener("click", openModal);
    if (closeBtn) closeBtn.addEventListener("click", closeModal);
    modalEl.addEventListener("click", (event) => {
      if (event.target === modalEl) closeModal();
    });
    modalEl.addEventListener("close", () => {
      modalEl.classList.remove("is-closing");
      document.body.style.overflow = "";
    });
  }

  function closeModal() {
    if (prefersReducedMotion()) {
      finishClose();
      return;
    }
    modalEl.classList.add("is-closing");
    let closed = false;
    function onClose() {
      if (closed) return;
      closed = true;
      finishClose();
    }
    modalEl.addEventListener("animationend", onClose, { once: true });
    window.setTimeout(onClose, 260);
  }

  function finishClose() {
    if (typeof modalEl.close === "function" && modalEl.open) {
      modalEl.close();
    } else {
      modalEl.style.display = "none";
    }
    modalEl.classList.remove("is-closing");
    document.body.style.overflow = "";
  }

  function ensureLeaflet() {
    if (window.L) return Promise.resolve();
    if (leafletAssetsPromise) return leafletAssetsPromise;

    loadLeafletStylesheet();
    leafletAssetsPromise = new Promise((resolve, reject) => {
      const script = document.createElement("script");
      script.src = LEAFLET_JS_URL;
      script.async = true;
      script.defer = true;
      script.crossOrigin = "anonymous";
      script.setAttribute("fetchpriority", "low");
      script.onload = () => {
        if (window.L) {
          resolve();
        } else {
          leafletAssetsPromise = null;
          reject(new Error("Leaflet did not initialize"));
        }
      };
      script.onerror = () => {
        leafletAssetsPromise = null;
        reject(new Error("Leaflet failed to load"));
      };
      document.head.appendChild(script);
    });
    return leafletAssetsPromise;
  }

  function loadLeafletStylesheet() {
    if (document.querySelector('link[data-foedus-leaflet-css="true"]')) return;
    const link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = LEAFLET_CSS_URL;
    link.crossOrigin = "anonymous";
    link.dataset.foedusLeafletCss = "true";
    link.setAttribute("fetchpriority", "low");
    document.head.appendChild(link);
  }

  function parseSpots(el) {
    try {
      const parsed = JSON.parse(el.textContent);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }

  function prefersReducedMotion() {
    return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  }
})();
