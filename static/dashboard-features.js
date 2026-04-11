function initDashboardFeatures() {
  "use strict";

  // ---------------------------------------------------------------
  // playlists management
  // ---------------------------------------------------------------
  var playlistsContainer = document.getElementById("playlists-container");
  var addPlaylistBtn = document.getElementById("add-playlist-btn");

  if (playlistsContainer && addPlaylistBtn && !playlistsContainer.dataset.bound) {
    playlistsContainer.dataset.bound = "true";
    addPlaylistBtn.addEventListener("click", function () {
      addPlaylistRow("");
      reindexPlaylists();
    });

    playlistsContainer.addEventListener("click", function (e) {
      if (e.target.classList.contains("remove-playlist-btn")) {
        e.target.closest(".playlist-row").remove();
        reindexPlaylists();
      }
    });
  }

  function addPlaylistRow(value) {
    var idx = playlistsContainer.querySelectorAll(".playlist-row").length;
    var row = document.createElement("div");
    row.className = "playlist-row";
    row.innerHTML =
      '<input type="text" name="spotify_playlist_' + idx + '" value="' + escapeAttr(value) + '" placeholder="https://open.spotify.com/playlist/..."/>' +
      '<button type="button" class="outline secondary remove-playlist-btn" aria-label="Remove">&times;</button>';
    playlistsContainer.appendChild(row);
  }

  function reindexPlaylists() {
    var rows = playlistsContainer.querySelectorAll(".playlist-row");
    rows.forEach(function (row, idx) {
      var input = row.querySelector('input[type="text"]');
      if (input) input.name = "spotify_playlist_" + idx;
    });
  }

  // ---------------------------------------------------------------
  // places management
  // ---------------------------------------------------------------
  var placesContainer = document.getElementById("places-container");
  var addPlaceBtn = document.getElementById("add-place-btn");

  if (placesContainer && addPlaceBtn && !placesContainer.dataset.bound) {
    placesContainer.dataset.bound = "true";
    addPlaceBtn.addEventListener("click", function () {
      addPlaceCard();
      reindexPlaces();
    });

    placesContainer.addEventListener("click", function (e) {
      var target = e.target;
      if (target.classList.contains("place-remove")) {
        target.closest(".place-card").remove();
        reindexPlaces();
      } else if (target.classList.contains("place-move-up")) {
        var card = target.closest(".place-card");
        var prev = card.previousElementSibling;
        if (prev) {
          placesContainer.insertBefore(card, prev);
          reindexPlaces();
        }
      } else if (target.classList.contains("place-move-down")) {
        var card = target.closest(".place-card");
        var next = card.nextElementSibling;
        if (next) {
          placesContainer.insertBefore(next, card);
          reindexPlaces();
        }
      }
    });

    // initialize autocomplete on existing place cards
    placesContainer.querySelectorAll(".place-card").forEach(function (card) {
      initPlaceAutocomplete(card);
    });
  }

  function addPlaceCard() {
    var idx = placesContainer.querySelectorAll(".place-card").length;
    var card = document.createElement("div");
    card.className = "place-card";
    card.dataset.index = idx;
    card.innerHTML =
      '<div class="place-card-header">' +
        '<span class="place-number">' + (idx + 1) + "</span>" +
        '<div class="place-card-actions">' +
          '<button type="button" class="place-move-up" aria-label="Move up">&uarr;</button>' +
          '<button type="button" class="place-move-down" aria-label="Move down">&darr;</button>' +
          '<button type="button" class="place-remove outline secondary" aria-label="Remove">&times;</button>' +
        "</div>" +
      "</div>" +
      '<div class="grid">' +
        "<div>" +
          "<label>Label</label>" +
          '<input type="text" name="place_label_' + idx + '" placeholder="e.g. First date"/>' +
        "</div>" +
        '<div style="position:relative">' +
          "<label>Location</label>" +
          '<input type="text" class="place-location-input" autocomplete="off" placeholder="Search place..."/>' +
          '<div class="autocomplete-dropdown place-dropdown"></div>' +
          '<input type="hidden" name="place_name_' + idx + '" class="place-name-hidden"/>' +
          '<input type="hidden" name="place_address_' + idx + '" class="place-address-hidden"/>' +
          '<input type="hidden" name="place_lat_' + idx + '" class="place-lat-hidden" value="0"/>' +
          '<input type="hidden" name="place_lng_' + idx + '" class="place-lng-hidden" value="0"/>' +
        "</div>" +
      "</div>";
    placesContainer.appendChild(card);
    initPlaceAutocomplete(card);
  }

  function reindexPlaces() {
    var cards = placesContainer.querySelectorAll(".place-card");
    cards.forEach(function (card, idx) {
      card.dataset.index = idx;
      var num = card.querySelector(".place-number");
      if (num) num.textContent = idx + 1;
      var label = card.querySelector('input[name^="place_label_"]');
      if (label) label.name = "place_label_" + idx;
      var name = card.querySelector(".place-name-hidden");
      if (name) name.name = "place_name_" + idx;
      var addr = card.querySelector(".place-address-hidden");
      if (addr) addr.name = "place_address_" + idx;
      var lat = card.querySelector(".place-lat-hidden");
      if (lat) lat.name = "place_lat_" + idx;
      var lng = card.querySelector(".place-lng-hidden");
      if (lng) lng.name = "place_lng_" + idx;
    });
  }

  // ---------------------------------------------------------------
  // accommodation suggestions management
  // ---------------------------------------------------------------
  var accommodationsContainer = document.getElementById("accommodations-container");
  var addAccommodationBtn = document.getElementById("add-accommodation-btn");

  if (accommodationsContainer && addAccommodationBtn && !accommodationsContainer.dataset.bound) {
    accommodationsContainer.dataset.bound = "true";
    addAccommodationBtn.addEventListener("click", function () {
      addAccommodationCard();
      reindexAccommodations();
    });

    accommodationsContainer.addEventListener("click", function (e) {
      if (e.target.classList.contains("accommodation-remove")) {
        e.target.closest(".accommodation-card").remove();
        reindexAccommodations();
      }
    });
  }

  function addAccommodationCard() {
    var idx = accommodationsContainer.querySelectorAll(".accommodation-card").length;
    var card = document.createElement("div");
    card.className = "accommodation-card";
    card.innerHTML =
      '<div class="accommodation-card-header">' +
        '<span class="accommodation-number">' + (idx + 1) + "</span>" +
        '<button type="button" class="accommodation-remove outline secondary" aria-label="Remove">&times;</button>' +
      "</div>" +
      '<div class="grid">' +
        "<div>" +
          "<label>Name</label>" +
          '<input type="text" name="accommodation_name_' + idx + '" placeholder="e.g. Agriturismo Il Gelsomino"/>' +
        "</div>" +
        "<div>" +
          "<label>Link</label>" +
          '<input type="url" name="accommodation_url_' + idx + '" placeholder="https://..."/>' +
        "</div>" +
      "</div>" +
      "<div>" +
        "<label>Description</label>" +
        '<textarea name="accommodation_description_' + idx + '" rows="3" placeholder="Optional note for guests"></textarea>' +
      "</div>";
    accommodationsContainer.appendChild(card);
  }

  function reindexAccommodations() {
    var cards = accommodationsContainer.querySelectorAll(".accommodation-card");
    cards.forEach(function (card, idx) {
      var num = card.querySelector(".accommodation-number");
      if (num) num.textContent = idx + 1;
      var name = card.querySelector('input[name^="accommodation_name_"]');
      if (name) name.name = "accommodation_name_" + idx;
      var url = card.querySelector('input[name^="accommodation_url_"]');
      if (url) url.name = "accommodation_url_" + idx;
      var description = card.querySelector('textarea[name^="accommodation_description_"]');
      if (description) description.name = "accommodation_description_" + idx;
    });
  }

  // ---------------------------------------------------------------
  // place autocomplete (reuses Nominatim, same pattern as autocomplete.js)
  // ---------------------------------------------------------------
  function initPlaceAutocomplete(card) {
    var input = card.querySelector(".place-location-input");
    var dropdown = card.querySelector(".place-dropdown");
    var nameHidden = card.querySelector(".place-name-hidden");
    var addressHidden = card.querySelector(".place-address-hidden");
    var latHidden = card.querySelector(".place-lat-hidden");
    var lngHidden = card.querySelector(".place-lng-hidden");
    if (!input || !dropdown || !nameHidden || !addressHidden) return;

    var debounceTimer = null;
    var activeIndex = -1;
    var results = [];

    function hide() {
      dropdown.classList.add("hidden");
      dropdown.innerHTML = "";
      activeIndex = -1;
      results = [];
    }

    function render(items) {
      results = items;
      activeIndex = -1;
      dropdown.innerHTML = "";
      if (items.length === 0) { hide(); return; }
      items.forEach(function (item, idx) {
        var div = document.createElement("div");
        var name = extractName(item);
        var address = extractAddress(item, name);
        div.innerHTML =
          '<span style="font-weight:500;color:var(--charcoal-dark)">' + escapeHtml(name) + "</span>" +
          (address ? '<br><span style="font-size:var(--fs-xs);color:var(--sage-dark)">' + escapeHtml(address) + "</span>" : "");
        div.addEventListener("mousedown", function (e) {
          e.preventDefault();
          select(idx);
        });
        dropdown.appendChild(div);
      });
      dropdown.classList.remove("hidden");
    }

    function highlight(idx) {
      var children = dropdown.children;
      for (var i = 0; i < children.length; i++) {
        children[i].style.background = i === idx ? "var(--ivory-deep)" : "";
      }
      activeIndex = idx;
    }

    function select(idx) {
      var item = results[idx];
      if (!item) return;
      var name = extractName(item);
      var address = extractAddress(item, name);
      nameHidden.value = name;
      addressHidden.value = address;
      latHidden.value = item.lat || "0";
      lngHidden.value = item.lon || "0";
      input.value = name + (address ? ", " + address : "");
      // show coords confirmation
      var existing = card.querySelector(".place-coords");
      if (existing) existing.remove();
      if (item.lat && item.lon) {
        var coordsDiv = document.createElement("div");
        coordsDiv.className = "place-coords";
        coordsDiv.innerHTML =
          '<span class="place-coords-check">&#10003;</span>' +
          "<code>" + parseFloat(item.lat).toFixed(4) + ", " + parseFloat(item.lon).toFixed(4) + "</code>";
        card.appendChild(coordsDiv);
      }
      hide();
    }

    function search(query) {
      var lang = document.documentElement.lang || "en";
      var url =
        "https://nominatim.openstreetmap.org/search?q=" +
        encodeURIComponent(query) +
        "&format=jsonv2&addressdetails=1&namedetails=1&limit=5";
      fetch(url, { headers: { "Accept-Language": lang } })
        .then(function (res) { return res.json(); })
        .then(function (data) { render(data); })
        .catch(function () { hide(); });
    }

    input.addEventListener("input", function () {
      nameHidden.value = "";
      addressHidden.value = "";
      latHidden.value = "0";
      lngHidden.value = "0";
      clearTimeout(debounceTimer);
      var q = input.value.trim();
      if (q.length < 3) { hide(); return; }
      debounceTimer = setTimeout(function () { search(q); }, 350);
    });

    input.addEventListener("keydown", function (e) {
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

    input.addEventListener("blur", function () {
      setTimeout(hide, 200);
    });
  }

  // ---------------------------------------------------------------
  // shared helpers
  // ---------------------------------------------------------------
  function extractName(item) {
    if (item.namedetails && item.namedetails.name) return item.namedetails.name;
    return item.display_name;
  }

  function extractAddress(item, name) {
    if (!item.display_name) return "";
    var dn = item.display_name;
    if (name && dn.indexOf(name) === 0) {
      var rest = dn.substring(name.length).replace(/^,\s*/, "");
      return rest;
    }
    return dn;
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return str.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }

  // ---------------------------------------------------------------
  // impersonations management
  // ---------------------------------------------------------------
  var impersonationsContainer = document.getElementById("impersonations-container");
  var addImpersonationBtn = document.getElementById("add-impersonation-btn");

  if (impersonationsContainer && addImpersonationBtn && !impersonationsContainer.dataset.bound) {
    impersonationsContainer.dataset.bound = "true";
    addImpersonationBtn.addEventListener("click", function () {
      addImpersonationCard();
      reindexImpersonations();
    });

    impersonationsContainer.addEventListener("click", function (e) {
      if (e.target.classList.contains("impersonation-remove")) {
        e.target.closest(".impersonation-card").remove();
        reindexImpersonations();
      }
    });
  }

  function addImpersonationCard() {
    var idx = impersonationsContainer.querySelectorAll(".impersonation-card").length;
    var card = document.createElement("div");
    card.className = "impersonation-card";
    card.innerHTML =
      '<div class="impersonation-card-header">' +
        '<span class="impersonation-number">' + (idx + 1) + "</span>" +
        '<button type="button" class="impersonation-remove outline secondary" aria-label="Remove">&times;</button>' +
      "</div>" +
      '<input type="text" name="impersonation_codename_' + idx + '" placeholder="e.g. Anna"/>' +
      '<textarea name="impersonation_profile_' + idx + '" rows="3" placeholder="Describe how this person writes..."></textarea>';
    impersonationsContainer.appendChild(card);
  }

  function reindexImpersonations() {
    var cards = impersonationsContainer.querySelectorAll(".impersonation-card");
    cards.forEach(function (card, idx) {
      var num = card.querySelector(".impersonation-number");
      if (num) num.textContent = idx + 1;
      var codename = card.querySelector('input[type="text"]');
      if (codename) codename.name = "impersonation_codename_" + idx;
      var profile = card.querySelector("textarea");
      if (profile) profile.name = "impersonation_profile_" + idx;
    });
  }
}

window.initDashboardFeatures = initDashboardFeatures;
initDashboardFeatures();
