(function () {
  "use strict";

  var chatPanel = document.getElementById("chat-panel");
  if (!chatPanel) return;

  var MSG_ERROR = chatPanel.dataset.msgError;
  var MSG_RATE  = chatPanel.dataset.msgRate;
  var HISTORY_KEY = "foedus_chat_history";
  var MAX_HISTORY = 20;
  var bubble   = document.getElementById("chat-bubble");
  var closeBtn = document.getElementById("chat-close");
  var form     = document.getElementById("chat-form");
  var input    = document.getElementById("chat-input");
  var msgs     = document.getElementById("chat-messages");

  var history = [];
  var replaying = true;
  try { history = JSON.parse(localStorage.getItem(HISTORY_KEY) || "[]"); } catch(e) {}
  history.forEach(function(m) {
    if (m.role === "assistant") {
      var tempBubble = appendBubble("assistant", "", false);
      finalize(m.content, tempBubble, false);
    } else {
      appendBubble(m.role, m.content, false);
    }
  });
  replaying = false;

  bubble.addEventListener("click", function() {
    if (chatPanel.style.display === "none") {
      openPanel();
    } else {
      closePanel();
    }
  });
  closeBtn.addEventListener("click", closePanel);

  function openPanel() {
    chatPanel.style.display = "";
    chatPanel.classList.remove("chat-panel--closing");
    chatPanel.classList.add("chat-panel--opening");
    bubble.classList.add("chat-bubble--covered");
    input.focus();
    msgs.scrollTop = msgs.scrollHeight;
  }

  function closePanel() {
    chatPanel.classList.remove("chat-panel--opening");
    chatPanel.classList.add("chat-panel--closing");
    chatPanel.addEventListener("animationend", function hide() {
      chatPanel.style.display = "none";
      chatPanel.classList.remove("chat-panel--closing");
      bubble.classList.remove("chat-bubble--covered");
      chatPanel.removeEventListener("animationend", hide);
    });
  }

  form.addEventListener("submit", function(e) {
    e.preventDefault();
    var text = input.value.trim();
    if (!text || text.length > 500) return;
    input.value = "";
    appendBubble("user", text, false);
    addMessage("user", text);

    var assistantBubble = appendBubble("assistant", "", true);
    var full = "";

    fetch("/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message: text, history: history.slice(-20) })
    }).then(function(res) {
      if (res.status === 429) { finalize(MSG_RATE, assistantBubble, true); return; }
      if (!res.ok) { finalize(MSG_ERROR, assistantBubble, true); return; }
      var reader = res.body.getReader();
      var decoder = new TextDecoder();
      var buf = "";
      function pump() {
        reader.read().then(function(chunk) {
          if (chunk.done) { finalize(full, assistantBubble, false); return; }
          buf += decoder.decode(chunk.value, { stream: true });
          var lines = buf.split("\n");
          buf = lines.pop();
          lines.forEach(function(line) {
            if (!line.startsWith("data:")) return;
            var raw = line.slice(5).trim();
            if (raw === "[DONE]") return;
            try {
              var obj = JSON.parse(raw);
              var delta = (obj.choices && obj.choices[0] && obj.choices[0].delta && obj.choices[0].delta.content) || "";
              full += delta;
              assistantBubble.textContent = full;
              msgs.scrollTop = msgs.scrollHeight;
            } catch(e) {}
          });
          pump();
        }).catch(function() { finalize(MSG_ERROR, assistantBubble, true); });
      }
      pump();
    }).catch(function() { finalize(MSG_ERROR, assistantBubble, true); });
  });

  function finalize(text, bubbleEl, isError) {
    if (isError) {
      bubbleEl.textContent = text;
      bubbleEl.classList.remove("chat-bubble--streaming");
      return;
    }
    // extract trailing signature like "— Anna" or "- Anna"
    var match = text.match(/[\u2014\-]\s*(\S.+?)\s*$/);
    var label = null;
    var display = text;
    if (match) {
      var candidate = match[1].trim();
      if (candidate.length > 0 && candidate.length < 40) {
        label = capitalizePersonaLabel(candidate);
        display = text.slice(0, match.index).trim();
      }
    }
    bubbleEl.textContent = display;
    bubbleEl.classList.remove("chat-bubble--streaming");
    if (label) {
      var span = document.createElement("span");
      span.className = "chat-persona-label";
      span.textContent = "\u2014 " + label;
      bubbleEl.parentNode.appendChild(span);
    }
    addMessage("assistant", text);
  }

  function addMessage(role, content) {
    if (replaying) return;
    history.push({ role: role, content: content });
    if (history.length > MAX_HISTORY) history = history.slice(-MAX_HISTORY);
    persistHistory();
  }

  function appendBubble(role, text, streaming) {
    var wrap = document.createElement("div");
    wrap.className = "chat-msg chat-msg--" + role;
    var p = document.createElement("p");
    p.className = "chat-bubble";
    p.textContent = text;
    if (streaming) p.classList.add("chat-bubble--streaming");
    wrap.appendChild(p);
    msgs.appendChild(wrap);
    msgs.scrollTop = msgs.scrollHeight;
    return p;
  }

  function capitalizePersonaLabel(label) {
    if (!label) return label;
    return label.replace(/(^|[\s-])([A-Za-zÀ-ÖØ-öø-ÿ])/g, function(_, prefix, letter) {
      return prefix + letter.toUpperCase();
    });
  }

  function persistHistory() {
    try { localStorage.setItem(HISTORY_KEY, JSON.stringify(history)); } catch(e) {}
  }
})();
