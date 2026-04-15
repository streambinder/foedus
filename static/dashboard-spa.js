(function () {
  "use strict";

  var selectedGuestIds = new Set();
  var guestSearchTimer = null;
  var SECTION_IDS = [
    "dashboard-flash",
    "dashboard-counters",
    "dashboard-settings",
    "dashboard-guests",
    "dashboard-invitations",
    "dashboard-registry"
  ];

  function init() {
    bindGlobalListeners();
    initDashboardFeatures();
    initImageResizers();
    syncGuestSelectionUI();
  }

  function bindGlobalListeners() {
    if (document.body.dataset.dashboardSpaBound === "true") return;
    document.body.dataset.dashboardSpaBound = "true";

    document.addEventListener("click", handleClick);
    document.addEventListener("submit", handleSubmit);
    document.addEventListener("change", handleChange);
    document.addEventListener("input", handleInput);
    window.addEventListener("popstate", function () {
      refreshSections(window.location.href, ["dashboard-guests"]);
    });
  }

  function handleClick(event) {
    var editLink = event.target.closest('.actions a[href^="/dashboard/"][href$="/edit"]');
    if (editLink) {
      event.preventDefault();
      openDashboardEditModal(editLink.href);
      return;
    }

    var paginationLink = event.target.closest('#dashboard-guests .pagination a[href^="/dashboard"]');
    if (paginationLink) {
      event.preventDefault();
      history.pushState({}, "", paginationLink.href);
      refreshSections(paginationLink.href, ["dashboard-guests"]);
      return;
    }

    var modalClose = event.target.closest("[data-dashboard-modal-close]");
    if (modalClose) {
      event.preventDefault();
      closeDashboardModal();
    }
  }

  function handleChange(event) {
    if (!event.target.classList.contains("guest-checkbox")) return;
    var id = event.target.value;
    if (event.target.checked) {
      selectedGuestIds.add(id);
    } else {
      selectedGuestIds.delete(id);
    }
    syncGuestSelectionUI();
  }

  function handleInput(event) {
    if (event.target.id !== "guest-search") return;
    clearTimeout(guestSearchTimer);
    guestSearchTimer = setTimeout(function () {
      var url = new URL(window.location.href);
      var value = event.target.value.trim();
      if (value) {
        url.searchParams.set("q", value);
      } else {
        url.searchParams.delete("q");
      }
      url.searchParams.delete("page");
      history.replaceState({}, "", url);
      refreshSections(url.toString(), ["dashboard-guests"]);
    }, 250);
  }

  function handleSubmit(event) {
    var form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    if (event.defaultPrevented) return;
    if (!isDashboardForm(form)) return;

    event.preventDefault();
    submitForm(form).catch(function (err) {
      console.error(err);
      window.alert("Dashboard request failed.");
    });
  }

  function isDashboardForm(form) {
    var action = form.getAttribute("action") || "";
    return action.indexOf("/dashboard") === 0;
  }

  async function submitForm(form) {
    if (form.id === "invitation-form") {
      var hidden = form.querySelector("#guest-ids-input");
      if (hidden) hidden.value = Array.from(selectedGuestIds).join(",");
      if (selectedGuestIds.size === 0) return;
    }

    var action = form.getAttribute("action") || window.location.pathname;
    var method = (form.getAttribute("method") || "GET").toUpperCase();
    var formData = new FormData(form);
    var csrfToken = getCsrfToken(form);
    var response = await fetch(action, {
      method: method,
      body: method === "GET" ? null : formData,
      credentials: "same-origin",
      headers: buildRequestHeaders(csrfToken)
    });
    if (!response.ok) {
      throw new Error("Request failed: " + response.status);
    }

    if (form.id === "invitation-form") {
      selectedGuestIds.clear();
    }

    if (isGuestDeleteForm(form)) {
      selectedGuestIds.delete(extractTrailingID(action));
    }

    await refreshSections(window.location.href, getRefreshSections(action, form));
    closeDashboardModal();
  }

  function getCsrfToken(form) {
    var input = form.querySelector('input[name="_csrf"]');
    return input ? input.value : "";
  }

  function buildRequestHeaders(csrfToken) {
    var headers = { "X-Requested-With": "fetch" };
    if (csrfToken) {
      headers["X-Csrf-Token"] = csrfToken;
    }
    return headers;
  }

  function getRefreshSections(action, form) {
    if (/\/dashboard\/guests\/\d+\/confirm\/(ceremony|reception)$/.test(action)) {
      return ["dashboard-counters", "dashboard-guests"];
    }
    if (/\/dashboard\/guests(\/import)?$/.test(action) || /\/dashboard\/guests\/\d+$/.test(action) || /\/dashboard\/guests\/\d+\/delete$/.test(action)) {
      return ["dashboard-flash", "dashboard-counters", "dashboard-guests", "dashboard-invitations"];
    }
    if (/\/dashboard\/invitations$/.test(action) || /\/dashboard\/invitations\/\d+\/delete$/.test(action)) {
      return ["dashboard-flash", "dashboard-guests", "dashboard-invitations"];
    }
    if (/\/dashboard\/polls(\/\d+\/delete)?$/.test(action)) {
      return ["dashboard-flash", "dashboard-invitations"];
    }
    if (/\/dashboard\/gifts\/\d+(\/delete)?$/.test(action)) {
      return ["dashboard-flash", "dashboard-registry"];
    }
    if (/\/dashboard\/registry(?:\/\d+(?:\/(?:move\/(?:up|down)|delete))?)?$/.test(action)) {
      return ["dashboard-flash", "dashboard-registry"];
    }
    if (/\/dashboard\/settings$/.test(action)) {
      return ["dashboard-flash", "dashboard-counters", "dashboard-settings", "dashboard-invitations", "dashboard-registry"];
    }
    return SECTION_IDS;
  }

  function isGuestDeleteForm(form) {
    var action = form.getAttribute("action") || "";
    return /\/dashboard\/guests\/\d+\/delete$/.test(action);
  }

  function extractTrailingID(action) {
    var match = action.match(/\/(\d+)(?:\/[^/]+)?$/);
    return match ? match[1] : "";
  }

  async function refreshSections(url, sectionIds) {
    var openDetails = getOpenAccordionKeys();
    var doc = await fetchDocument(url);
    sectionIds.forEach(function (id) {
      replaceSection(id, doc);
    });
    restoreOpenAccordions(openDetails);
    initDashboardFeatures();
    initImageResizers();
    syncGuestSelectionUI();
    if (sectionIds.indexOf("dashboard-guests") !== -1) {
      focusGuestSearchIfPresent();
    }
  }

  async function fetchDocument(url) {
    var response = await fetch(url, {
      credentials: "same-origin",
      headers: { "X-Requested-With": "fetch" }
    });
    if (!response.ok) {
      throw new Error("Request failed: " + response.status);
    }
    var html = await response.text();
    return new DOMParser().parseFromString(html, "text/html");
  }

  function replaceSection(id, doc) {
    var current = document.getElementById(id);
    var next = doc.getElementById(id);
    if (current && next) {
      current.replaceWith(next);
    }
  }

  function syncGuestSelectionUI() {
    var checkboxes = document.querySelectorAll(".guest-checkbox");
    checkboxes.forEach(function (checkbox) {
      checkbox.checked = selectedGuestIds.has(checkbox.value);
    });

    var hidden = document.getElementById("guest-ids-input");
    if (hidden) {
      hidden.value = Array.from(selectedGuestIds).join(",");
    }

    var button = document.getElementById("create-invitation-btn");
    if (button) {
      button.disabled = selectedGuestIds.size === 0;
    }
  }

  function focusGuestSearchIfPresent() {
    var input = document.getElementById("guest-search");
    if (!input) return;
    input.focus();
    var end = input.value.length;
    input.setSelectionRange(end, end);
  }

  function getOpenAccordionKeys() {
    return Array.from(document.querySelectorAll(".accordion-collapse.show[data-dashboard-key]")).map(function (el) {
      return el.dataset.dashboardKey;
    });
  }

  function restoreOpenAccordions(keys) {
    keys.forEach(function (key) {
      var panel = document.querySelector('.accordion-collapse[data-dashboard-key="' + key + '"]');
      if (!panel) return;
      panel.classList.add("show");
      var button = document.querySelector('[data-bs-target="#' + panel.id + '"]');
      if (!button) return;
      button.classList.remove("collapsed");
      button.setAttribute("aria-expanded", "true");
    });
  }

  async function openDashboardEditModal(url) {
    var doc = await fetchDocument(url);
    var title = doc.querySelector("h1");
    var article = doc.querySelector("article");
    if (!article) return;

    var root = document.getElementById("dashboard-modal-root");
    if (!root) return;

    root.innerHTML =
      '<div class="modal-overlay" data-dashboard-modal-overlay>' +
        '<div class="modal-box dashboard-edit-modal-box">' +
          '<div class="dashboard-modal-header">' +
            '<h3>' + escapeHtml(title ? title.textContent.trim() : "Edit") + '</h3>' +
            '<button type="button" class="outline secondary" data-dashboard-modal-close>Close</button>' +
          '</div>' +
          article.innerHTML +
        '</div>' +
      '</div>';
    bindManagedImageResizers(root);
  }

  function closeDashboardModal() {
    var root = document.getElementById("dashboard-modal-root");
    if (root) root.innerHTML = "";
  }

  function initImageResizers() {
    bindImageResize("registry-file", "registry-image-data", null, null, null);
    bindImageResize("ceremony-file", "ceremony-image-data", "ceremony-preview", "image/jpeg", 0.7, false, "ceremony-image-token");
    bindImageResize("reception-file", "reception-image-data", "reception-preview", "image/jpeg", 0.7, false, "reception-image-token");
    bindImageResize("share-preview-file", "share-preview-image-data", "share-preview-preview", null, null, true, "share-preview-image-token");
    bindManagedImageResizers();
  }

  function bindImageResize(fileId, dataId, previewId, format, quality, withRemove, tokenId) {
    var fileInput = document.getElementById(fileId);
    var dataInput = document.getElementById(dataId);
    var tokenInput = tokenId ? document.getElementById(tokenId) : null;
    var previewImg = previewId ? document.getElementById(previewId) : null;
    if (!fileInput || !dataInput || fileInput.dataset.resizeBound === "true") return;
    fileInput.dataset.resizeBound = "true";

    fileInput.addEventListener("change", function () {
      var file = fileInput.files && fileInput.files[0];
      if (!file) return;
      var img = new Image();
      img.onload = function () {
        var max = 400;
        var w = img.width;
        var h = img.height;
        if (w > max || h > max) {
          var ratio = Math.min(max / w, max / h);
          w = Math.round(w * ratio);
          h = Math.round(h * ratio);
        }
        var canvas = document.createElement("canvas");
        canvas.width = w;
        canvas.height = h;
        canvas.getContext("2d").drawImage(img, 0, 0, w, h);
        dataInput.value = canvas.toDataURL(format || "image/png", quality);
        if (tokenInput) tokenInput.value = "";
        if (previewImg) {
          previewImg.src = dataInput.value;
          previewImg.style.display = "";
        }
        if (withRemove) {
          var removeBtn = document.getElementById("share-preview-remove");
          if (removeBtn) removeBtn.style.display = "";
        }
      };
      img.src = URL.createObjectURL(file);
    });

    if (withRemove) {
      var removeBtn = document.getElementById("share-preview-remove");
      if (removeBtn && removeBtn.dataset.bound !== "true") {
        removeBtn.dataset.bound = "true";
        removeBtn.addEventListener("click", function () {
          dataInput.value = "";
          if (tokenInput) tokenInput.value = "";
          if (previewImg) {
            previewImg.src = "";
            previewImg.style.display = "none";
          }
          removeBtn.style.display = "none";
          fileInput.value = "";
        });
      }
    }
  }

  function bindManagedImageResizers(root) {
    var scope = root instanceof Element ? root : document;
    scope.querySelectorAll(".managed-image-file").forEach(function (fileInput) {
      if (fileInput.dataset.resizeBound === "true") return;
      fileInput.dataset.resizeBound = "true";
      fileInput.addEventListener("change", function () {
        var file = fileInput.files && fileInput.files[0];
        if (!file) return;
        var targetInput = document.getElementById(fileInput.dataset.targetInput || "");
        var tokenInput = document.getElementById(fileInput.dataset.tokenInput || "");
        var previewImg = document.getElementById(fileInput.dataset.previewTarget || "");
        if (!targetInput) return;

        var img = new Image();
        img.onload = function () {
          var w = img.width;
          var h = img.height;
          var format = fileInput.dataset.format || "image/png";
          var quality = parseFloat(fileInput.dataset.quality || "0.92");
          var maxBytes = parseInt(fileInput.dataset.maxBytes || "0", 10);
          var maxWidth = parseInt(fileInput.dataset.maxWidth || "0", 10);
          var maxHeight = parseInt(fileInput.dataset.maxHeight || "0", 10);
          if (maxWidth > 0 || maxHeight > 0) {
            var widthRatio = maxWidth > 0 ? maxWidth / w : 1;
            var heightRatio = maxHeight > 0 ? maxHeight / h : 1;
            var ratio = Math.min(widthRatio, heightRatio, 1);
            w = Math.max(1, Math.round(w * ratio));
            h = Math.max(1, Math.round(h * ratio));
          }
          var canvas = document.createElement("canvas");
          canvas.width = w;
          canvas.height = h;
          canvas.getContext("2d").drawImage(img, 0, 0, w, h);
          targetInput.value = encodeManagedImage(canvas, format, quality, maxBytes);
          if (tokenInput) tokenInput.value = "";
          if (previewImg) {
            previewImg.src = targetInput.value;
            previewImg.style.display = "";
          }
        };
        img.src = URL.createObjectURL(file);
      });
    });
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function encodeManagedImage(canvas, format, quality, maxBytes) {
    var mimeType = format || "image/png";
    var currentQuality = normalizeQuality(quality);
    var supportsQuality = mimeType === "image/jpeg" || mimeType === "image/webp";
    var dataUrl = canvas.toDataURL(mimeType, currentQuality);
    if (!maxBytes || estimateDataURLBytes(dataUrl) <= maxBytes) {
      return dataUrl;
    }

    var workCanvas = document.createElement("canvas");
    var workCtx = workCanvas.getContext("2d");
    var currentWidth = canvas.width;
    var currentHeight = canvas.height;

    while (estimateDataURLBytes(dataUrl) > maxBytes && currentWidth > 80 && currentHeight > 80) {
      if (supportsQuality) {
        for (var nextQuality = currentQuality - 0.07; nextQuality >= 0.45; nextQuality -= 0.07) {
          dataUrl = renderManagedImage(canvas, workCanvas, workCtx, currentWidth, currentHeight, mimeType, nextQuality);
          if (estimateDataURLBytes(dataUrl) <= maxBytes) {
            return dataUrl;
          }
        }
      }

      currentWidth = Math.max(80, Math.round(currentWidth * 0.85));
      currentHeight = Math.max(80, Math.round(currentHeight * 0.85));
      dataUrl = renderManagedImage(canvas, workCanvas, workCtx, currentWidth, currentHeight, mimeType, currentQuality);
    }

    return dataUrl;
  }

  function renderManagedImage(sourceCanvas, targetCanvas, targetContext, width, height, format, quality) {
    targetCanvas.width = width;
    targetCanvas.height = height;
    targetContext.clearRect(0, 0, width, height);
    targetContext.drawImage(sourceCanvas, 0, 0, width, height);
    return targetCanvas.toDataURL(format || "image/png", quality);
  }

  function normalizeQuality(quality) {
    var parsed = typeof quality === "number" ? quality : parseFloat(quality || "0.92");
    if (!isFinite(parsed)) return 0.92;
    return Math.min(Math.max(parsed, 0.1), 0.92);
  }

  function estimateDataURLBytes(dataUrl) {
    var idx = dataUrl.indexOf(",");
    if (idx === -1) return 0;
    var base64 = dataUrl.slice(idx + 1);
    var padding = 0;
    if (base64.endsWith("==")) padding = 2;
    else if (base64.endsWith("=")) padding = 1;
    return Math.floor(base64.length * 3 / 4) - padding;
  }

  window.dashboardSubmitForm = function (form) {
    submitForm(form).catch(function (err) {
      console.error(err);
      window.alert("Dashboard request failed.");
    });
  };
  window.initDashboardImageResizers = bindManagedImageResizers;

  init();
})();
