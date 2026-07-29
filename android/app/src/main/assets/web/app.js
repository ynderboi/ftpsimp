(() => {
  const listEl = document.getElementById("list");
  const emptyEl = document.getElementById("empty");
  const crumbsEl = document.getElementById("crumbs");
  const statusEl = document.getElementById("status");
  const rootLabel = document.getElementById("root-label");
  const overlay = document.getElementById("drop-overlay");
  const fileInput = document.getElementById("file-input");
  const btnMkdir = document.getElementById("btn-mkdir");
  const btnSettings = document.getElementById("btn-settings");
  const settingsDialog = document.getElementById("settings-dialog");
  const settingsForm = document.getElementById("settings-form");
  const settingsRoot = document.getElementById("settings-root");
  const settingsError = document.getElementById("settings-error");

  let currentPath = "";

  function setRootLabel(root) {
    if (root) rootLabel.textContent = root;
  }

  async function openSettings() {
    settingsError.hidden = true;
    settingsError.textContent = "";
    try {
      const res = await fetch("/api/settings");
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      settingsRoot.value = data.root || "";
    } catch (e) {
      settingsRoot.value = rootLabel.textContent || "";
    }
    settingsDialog.showModal();
    settingsRoot.focus();
    settingsRoot.select();
  }

  btnSettings.addEventListener("click", openSettings);

  settingsForm.addEventListener("submit", async (ev) => {
    const submitter = ev.submitter;
    if (!submitter || submitter.value !== "save") return;
    ev.preventDefault();
    settingsError.hidden = true;
    const root = settingsRoot.value.trim();
    if (!root) {
      settingsError.textContent = "Укажите путь к папке";
      settingsError.hidden = false;
      return;
    }
    try {
      const res = await fetch("/api/settings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ root }),
      });
      if (!res.ok) throw new Error((await res.text()).trim() || "Ошибка сохранения");
      const data = await res.json();
      setRootLabel(data.root);
      settingsDialog.close();
      location.hash = "#/";
      currentPath = "";
      showStatus("Корневая папка изменена", "ok");
      load();
    } catch (e) {
      settingsError.textContent = String(e.message || e);
      settingsError.hidden = false;
    }
  });

  function showStatus(msg, kind = "") {
    statusEl.hidden = !msg;
    statusEl.textContent = msg || "";
    statusEl.className = "status" + (kind ? " " + kind : "");
    if (msg && kind !== "error") {
      clearTimeout(showStatus._t);
      showStatus._t = setTimeout(() => {
        statusEl.hidden = true;
      }, 3200);
    }
  }

  function formatSize(n) {
    if (n < 1024) return n + " B";
    if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
    if (n < 1024 * 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + " MB";
    return (n / (1024 * 1024 * 1024)).toFixed(2) + " GB";
  }

  function formatDate(iso) {
    try {
      const d = new Date(iso);
      return d.toLocaleString(undefined, {
        day: "2-digit",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch {
      return "";
    }
  }

  function pathParts(p) {
    if (!p) return [];
    return p.split("/").filter(Boolean);
  }

  function renderCrumbs() {
    const parts = pathParts(currentPath);
    let html =
      '<a href="#/" data-path="">Корень</a>';
    let acc = "";
    parts.forEach((part, i) => {
      acc = acc ? acc + "/" + part : part;
      html += '<span class="sep">/</span>';
      if (i === parts.length - 1) {
        html += `<span class="current">${escapeHtml(part)}</span>`;
      } else {
        html += `<a href="#/${acc}" data-path="${escapeAttr(acc)}">${escapeHtml(part)}</a>`;
      }
    });
    crumbsEl.innerHTML = html;
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, "&#39;");
  }

  async function load() {
    renderCrumbs();
    showStatus("Загрузка…");
    try {
      const res = await fetch("/api/list?path=" + encodeURIComponent(currentPath));
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      renderList(data.entries || []);
      showStatus("");
    } catch (e) {
      showStatus(String(e.message || e), "error");
      listEl.innerHTML = "";
      emptyEl.hidden = true;
    }
  }

  function renderList(entries) {
    if (!entries.length) {
      listEl.innerHTML = "";
      listEl.hidden = true;
      emptyEl.hidden = false;
      return;
    }
    emptyEl.hidden = true;
    listEl.hidden = false;
    listEl.innerHTML = entries
      .map((e, i) => {
        const delay = Math.min(i * 20, 200);
        const icon = e.isDir ? "D" : "F";
        const iconClass = e.isDir ? "icon dir" : "icon";
        const href = e.isDir
          ? `#/${escapeAttr(e.path)}`
          : `/api/download?path=${encodeURIComponent(e.path)}`;
        const nameHtml = e.isDir
          ? `<a href="${href}">${escapeHtml(e.name)}</a>`
          : `<a href="${href}" download>${escapeHtml(e.name)}</a>`;
        const meta = e.isDir
          ? "папка · " + formatDate(e.modTime)
          : formatSize(e.size) + " · " + formatDate(e.modTime);
        const openBtn = e.isDir
          ? `<a class="btn sm ghost" href="#/${escapeAttr(e.path)}">Открыть</a>`
          : `<a class="btn sm ghost" href="/api/download?path=${encodeURIComponent(e.path)}" download>Скачать</a>`;
        return `
          <div class="row" style="animation-delay:${delay}ms">
            <div class="file-main">
              <div class="${iconClass}" aria-hidden="true">${icon}</div>
              <div class="name">${nameHtml}</div>
            </div>
            <div class="meta">${meta}</div>
            <div class="row-actions">
              ${openBtn}
              <button type="button" class="btn sm danger" data-del="${escapeAttr(e.path)}" data-name="${escapeAttr(e.name)}">Удалить</button>
            </div>
          </div>`;
      })
      .join("");
  }

  listEl.addEventListener("click", async (ev) => {
    const btn = ev.target.closest("[data-del]");
    if (!btn) return;
    const p = btn.getAttribute("data-del");
    const name = btn.getAttribute("data-name");
    if (!confirm(`Удалить «${name}»?`)) return;
    try {
      const res = await fetch("/api/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: p }),
      });
      if (!res.ok) throw new Error(await res.text());
      showStatus("Удалено", "ok");
      load();
    } catch (e) {
      showStatus(String(e.message || e), "error");
    }
  });

  async function uploadFiles(fileList) {
    const files = Array.from(fileList || []);
    if (!files.length) return;
    const fd = new FormData();
    files.forEach((f) => fd.append("files", f, f.name));
    showStatus(`Загрузка ${files.length} файл(ов)…`);
    try {
      const res = await fetch(
        "/api/upload?path=" + encodeURIComponent(currentPath),
        { method: "POST", body: fd }
      );
      if (!res.ok) throw new Error(await res.text());
      showStatus("Загружено", "ok");
      load();
    } catch (e) {
      showStatus(String(e.message || e), "error");
    }
  }

  fileInput.addEventListener("change", () => {
    uploadFiles(fileInput.files);
    fileInput.value = "";
  });

  btnMkdir.addEventListener("click", async () => {
    const name = prompt("Имя новой папки:");
    if (!name) return;
    try {
      const res = await fetch("/api/mkdir", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: currentPath, name }),
      });
      if (!res.ok) throw new Error(await res.text());
      showStatus("Папка создана", "ok");
      load();
    } catch (e) {
      showStatus(String(e.message || e), "error");
    }
  });

  function syncPathFromHash() {
    const h = location.hash.replace(/^#\/?/, "");
    currentPath = decodeURIComponent(h);
    load();
  }

  window.addEventListener("hashchange", syncPathFromHash);

  let dragDepth = 0;
  window.addEventListener("dragenter", (e) => {
    e.preventDefault();
    dragDepth++;
    overlay.hidden = false;
  });
  window.addEventListener("dragleave", (e) => {
    e.preventDefault();
    dragDepth = Math.max(0, dragDepth - 1);
    if (dragDepth === 0) overlay.hidden = true;
  });
  window.addEventListener("dragover", (e) => e.preventDefault());
  window.addEventListener("drop", (e) => {
    e.preventDefault();
    dragDepth = 0;
    overlay.hidden = true;
    if (e.dataTransfer && e.dataTransfer.files) {
      uploadFiles(e.dataTransfer.files);
    }
  });

  fetch("/api/info")
    .then((r) => r.json())
    .then((d) => setRootLabel(d.root))
    .catch(() => {});

  syncPathFromHash();
})();
