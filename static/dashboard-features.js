function initDashboardFeatures() {
  initDashboardLazyImages();

  // ---------------------------------------------------------------
  // lazy dashboard previews
  // ---------------------------------------------------------------
  function initDashboardLazyImages() {
    loadDashboardLazyImages(document, false);

    if (
      !document.body ||
      document.body.dataset.dashboardLazyImagesBound === "true"
    )
      return;
    document.body.dataset.dashboardLazyImagesBound = "true";
    document.addEventListener("shown.bs.collapse", (event) => {
      loadDashboardLazyImages(event.target, true);
    });
  }

  function loadDashboardLazyImages(root, force) {
    const scope = root instanceof Element ? root : document;
    scope
      .querySelectorAll("img.dashboard-lazy-image[data-src]")
      .forEach((img) => {
        if (!force && img.closest(".accordion-collapse:not(.show)")) return;
        const src = img.getAttribute("data-src");
        if (!src) return;
        img.src = src;
        img.removeAttribute("data-src");
      });
  }

  // ---------------------------------------------------------------
  // places / honeymoon management
  // ---------------------------------------------------------------
  bindLocationCollection({
    container: document.getElementById("places-container"),
    addButton: document.getElementById("add-place-btn"),
    prefix: "place",
    imageIdPrefix: "place-image",
    imageFormat: "image/jpeg",
    imageMaxWidth: 1200,
    imageMaxHeight: 1200,
    imageMaxBytes: 524288,
    imageAlt: "Place preview",
    includeDate: true,
  });

  bindLocationCollection({
    container: document.getElementById("honeymoon-container"),
    addButton: document.getElementById("add-honeymoon-btn"),
    prefix: "honeymoon",
    imageIdPrefix: "honeymoon-image",
    imageFormat: "image/png",
    imageMaxWidth: 900,
    imageMaxHeight: 900,
    imageMaxBytes: 614400,
    imageAlt: "Honeymoon preview",
    includeDate: false,
  });

  // ---------------------------------------------------------------
  // parking spots management (coords-only, so its own lean binder
  // instead of bindLocationCollection which hardcodes label/image markup)
  // ---------------------------------------------------------------
  bindParkingCollection(
    document.getElementById("parking-container"),
    document.getElementById("add-parking-btn"),
  );

  function bindParkingCollection(container, addButton) {
    if (!container || !addButton || container.dataset.bound) return;
    container.dataset.bound = "true";

    addButton.addEventListener("click", () => {
      addParkingCard(container);
      reindexParkingCards(container);
    });

    container.addEventListener("click", (e) => {
      if (e.target.classList.contains("place-remove")) {
        e.target.closest(".parking-card").remove();
        reindexParkingCards(container);
      }
    });

    container.querySelectorAll(".parking-card").forEach((card) => {
      initPlaceAutocomplete(card);
    });
  }

  function addParkingCard(container) {
    const idx = container.querySelectorAll(".parking-card").length;
    const card = document.createElement("div");
    card.className = "place-card parking-card";
    card.dataset.index = idx;
    card.innerHTML =
      '<div class="place-card-header">' +
      '<div class="place-card-meta">' +
      '<span class="place-number">' +
      (idx + 1) +
      "</span>" +
      "</div>" +
      '<div class="place-card-actions">' +
      '<button type="button" class="place-remove outline secondary" aria-label="Remove">&times;</button>' +
      "</div>" +
      "</div>" +
      '<div style="position:relative">' +
      "<label>Address</label>" +
      '<input type="text" class="place-location-input" autocomplete="off" placeholder="Search for an address..."/>' +
      '<div class="autocomplete-dropdown place-dropdown"></div>' +
      '<input type="hidden" name="parking_lat_' +
      idx +
      '" class="place-lat-hidden" value="0"/>' +
      '<input type="hidden" name="parking_lng_' +
      idx +
      '" class="place-lng-hidden" value="0"/>' +
      "</div>";
    container.appendChild(card);
    initPlaceAutocomplete(card);
  }

  function reindexParkingCards(container) {
    container.querySelectorAll(".parking-card").forEach((card, idx) => {
      card.dataset.index = idx;
      const num = card.querySelector(".place-number");
      if (num) num.textContent = idx + 1;
      const lat = card.querySelector(".place-lat-hidden");
      if (lat) lat.name = `parking_lat_${idx}`;
      const lng = card.querySelector(".place-lng-hidden");
      if (lng) lng.name = `parking_lng_${idx}`;
    });
  }

  function bindLocationCollection(config) {
    const container = config.container;
    const addButton = config.addButton;
    if (!container || !addButton || container.dataset.bound) return;

    container.dataset.bound = "true";
    addButton.addEventListener("click", () => {
      addLocationCard(container, config);
      reindexLocationCards(container, config);
    });

    container.addEventListener("click", (e) => {
      const target = e.target;
      if (target.classList.contains("place-remove")) {
        target.closest(".place-card").remove();
        reindexLocationCards(container, config);
      } else if (target.classList.contains("place-move-up")) {
        const card = target.closest(".place-card");
        const prev = card.previousElementSibling;
        if (prev) {
          container.insertBefore(card, prev);
          reindexLocationCards(container, config);
        }
      } else if (target.classList.contains("place-move-down")) {
        const movingCard = target.closest(".place-card");
        const next = movingCard.nextElementSibling;
        if (next) {
          container.insertBefore(next, movingCard);
          reindexLocationCards(container, config);
        }
      }
    });

    container.querySelectorAll(".place-card").forEach((card) => {
      initPlaceAutocomplete(card);
    });
  }

  function addLocationCard(container, config) {
    const idx = container.querySelectorAll(".place-card").length;
    const card = document.createElement("div");
    card.className = "place-card";
    card.dataset.index = idx;
    card.innerHTML =
      '<div class="place-card-header">' +
      '<div class="place-card-meta">' +
      '<span class="place-number">' +
      (idx + 1) +
      "</span>" +
      "</div>" +
      '<div class="place-card-actions">' +
      '<button type="button" class="place-move-up" aria-label="Move up">&uarr;</button>' +
      '<button type="button" class="place-move-down" aria-label="Move down">&darr;</button>' +
      '<button type="button" class="place-remove outline secondary" aria-label="Remove">&times;</button>' +
      "</div>" +
      "</div>" +
      (config.includeDate
        ? '<div class="grid">' +
          "<div>" +
          "<label>Label</label>" +
          '<input type="text" name="' +
          config.prefix +
          "_label_" +
          idx +
          '" placeholder="e.g. First date"/>' +
          "</div>" +
          "<div>" +
          "<label>Date</label>" +
          '<input type="date" name="' +
          config.prefix +
          "_date_" +
          idx +
          '"/>' +
          "</div>" +
          "</div>"
        : "<div>" +
          "<label>Label</label>" +
          '<input type="text" name="' +
          config.prefix +
          "_label_" +
          idx +
          '" placeholder="e.g. First date"/>' +
          "</div>") +
      '<div class="grid">' +
      '<div style="position:relative">' +
      "<label>Address</label>" +
      '<input type="text" class="place-location-input" autocomplete="off" placeholder="Search for an address..."/>' +
      '<div class="autocomplete-dropdown place-dropdown"></div>' +
      '<input type="hidden" name="' +
      config.prefix +
      "_name_" +
      idx +
      '" class="place-name-hidden"/>' +
      '<input type="hidden" name="' +
      config.prefix +
      "_address_" +
      idx +
      '" class="place-address-hidden"/>' +
      '<input type="hidden" name="' +
      config.prefix +
      "_lat_" +
      idx +
      '" class="place-lat-hidden" value="0"/>' +
      '<input type="hidden" name="' +
      config.prefix +
      "_lng_" +
      idx +
      '" class="place-lng-hidden" value="0"/>' +
      "</div>" +
      "<div>" +
      "<label>Image</label>" +
      '<div class="managed-image-field">' +
      '<input type="file" accept="image/*" class="managed-image-file" data-target-input="' +
      config.imageIdPrefix +
      "-data-" +
      idx +
      '" data-preview-target="' +
      config.imageIdPrefix +
      "-preview-" +
      idx +
      '" data-media-id-input="' +
      config.imageIdPrefix +
      "-media-id-" +
      idx +
      '" data-format="' +
      escapeAttr(config.imageFormat || "image/jpeg") +
      '" data-quality="0.88" data-max-width="' +
      config.imageMaxWidth +
      '" data-max-height="' +
      config.imageMaxHeight +
      '" data-max-bytes="' +
      (config.imageMaxBytes || 0) +
      '"/>' +
      '<input type="hidden" name="' +
      config.prefix +
      "_image_" +
      idx +
      '" id="' +
      config.imageIdPrefix +
      "-data-" +
      idx +
      '" class="place-image-hidden"/>' +
      '<input type="hidden" name="' +
      config.prefix +
      "_media_id_" +
      idx +
      '" id="' +
      config.imageIdPrefix +
      "-media-id-" +
      idx +
      '" class="place-media-id-hidden"/>' +
      '<img class="venue-image-preview place-image-preview" id="' +
      config.imageIdPrefix +
      "-preview-" +
      idx +
      '" style="display:none" alt="' +
      escapeAttr(config.imageAlt) +
      '"/>' +
      "</div>" +
      "</div>" +
      "</div>";
    container.appendChild(card);
    initPlaceAutocomplete(card);
    if (window.initDashboardImageResizers) {
      window.initDashboardImageResizers(card);
    }
  }

  function reindexLocationCards(container, config) {
    const cards = container.querySelectorAll(".place-card");
    cards.forEach((card, idx) => {
      card.dataset.index = idx;
      const num = card.querySelector(".place-number");
      if (num) num.textContent = idx + 1;
      const label = card.querySelector(
        `input[name^="${config.prefix}_label_"]`,
      );
      if (label) label.name = `${config.prefix}_label_${idx}`;
      const date = card.querySelector(`input[name^="${config.prefix}_date_"]`);
      if (date) date.name = `${config.prefix}_date_${idx}`;
      const name = card.querySelector(".place-name-hidden");
      if (name) name.name = `${config.prefix}_name_${idx}`;
      const addr = card.querySelector(".place-address-hidden");
      if (addr) addr.name = `${config.prefix}_address_${idx}`;
      const lat = card.querySelector(".place-lat-hidden");
      if (lat) lat.name = `${config.prefix}_lat_${idx}`;
      const lng = card.querySelector(".place-lng-hidden");
      if (lng) lng.name = `${config.prefix}_lng_${idx}`;
      const image = card.querySelector(".place-image-hidden");
      if (image) {
        image.name = `${config.prefix}_image_${idx}`;
        image.id = `${config.imageIdPrefix}-data-${idx}`;
      }
      const imageToken = card.querySelector(".place-media-id-hidden");
      if (imageToken) {
        imageToken.name = `${config.prefix}_media_id_${idx}`;
        imageToken.id = `${config.imageIdPrefix}-media-id-${idx}`;
      }
      const imageFile = card.querySelector(".managed-image-file");
      if (imageFile) {
        imageFile.dataset.targetInput = `${config.imageIdPrefix}-data-${idx}`;
        imageFile.dataset.previewTarget = `${config.imageIdPrefix}-preview-${idx}`;
        imageFile.dataset.mediaIdInput = `${config.imageIdPrefix}-media-id-${idx}`;
        imageFile.dataset.format = config.imageFormat || "image/jpeg";
        imageFile.dataset.maxBytes = String(config.imageMaxBytes || 0);
      }
      const imagePreview = card.querySelector(".place-image-preview");
      if (imagePreview)
        imagePreview.id = `${config.imageIdPrefix}-preview-${idx}`;
    });
  }

  // ---------------------------------------------------------------
  // accommodation suggestions management
  // ---------------------------------------------------------------
  const accommodationsContainer = document.getElementById(
    "accommodations-container",
  );
  const addAccommodationBtn = document.getElementById("add-accommodation-btn");

  if (
    accommodationsContainer &&
    addAccommodationBtn &&
    !accommodationsContainer.dataset.bound
  ) {
    accommodationsContainer.dataset.bound = "true";
    addAccommodationBtn.addEventListener("click", () => {
      addAccommodationCard();
      reindexAccommodations();
    });

    accommodationsContainer.addEventListener("click", (e) => {
      if (e.target.classList.contains("accommodation-remove")) {
        e.target.closest(".accommodation-card").remove();
        reindexAccommodations();
      }
    });
  }

  function addAccommodationCard() {
    const idx = accommodationsContainer.querySelectorAll(
      ".accommodation-card",
    ).length;
    const card = document.createElement("div");
    card.className = "accommodation-card";
    card.innerHTML =
      '<div class="accommodation-card-header">' +
      '<span class="accommodation-number">' +
      (idx + 1) +
      "</span>" +
      '<button type="button" class="accommodation-remove outline secondary" aria-label="Remove">&times;</button>' +
      "</div>" +
      '<div class="grid">' +
      "<div>" +
      "<label>Name</label>" +
      '<input type="text" name="accommodation_name_' +
      idx +
      '" placeholder="e.g. Agriturismo Il Gelsomino"/>' +
      "</div>" +
      "<div>" +
      "<label>Link</label>" +
      '<input type="url" name="accommodation_url_' +
      idx +
      '" placeholder="https://..."/>' +
      "</div>" +
      "</div>" +
      "<div>" +
      "<label>Description</label>" +
      '<textarea name="accommodation_description_' +
      idx +
      '" rows="3" placeholder="Optional note for guests"></textarea>' +
      "</div>";
    accommodationsContainer.appendChild(card);
  }

  function reindexAccommodations() {
    const cards = accommodationsContainer.querySelectorAll(
      ".accommodation-card",
    );
    cards.forEach((card, idx) => {
      const num = card.querySelector(".accommodation-number");
      if (num) num.textContent = idx + 1;
      const name = card.querySelector('input[name^="accommodation_name_"]');
      if (name) name.name = `accommodation_name_${idx}`;
      const url = card.querySelector('input[name^="accommodation_url_"]');
      if (url) url.name = `accommodation_url_${idx}`;
      const description = card.querySelector(
        'textarea[name^="accommodation_description_"]',
      );
      if (description) description.name = `accommodation_description_${idx}`;
    });
  }

  // ---------------------------------------------------------------
  // place autocomplete (shared FoedusSearchPlaces from autocomplete.js)
  // ---------------------------------------------------------------
  function initPlaceAutocomplete(card) {
    const input = card.querySelector(".place-location-input");
    const dropdown = card.querySelector(".place-dropdown");
    const nameHidden = card.querySelector(".place-name-hidden");
    const addressHidden = card.querySelector(".place-address-hidden");
    const latHidden = card.querySelector(".place-lat-hidden");
    const lngHidden = card.querySelector(".place-lng-hidden");
    // name/address are optional: parking cards store coords only
    if (!input || !dropdown || !latHidden || !lngHidden) return;

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
        div.textContent = window.FoedusNoResultsLabel(dropdown);
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
      if (nameHidden) nameHidden.value = item.name;
      if (addressHidden) addressHidden.value = item.address;
      latHidden.value = item.lat || "0";
      lngHidden.value = item.lon || "0";
      input.value = item.name + (item.address ? `, ${item.address}` : "");
      renderCoordsBadge(card, item.lat, item.lon);
      hide();
    }

    function search(query) {
      const seq = ++searchSeq;
      window.FoedusSearchPlaces(query).then((items) => {
        if (seq === searchSeq) render(items);
      });
    }

    input.addEventListener("input", () => {
      if (nameHidden) nameHidden.value = "";
      if (addressHidden) addressHidden.value = "";
      latHidden.value = "0";
      lngHidden.value = "0";
      renderCoordsBadge(card, "", "");
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
  }

  // ---------------------------------------------------------------
  // shared helpers
  // ---------------------------------------------------------------
  function renderCoordsBadge(card, lat, lng) {
    const meta = card.querySelector(".place-card-meta");
    if (!meta) return;

    const existing = meta.querySelector(".place-coords");
    if (existing) existing.remove();
    if (!lat || !lng) return;

    const coordsDiv = document.createElement("div");
    coordsDiv.className = "place-coords";
    coordsDiv.innerHTML =
      '<span class="place-coords-check">&#10003;</span>' +
      "<code>" +
      parseFloat(lat).toFixed(4) +
      ", " +
      parseFloat(lng).toFixed(4) +
      "</code>";
    meta.appendChild(coordsDiv);
  }

  function escapeHtml(str) {
    const div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return str
      .replace(/&/g, "&amp;")
      .replace(/"/g, "&quot;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  // ---------------------------------------------------------------
  // impersonations management
  // ---------------------------------------------------------------
  const impersonationsContainer = document.getElementById(
    "impersonations-container",
  );
  const addImpersonationBtn = document.getElementById("add-impersonation-btn");

  if (
    impersonationsContainer &&
    addImpersonationBtn &&
    !impersonationsContainer.dataset.bound
  ) {
    impersonationsContainer.dataset.bound = "true";
    addImpersonationBtn.addEventListener("click", () => {
      addImpersonationCard();
      reindexImpersonations();
    });

    impersonationsContainer.addEventListener("click", (e) => {
      if (e.target.classList.contains("impersonation-remove")) {
        e.target.closest(".impersonation-card").remove();
        reindexImpersonations();
      }
    });
  }

  function addImpersonationCard() {
    const idx = impersonationsContainer.querySelectorAll(
      ".impersonation-card",
    ).length;
    const card = document.createElement("div");
    card.className = "impersonation-card";
    card.innerHTML =
      '<div class="impersonation-card-header">' +
      '<span class="impersonation-number">' +
      (idx + 1) +
      "</span>" +
      '<button type="button" class="impersonation-remove outline secondary" aria-label="Remove">&times;</button>' +
      "</div>" +
      '<input type="text" name="impersonation_codename_' +
      idx +
      '" placeholder="e.g. Anna"/>' +
      '<textarea name="impersonation_profile_' +
      idx +
      '" rows="3" placeholder="Describe how this person writes..."></textarea>';
    impersonationsContainer.appendChild(card);
  }

  function reindexImpersonations() {
    const cards = impersonationsContainer.querySelectorAll(
      ".impersonation-card",
    );
    cards.forEach((card, idx) => {
      const num = card.querySelector(".impersonation-number");
      if (num) num.textContent = idx + 1;
      const codename = card.querySelector('input[type="text"]');
      if (codename) codename.name = `impersonation_codename_${idx}`;
      const profile = card.querySelector("textarea");
      if (profile) profile.name = `impersonation_profile_${idx}`;
    });
  }
}

window.initDashboardFeatures = initDashboardFeatures;
initDashboardFeatures();

// ---------------------------------------------------------------
// dashboard glue: form interception, Datastar refresh dispatch,
// edit and counter modals, image resizers. This block used to live
// in dashboard-spa.js; the section-replacement machinery it was
// named after is gone now that every section morphs through
// Datastar fragments.
// ---------------------------------------------------------------
(() => {
  function init() {
    bindGlobalListeners();
    initDashboardFeatures();
    initImageResizers();
  }

  // The settings section effect calls this hook after Datastar morphs
  // the settings form, so feature bindings (place autocomplete, image
  // editors) are re-applied to the fresh DOM.
  window.FoedusRebindDashboard = () => {
    initDashboardFeatures();
    initImageResizers();
  };

  function bindGlobalListeners() {
    if (document.body.dataset.dashboardGlueBound === "true") return;
    document.body.dataset.dashboardGlueBound = "true";

    document.addEventListener("click", handleClick);
    document.addEventListener("submit", handleSubmit);
  }

  function handleClick(event) {
    const editLink = event.target.closest(
      '.actions a[href^="/dashboard/"][href$="/edit"]',
    );
    if (editLink) {
      event.preventDefault();
      openDashboardEditModal(editLink.href);
      return;
    }

    const counterTrigger = event.target.closest(".counter-names-trigger");
    if (counterTrigger) {
      event.preventDefault();
      openCounterNamesModal(
        counterTrigger.dataset.counterCategory,
        counterTrigger.dataset.counterLabel,
        counterTrigger.textContent.trim(),
      );
      return;
    }

    const modalOverlay = event.target.closest("[data-dashboard-modal-overlay]");
    if (modalOverlay && event.target === modalOverlay) {
      // click landed on the backdrop itself, not on inner modal content
      event.preventDefault();
      closeDashboardModal();
      return;
    }

    const modalClose = event.target.closest("[data-dashboard-modal-close]");
    if (modalClose) {
      event.preventDefault();
      closeDashboardModal();
    }
  }

  function handleSubmit(event) {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    if (event.defaultPrevented) return;
    if (!isDashboardForm(form)) return;

    event.preventDefault();
    submitForm(form).catch((err) => {
      console.error(err);
      window.alert("Dashboard request failed.");
    });
  }

  function isDashboardForm(form) {
    const action = form.getAttribute("action") || "";
    return action.indexOf("/dashboard") === 0;
  }

  async function submitForm(form) {
    const action = form.getAttribute("action") || window.location.pathname;
    const method = (form.getAttribute("method") || "GET").toUpperCase();
    const formData = new FormData(form);
    const csrfToken = getCsrfToken(form);
    const response = await fetch(action, {
      method: method,
      body: method === "GET" ? null : formData,
      credentials: "same-origin",
      headers: buildRequestHeaders(csrfToken),
    });
    if (!response.ok) {
      throw new Error(`Request failed: ${response.status}`);
    }

    if (form.id === "invitation-form") {
      const code = response.headers.get("X-Invitation-Code");
      if (code && typeof window.copyInvitationURL === "function") {
        window.copyInvitationURL(code);
      }
    }

    applyFlash(await response.text());
    if (affectsCounters(action)) {
      dispatchSectionRefresh("dashboard-counters", "counters-refresh");
    }
    if (affectsPolls(action)) {
      dispatchSectionRefresh("polls-results", "polls-refresh");
    }
    if (affectsGuests(action)) {
      // the guests section is Datastar-driven: ask it to refetch its
      // fragment with its current signals instead of replacing its DOM
      const detail = {};
      if (form.id === "invitation-form") detail.reset = true;
      if (isGuestDeleteForm(form)) {
        detail.removeId = extractTrailingID(action);
      }
      dispatchGuestsRefresh(detail);
    }
    // the invitations, registry and soundtrack sections are Datastar-driven
    // too: their tables refetch their fragments instead of being replaced
    if (affectsInvitations(action)) {
      dispatchSectionRefresh("dashboard-invitations", "invitations-refresh");
    }
    if (affectsRegistry(action)) {
      dispatchSectionRefresh("dashboard-registry", "registry-refresh");
    }
    if (affectsSoundtrack(action)) {
      dispatchSectionRefresh(
        "dashboard-soundtrack-events",
        "soundtrack-refresh",
      );
    }
    if (/\/dashboard\/settings$/.test(action)) {
      dispatchSectionRefresh("dashboard-settings", "settings-refresh");
    }
    closeDashboardModal();
  }

  function affectsGuests(action) {
    return (
      /\/dashboard\/guests\/\d+\/confirm\/(ceremony|reception)$/.test(action) ||
      /\/dashboard\/guests(\/import)?$/.test(action) ||
      /\/dashboard\/guests\/\d+$/.test(action) ||
      /\/dashboard\/guests\/\d+\/delete$/.test(action) ||
      /\/dashboard\/invitations$/.test(action) ||
      /\/dashboard\/invitations\/\d+\/delete$/.test(action) ||
      /\/dashboard\/invitations\/\d+\/viewed\/reset$/.test(action)
    );
  }

  function dispatchGuestsRefresh(detail) {
    const section = document.getElementById("dashboard-guests");
    if (section) {
      section.dispatchEvent(
        new CustomEvent("guests-refresh", { detail: detail || {} }),
      );
    }
  }

  function dispatchSectionRefresh(sectionId, eventName) {
    const section = document.getElementById(sectionId);
    if (section) {
      section.dispatchEvent(new CustomEvent(eventName));
    }
  }

  function affectsInvitations(action) {
    return (
      /\/dashboard\/invitations(\/\d+\/(delete|viewed\/reset))?$/.test(
        action,
      ) ||
      /\/dashboard\/guests(\/import)?$/.test(action) ||
      /\/dashboard\/guests\/\d+$/.test(action) ||
      /\/dashboard\/guests\/\d+\/delete$/.test(action) ||
      /\/dashboard\/settings$/.test(action)
    );
  }

  function affectsRegistry(action) {
    return (
      /\/dashboard\/registry(\/\d+(\/(move\/(up|down)|delete))?)?$/.test(
        action,
      ) ||
      /\/dashboard\/gifts\/\d+(\/delete)?$/.test(action) ||
      /\/dashboard\/settings$/.test(action)
    );
  }

  function affectsSoundtrack(action) {
    return /\/dashboard\/soundtrack\/\d+\/delete$/.test(action);
  }

  function affectsCounters(action) {
    return (
      /\/dashboard\/guests\/\d+\/confirm\/(ceremony|reception)$/.test(action) ||
      /\/dashboard\/guests(\/import)?$/.test(action) ||
      /\/dashboard\/guests\/\d+$/.test(action) ||
      /\/dashboard\/guests\/\d+\/delete$/.test(action) ||
      /\/dashboard\/settings$/.test(action)
    );
  }

  function affectsPolls(action) {
    return (
      /\/dashboard\/polls(\/\d+(\/delete)?)?$/.test(action) ||
      /\/dashboard\/guests(\/import)?$/.test(action) ||
      /\/dashboard\/guests\/\d+$/.test(action) ||
      /\/dashboard\/guests\/\d+\/delete$/.test(action)
    );
  }

  // The submit response is the dashboard page after the redirect, and its
  // flash element carries the message the handler set. Applying it from
  // the response (instead of refetching the page) is what makes the flash
  // show at all: the redirect-following fetch already consumed the flash
  // cookie, so a second fetch would render an empty flash.
  function applyFlash(html) {
    const doc = new DOMParser().parseFromString(html, "text/html");
    const next = doc.getElementById("dashboard-flash");
    const current = document.getElementById("dashboard-flash");
    if (next && current) {
      current.replaceWith(next);
    }
  }

  function getCsrfToken(form) {
    const input = form.querySelector('input[name="_csrf"]');
    return input ? input.value : "";
  }

  function buildRequestHeaders(csrfToken) {
    const headers = { "X-Requested-With": "fetch" };
    if (csrfToken) {
      headers["X-Csrf-Token"] = csrfToken;
    }
    return headers;
  }

  function isGuestDeleteForm(form) {
    const action = form.getAttribute("action") || "";
    return /\/dashboard\/guests\/\d+\/delete$/.test(action);
  }

  function extractTrailingID(action) {
    const match = action.match(/\/(\d+)(?:\/[^/]+)?$/);
    return match ? match[1] : "";
  }

  async function fetchDocument(url) {
    const response = await fetch(url, {
      credentials: "same-origin",
      headers: { "X-Requested-With": "fetch" },
    });
    if (!response.ok) {
      throw new Error(`Request failed: ${response.status}`);
    }
    const html = await response.text();
    return new DOMParser().parseFromString(html, "text/html");
  }

  async function openDashboardEditModal(url) {
    const doc = await fetchDocument(url);
    const title = doc.querySelector("h1");
    const article = doc.querySelector("article");
    if (!article) return;

    const root = document.getElementById("dashboard-modal-root");
    if (!root) return;

    root.innerHTML =
      '<div class="modal-overlay" data-dashboard-modal-overlay>' +
      '<div class="modal-box dashboard-edit-modal-box">' +
      '<div class="dashboard-modal-header">' +
      "<h3>" +
      escapeHtml(title ? title.textContent.trim() : "Edit") +
      "</h3>" +
      '<button type="button" class="outline secondary" data-dashboard-modal-close>Close</button>' +
      "</div>" +
      article.innerHTML +
      "</div>" +
      "</div>";
    bindManagedImageResizers(root);
    lockBodyScrollForModal();
  }

  function closeDashboardModal() {
    const root = document.getElementById("dashboard-modal-root");
    if (root) root.innerHTML = "";
    document.body.classList.remove("dashboard-modal-open");
  }

  function lockBodyScrollForModal() {
    document.body.classList.add("dashboard-modal-open");
  }

  async function openCounterNamesModal(category, label, count) {
    if (!category) return;
    const root = document.getElementById("dashboard-modal-root");
    if (!root) return;

    // skeleton first so the user sees immediate feedback
    root.innerHTML =
      '<div class="modal-overlay" data-dashboard-modal-overlay>' +
      '<div class="modal-box dashboard-counter-modal-box">' +
      '<div class="dashboard-modal-header">' +
      "<h3>" +
      escapeHtml(label || "") +
      " <small>(" +
      escapeHtml(count || "") +
      ")</small></h3>" +
      '<button type="button" class="outline secondary" data-dashboard-modal-close>Close</button>' +
      "</div>" +
      '<div class="counter-names-list">…</div>' +
      "</div>" +
      "</div>";
    lockBodyScrollForModal();

    try {
      const res = await fetch(
        `/dashboard/counters/${encodeURIComponent(category)}`,
        {
          credentials: "same-origin",
          headers: { "X-Requested-With": "fetch" },
        },
      );
      if (!res.ok) throw new Error(`status ${res.status}`);
      const data = await res.json();
      const groups = Array.isArray(data.groups) ? data.groups : [];
      const list = root.querySelector(".counter-names-list");
      if (!list) return;
      if (groups.length === 0) {
        list.textContent = "—";
        return;
      }
      // continuous numbering across groups, separated visually
      let start = 1;
      list.innerHTML = groups
        .map((names) => {
          const ol =
            `<ol start=${start}>` +
            names.map((n) => `<li>${escapeHtml(n)}</li>`).join("") +
            "</ol>";
          start += names.length;
          return ol;
        })
        .join('<hr class="counter-group-sep">');
    } catch (err) {
      const list = root.querySelector(".counter-names-list");
      if (list) list.textContent = `Error: ${err.message}`;
    }
  }

  function initImageResizers() {
    bindImageResize(
      "registry-file",
      "registry-image-data",
      null,
      null,
      null,
      null,
      null,
      null,
      false,
    );
    bindImageResize(
      "ceremony-file",
      "ceremony-image-data",
      "ceremony-preview",
      "image/webp",
      0.9,
      null,
      "ceremony-image-token",
      1600,
      true,
    );
    bindImageResize(
      "reception-file",
      "reception-image-data",
      "reception-preview",
      "image/webp",
      0.9,
      null,
      "reception-image-token",
      1600,
      true,
    );
    bindImageResize(
      "share-preview-file",
      "share-preview-image-data",
      "share-preview-preview",
      null,
      null,
      "share-preview-remove",
      "share-preview-image-token",
      null,
      true,
    );
    bindImageResize(
      "stamp-file",
      "stamp-image-data",
      "stamp-preview",
      null,
      null,
      "stamp-remove",
      null,
      null,
      true,
    );
    bindImageResize(
      "flora-bl-file",
      "flora-bl-image-data",
      "flora-bl-preview",
      null,
      null,
      "flora-bl-remove",
      null,
      null,
      true,
    );
    bindImageResize(
      "flora-tr-file",
      "flora-tr-image-data",
      "flora-tr-preview",
      null,
      null,
      "flora-tr-remove",
      null,
      null,
      true,
    );
    bindManagedImageResizers();
    initAppearanceColorFields();
  }

  // Native colour pickers in the Appearance section: keep the hex readout next
  // to each picker in sync while the user drags, and let the reset button
  // restore the default colour for that field.
  function initAppearanceColorFields() {
    document
      .querySelectorAll('input[type="color"][data-hex-target]')
      .forEach((input) => {
        if (input.dataset.colorBound === "true") return;
        input.dataset.colorBound = "true";
        input.addEventListener("input", () => {
          const target = document.getElementById(input.dataset.hexTarget);
          if (target) target.textContent = input.value;
        });
      });
    document.querySelectorAll("button[data-color-reset]").forEach((button) => {
      if (button.dataset.resetBound === "true") return;
      button.dataset.resetBound = "true";
      button.addEventListener("click", () => {
        const input = document.getElementById(button.dataset.colorReset);
        if (!input || !button.dataset.defaultColor) return;
        input.value = button.dataset.defaultColor;
        const target = document.getElementById(input.dataset.hexTarget);
        if (target) target.textContent = input.value;
      });
    });
  }

  function bindImageResize(
    fileId,
    dataId,
    previewId,
    format,
    quality,
    removeBtnId,
    tokenId,
    maxDim,
    passthrough,
  ) {
    const fileInput = document.getElementById(fileId);
    const dataInput = document.getElementById(dataId);
    const tokenInput = tokenId ? document.getElementById(tokenId) : null;
    const previewImg = previewId ? document.getElementById(previewId) : null;
    const removeBtn = removeBtnId ? document.getElementById(removeBtnId) : null;
    if (!fileInput || !dataInput || fileInput.dataset.resizeBound === "true")
      return;
    fileInput.dataset.resizeBound = "true";

    const clearForRemove = () => {
      dataInput.value = "";
      // dropping the media id too is what actually restores the default: the
      // server deletes the stored media row when both fields arrive empty.
      const mediaIdInput = document.getElementById(
        fileId.replace(/-file$/, "-media-id"),
      );
      if (mediaIdInput) mediaIdInput.value = "";
      if (tokenInput) tokenInput.value = "";
      if (previewImg) {
        clearPreviewImage(previewImg);
      }
      if (removeBtn) removeBtn.style.display = "none";
      fileInput.value = "";
    };

    fileInput.addEventListener("change", () => {
      const file = fileInput.files?.[0];
      if (!file) return;

      // skip canvas re-encode when source already fits server cap — preserves original quality
      const serverCapBytes = 5 * 1024 * 1024;
      if (
        passthrough &&
        file.type &&
        file.type.indexOf("image/") === 0 &&
        file.size <= serverCapBytes
      ) {
        const reader = new FileReader();
        reader.onload = () => {
          dataInput.value = reader.result;
          if (tokenInput) tokenInput.value = "";
          if (previewImg) {
            setPreviewImage(previewImg, reader.result);
          }
          if (removeBtn) removeBtn.style.display = "";
        };
        reader.readAsDataURL(file);
        return;
      }

      const img = new Image();
      img.onload = () => {
        const max = maxDim || 400;
        let w = img.width;
        let h = img.height;
        if (w > max || h > max) {
          const ratio = Math.min(max / w, max / h);
          w = Math.round(w * ratio);
          h = Math.round(h * ratio);
        }
        const canvas = document.createElement("canvas");
        canvas.width = w;
        canvas.height = h;
        canvas.getContext("2d").drawImage(img, 0, 0, w, h);
        dataInput.value = canvas.toDataURL(format || "image/png", quality);
        if (tokenInput) tokenInput.value = "";
        if (previewImg) {
          setPreviewImage(previewImg, dataInput.value);
        }
        if (removeBtn) removeBtn.style.display = "";
      };
      img.src = URL.createObjectURL(file);
    });

    if (removeBtn && removeBtn.dataset.bound !== "true") {
      removeBtn.dataset.bound = "true";
      removeBtn.addEventListener("click", clearForRemove);
    }
  }

  function bindManagedImageResizers(root) {
    const scope = root instanceof Element ? root : document;
    scope.querySelectorAll(".managed-image-file").forEach((fileInput) => {
      if (fileInput.dataset.resizeBound === "true") return;
      fileInput.dataset.resizeBound = "true";
      fileInput.addEventListener("change", () => {
        const file = fileInput.files?.[0];
        if (!file) return;
        const targetInput = document.getElementById(
          fileInput.dataset.targetInput || "",
        );
        const tokenInput = document.getElementById(
          fileInput.dataset.mediaIdInput || "",
        );
        const previewImg = document.getElementById(
          fileInput.dataset.previewTarget || "",
        );
        if (!targetInput) return;

        const img = new Image();
        img.onload = () => {
          let w = img.width;
          let h = img.height;
          const format = fileInput.dataset.format || "image/png";
          const quality = parseFloat(fileInput.dataset.quality || "0.92");
          const maxBytes = parseInt(fileInput.dataset.maxBytes || "0", 10);
          const maxWidth = parseInt(fileInput.dataset.maxWidth || "0", 10);
          const maxHeight = parseInt(fileInput.dataset.maxHeight || "0", 10);
          if (maxWidth > 0 || maxHeight > 0) {
            const widthRatio = maxWidth > 0 ? maxWidth / w : 1;
            const heightRatio = maxHeight > 0 ? maxHeight / h : 1;
            const ratio = Math.min(widthRatio, heightRatio, 1);
            w = Math.max(1, Math.round(w * ratio));
            h = Math.max(1, Math.round(h * ratio));
          }
          const canvas = document.createElement("canvas");
          canvas.width = w;
          canvas.height = h;
          canvas.getContext("2d").drawImage(img, 0, 0, w, h);
          targetInput.value = encodeManagedImage(
            canvas,
            format,
            quality,
            maxBytes,
          );
          if (tokenInput) tokenInput.value = "";
          if (previewImg) {
            setPreviewImage(previewImg, targetInput.value);
          }
        };
        img.src = URL.createObjectURL(file);
      });
    });
  }

  function setPreviewImage(previewImg, src) {
    previewImg.removeAttribute("data-src");
    previewImg.classList.remove("dashboard-lazy-image");
    previewImg.src = src;
    previewImg.style.display = "";
  }

  function clearPreviewImage(previewImg) {
    previewImg.removeAttribute("data-src");
    previewImg.classList.remove("dashboard-lazy-image");
    previewImg.removeAttribute("src");
    previewImg.style.display = "none";
  }

  function escapeHtml(str) {
    const div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function encodeManagedImage(canvas, format, quality, maxBytes) {
    const mimeType = format || "image/png";
    const currentQuality = normalizeQuality(quality);
    const supportsQuality =
      mimeType === "image/jpeg" || mimeType === "image/webp";
    let dataUrl = canvas.toDataURL(mimeType, currentQuality);
    if (!maxBytes || estimateDataURLBytes(dataUrl) <= maxBytes) {
      return dataUrl;
    }

    const workCanvas = document.createElement("canvas");
    const workCtx = workCanvas.getContext("2d");
    let currentWidth = canvas.width;
    let currentHeight = canvas.height;

    while (
      estimateDataURLBytes(dataUrl) > maxBytes &&
      currentWidth > 80 &&
      currentHeight > 80
    ) {
      if (supportsQuality) {
        for (
          let nextQuality = currentQuality - 0.07;
          nextQuality >= 0.45;
          nextQuality -= 0.07
        ) {
          dataUrl = renderManagedImage(
            canvas,
            workCanvas,
            workCtx,
            currentWidth,
            currentHeight,
            mimeType,
            nextQuality,
          );
          if (estimateDataURLBytes(dataUrl) <= maxBytes) {
            return dataUrl;
          }
        }
      }

      currentWidth = Math.max(80, Math.round(currentWidth * 0.85));
      currentHeight = Math.max(80, Math.round(currentHeight * 0.85));
      dataUrl = renderManagedImage(
        canvas,
        workCanvas,
        workCtx,
        currentWidth,
        currentHeight,
        mimeType,
        currentQuality,
      );
    }

    return dataUrl;
  }

  function renderManagedImage(
    sourceCanvas,
    targetCanvas,
    targetContext,
    width,
    height,
    format,
    quality,
  ) {
    targetCanvas.width = width;
    targetCanvas.height = height;
    targetContext.clearRect(0, 0, width, height);
    targetContext.drawImage(sourceCanvas, 0, 0, width, height);
    return targetCanvas.toDataURL(format || "image/png", quality);
  }

  function normalizeQuality(quality) {
    const parsed =
      typeof quality === "number" ? quality : parseFloat(quality || "0.92");
    if (!Number.isFinite(parsed)) return 0.92;
    return Math.min(Math.max(parsed, 0.1), 0.92);
  }

  function estimateDataURLBytes(dataUrl) {
    const idx = dataUrl.indexOf(",");
    if (idx === -1) return 0;
    const base64 = dataUrl.slice(idx + 1);
    let padding = 0;
    if (base64.endsWith("==")) padding = 2;
    else if (base64.endsWith("=")) padding = 1;
    return Math.floor((base64.length * 3) / 4) - padding;
  }

  window.dashboardSubmitForm = (form) => {
    submitForm(form).catch((err) => {
      console.error(err);
      window.alert("Dashboard request failed.");
    });
  };
  window.initDashboardImageResizers = bindManagedImageResizers;

  init();
})();
