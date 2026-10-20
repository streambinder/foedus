(() => {
  // ---------------------------------------------------------------
  // shared place search, also used by dashboard-features.js.
  // Photon answers fuzzy and address-shaped queries that Nominatim
  // misses; Nominatim covers named places Photon ranks low. Results
  // merge into { name, address, city, lat, lon } items. Queries that
  // mix a name with a location hint also run the anchored two-stage
  // search below; other short queries retry once without generic
  // venue words, so "parrocchia san clemente i papa" also tries
  // "san clemente papa".
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

  // Every provider fetch shares one timeout: a stalled request must
  // degrade to an empty result, never hang the whole search. Nominatim
  // takes the language as a query parameter; a custom Accept-Language
  // header makes the request non-simple, and the CORS preflight to
  // nominatim.openstreetmap.org is answered with a redirect, which
  // browsers treat as a failure, so the header silently disabled
  // Nominatim from every real browser.
  const FETCH_TIMEOUT_MS = 3000;

  function fetchJSON(url) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);
    return fetch(url, { signal: controller.signal })
      .then((res) => res.json())
      .finally(() => clearTimeout(timer));
  }

  function fetchNominatim(query) {
    const lang = document.documentElement.lang || "en";
    const url = `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query)}&format=jsonv2&addressdetails=1&namedetails=1&limit=5&accept-language=${encodeURIComponent(lang)}`;
    return fetchJSON(url)
      .then((data) => data.map(normalizeNominatim))
      .catch(() => []);
  }

  function fetchPhoton(query) {
    const url = `https://photon.komoot.io/api/?q=${encodeURIComponent(query)}&limit=6`;
    return fetchJSON(url)
      .then((data) => (data.features || []).map(normalizePhoton))
      .catch(() => []);
  }

  function fetchProviders(query) {
    return Promise.all([fetchPhoton(query), fetchNominatim(query)]).then(
      ([photon, nominatim]) => mergeResults([photon, nominatim]),
    );
  }

  function tokenize(text) {
    return text
      .toLowerCase()
      .replace(/[.,;:'"]/g, " ")
      .split(/\s+/)
      .filter((token) => token);
  }

  function stripGeneric(text) {
    return tokenize(text)
      .filter((token) => !GENERIC_TOKENS.has(token))
      .join(" ");
  }

  function relaxQuery(query) {
    const relaxed = stripGeneric(query);
    return relaxed && relaxed !== query.toLowerCase() ? relaxed : "";
  }

  function distanceKm(a, b) {
    const lat1 = Number(a.lat);
    const lon1 = Number(a.lon);
    const lat2 = Number(b.lat);
    const lon2 = Number(b.lon);
    if (!lat1 || !lon1 || !lat2 || !lon2) return Infinity;
    const rad = Math.PI / 180;
    const dLat = (lat2 - lat1) * rad;
    const dLon = (lon2 - lon1) * rad;
    const h =
      Math.sin(dLat / 2) ** 2 +
      Math.cos(lat1 * rad) * Math.cos(lat2 * rad) * Math.sin(dLon / 2) ** 2;
    return 2 * 6371 * Math.asin(Math.sqrt(h));
  }

  function fetchPhotonAnchor(hint) {
    const url = `https://photon.komoot.io/api/?q=${encodeURIComponent(hint)}&limit=1`;
    return fetchJSON(url)
      .then((data) => {
        const first = (data.features || []).map(normalizePhoton)[0];
        return first?.lat ? first : null;
      })
      .catch(() => null);
  }

  function fetchPhotonBiased(nameQuery, anchor) {
    const url = `https://photon.komoot.io/api/?q=${encodeURIComponent(nameQuery)}&lat=${anchor.lat}&lon=${anchor.lon}&zoom=16&location_bias_scale=0.1&limit=10`;
    return fetchJSON(url)
      .then((data) => (data.features || []).map(normalizePhoton))
      .catch(() => []);
  }

  function fetchNominatimBounded(nameQuery, anchor) {
    const lang = document.documentElement.lang || "en";
    const lat = Number(anchor.lat);
    const lon = Number(anchor.lon);
    const viewbox = `${(lon - 0.01).toFixed(6)},${(lat + 0.01).toFixed(6)},${(lon + 0.01).toFixed(6)},${(lat - 0.01).toFixed(6)}`;
    const url = `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(nameQuery)}&format=jsonv2&addressdetails=1&namedetails=1&limit=10&layer=poi&viewbox=${viewbox}&bounded=1&accept-language=${encodeURIComponent(lang)}`;
    return fetchJSON(url)
      .then((data) => data.map(normalizeNominatim))
      .catch(() => []);
  }

  // Two-stage decomposition, the pattern production geocoders use for
  // "name + location hint" queries: split candidates are generated at
  // token boundaries and the data validates them. The hint suffix is
  // geocoded to an anchor; the name prefix is then searched around the
  // anchor with each provider's native bias. A split counts only when
  // the biased search finds the name near its own anchor.
  function anchoredSearch(query) {
    const tokens = query.trim().split(/\s+/);
    if (tokens.length < 3) return Promise.resolve(null);
    const hintLens = [3, 2, 1].filter((n) => n < tokens.length);
    function attempt(idx) {
      if (idx >= hintLens.length) return Promise.resolve(null);
      const hintLen = hintLens[idx];
      const nameQuery = stripGeneric(
        tokens.slice(0, tokens.length - hintLen).join(" "),
      );
      const nameTokens = nameQuery ? nameQuery.split(" ").length : 0;
      // a lone name word pairs only with a lone hint word, otherwise
      // the split eats too much of the name ("chiesa largo val ...")
      if (!nameQuery || (nameTokens < 2 && hintLen > 1)) {
        return attempt(idx + 1);
      }
      const hint = tokens.slice(tokens.length - hintLen).join(" ");
      return fetchPhotonAnchor(hint).then((anchor) => {
        if (!anchor) return attempt(idx + 1);
        return fetchPhotonBiased(nameQuery, anchor).then((biased) => {
          const near = biased.filter((item) => distanceKm(item, anchor) <= 2);
          if (!near.length) return attempt(idx + 1);
          return fetchNominatimBounded(nameQuery, anchor).then((bounded) => ({
            anchor: anchor,
            items: mergeResults([
              bounded,
              biased.filter((item) => distanceKm(item, anchor) <= 25),
            ]),
          }));
        });
      });
    }
    return attempt(0);
  }

  function tokenCoverage(tokens, text) {
    if (!tokens.length) return 0;
    const words = tokenize(text);
    let hits = 0;
    for (const token of tokens) {
      const hit = words.some(
        (word) =>
          word === token ||
          (token.length >= 3 &&
            (word.startsWith(token) || token.startsWith(word))),
      );
      if (hit) hits++;
    }
    return hits / tokens.length;
  }

  function rankWithAnchor(items, query, anchor) {
    const tokens = tokenize(stripGeneric(query));
    const scored = items.map((item, idx) => {
      let score =
        2 * tokenCoverage(tokens, item.name) +
        tokenCoverage(tokens, item.address);
      const dist = distanceKm(item, anchor);
      if (dist <= 0.3) score += 1;
      else if (dist <= 2) score += 0.5;
      else if (dist <= 20) score += 0.25;
      return { item: item, idx: idx, score: score };
    });
    scored.sort((a, b) => b.score - a.score || a.idx - b.idx);
    return scored.map((entry) => entry.item).slice(0, 8);
  }

  // Plain-path ranking: name coverage weighs double, address coverage
  // single, and an item matching the longest query token in its name
  // gets a bonus, because the longest token is usually the distinctive
  // one ("colonnetta", not "villa"). Ties keep provider order.
  function rankByCoverage(items, query) {
    const tokens = tokenize(stripGeneric(query));
    if (!tokens.length || items.length < 2) return items;
    const longest = tokens.reduce((a, b) => (b.length > a.length ? b : a));
    const scored = items.map((item, idx) => {
      let score =
        2 * tokenCoverage(tokens, item.name) +
        tokenCoverage(tokens, item.address);
      if (tokenCoverage([longest], item.name) === 1) score += 0.5;
      return { item: item, idx: idx, score: score };
    });
    scored.sort((a, b) => b.score - a.score || a.idx - b.idx);
    return scored.map((entry) => entry.item);
  }

  // Last-resort relaxation for a query written the way people know a
  // place ("villa colonnetta giardini e roseti") when the map data
  // knows a shorter name ("colonnetta la memoria"): drop one trailing
  // token at a time. A step counts only when some result matches the
  // step's longest token, so fuzzy matches on the generic words alone
  // ("villa", "giardini") never end the search early.
  function ladderSearch(query) {
    const tokens = tokenize(query);
    const steps = [];
    for (let n = tokens.length - 1; n >= 2; n--) {
      steps.push(tokens.slice(0, n));
    }
    function attempt(idx) {
      if (idx >= steps.length) return Promise.resolve([]);
      const step = steps[idx];
      const longest = step.reduce((a, b) => (b.length > a.length ? b : a));
      return fetchProviders(step.join(" ")).then((items) => {
        const accepted = items.filter(
          (item) =>
            tokenCoverage([longest], `${item.name} ${item.address}`) === 1,
        );
        if (accepted.length) return rankByCoverage(accepted, query);
        return attempt(idx + 1);
      });
    }
    return attempt(0);
  }

  function searchPlaces(query) {
    const providers = fetchProviders(query);
    const anchored = anchoredSearch(query);
    return Promise.all([providers, anchored]).then(([items, anch]) => {
      if (anch) {
        const pool = mergeResults([anch.items, items]);
        return rankWithAnchor(pool, query, anch.anchor);
      }
      if (items.length >= 3) return rankByCoverage(items, query);
      const relaxed = relaxQuery(query);
      const merged = relaxed
        ? fetchProviders(relaxed).then((more) => mergeResults([items, more]))
        : Promise.resolve(items);
      return merged.then((all) => {
        const ranked = rankByCoverage(all, query);
        return ranked.length ? ranked : ladderSearch(query);
      });
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

  // localized "searching" label: same data-attribute pattern as the
  // no-results label, so the template owns the wording
  function searchingLabel(dropdown) {
    if (dropdown.dataset.searchingLabel) return dropdown.dataset.searchingLabel;
    const any = document.querySelector(
      ".autocomplete-dropdown[data-searching-label]",
    );
    return any ? any.dataset.searchingLabel : "Searching…";
  }

  // placeholder row shown while a search runs: a multi-step search can
  // take seconds, and a silent field reads as a broken one
  function renderSearching(dropdown) {
    dropdown.innerHTML = "";
    const div = document.createElement("div");
    div.className = "autocomplete-no-results autocomplete-searching";
    div.textContent = searchingLabel(dropdown);
    dropdown.appendChild(div);
    dropdown.classList.remove("hidden");
  }

  window.FoedusSearchingLabel = searchingLabel;
  window.FoedusRenderSearching = renderSearching;

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
      results = [];
      activeIndex = -1;
      renderSearching(dropdown);
      searchPlaces(query)
        .then((items) => {
          if (seq === searchSeq) render(items);
        })
        .catch(() => {
          // the dropdown must always answer, even on a total failure:
          // silence reads as a broken field, the empty state does not
          if (seq === searchSeq) render([]);
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
