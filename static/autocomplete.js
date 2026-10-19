(() => {
  // ---------------------------------------------------------------
  // shared place search, also used by dashboard-features.js.
  // Photon answers fuzzy and address-shaped queries that Nominatim
  // misses; Nominatim covers named places Photon ranks low. Results
  // merge into { name, address, city, lat, lon } items. When a query
  // finds little, it retries once without generic venue words, so
  // "parrocchia san clemente i papa" also tries "san clemente papa".
  // ---------------------------------------------------------------
  const GENERIC_TOKENS = new Set([
    "chiesa",
    "parrocchia",
    "basilica",
    "cattedrale",
    "santuario",
    "oratorio",
    "abbazia",
    "di",
    "de",
    "del",
    "dello",
    "della",
    "dei",
    "degli",
    "delle",
    "il",
    "lo",
    "la",
    "i",
    "gli",
    "le",
    "un",
    "uno",
    "una",
    "church",
    "the",
    "of",
  ]);

  function normalizeNominatim(item) {
    const name =
      item.namedetails?.name || (item.display_name || "").split(",")[0] || "";
    let address = item.display_name || "";
    if (name && address.indexOf(name) === 0) {
      address = address.substring(name.length).replace(/^,\s*/, "");
    }
    const a = item.address || {};
    return {
      name: name,
      address: address,
      city: a.city || a.town || a.village || a.municipality || "",
      lat: item.lat || "",
      lon: item.lon || "",
    };
  }

  function normalizePhoton(feature) {
    const p = feature.properties || {};
    const coords = feature.geometry?.coordinates || [];
    const name =
      p.name || [p.street, p.city].filter(Boolean).join(" ") || p.city || "";
    const parts = [];
    const street = [p.street, p.housenumber].filter(Boolean).join(" ");
    if (street && street !== name) parts.push(street);
    const cityLine = [p.postcode, p.city || p.locality]
      .filter(Boolean)
      .join(" ");
    if (cityLine) parts.push(cityLine);
    if (p.state) parts.push(p.state);
    if (p.country) parts.push(p.country);
    return {
      name: name,
      address: parts.filter((part) => part !== name).join(", "),
      city: p.city || p.locality || "",
      lat: coords[1] != null ? String(coords[1]) : "",
      lon: coords[0] != null ? String(coords[0]) : "",
    };
  }

  function mergeResults(lists) {
    const seen = new Set();
    const merged = [];
    for (const list of lists) {
      for (const item of list) {
        if (!item.name && !item.address) continue;
        const key = `${item.name.toLowerCase()}@${item.lat ? Number(item.lat).toFixed(3) : ""},${item.lon ? Number(item.lon).toFixed(3) : ""}`;
        if (seen.has(key)) continue;
        seen.add(key);
        merged.push(item);
      }
    }
    return merged.slice(0, 8);
  }

  function fetchNominatim(query) {
    const lang = document.documentElement.lang || "en";
    const url = `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query)}&format=jsonv2&addressdetails=1&namedetails=1&limit=5`;
    return fetch(url, { headers: { "Accept-Language": lang } })
      .then((res) => res.json())
      .then((data) => data.map(normalizeNominatim))
      .catch(() => []);
  }

  function fetchPhoton(query) {
    const url = `https://photon.komoot.io/api/?q=${encodeURIComponent(query)}&limit=6`;
    return fetch(url)
      .then((res) => res.json())
      .then((data) => (data.features || []).map(normalizePhoton))
      .catch(() => []);
  }

  function fetchProviders(query) {
    return Promise.all([fetchPhoton(query), fetchNominatim(query)]).then(
      ([photon, nominatim]) => mergeResults([photon, nominatim]),
    );
  }

  function relaxQuery(query) {
    const tokens = query
      .toLowerCase()
      .replace(/[.,;:'"]/g, " ")
      .split(/\s+/)
      .filter((token) => token && !GENERIC_TOKENS.has(token));
    const relaxed = tokens.join(" ");
    return relaxed && relaxed !== query.toLowerCase() ? relaxed : "";
  }

  function searchPlaces(query) {
    return fetchProviders(query).then((items) => {
      if (items.length >= 3) return items;
      const relaxed = relaxQuery(query);
      if (!relaxed) return items;
      return fetchProviders(relaxed).then((more) =>
        mergeResults([items, more]),
      );
    });
  }

  window.FoedusSearchPlaces = searchPlaces;

  // localized "no results" label: the dashboard renders it as a data
  // attribute on each dropdown; fall back to any dropdown on the page
  function noResultsLabel(dropdown) {
    if (dropdown.dataset.noResultsLabel) return dropdown.dataset.noResultsLabel;
    const any = document.querySelector(
      ".autocomplete-dropdown[data-no-results-label]",
    );
    return any ? any.dataset.noResultsLabel : "No place found with that name";
  }

  window.FoedusNoResultsLabel = noResultsLabel;

  function initAutocomplete(config) {
    const input = document.getElementById(config.inputId);
    const dropdown = document.getElementById(config.dropdownId);
    const nameHidden = document.getElementById(config.nameHiddenId);
    const addressHidden = document.getElementById(config.addressHiddenId);
    const cityHidden = document.getElementById(config.cityHiddenId);
    const latHidden = config.latHiddenId
      ? document.getElementById(config.latHiddenId)
      : null;
    const lngHidden = config.lngHiddenId
      ? document.getElementById(config.lngHiddenId)
      : null;
    if (!input || !dropdown || !nameHidden || !addressHidden) return;

    let debounceTimer = null;
    let activeIndex = -1;
    let results = [];
    let searchSeq = 0;

    function hide() {
      searchSeq++;
      dropdown.classList.add("hidden");
      dropdown.innerHTML = "";
      activeIndex = -1;
      results = [];
    }

    function render(items) {
      results = items;
      activeIndex = -1;
      dropdown.innerHTML = "";
      if (items.length === 0) {
        const div = document.createElement("div");
        div.className = "autocomplete-no-results";
        div.textContent = noResultsLabel(dropdown);
        dropdown.appendChild(div);
        dropdown.classList.remove("hidden");
        return;
      }
      items.forEach((item, idx) => {
        const div = document.createElement("div");
        div.innerHTML =
          '<span style="font-weight:500;color:var(--charcoal-dark)">' +
          escapeHtml(item.name) +
          "</span>" +
          (item.address
            ? '<br><span style="font-size:var(--fs-xs);color:var(--sage-dark)">' +
              escapeHtml(item.address) +
              "</span>"
            : "");
        div.addEventListener("mousedown", (e) => {
          e.preventDefault();
          select(idx);
        });
        dropdown.appendChild(div);
      });
      dropdown.classList.remove("hidden");
    }

    function highlight(idx) {
      const children = dropdown.children;
      for (let i = 0; i < children.length; i++) {
        children[i].style.background = i === idx ? "var(--ivory-deep)" : "";
      }
      activeIndex = idx;
    }

    function select(idx) {
      const item = results[idx];
      if (!item) return;
      nameHidden.value = item.name;
      addressHidden.value = item.address;
      if (cityHidden) cityHidden.value = item.city;
      // providers return coordinates as strings; capture them for venues
      // that pin on the parking map (ceremony/reception)
      if (latHidden) latHidden.value = item.lat || "0";
      if (lngHidden) lngHidden.value = item.lon || "0";
      input.value = item.name + (item.address ? `, ${item.address}` : "");
      hide();
    }

    function escapeHtml(str) {
      const div = document.createElement("div");
      div.appendChild(document.createTextNode(str));
      return div.innerHTML;
    }

    function search(query) {
      const seq = ++searchSeq;
      searchPlaces(query).then((items) => {
        if (seq === searchSeq) render(items);
      });
    }

    input.addEventListener("input", () => {
      nameHidden.value = "";
      addressHidden.value = "";
      if (cityHidden) cityHidden.value = "";
      if (latHidden) latHidden.value = "0";
      if (lngHidden) lngHidden.value = "0";
      clearTimeout(debounceTimer);
      const q = input.value.trim();
      if (q.length < 3) {
        hide();
        return;
      }
      debounceTimer = setTimeout(() => {
        search(q);
      }, 350);
    });

    input.addEventListener("keydown", (e) => {
      if (results.length === 0) return;
      if (e.key === "ArrowDown") {
        e.preventDefault();
        highlight(Math.min(activeIndex + 1, results.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        highlight(Math.max(activeIndex - 1, 0));
      } else if (e.key === "Enter" && activeIndex >= 0) {
        e.preventDefault();
        select(activeIndex);
      } else if (e.key === "Escape") {
        hide();
      }
    });

    input.addEventListener("blur", () => {
      setTimeout(hide, 200);
    });

    document.addEventListener("click", (e) => {
      if (!input.contains(e.target) && !dropdown.contains(e.target)) {
        hide();
      }
    });
  }

  initAutocomplete({
    inputId: "ceremony-input",
    dropdownId: "ceremony-dropdown",
    nameHiddenId: "ceremony-address-hidden",
    addressHiddenId: "ceremony-location-hidden",
    cityHiddenId: "ceremony-city-hidden",
    latHiddenId: "ceremony-lat-hidden",
    lngHiddenId: "ceremony-lng-hidden",
  });
  initAutocomplete({
    inputId: "reception-input",
    dropdownId: "reception-dropdown",
    nameHiddenId: "reception-address-hidden",
    addressHiddenId: "reception-location-hidden",
    cityHiddenId: "reception-city-hidden",
    latHiddenId: "reception-lat-hidden",
    lngHiddenId: "reception-lng-hidden",
  });
})();
