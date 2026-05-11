(() => {
  const selectedGuestIds = new Set();
  let guestSearchTimer = null;
  let invitationSearchValue = "";
  let invitationLabelValue = "";
  let invitationLabelUserEdited = false;
  const guestFirstNames = {};
  let guestSearchSeq = 0;
  const SECTION_IDS = [
    "dashboard-flash",
    "dashboard-counters",
    "dashboard-settings",
    "dashboard-guests",
    "dashboard-invitations",
    "dashboard-registry",
    "dashboard-soundtrack-events",
  ];

  function init() {
    bindGlobalListeners();
    initDashboardFeatures();
    initImageResizers();
    syncGuestSelectionUI();
    applyInvitationFilter();
  }

  function bindGlobalListeners() {
    if (document.body.dataset.dashboardSpaBound === "true") return;
    document.body.dataset.dashboardSpaBound = "true";

    document.addEventListener("click", handleClick);
    document.addEventListener("submit", handleSubmit);
    document.addEventListener("change", handleChange);
    document.addEventListener("input", handleInput);
    window.addEventListener("popstate", () => {
      refreshSections(window.location.href, [
        "dashboard-guests",
        "dashboard-invitations",
      ]);
    });
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

    const paginationLink = event.target.closest(
      '#dashboard-guests .pagination a[href^="/dashboard"]',
    );
    if (paginationLink) {
      event.preventDefault();
      history.pushState({}, "", paginationLink.href);
      refreshSections(paginationLink.href, ["dashboard-guests"]);
      return;
    }

    const invitePaginationLink = event.target.closest(
      '#dashboard-invitations .pagination a[href^="/dashboard"]',
    );
    if (invitePaginationLink) {
      event.preventDefault();
      history.pushState({}, "", invitePaginationLink.href);
      refreshSections(invitePaginationLink.href, ["dashboard-invitations"]);
      return;
    }

    const modalClose = event.target.closest("[data-dashboard-modal-close]");
    if (modalClose) {
      event.preventDefault();
      closeDashboardModal();
    }
  }

  function handleChange(event) {
    if (!event.target.classList.contains("guest-checkbox")) return;
    const id = event.target.value;
    if (event.target.checked) {
      selectedGuestIds.add(id);
    } else {
      selectedGuestIds.delete(id);
    }
    syncGuestSelectionUI();
  }

  function handleInput(event) {
    if (event.target.id === "invitation-label-input") {
      invitationLabelValue = event.target.value;
      invitationLabelUserEdited = event.target.value.trim() !== "";
      event.target.dataset.userEdited = invitationLabelUserEdited ? "1" : "";
      return;
    }
    if (event.target.id === "guest-search") {
      clearTimeout(guestSearchTimer);
      guestSearchTimer = setTimeout(() => {
        const url = new URL(window.location.href);
        const value = event.target.value.trim();
        if (value) {
          url.searchParams.set("q", value);
        } else {
          url.searchParams.delete("q");
        }
        url.searchParams.delete("page");
        history.replaceState({}, "", url);
        const seq = ++guestSearchSeq;
        refreshSections(
          url.toString(),
          ["dashboard-guests"],
          () => seq === guestSearchSeq,
        );
      }, 350);
      return;
    }

    if (event.target.id === "invitation-search") {
      invitationSearchValue = event.target.value;
      applyInvitationFilter();
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
    if (form.id === "invitation-form") {
      const hidden = form.querySelector("#guest-ids-input");
      if (hidden) hidden.value = Array.from(selectedGuestIds).join(",");
      if (selectedGuestIds.size === 0) return;
    }

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
      selectedGuestIds.clear();
      invitationLabelValue = "";
      invitationLabelUserEdited = false;
      const labelInput = form.querySelector("#invitation-label-input");
      if (labelInput) {
        labelInput.value = "";
        labelInput.dataset.userEdited = "";
      }
      const code = response.headers.get("X-Invitation-Code");
      if (code && typeof window.copyInvitationURL === "function") {
        window.copyInvitationURL(code);
      }
    }

    if (isGuestDeleteForm(form)) {
      selectedGuestIds.delete(extractTrailingID(action));
    }

    await refreshSections(
      window.location.href,
      getRefreshSections(action, form),
    );
    closeDashboardModal();
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

  function getRefreshSections(action) {
    if (
      /\/dashboard\/guests\/\d+\/confirm\/(ceremony|reception)$/.test(action)
    ) {
      return ["dashboard-counters", "dashboard-guests"];
    }
    if (
      /\/dashboard\/guests(\/import)?$/.test(action) ||
      /\/dashboard\/guests\/\d+$/.test(action) ||
      /\/dashboard\/guests\/\d+\/delete$/.test(action)
    ) {
      return [
        "dashboard-flash",
        "dashboard-counters",
        "dashboard-guests",
        "dashboard-invitations",
      ];
    }
    if (
      /\/dashboard\/invitations$/.test(action) ||
      /\/dashboard\/invitations\/\d+\/delete$/.test(action) ||
      /\/dashboard\/invitations\/\d+\/viewed\/reset$/.test(action)
    ) {
      return ["dashboard-flash", "dashboard-guests", "dashboard-invitations"];
    }
    if (/\/dashboard\/polls(\/\d+\/delete)?$/.test(action)) {
      return ["dashboard-flash", "dashboard-invitations"];
    }
    if (/\/dashboard\/gifts\/\d+(\/delete)?$/.test(action)) {
      return ["dashboard-flash", "dashboard-registry"];
    }
    if (
      /\/dashboard\/registry(?:\/\d+(?:\/(?:move\/(?:up|down)|delete))?)?$/.test(
        action,
      )
    ) {
      return ["dashboard-flash", "dashboard-registry"];
    }
    if (/\/dashboard\/soundtrack\/\d+\/delete$/.test(action)) {
      return ["dashboard-flash", "dashboard-soundtrack-events"];
    }
    if (/\/dashboard\/settings$/.test(action)) {
      return [
        "dashboard-flash",
        "dashboard-counters",
        "dashboard-settings",
        "dashboard-invitations",
        "dashboard-registry",
      ];
    }
    return SECTION_IDS;
  }

  function isGuestDeleteForm(form) {
    const action = form.getAttribute("action") || "";
    return /\/dashboard\/guests\/\d+\/delete$/.test(action);
  }

  function extractTrailingID(action) {
    const match = action.match(/\/(\d+)(?:\/[^/]+)?$/);
    return match ? match[1] : "";
  }

  async function refreshSections(url, sectionIds, isStillCurrent) {
    const openDetails = getOpenAccordionKeys();
    const doc = await fetchDocument(url);
    if (typeof isStillCurrent === "function" && !isStillCurrent()) return;
    sectionIds.forEach((id) => {
      replaceSection(id, doc);
    });
    restoreOpenAccordions(openDetails);
    initDashboardFeatures();
    initImageResizers();
    syncGuestSelectionUI();
    applyInvitationFilter();
    if (sectionIds.indexOf("dashboard-guests") !== -1) {
      focusGuestSearchIfPresent();
    }
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

  function replaceSection(id, doc) {
    const current = document.getElementById(id);
    const next = doc.getElementById(id);
    if (current && next) {
      current.replaceWith(next);
    }
  }

  function syncGuestSelectionUI() {
    const checkboxes = document.querySelectorAll(".guest-checkbox");
    checkboxes.forEach((checkbox) => {
      checkbox.checked = selectedGuestIds.has(checkbox.value);
    });

    const hidden = document.getElementById("guest-ids-input");
    if (hidden) {
      hidden.value = Array.from(selectedGuestIds).join(",");
    }

    const button = document.getElementById("create-invitation-btn");
    if (button) {
      button.disabled = selectedGuestIds.size === 0;
    }

    syncInvitationLabelDefault();
  }

  function syncInvitationLabelDefault() {
    const input = document.getElementById("invitation-label-input");
    if (!input) return;

    // cache first names from currently rendered guests so filtered-out selections keep their names
    document.querySelectorAll(".guest-checkbox").forEach((checkbox) => {
      const name = (checkbox.dataset.firstName || "").trim();
      if (name) guestFirstNames[checkbox.value] = name;
    });

    // restore user-typed label across guest section refreshes (search/pagination)
    if (invitationLabelUserEdited) {
      input.dataset.userEdited = "1";
      if (input.value !== invitationLabelValue) {
        input.value = invitationLabelValue;
      }
      return;
    }
    if (input.dataset.userEdited === "1") return;

    const firstNames = [];
    selectedGuestIds.forEach((id) => {
      const name = guestFirstNames[id];
      if (name) firstNames.push(name);
    });
    input.value = composeDefaultInvitationLabel(firstNames);
    invitationLabelValue = input.value;
  }

  function composeDefaultInvitationLabel(firstNames) {
    if (firstNames.length === 0) return "";
    if (firstNames.length === 1) return firstNames[0];
    if (firstNames.length === 2) return `${firstNames[0]} & ${firstNames[1]}`;
    return `${firstNames[0]} + ${firstNames.length - 1}`;
  }

  function focusGuestSearchIfPresent() {
    const input = document.getElementById("guest-search");
    if (!input) return;
    input.focus();
    const end = input.value.length;
    input.setSelectionRange(end, end);
  }

  function applyInvitationFilter() {
    const input = document.getElementById("invitation-search");
    if (!input) return;

    if (input.value !== invitationSearchValue) {
      input.value = invitationSearchValue;
    }

    const query = invitationSearchValue.trim().toLowerCase();
    const rows = document.querySelectorAll(
      "#dashboard-invitations tbody tr[data-invitation-guests]",
    );
    rows.forEach((row) => {
      const guestNames = (row.dataset.invitationGuests || "").toLowerCase();
      row.hidden = query !== "" && guestNames.indexOf(query) === -1;
    });
  }

  function getOpenAccordionKeys() {
    return Array.from(
      document.querySelectorAll(".accordion-collapse.show[data-dashboard-key]"),
    ).map((el) => el.dataset.dashboardKey);
  }

  function restoreOpenAccordions(keys) {
    keys.forEach((key) => {
      const panel = document.querySelector(
        `.accordion-collapse[data-dashboard-key="${key}"]`,
      );
      if (!panel) return;
      panel.classList.add("show");
      const button = document.querySelector(`[data-bs-target="#${panel.id}"]`);
      if (!button) return;
      button.classList.remove("collapsed");
      button.setAttribute("aria-expanded", "true");
    });
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
  }

  function closeDashboardModal() {
    const root = document.getElementById("dashboard-modal-root");
    if (root) root.innerHTML = "";
  }

  function initImageResizers() {
    bindImageResize(
      "registry-file",
      "registry-image-data",
      null,
      null,
      null,
      false,
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
      false,
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
      false,
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
      true,
      "share-preview-image-token",
      null,
      true,
    );
    bindManagedImageResizers();
  }

  function bindImageResize(
    fileId,
    dataId,
    previewId,
    format,
    quality,
    withRemove,
    tokenId,
    maxDim,
    passthrough,
  ) {
    const fileInput = document.getElementById(fileId);
    const dataInput = document.getElementById(dataId);
    const tokenInput = tokenId ? document.getElementById(tokenId) : null;
    const previewImg = previewId ? document.getElementById(previewId) : null;
    if (!fileInput || !dataInput || fileInput.dataset.resizeBound === "true")
      return;
    fileInput.dataset.resizeBound = "true";

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
          if (withRemove) {
            const removeBtn = document.getElementById("share-preview-remove");
            if (removeBtn) removeBtn.style.display = "";
          }
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
        if (withRemove) {
          const removeBtn = document.getElementById("share-preview-remove");
          if (removeBtn) removeBtn.style.display = "";
        }
      };
      img.src = URL.createObjectURL(file);
    });

    if (withRemove) {
      const removeBtn = document.getElementById("share-preview-remove");
      if (removeBtn && removeBtn.dataset.bound !== "true") {
        removeBtn.dataset.bound = "true";
        removeBtn.addEventListener("click", () => {
          dataInput.value = "";
          if (tokenInput) tokenInput.value = "";
          if (previewImg) {
            clearPreviewImage(previewImg);
          }
          removeBtn.style.display = "none";
          fileInput.value = "";
        });
      }
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
