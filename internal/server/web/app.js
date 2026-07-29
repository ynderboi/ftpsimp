(() => {
  const TOKEN_KEY = "ftpsimp_token";
  const LANG_KEY = "ftpsimp_lang";

  const STR = {
    en: {
      loginTitle: "ftpsimp — Log On",
      loginHelp: "Enter the PIN shown on the host computer.",
      loginPinRequired: "Enter PIN.",
      loginBadPin: "Invalid PIN",
      loginBadResp: "Bad login response",
      sessionExpired: "Session expired. Enter PIN again.",
      settings: "Settings",
      newFolder: "New Folder",
      upload: "Upload",
      address: "Address",
      refresh: "Refresh",
      refreshed: "Refreshed.",
      colName: "Name",
      colSize: "Size",
      colDate: "Modified",
      emptyFolder: "This folder is empty.",
      emptyHint: "Drag files here or choose Upload.",
      ready: "Ready",
      readWrite: "Read/Write",
      readOnly: "Read-only",
      readOnlyUpload: "Read-only mode: uploads are disabled on the host.",
      readOnlyMkdir: "Read-only mode: cannot create folders.",
      copying: "Copying…",
      dropHint: "Drop files to upload to this folder.",
      rootLabel: "Root folder (this PC only):",
      cancel: "Cancel",
      logout: "Log off",
      lang: "Lang",
      open: "Open",
      get: "Get",
      del: "Del",
      confirmDel: "Delete “{name}”?",
      deleted: "Deleted.",
      folderCreated: "Folder created.",
      mkdirPrompt: "New folder name:",
      uploading: "Uploading {n} file(s)…",
      uploadDone: "Upload complete.",
      uploadCancel: "Upload cancelled.",
      overwrite: "File already exists. Overwrite?",
      opening: "Opening…",
      objects: "{n} object(s)",
      rootUpdated: "Root folder updated.",
      enterPath: "Enter a folder path.",
      hintCan: "Enter the full path on this computer. Saved for next start.",
      hintCannot: "Root can only be changed from the host PC.",
      exploring: "ftpsimp — {root}",
      shareRoot: "Shared folder",
      taskLogin: "Log On",
      taskExplorer: "Explorer",
      taskSettings: "Settings",
    },
    ru: {
      loginTitle: "ftpsimp — Вход",
      loginHelp: "Введите PIN с экрана или консоли компьютера, где запущен сервер.",
      loginPinRequired: "Введите PIN.",
      loginBadPin: "Неверный PIN",
      loginBadResp: "Ошибка ответа сервера",
      sessionExpired: "Сессия истекла. Введите PIN снова.",
      settings: "Настройки",
      newFolder: "Новая папка",
      upload: "Загрузить",
      address: "Адрес",
      refresh: "Обновить",
      refreshed: "Обновлено.",
      colName: "Имя",
      colSize: "Размер",
      colDate: "Изменён",
      emptyFolder: "Эта папка пуста.",
      emptyHint: "Перетащите файлы сюда или нажмите «Загрузить».",
      ready: "Готово",
      readWrite: "Чтение/запись",
      readOnly: "Только чтение",
      readOnlyUpload: "Режим только чтения: загрузка файлов отключена на хосте.",
      readOnlyMkdir: "Режим только чтения: нельзя создавать папки.",
      copying: "Копирование…",
      dropHint: "Отпустите файлы для загрузки в эту папку.",
      rootLabel: "Корневая папка (только с этого ПК):",
      cancel: "Отмена",
      logout: "Выйти",
      lang: "Язык",
      open: "Откр.",
      get: "Скач.",
      del: "Удал.",
      confirmDel: "Удалить «{name}»?",
      deleted: "Удалено.",
      folderCreated: "Папка создана.",
      mkdirPrompt: "Имя новой папки:",
      uploading: "Загрузка {n} файл(ов)…",
      uploadDone: "Загрузка завершена.",
      uploadCancel: "Загрузка отменена.",
      overwrite: "Файл уже есть. Перезаписать?",
      opening: "Открытие…",
      objects: "{n} объект(ов)",
      rootUpdated: "Корневая папка изменена.",
      enterPath: "Укажите путь к папке.",
      hintCan: "Укажите полный путь на этом компьютере. Сохранится для следующего запуска.",
      hintCannot: "Смена корня доступна только с хоста (откройте UI на ПК с сервером).",
      exploring: "ftpsimp — {root}",
      shareRoot: "Общая папка",
      taskLogin: "Вход",
      taskExplorer: "Проводник",
      taskSettings: "Настройки",
    },
  };

  let lang = "en";
  {
    const stored = localStorage.getItem(LANG_KEY);
    if (stored === "ru" || stored === "en") lang = stored;
    else lang = (navigator.language || "").toLowerCase().startsWith("ru") ? "ru" : "en";
  }

  function t(key, vars) {
    let s = (STR[lang] && STR[lang][key]) || STR.en[key] || key;
    if (vars) {
      Object.keys(vars).forEach((k) => {
        s = s.replace(new RegExp("\\{" + k + "\\}", "g"), vars[k]);
      });
    }
    return s;
  }

  function applyI18n() {
    document.documentElement.lang = lang;
    document.querySelectorAll("[data-i18n]").forEach((el) => {
      const key = el.getAttribute("data-i18n");
      if (key) el.textContent = t(key);
    });
    document.querySelectorAll("[data-i18n-title]").forEach((el) => {
      const key = el.getAttribute("data-i18n-title");
      if (key) el.title = t(key);
    });
    if (langSelect) langSelect.value = lang;
    if (langTray) langTray.value = lang;
    applyReadOnlyUI();
    updateStatusline();
    updateTaskbar();
  }

  const gate = document.getElementById("gate");
  const app = document.getElementById("app");
  const loginForm = document.getElementById("login-form");
  const pinInput = document.getElementById("pin-input");
  const gateError = document.getElementById("gate-error");
  const listEl = document.getElementById("list");
  const listHead = document.getElementById("list-head");
  const emptyEl = document.getElementById("empty");
  const crumbsEl = document.getElementById("crumbs");
  const statusEl = document.getElementById("status");
  const statusline = document.getElementById("statusline");
  const windowTitle = document.getElementById("window-title");
  const overlay = document.getElementById("drop-overlay");
  const fileInput = document.getElementById("file-input");
  const btnMkdir = document.getElementById("btn-mkdir");
  const btnRefresh = document.getElementById("btn-refresh");
  const btnRefreshTool = document.getElementById("btn-refresh-tool");
  const btnSettings = document.getElementById("btn-settings");
  const btnLogout = document.getElementById("btn-logout");
  const btnUploadWrap = document.getElementById("btn-upload-wrap");
  const settingsModal = document.getElementById("settings-modal");
  const settingsWindow = document.getElementById("settings-window");
  const settingsForm = document.getElementById("settings-form");
  const settingsRoot = document.getElementById("settings-root");
  const settingsError = document.getElementById("settings-error");
  const settingsHint = document.getElementById("settings-hint");
  const settingsSave = document.getElementById("settings-save");
  const settingsCancel = document.getElementById("settings-cancel");
  const settingsClose = document.getElementById("settings-close");
  const settingsBackdrop = document.getElementById("settings-backdrop");
  const footLeft = document.getElementById("foot-left");
  const footRight = document.getElementById("foot-right");
  const taskButtons = document.getElementById("task-buttons");
  const clockEl = document.getElementById("clock");
  const langSelect = document.getElementById("lang-select");
  const langTray = document.getElementById("lang-select-tray");

  let currentPath = "";
  let readOnly = false;
  let authRequired = true;
  let rootPath = "";
  let token = sessionStorage.getItem(TOKEN_KEY) || "";
  let zTop = 10;
  let lastEntries = [];

  const winState = new WeakMap();

  function setToken(v) {
    token = v || "";
    if (token) sessionStorage.setItem(TOKEN_KEY, token);
    else sessionStorage.removeItem(TOKEN_KEY);
  }

  function authHeaders(extra) {
    const h = Object.assign({}, extra || {});
    if (token) h.Authorization = "Bearer " + token;
    return h;
  }

  function api(url, opts) {
    const o = Object.assign({ credentials: "same-origin" }, opts || {});
    o.headers = authHeaders(o.headers);
    return fetch(url, o);
  }

  function withToken(url) {
    if (!token) return url;
    return url + (url.includes("?") ? "&" : "?") + "token=" + encodeURIComponent(token);
  }

  function deskH() {
    return window.innerHeight - 28;
  }

  function getState(win) {
    let s = winState.get(win);
    if (!s) {
      s = { restored: null, maximized: false, minimized: false };
      winState.set(win, s);
    }
    return s;
  }

  function bringToFront(win) {
    document.querySelectorAll(".window.active").forEach((w) => w.classList.remove("active"));
    win.classList.add("active");
    zTop += 1;
    win.style.zIndex = String(zTop);
    updateTaskbar();
  }

  function isVisibleWin(win) {
    if (win === settingsWindow) return !settingsModal.hasAttribute("hidden") && !win.classList.contains("minimized");
    return !win.hasAttribute("hidden") && !win.classList.contains("minimized");
  }

  function snapshotRect(win) {
    const r = win.getBoundingClientRect();
    return { left: r.left, top: r.top, width: r.width, height: r.height };
  }

  function applyRect(win, rect) {
    win.style.left = rect.left + "px";
    win.style.top = rect.top + "px";
    if (rect.width) win.style.width = rect.width + "px";
    if (rect.height && win.classList.contains("explorer")) win.style.height = rect.height + "px";
    else if (rect.height && win.dataset.win === "settings") win.style.height = rect.height + "px";
  }

  function placeWindow(win, preferX, preferY) {
    const pad = 8;
    const vw = window.innerWidth;
    const vh = deskH();
    const st = getState(win);
    st.minimized = false;
    win.classList.remove("minimized");
    if (win !== settingsWindow) win.removeAttribute("hidden");
    win.style.visibility = "hidden";
    if (win === settingsWindow) settingsModal.removeAttribute("hidden");

    if (!win.style.width) {
      /* keep CSS default */
    }
    const rect = win.getBoundingClientRect();
    const w = rect.width || 380;
    const h = rect.height || 240;
    let x = preferX != null ? preferX : Math.max(pad, Math.round((vw - w) / 2));
    let y = preferY != null ? preferY : Math.max(pad, Math.round((vh - h) / 3));
    x = Math.min(Math.max(pad, x), Math.max(pad, vw - Math.min(w, vw - pad)));
    y = Math.min(Math.max(pad, y), Math.max(pad, vh - Math.min(h, vh - pad)));
    win.style.left = x + "px";
    win.style.top = y + "px";
    win.style.visibility = "";
    bringToFront(win);
    updateTaskbar();
  }

  function maximizeWindow(win) {
    // Login dialog stays compact; full-screen maximize breaks the PIN field on phones.
    if (win === gate) return;
    const st = getState(win);
    if (st.maximized) {
      restoreWindow(win);
      return;
    }
    st.restored = snapshotRect(win);
    st.maximized = true;
    win.classList.add("maximized");
    win.style.left = "0px";
    win.style.top = "0px";
    win.style.width = window.innerWidth + "px";
    win.style.height = deskH() + "px";
    const maxBtn = win.querySelector("[data-win-max]");
    if (maxBtn) maxBtn.textContent = "❐";
    bringToFront(win);
  }

  function restoreWindow(win) {
    const st = getState(win);
    st.maximized = false;
    win.classList.remove("maximized", "minimized");
    st.minimized = false;
    if (st.restored) applyRect(win, st.restored);
    const maxBtn = win.querySelector("[data-win-max]");
    if (maxBtn) maxBtn.textContent = "□";
    if (win === settingsWindow) settingsModal.removeAttribute("hidden");
    else win.removeAttribute("hidden");
    bringToFront(win);
  }

  function minimizeWindow(win) {
    const st = getState(win);
    if (!st.maximized) st.restored = snapshotRect(win);
    st.minimized = true;
    win.classList.add("minimized");
    if (win === settingsWindow) {
      // keep modal open state but hide window via minimized class; hide backdrop clutter
      settingsModal.setAttribute("hidden", "");
    }
    updateTaskbar();
  }

  function toggleWindow(win) {
    const st = getState(win);
    if (st.minimized || (win === settingsWindow && settingsModal.hasAttribute("hidden"))) {
      if (win === settingsWindow) settingsModal.removeAttribute("hidden");
      restoreWindow(win);
      return;
    }
    if (win.classList.contains("active") && isVisibleWin(win)) minimizeWindow(win);
    else bringToFront(win);
  }

  function updateTaskbar() {
    const items = [];
    const gateSt = getState(gate);
    const appSt = getState(app);
    const setSt = getState(settingsWindow);

    if (!gate.hasAttribute("hidden") || gateSt.minimized) {
      items.push({ id: "gate", label: t("taskLogin"), win: gate, st: gateSt });
    }
    if (!app.hasAttribute("hidden") || appSt.minimized) {
      items.push({ id: "app", label: t("taskExplorer"), win: app, st: appSt });
    }
    if (!settingsModal.hasAttribute("hidden") || setSt.minimized) {
      items.push({ id: "settings", label: t("taskSettings"), win: settingsWindow, st: setSt });
    }

    taskButtons.innerHTML = items
      .map((it) => {
        const active = it.win.classList.contains("active") && !it.st.minimized;
        return `<button type="button" class="task-btn${active ? " active" : ""}" data-task="${it.id}">${escapeHtml(it.label)}</button>`;
      })
      .join("");
  }

  taskButtons.addEventListener("click", (e) => {
    const btn = e.target.closest("[data-task]");
    if (!btn) return;
    const id = btn.getAttribute("data-task");
    const win = id === "gate" ? gate : id === "app" ? app : settingsWindow;
    toggleWindow(win);
  });

  function makeWindowControls(win) {
    win.querySelectorAll("[data-win-min]").forEach((b) =>
      b.addEventListener("click", (e) => {
        e.stopPropagation();
        minimizeWindow(win);
      })
    );
    win.querySelectorAll("[data-win-max]").forEach((b) =>
      b.addEventListener("click", (e) => {
        e.stopPropagation();
        maximizeWindow(win);
      })
    );
    const handle = win.querySelector("[data-drag-handle]");
    if (handle) {
      handle.addEventListener("dblclick", (e) => {
        if (e.target.closest(".title-bar-controls")) return;
        maximizeWindow(win);
      });
    }
  }

  function makeDraggable(win) {
    const handle = win.querySelector("[data-drag-handle]");
    if (!handle) return;
    let dragging = false;
    let startX = 0;
    let startY = 0;
    let origLeft = 0;
    let origTop = 0;

    handle.addEventListener("pointerdown", (e) => {
      if (e.button != null && e.button !== 0) return;
      if (e.target.closest(".title-bar-controls, button, input, a, label, select")) return;
      if (getState(win).maximized) return;
      bringToFront(win);
      const rect = win.getBoundingClientRect();
      win.style.left = rect.left + "px";
      win.style.top = rect.top + "px";
      dragging = true;
      startX = e.clientX;
      startY = e.clientY;
      origLeft = rect.left;
      origTop = rect.top;
      document.body.style.userSelect = "none";
      try {
        handle.setPointerCapture(e.pointerId);
      } catch (_) {}
      e.preventDefault();
    });

    handle.addEventListener("pointermove", (e) => {
      if (!dragging) return;
      let x = origLeft + (e.clientX - startX);
      let y = origTop + (e.clientY - startY);
      x = Math.min(Math.max(-win.offsetWidth + 80, x), window.innerWidth - 48);
      y = Math.min(Math.max(0, y), deskH() - 24);
      win.style.left = x + "px";
      win.style.top = y + "px";
    });

    const stop = () => {
      dragging = false;
      document.body.style.userSelect = "";
    };
    handle.addEventListener("pointerup", stop);
    handle.addEventListener("pointercancel", stop);
    win.addEventListener("mousedown", () => bringToFront(win));
  }

  function makeResizable(win) {
    const handles = win.querySelectorAll("[data-resize]");
    handles.forEach((el) => {
      el.addEventListener("pointerdown", (e) => {
        if (getState(win).maximized) return;
        e.preventDefault();
        e.stopPropagation();
        bringToFront(win);
        const dir = el.getAttribute("data-resize");
        const start = { x: e.clientX, y: e.clientY, ...snapshotRect(win) };
        const minW = 280;
        const minH = win.classList.contains("login-window") ? 140 : 200;

        function move(ev) {
          let { left, top, width, height } = start;
          const dx = ev.clientX - start.x;
          const dy = ev.clientY - start.y;
          if (dir.includes("e")) width = Math.max(minW, start.width + dx);
          if (dir.includes("s")) height = Math.max(minH, start.height + dy);
          if (dir.includes("w")) {
            width = Math.max(minW, start.width - dx);
            left = start.left + (start.width - width);
          }
          if (dir.includes("n")) {
            height = Math.max(minH, start.height - dy);
            top = start.top + (start.height - height);
          }
          width = Math.min(width, window.innerWidth - left);
          height = Math.min(height, deskH() - top);
          win.style.left = left + "px";
          win.style.top = top + "px";
          win.style.width = width + "px";
          if (!win.classList.contains("login-window") || height > 160) {
            win.style.height = height + "px";
          }
        }

        function up() {
          window.removeEventListener("pointermove", move);
          window.removeEventListener("pointerup", up);
          document.body.style.userSelect = "";
        }
        document.body.style.userSelect = "none";
        window.addEventListener("pointermove", move);
        window.addEventListener("pointerup", up);
      });
    });
  }

  [gate, app, settingsWindow].forEach((w) => {
    makeDraggable(w);
    makeResizable(w);
    makeWindowControls(w);
  });

  function isMobile() {
    return window.matchMedia("(max-width: 720px), (pointer: coarse)").matches && window.innerWidth <= 900;
  }

  function showGate(show) {
    if (show) {
      app.setAttribute("hidden", "");
      getState(app).minimized = false;
      gate.classList.remove("maximized", "minimized");
      getState(gate).maximized = false;
      getState(gate).minimized = false;
      gate.style.width = "";
      gate.style.height = "";
      placeWindow(gate);
      // Keep login compact on phones — maximize stretches the PIN field oddly.
      const maxBtn = gate.querySelector("[data-win-max]");
      if (maxBtn) maxBtn.textContent = "□";
    } else {
      gate.setAttribute("hidden", "");
      getState(gate).minimized = false;
      placeWindow(app, 16, 16);
      if (isMobile()) maximizeWindow(app);
    }
    updateTaskbar();
  }

  function closeSettings() {
    getState(settingsWindow).minimized = false;
    settingsWindow.classList.remove("minimized", "maximized");
    settingsModal.setAttribute("hidden", "");
    updateTaskbar();
  }

  function openSettingsModal() {
    getState(settingsWindow).minimized = false;
    settingsWindow.classList.remove("minimized");
    placeWindow(settingsWindow);
  }

  document.querySelector("[data-close-gate]")?.addEventListener("click", () => {
    minimizeWindow(gate);
  });

  function applyReadOnlyUI() {
    btnMkdir.hidden = false;
    btnUploadWrap.hidden = false;
    btnMkdir.classList.toggle("ro-disabled", readOnly);
    btnUploadWrap.classList.toggle("ro-disabled", readOnly);
    footRight.textContent = readOnly ? t("readOnly") : t("readWrite");
  }

  function warnReadOnly(kind) {
    showStatus(t(kind === "mkdir" ? "readOnlyMkdir" : "readOnlyUpload"), "error");
  }

  function updateStatusline() {
    const host = location.host || "local";
    statusline.textContent = host + (rootPath ? " — " + rootPath : "");
    windowTitle.textContent = t("exploring", { root: rootPath || t("shareRoot") });
  }

  function setLang(next) {
    lang = next === "ru" ? "ru" : "en";
    localStorage.setItem(LANG_KEY, lang);
    applyI18n();
    renderList(lastEntries);
    renderCrumbs();
  }

  langSelect.addEventListener("change", () => setLang(langSelect.value));
  langTray.addEventListener("change", () => setLang(langTray.value));

  async function bootstrap() {
    applyI18n();
    try {
      const res = await api("/api/status");
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      authRequired = !!data.authRequired;
      readOnly = !!data.readOnly;
      btnLogout.hidden = !authRequired;

      if (data.authenticated) {
        rootPath = data.root || "";
        showGate(false);
        applyReadOnlyUI();
        updateStatusline();
        syncPathFromHash();
        return;
      }
      if (token) setToken("");
      if (!authRequired) {
        showGate(false);
        applyReadOnlyUI();
        updateStatusline();
        try {
          const info = await api("/api/info");
          if (info.ok) {
            const d = await info.json();
            rootPath = d.root || "";
            readOnly = !!d.readOnly || readOnly;
            applyReadOnlyUI();
            updateStatusline();
          }
        } catch (_) {}
        syncPathFromHash();
        return;
      }
      showGate(true);
    } catch (e) {
      showGate(true);
      gateError.textContent = String(e.message || e);
      gateError.hidden = false;
    }
  }

  pinInput.addEventListener("input", () => {
    const cleaned = pinInput.value.replace(/\D+/g, "").slice(0, 12);
    if (pinInput.value !== cleaned) pinInput.value = cleaned;
  });

  loginForm.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    gateError.hidden = true;
    const pin = pinInput.value.replace(/\D+/g, "").trim();
    if (!pin) {
      gateError.textContent = t("loginPinRequired");
      gateError.hidden = false;
      return;
    }
    try {
      const res = await fetch("/api/login", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pin }),
      });
      const raw = (await res.text()).trim();
      if (!res.ok) throw new Error(raw || t("loginBadPin"));
      const data = JSON.parse(raw);
      if (data.token) setToken(data.token);
      rootPath = data.root || "";
      readOnly = !!data.readOnly;
      pinInput.value = "";
      showGate(false);
      applyReadOnlyUI();
      updateStatusline();
      syncPathFromHash();
    } catch (e) {
      gateError.textContent = String(e.message || e);
      gateError.hidden = false;
      pinInput.select();
    }
  });

  btnLogout.addEventListener("click", async () => {
    try {
      await api("/api/logout", { method: "POST" });
    } catch (_) {}
    setToken("");
    closeSettings();
    location.hash = "#/";
    currentPath = "";
    listEl.innerHTML = "";
    showGate(true);
  });

  async function openSettings() {
    settingsError.hidden = true;
    settingsError.textContent = "";
    try {
      const res = await api("/api/settings");
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      settingsRoot.value = data.root || "";
      const can = !!data.canChangeRoot;
      settingsRoot.disabled = !can;
      settingsSave.disabled = !can;
      settingsHint.textContent = can ? t("hintCan") : t("hintCannot");
    } catch (e) {
      settingsRoot.value = rootPath || "";
      settingsRoot.disabled = true;
      settingsSave.disabled = true;
      settingsHint.textContent = t("hintCannot");
    }
    openSettingsModal();
    if (!settingsRoot.disabled) {
      settingsRoot.focus();
      settingsRoot.select();
    }
  }

  btnSettings.addEventListener("click", openSettings);
  btnRefresh.addEventListener("click", refresh);
  btnRefreshTool.addEventListener("click", refresh);
  settingsCancel.addEventListener("click", closeSettings);
  settingsClose.addEventListener("click", closeSettings);
  settingsBackdrop.addEventListener("click", closeSettings);

  settingsForm.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    if (settingsSave.disabled) return;
    settingsError.hidden = true;
    const root = settingsRoot.value.trim();
    if (!root) {
      settingsError.textContent = t("enterPath");
      settingsError.hidden = false;
      return;
    }
    try {
      const res = await api("/api/settings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ root }),
      });
      if (!res.ok) throw new Error((await res.text()).trim() || "Error");
      const data = await res.json();
      rootPath = data.root || root;
      updateStatusline();
      closeSettings();
      location.hash = "#/";
      currentPath = "";
      showStatus(t("rootUpdated"), "ok");
      load();
    } catch (e) {
      settingsError.textContent = String(e.message || e);
      settingsError.hidden = false;
    }
  });

  function showStatus(msg, kind = "") {
    statusEl.hidden = !msg;
    statusEl.textContent = msg || "";
    statusEl.className = "status-banner" + (kind ? " " + kind : "");
    footLeft.textContent = msg || t("ready");
    if (msg && kind !== "error") {
      clearTimeout(showStatus._t);
      showStatus._t = setTimeout(() => {
        statusEl.hidden = true;
        footLeft.textContent = t("ready");
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
      return new Date(iso).toLocaleString(lang === "ru" ? "ru-RU" : "en-US", {
        day: "2-digit",
        month: "2-digit",
        year: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch {
      return "";
    }
  }

  function pathParts(p) {
    return p ? p.split("/").filter(Boolean) : [];
  }

  function fileIconClass(isDir, name) {
    if (isDir) return "icon dir";
    const ext = (name.split(".").pop() || "").toLowerCase();
    if (["png", "jpg", "jpeg", "gif", "bmp", "ico", "webp"].includes(ext)) return "icon img";
    return "icon file";
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

  function pathSep() {
    return rootPath && rootPath.indexOf("\\") !== -1 ? "\\" : "/";
  }

  function splitRootSegments(root) {
    if (!root) return [];
    // Windows: C:\Users\...  or \\server\share
    if (/^[A-Za-z]:[\\/]/.test(root) || root.startsWith("\\\\")) {
      const norm = root.replace(/\//g, "\\");
      const m = norm.match(/^([A-Za-z]:)\\(.*)$/);
      if (m) {
        const rest = m[2] ? m[2].split("\\").filter(Boolean) : [];
        return [m[1] + "\\"].concat(rest);
      }
      return norm.split("\\").filter(Boolean);
    }
    // Unix absolute
    if (root.startsWith("/")) {
      const rest = root.split("/").filter(Boolean);
      return ["/"].concat(rest);
    }
    return root.split(/[\\/]/).filter(Boolean);
  }

  function renderCrumbs() {
    const rootSegs = splitRootSegments(rootPath);
    const relParts = pathParts(currentPath);
    const sep = pathSep();
    let html = "";

    if (!rootSegs.length) {
      html = `<a href="#/">${escapeHtml(t("shareRoot"))}</a>`;
    } else {
      // First clickable = root share
      html = `<a href="#/" title="${escapeAttr(rootPath)}">${escapeHtml(rootSegs[0])}</a>`;
      for (let i = 1; i < rootSegs.length; i++) {
        html += `<span class="sep">${escapeHtml(sep === "\\" ? "\\" : "/")}</span>`;
        // Intermediate root segments still go to share root (can't navigate above share)
        const isLastRoot = i === rootSegs.length - 1 && relParts.length === 0;
        if (isLastRoot) {
          html += `<span class="current">${escapeHtml(rootSegs[i])}</span>`;
        } else {
          html += `<a href="#/">${escapeHtml(rootSegs[i])}</a>`;
        }
      }
    }

    let acc = "";
    relParts.forEach((part, i) => {
      acc = acc ? acc + "/" + part : part;
      html += `<span class="sep">${escapeHtml(sep === "\\" ? "\\" : "/")}</span>`;
      if (i === relParts.length - 1) {
        html += `<span class="current">${escapeHtml(part)}</span>`;
      } else {
        html += `<a href="#/${escapeAttr(acc)}">${escapeHtml(part)}</a>`;
      }
    });

    // Full path tooltip on the bar
    const full =
      rootPath
        ? rootPath.replace(/[\\/]+$/, "") +
          (relParts.length ? (pathSep() + relParts.join(pathSep())) : "")
        : relParts.join(pathSep()) || t("shareRoot");
    crumbsEl.setAttribute("title", full);
    crumbsEl.innerHTML = html;
  }

  function refresh() {
    load(true);
  }

  async function load(fromRefresh) {
    renderCrumbs();
    showStatus(fromRefresh ? t("refresh") + "…" : t("opening"));
    try {
      const res = await api("/api/list?path=" + encodeURIComponent(currentPath));
      if (res.status === 401) {
        setToken("");
        showGate(true);
        gateError.textContent = t("sessionExpired");
        gateError.hidden = false;
        return;
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      lastEntries = data.entries || [];
      renderList(lastEntries);
      showStatus(fromRefresh ? t("refreshed") : "");
      footLeft.textContent = t("objects", { n: lastEntries.length });
    } catch (e) {
      showStatus(String(e.message || e), "error");
      listEl.innerHTML = "";
      listHead.hidden = true;
      emptyEl.hidden = true;
    }
  }

  function renderList(entries) {
    lastEntries = entries || [];
    if (!lastEntries.length) {
      listEl.innerHTML = "";
      listEl.hidden = true;
      listHead.hidden = true;
      emptyEl.hidden = false;
      return;
    }
    emptyEl.hidden = true;
    listEl.hidden = false;
    listHead.hidden = false;
    listEl.innerHTML = lastEntries
      .map((e) => {
        const href = e.isDir
          ? `#/${escapeAttr(e.path)}`
          : withToken(`/api/download?path=${encodeURIComponent(e.path)}`);
        const nameHtml = e.isDir
          ? `<a href="${href}">${escapeHtml(e.name)}</a>`
          : `<a href="${href}" download>${escapeHtml(e.name)}</a>`;
        const openBtn = e.isDir
          ? `<a class="btn sm" href="#/${escapeAttr(e.path)}">${escapeHtml(t("open"))}</a>`
          : `<a class="btn sm" href="${withToken(`/api/download?path=${encodeURIComponent(e.path)}`)}" download>${escapeHtml(t("get"))}</a>`;
        const delBtn = readOnly
          ? ""
          : `<button type="button" class="btn sm danger" data-del="${escapeAttr(e.path)}" data-name="${escapeAttr(e.name)}">${escapeHtml(t("del"))}</button>`;
        return `<div class="row">
          <div class="file-main"><div class="${fileIconClass(e.isDir, e.name)}" aria-hidden="true"></div><div class="name">${nameHtml}</div></div>
          <div class="meta">${e.isDir ? "" : formatSize(e.size)}</div>
          <div class="date">${formatDate(e.modTime)}</div>
          <div class="row-actions">${openBtn}${delBtn}</div>
        </div>`;
      })
      .join("");
  }

  listEl.addEventListener("click", async (ev) => {
    const row = ev.target.closest(".row");
    if (row) {
      listEl.querySelectorAll(".row.selected").forEach((r) => r.classList.remove("selected"));
      row.classList.add("selected");
    }
    const btn = ev.target.closest("[data-del]");
    if (!btn) return;
    const name = btn.getAttribute("data-name");
    if (!confirm(t("confirmDel", { name }))) return;
    try {
      const res = await api("/api/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: btn.getAttribute("data-del") }),
      });
      if (!res.ok) throw new Error(await res.text());
      showStatus(t("deleted"), "ok");
      load();
    } catch (e) {
      showStatus(String(e.message || e), "error");
    }
  });

  async function uploadFiles(fileList, overwrite = false) {
    const files = Array.from(fileList || []);
    if (!files.length) return;
    if (readOnly) {
      warnReadOnly("upload");
      return;
    }
    const fd = new FormData();
    files.forEach((f) => fd.append("files", f, f.name));
    showStatus(t("uploading", { n: files.length }));
    const q = "/api/upload?path=" + encodeURIComponent(currentPath) + (overwrite ? "&overwrite=1" : "");
    try {
      const res = await api(q, { method: "POST", body: fd });
      if (res.status === 409) {
        if (confirm(t("overwrite"))) return uploadFiles(fileList, true);
        showStatus(t("uploadCancel"));
        return;
      }
      if (!res.ok) throw new Error(await res.text());
      showStatus(t("uploadDone"), "ok");
      load();
    } catch (e) {
      showStatus(String(e.message || e), "error");
    }
  }

  btnUploadWrap.addEventListener(
    "click",
    (e) => {
      if (!readOnly) return;
      e.preventDefault();
      e.stopPropagation();
      warnReadOnly("upload");
    },
    true
  );

  fileInput.addEventListener("change", () => {
    uploadFiles(fileInput.files);
    fileInput.value = "";
  });

  btnMkdir.addEventListener("click", async () => {
    if (readOnly) {
      warnReadOnly("mkdir");
      return;
    }
    const name = prompt(t("mkdirPrompt"));
    if (!name) return;
    try {
      const res = await api("/api/mkdir", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: currentPath, name }),
      });
      if (!res.ok) throw new Error(await res.text());
      showStatus(t("folderCreated"), "ok");
      load();
    } catch (e) {
      showStatus(String(e.message || e), "error");
    }
  });

  function syncPathFromHash() {
    currentPath = decodeURIComponent(location.hash.replace(/^#\/?/, ""));
    load();
  }

  window.addEventListener("hashchange", () => {
    if (app.hasAttribute("hidden")) return;
    syncPathFromHash();
  });

  function tickClock() {
    const d = new Date();
    clockEl.textContent = d.toLocaleTimeString(lang === "ru" ? "ru-RU" : "en-US", {
      hour: "2-digit",
      minute: "2-digit",
    });
  }
  tickClock();
  setInterval(tickClock, 15000);

  let dragDepth = 0;
  window.addEventListener("dragenter", (e) => {
    if (app.hasAttribute("hidden")) return;
    e.preventDefault();
    if (readOnly) return;
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
    if (app.hasAttribute("hidden")) return;
    if (readOnly) {
      warnReadOnly("upload");
      return;
    }
    if (e.dataTransfer && e.dataTransfer.files) uploadFiles(e.dataTransfer.files);
  });

  bootstrap();
})();
