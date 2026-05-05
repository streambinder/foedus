(function () {
  "use strict";

  var container = document.querySelector(".soundtrack-search");
  if (!container) { return; }

  var MSG_ADDED = container.dataset.msgAdded;
  var MSG_ERROR = container.dataset.msgError;
  var MSG_RATE = container.dataset.msgRate;
  var input = container.querySelector(".soundtrack-search-input");
  var results = container.querySelector(".soundtrack-results");
  var debounceTimer = null;
  var searchController = null;

  // move results to body so it escapes overflow:hidden ancestors
  document.body.appendChild(results);
  function positionResults() {
    var rect = input.getBoundingClientRect();
    var viewport = window.visualViewport || null;
    var viewportTop = viewport ? viewport.offsetTop : 0;
    var viewportLeft = viewport ? viewport.offsetLeft : 0;
    var viewportWidth = viewport ? viewport.width : window.innerWidth;
    var viewportHeight = viewport ? viewport.height : window.innerHeight;
    var margin = 12;
    var gap = 4;
    var width = Math.min(rect.width, Math.max(160, viewportWidth - margin * 2));
    var left = Math.max(viewportLeft + margin, Math.min(rect.left, viewportLeft + viewportWidth - width - margin));
    var top = rect.bottom + gap;
    var availableBelow = viewportTop + viewportHeight - top - margin;
    var availableAbove = rect.top - viewportTop - margin;
    var maxHeight = Math.min(288, Math.max(160, Math.max(availableBelow, availableAbove)));

    if (availableBelow < 160 && availableAbove > availableBelow) {
      top = Math.max(viewportTop + margin, rect.top - gap - maxHeight);
    }

    results.style.top = Math.round(top) + "px";
    results.style.bottom = "auto";
    results.style.left = Math.round(left) + "px";
    results.style.width = Math.round(width) + "px";
    results.style.maxHeight = Math.round(maxHeight) + "px";
    results.style.transform = "none";
  }

  input.addEventListener("input", function () {
    clearTimeout(debounceTimer);
    var query = input.value.trim();
    if (query.length < 2) {
      results.innerHTML = "";
      results.style.display = "none";
      return;
    }
    debounceTimer = setTimeout(function () {
      search(query);
    }, 300);
  });

  // close results on outside click, blur, or scroll
  function dismissResults() {
    results.style.display = "none";
  }

  document.addEventListener("click", function (e) {
    if (!container.contains(e.target) && !results.contains(e.target)) {
      dismissResults();
    }
  });

  input.addEventListener("focus", function () {
    if (results.children.length > 0) {
      positionResults();
      results.style.display = "block";
    }
  });

  input.addEventListener("blur", function () {
    setTimeout(function () {
      dismissResults();
    }, 150);
  });

  // reposition on viewport changes (mobile keyboard show/hide, scroll, resize)
  function repositionIfVisible() {
    if (results.style.display === "block") positionResults();
  }
  window.addEventListener("resize", repositionIfVisible, { passive: true });
  window.addEventListener("scroll", repositionIfVisible, { capture: true, passive: true });
  if (window.visualViewport) {
    window.visualViewport.addEventListener("resize", repositionIfVisible, { passive: true });
    window.visualViewport.addEventListener("scroll", repositionIfVisible, { passive: true });
  }

  function search(query) {
    if (searchController) {
      searchController.abort();
    }
    searchController = window.AbortController ? new AbortController() : null;
    fetch(
      "/soundtrack/search?q=" + encodeURIComponent(query),
      searchController ? { signal: searchController.signal } : undefined
    )
      .then(function (res) {
        if (!res.ok) return [];
        return res.json();
      })
      .then(function (tracks) {
        results.innerHTML = "";
        if (!tracks || tracks.length === 0) {
          results.style.display = "none";
          return;
        }
        tracks.forEach(function (track) {
          var item = document.createElement("div");
          item.className = "soundtrack-result-item";

          var img = "";
          if (track.image_url) {
            img =
              '<img class="soundtrack-result-art" src="' +
              escapeAttr(track.image_url) +
              '" alt="" width="40" height="40" loading="lazy" decoding="async" fetchpriority="low"/>';
          }

          item.innerHTML =
            img +
            '<div class="soundtrack-result-info">' +
            '<span class="soundtrack-result-name">' +
            escapeHtml(track.name) +
            "</span>" +
            '<span class="soundtrack-result-artist">' +
            escapeHtml(track.artist) +
            "</span>" +
            "</div>" +
            '<button type="button" class="soundtrack-add-btn" aria-label="Add">+</button>';

          var btn = item.querySelector(".soundtrack-add-btn");
          btn.addEventListener("click", function (e) {
            e.stopPropagation();
            addTrack(track, btn, item);
          });
          item.addEventListener("click", function () {
            addTrack(track, btn, item);
          });
          item.style.cursor = "pointer";

          results.appendChild(item);
        });
        results.style.display = "block";
        positionResults();
      })
      .catch(function (err) {
        if (err && err.name === "AbortError") return;
        results.style.display = "none";
      });
  }

  function addTrack(track, btn, row) {
    var inviteID = "";
    try {
      inviteID = new URLSearchParams(window.location.search).get("invite") || "";
    } catch (_) {}

    btn.disabled = true;
    btn.textContent = "...";
    fetch("/soundtrack/add", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        uri: track.uri,
        title: track.name,
        artist: track.artist,
        url: track.id ? ("https://open.spotify.com/track/" + track.id) : "",
        invite_id: inviteID
      }),
    })
      .then(function (res) {
        if (res.status === 429) {
          showToast(MSG_RATE, true);
          btn.disabled = false;
          btn.textContent = "+";
          return;
        }
        if (!res.ok) {
          showToast(MSG_ERROR, true);
          btn.disabled = false;
          btn.textContent = "+";
          return;
        }
        row.remove();
        if (results.children.length === 0) {
          results.style.display = "none";
        }
        var embed = document.querySelector(".soundtrack-embed");
        if (embed) {
          setTimeout(function () { embed.src = embed.src; }, 2000);
        }
        showToast(MSG_ADDED, false);
      })
      .catch(function () {
        showToast(MSG_ERROR, true);
        btn.disabled = false;
        btn.textContent = "+";
      });
  }

  function showToast(msg, isError) {
    var existing = document.querySelector(".soundtrack-toast");
    if (existing) existing.remove();

    var toast = document.createElement("div");
    toast.className = "soundtrack-toast" + (isError ? " soundtrack-toast--error" : "");
    toast.textContent = msg;
    document.body.appendChild(toast);

    setTimeout(function () {
      toast.classList.add("soundtrack-toast--fade");
    }, 2000);
    setTimeout(function () {
      toast.remove();
    }, 2600);
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return str.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }
})();
