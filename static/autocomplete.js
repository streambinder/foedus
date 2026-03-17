(function () {
  "use strict";

  function initAutocomplete(config) {
    var input = document.getElementById(config.inputId);
    var dropdown = document.getElementById(config.dropdownId);
    var nameHidden = document.getElementById(config.nameHiddenId);
    var addressHidden = document.getElementById(config.addressHiddenId);
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
      if (items.length === 0) {
        hide();
        return;
      }
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
      input.value = name + (address ? ", " + address : "");
      hide();
    }

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

    function search(query) {
      var lang = document.documentElement.lang || "en";
      var url =
        "https://nominatim.openstreetmap.org/search?q=" +
        encodeURIComponent(query) +
        "&format=jsonv2&addressdetails=1&namedetails=1&limit=5";

      fetch(url, {
        headers: { "Accept-Language": lang },
      })
        .then(function (res) { return res.json(); })
        .then(function (data) { render(data); })
        .catch(function () { hide(); });
    }

    input.addEventListener("input", function () {
      nameHidden.value = "";
      addressHidden.value = "";
      clearTimeout(debounceTimer);
      var q = input.value.trim();
      if (q.length < 3) {
        hide();
        return;
      }
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

    document.addEventListener("click", function (e) {
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
  });
  initAutocomplete({
    inputId: "reception-input",
    dropdownId: "reception-dropdown",
    nameHiddenId: "reception-address-hidden",
    addressHiddenId: "reception-location-hidden",
  });
})();
