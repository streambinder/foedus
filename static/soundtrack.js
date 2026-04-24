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

  // move results to body so it escapes overflow:hidden ancestors
  document.body.appendChild(results);
  function positionResults() {
    var rect = input.getBoundingClientRect();
    results.style.top = (rect.bottom + 4) + "px";
    results.style.bottom = "auto";
    results.style.left = rect.left + "px";
    results.style.width = rect.width + "px";
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

  function search(query) {
    fetch("/soundtrack/search?q=" + encodeURIComponent(query))
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
              '" alt="" loading="lazy"/>';
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
