(() => {
  const container = document.querySelector(".soundtrack-search");
  if (!container) {
    return;
  }

  const MSG_ADDED = container.dataset.msgAdded;
  const MSG_ERROR = container.dataset.msgError;
  const MSG_RATE = container.dataset.msgRate;
  const input = container.querySelector(".soundtrack-search-input");
  const results = container.querySelector(".soundtrack-results");
  let debounceTimer = null;
  let searchController = null;

  // move results to body so it escapes overflow:hidden ancestors
  document.body.appendChild(results);
  function positionResults() {
    const rect = input.getBoundingClientRect();
    const viewport = window.visualViewport || null;
    const viewportTop = viewport ? viewport.offsetTop : 0;
    const viewportLeft = viewport ? viewport.offsetLeft : 0;
    const viewportWidth = viewport ? viewport.width : window.innerWidth;
    const viewportHeight = viewport ? viewport.height : window.innerHeight;
    const margin = 12;
    const gap = 4;
    const width = Math.min(
      rect.width,
      Math.max(160, viewportWidth - margin * 2),
    );
    const left = Math.max(
      viewportLeft + margin,
      Math.min(rect.left, viewportLeft + viewportWidth - width - margin),
    );
    let top = rect.bottom + gap;
    const availableBelow = viewportTop + viewportHeight - top - margin;
    const availableAbove = rect.top - viewportTop - margin;
    const maxHeight = Math.min(
      288,
      Math.max(160, Math.max(availableBelow, availableAbove)),
    );

    if (availableBelow < 160 && availableAbove > availableBelow) {
      top = Math.max(viewportTop + margin, rect.top - gap - maxHeight);
    }

    results.style.top = `${Math.round(top)}px`;
    results.style.bottom = "auto";
    results.style.left = `${Math.round(left)}px`;
    results.style.width = `${Math.round(width)}px`;
    results.style.maxHeight = `${Math.round(maxHeight)}px`;
    results.style.transform = "none";
  }

  input.addEventListener("input", () => {
    clearTimeout(debounceTimer);
    const query = input.value.trim();
    if (query.length < 2) {
      results.innerHTML = "";
      results.style.display = "none";
      return;
    }
    debounceTimer = setTimeout(() => {
      search(query);
    }, 300);
  });

  // close results on outside click, blur, or scroll
  function dismissResults() {
    results.style.display = "none";
  }

  document.addEventListener("click", (e) => {
    if (!container.contains(e.target) && !results.contains(e.target)) {
      dismissResults();
    }
  });

  input.addEventListener("focus", () => {
    if (results.children.length > 0) {
      positionResults();
      results.style.display = "block";
    }
  });

  input.addEventListener("blur", () => {
    setTimeout(() => {
      dismissResults();
    }, 150);
  });

  // reposition on viewport changes (mobile keyboard show/hide, scroll, resize)
  function repositionIfVisible() {
    if (results.style.display === "block") positionResults();
  }
  window.addEventListener("resize", repositionIfVisible, { passive: true });
  window.addEventListener("scroll", repositionIfVisible, {
    capture: true,
    passive: true,
  });
  if (window.visualViewport) {
    window.visualViewport.addEventListener("resize", repositionIfVisible, {
      passive: true,
    });
    window.visualViewport.addEventListener("scroll", repositionIfVisible, {
      passive: true,
    });
  }

  function search(query) {
    if (searchController) {
      searchController.abort();
    }
    searchController = window.AbortController ? new AbortController() : null;
    fetch(
      `/soundtrack/search?q=${encodeURIComponent(query)}`,
      searchController ? { signal: searchController.signal } : undefined,
    )
      .then((res) => {
        if (!res.ok) return [];
        return res.json();
      })
      .then((tracks) => {
        results.innerHTML = "";
        if (!tracks || tracks.length === 0) {
          results.style.display = "none";
          return;
        }
        tracks.forEach((track) => {
          const item = document.createElement("div");
          item.className = "soundtrack-result-item";

          let img = "";
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

          const btn = item.querySelector(".soundtrack-add-btn");
          btn.addEventListener("click", (e) => {
            e.stopPropagation();
            addTrack(track, btn, item);
          });
          item.addEventListener("click", () => {
            addTrack(track, btn, item);
          });
          item.style.cursor = "pointer";

          results.appendChild(item);
        });
        results.style.display = "block";
        positionResults();
      })
      .catch((err) => {
        if (err && err.name === "AbortError") return;
        results.style.display = "none";
      });
  }

  function addTrack(track, btn, row) {
    let inviteID = "";
    try {
      inviteID =
        new URLSearchParams(window.location.search).get("invite") || "";
    } catch {
      // ignore
    }

    btn.disabled = true;
    btn.textContent = "...";
    fetch("/soundtrack/add", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        uri: track.uri,
        title: track.name,
        artist: track.artist,
        url: track.id ? `https://open.spotify.com/track/${track.id}` : "",
        invite_id: inviteID,
      }),
    })
      .then((res) => {
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
        const embed = document.querySelector(".soundtrack-embed");
        if (embed) {
          setTimeout(() => {
            // force iframe reload by re-setting src via attribute
            embed.setAttribute("src", embed.getAttribute("src"));
          }, 2000);
        }
        showToast(MSG_ADDED, false);
      })
      .catch(() => {
        showToast(MSG_ERROR, true);
        btn.disabled = false;
        btn.textContent = "+";
      });
  }

  function showToast(msg, isError) {
    const existing = document.querySelector(".soundtrack-toast");
    if (existing) existing.remove();

    const toast = document.createElement("div");
    toast.className = `soundtrack-toast${isError ? " soundtrack-toast-error" : ""}`;
    toast.textContent = msg;
    document.body.appendChild(toast);

    setTimeout(() => {
      toast.classList.add("soundtrack-toast-fade");
    }, 2000);
    setTimeout(() => {
      toast.remove();
    }, 2600);
  }

  function escapeHtml(str) {
    const div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return str
      .replace(/&/g, "&amp;")
      .replace(/"/g, "&quot;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }
})();
