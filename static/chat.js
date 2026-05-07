(() => {
  const chatPanel = document.getElementById("chat-panel");
  if (!chatPanel) return;

  const MSG_ERROR = chatPanel.dataset.msgError;
  const MSG_RATE = chatPanel.dataset.msgRate;
  const HISTORY_KEY = "foedus_chat_history";
  const MAX_HISTORY = 20;
  const bubble = document.getElementById("chat-bubble");
  const closeBtn = document.getElementById("chat-close");
  const form = document.getElementById("chat-form");
  const input = document.getElementById("chat-input");
  const msgs = document.getElementById("chat-messages");

  let history = [];
  let replaying = true;
  try {
    history = JSON.parse(localStorage.getItem(HISTORY_KEY) || "[]");
  } catch {
    // ignore
  }
  history.forEach((m) => {
    if (m.role === "assistant") {
      const tempBubble = appendBubble("assistant", "", false);
      finalize(m.content, tempBubble, false);
    } else {
      appendBubble(m.role, m.content, false);
    }
  });
  replaying = false;

  bubble.addEventListener("click", () => {
    if (chatPanel.style.display === "none") {
      openPanel();
    } else {
      closePanel();
    }
  });
  closeBtn.addEventListener("click", closePanel);

  document.addEventListener("click", (e) => {
    if (chatPanel.style.display === "none") return;
    if (chatPanel.classList.contains("chat-panel-closing")) return;
    if (chatPanel.contains(e.target) || bubble.contains(e.target)) return;
    closePanel();
  });

  function openPanel() {
    chatPanel.style.display = "";
    chatPanel.classList.remove("chat-panel-closing");
    chatPanel.classList.add("chat-panel-opening");
    bubble.classList.add("chat-bubble-covered");
    input.focus();
    msgs.scrollTop = msgs.scrollHeight;
  }

  function closePanel() {
    chatPanel.classList.remove("chat-panel-opening");
    chatPanel.classList.add("chat-panel-closing");
    chatPanel.addEventListener("animationend", function hide() {
      chatPanel.style.display = "none";
      chatPanel.classList.remove("chat-panel-closing");
      bubble.classList.remove("chat-bubble-covered");
      chatPanel.removeEventListener("animationend", hide);
    });
  }

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const text = input.value.trim();
    if (!text || text.length > 500) return;
    input.value = "";
    appendBubble("user", text, false);
    addMessage("user", text);

    const assistantBubble = appendBubble("assistant", "", true);
    let full = "";

    fetch("/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message: text, history: history.slice(-20) }),
    })
      .then((res) => {
        if (res.status === 429) {
          finalize(MSG_RATE, assistantBubble, true);
          return;
        }
        if (!res.ok) {
          finalize(MSG_ERROR, assistantBubble, true);
          return;
        }
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";
        function pump() {
          reader
            .read()
            .then((chunk) => {
              if (chunk.done) {
                finalize(full, assistantBubble, false);
                return;
              }
              buf += decoder.decode(chunk.value, { stream: true });
              const lines = buf.split("\n");
              buf = lines.pop();
              lines.forEach((line) => {
                if (!line.startsWith("data:")) return;
                const raw = line.slice(5).trim();
                if (raw === "[DONE]") return;
                try {
                  const obj = JSON.parse(raw);
                  const delta = obj.choices?.[0]?.delta?.content || "";
                  full += delta;
                  assistantBubble.textContent = full;
                  msgs.scrollTop = msgs.scrollHeight;
                } catch {
                  // ignore
                }
              });
              pump();
            })
            .catch(() => {
              finalize(MSG_ERROR, assistantBubble, true);
            });
        }
        pump();
      })
      .catch(() => {
        finalize(MSG_ERROR, assistantBubble, true);
      });
  });

  function finalize(text, bubbleEl, isError) {
    if (isError) {
      bubbleEl.textContent = text;
      bubbleEl.classList.remove("chat-bubble-streaming");
      return;
    }
    // extract trailing signature like "— Anna" or "- Anna"
    const match = text.match(/[\u2014-]\s*(\S.+?)\s*$/);
    let label = null;
    let display = text;
    if (match) {
      const candidate = match[1].trim();
      if (candidate.length > 0 && candidate.length < 40) {
        label = capitalizePersonaLabel(candidate);
        display = text.slice(0, match.index).trim();
      }
    }
    bubbleEl.textContent = display;
    bubbleEl.classList.remove("chat-bubble-streaming");
    if (label) {
      const span = document.createElement("span");
      span.className = "chat-persona-label";
      span.textContent = `\u2014 ${label}`;
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
    const wrap = document.createElement("div");
    wrap.className = `chat-msg chat-msg--${role}`;
    const p = document.createElement("p");
    p.className = "chat-bubble";
    p.textContent = text;
    if (streaming) p.classList.add("chat-bubble-streaming");
    wrap.appendChild(p);
    msgs.appendChild(wrap);
    msgs.scrollTop = msgs.scrollHeight;
    return p;
  }

  function capitalizePersonaLabel(label) {
    if (!label) return label;
    return label.replace(
      /(^|[\s-])([A-Za-zÀ-ÖØ-öø-ÿ])/g,
      (_, prefix, letter) => prefix + letter.toUpperCase(),
    );
  }

  function persistHistory() {
    try {
      localStorage.setItem(HISTORY_KEY, JSON.stringify(history));
    } catch {
      // ignore
    }
  }
})();
