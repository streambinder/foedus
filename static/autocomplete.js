(() => {
  function initAutocomplete(config) {
    const input = document.getElementById(config.inputId);
    const dropdown = document.getElementById(config.dropdownId);
    const nameHidden = document.getElementById(config.nameHiddenId);
    const addressHidden = document.getElementById(config.addressHiddenId);
    const cityHidden = document.getElementById(config.cityHiddenId);
    if (!input || !dropdown || !nameHidden || !addressHidden) return;

    let debounceTimer = null;
    let activeIndex = -1;
    let results = [];

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
      items.forEach((item, idx) => {
        const div = document.createElement("div");
        const name = extractName(item);
        const address = extractAddress(item, name);
        div.innerHTML =
          '<span style="font-weight:500;color:var(--charcoal-dark)">' +
          escapeHtml(name) +
          "</span>" +
          (address
            ? '<br><span style="font-size:var(--fs-xs);color:var(--sage-dark)">' +
              escapeHtml(address) +
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

    function extractCity(item) {
      if (!item.address) return "";
      return (
        item.address.city ||
        item.address.town ||
        item.address.village ||
        item.address.municipality ||
        ""
      );
    }

    function select(idx) {
      const item = results[idx];
      if (!item) return;
      const name = extractName(item);
      const address = extractAddress(item, name);
      nameHidden.value = name;
      addressHidden.value = address;
      if (cityHidden) cityHidden.value = extractCity(item);
      input.value = name + (address ? `, ${address}` : "");
      hide();
    }

    function extractName(item) {
      if (item.namedetails?.name) return item.namedetails.name;
      return item.display_name;
    }

    function extractAddress(item, name) {
      if (!item.display_name) return "";
      const dn = item.display_name;
      if (name && dn.indexOf(name) === 0) {
        const rest = dn.substring(name.length).replace(/^,\s*/, "");
        return rest;
      }
      return dn;
    }

    function escapeHtml(str) {
      const div = document.createElement("div");
      div.appendChild(document.createTextNode(str));
      return div.innerHTML;
    }

    function search(query) {
      const lang = document.documentElement.lang || "en";
      const url =
        "https://nominatim.openstreetmap.org/search?q=" +
        encodeURIComponent(query) +
        "&format=jsonv2&addressdetails=1&namedetails=1&limit=5";

      fetch(url, {
        headers: { "Accept-Language": lang },
      })
        .then((res) => res.json())
        .then((data) => {
          render(data);
        })
        .catch(() => {
          hide();
        });
    }

    input.addEventListener("input", () => {
      nameHidden.value = "";
      addressHidden.value = "";
      if (cityHidden) cityHidden.value = "";
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
  });
  initAutocomplete({
    inputId: "reception-input",
    dropdownId: "reception-dropdown",
    nameHiddenId: "reception-address-hidden",
    addressHiddenId: "reception-location-hidden",
    cityHiddenId: "reception-city-hidden",
  });
})();
