/**
 * um-react bridge (non-destructive injection)
 *
 * Adds an "导入" (Import) button next to the "下载" (Download) button on each
 * decrypted-audio card in um-react. Clicking it uploads the decrypted audio to
 * the BCTools backend and opens the main app's song-import page IN A SEPARATE
 * TAB, leaving um-react open so the user can import more songs.
 *
 * The application backend owns the temporary upload and its import receipt.
 * The import tab or desktop window opens with the stage ID in its URL.
 *
 * Import-success feedback: after the main app imports the staged file it POSTs
 * /api/decrypt/stage/{id}/imported. This script polls that flag; once it sees
 * imported=true it removes the decrypted card using um-react's OWN delete logic
 * (it finds the card's native 删除 button and clicks it), then deletes the stage.
 * um-react is never closed or navigated away.
 *
 * This script is injected into um-react's index.html by the root npm build and does NOT
 * modify any um-react source, so updating um-react from upstream won't break it.
 */
(function () {
  "use strict";

  var STAGE_URL = "/api/decrypt/stage";
  var IMPORT_URL = "/#/dorm/manage?stage=";
  var LOGO_URL = "/logo.png";
  var IMPORT_TAB_NAME = "bctools_import";
  // 轮询间隔直接决定「主应用导入成功 → um-react 这边卡片消失」的观感延迟。
  // 1500ms 会让用户觉得卡了一下；请求本身只是读一个几十字节的 meta，很轻。
  var POLL_INTERVAL = 300; // ms
  var POLL_MAX_ATTEMPTS = 1000; // ~5 min before giving up

  // stageId -> { timer, attempts }
  var pending = {};

  function findDownloadAnchor(row) {
    return row.querySelector('[data-testid="audio-download"]');
  }

  function createImportButton(downloadAnchor) {
    var container = downloadAnchor.parentElement;
    if (!container || container.querySelector("[data-bctools-import]")) return null;

    var btn = document.createElement("button");
    btn.type = "button";
    btn.setAttribute("data-bctools-import", "");
    // Mirror um-react's daisyUI button classes so the injected button matches the
    // existing 下载 button.
    btn.className = "btn btn-outline btn-sm gap-2";
    btn.style.marginLeft = "0.5rem";

    var img = document.createElement("img");
    img.src = LOGO_URL;
    img.alt = "";
    img.style.width = "16px";
    img.style.height = "16px";
    img.style.verticalAlign = "middle";
    btn.appendChild(img);

    var label = document.createElement("span");
    label.textContent = "导入";
    btn.appendChild(label);

    btn.addEventListener("click", function (e) {
      e.preventDefault();
      e.stopPropagation();
      handleImport(downloadAnchor, btn);
    });

    container.appendChild(btn);
    return btn;
  }

  function setBusy(btn, text, disabled) {
    if (!btn) return;
    btn.disabled = !!disabled;
    var span = btn.querySelector("span");
    if (span) span.textContent = text;
  }

  function handleImport(downloadAnchor, btn) {
    var blobUrl = downloadAnchor.getAttribute("href");
    var fileName = downloadAnchor.getAttribute("download") || "decrypted.mp3";
    if (!blobUrl) {
      alert("未找到解密后的音频，请稍后再试。");
      return;
    }
    var desktop = location.hostname === "wails.localhost" || location.hostname.startsWith("wails.localhost.");
    // Open under the click's user activation, before asynchronous upload.
    var importTab = desktop ? null : window.open("", IMPORT_TAB_NAME);
    if (!desktop && !importTab) {
      alert("无法打开导入页面，请允许本站的弹出式窗口后重试。");
      return;
    }
    var uploadedStage;
    setBusy(btn, "上传中…", true);
    fetch(blobUrl)
      .then(function (res) {
        if (!res.ok) throw new Error("读取音频失败");
        return res.blob();
      })
      .then(function (blob) {
        var form = new FormData();
        form.append("file", blob, fileName);
        return fetch(STAGE_URL, { method: "POST", body: form });
      })
      .then(function (res) {
        if (!res.ok) throw new Error("上传失败");
        return res.json();
      })
      .then(function (data) {
        if (!data || !data.stageId) throw new Error("上传响应无效");
        uploadedStage = data.stageId;
        // Open the main app's import page in its own tab; um-react stays open.
        // A named tab is reused for successive imports so no tab accumulates.
        if (desktop) {
          return fetch("/api/desktop/window?view=import&stage=" + encodeURIComponent(data.stageId), {
            method: "POST", headers: { "X-BCTools-Local": "1" }
          }).then(function (res) {
            if (!res.ok) throw new Error("无法打开导入页面");
            setBusy(btn, "等待导入…", true);
            startPolling(data.stageId, btn, blobUrl);
          });
        }
        if (importTab.closed) throw new Error("导入页面已关闭");
        importTab.location.href = new URL(IMPORT_URL + encodeURIComponent(data.stageId), location.origin).href;
        setBusy(btn, "等待导入…", true);
        startPolling(data.stageId, btn, blobUrl);
      })
      .catch(function (err) {
        if (uploadedStage) {
          fetch(STAGE_URL + "/" + encodeURIComponent(uploadedStage), { method: "DELETE" }).catch(function () {});
        }
        if (importTab && !importTab.closed) {
          try { if (importTab.location.href === "about:blank") importTab.close(); } catch {}
        }
        console.error("[bctools-bridge] import failed:", err);
        setBusy(btn, "导入", false);
        alert("导入失败：" + (err && err.message ? err.message : "未知错误"));
      });
  }

  // startPolling watches the staged file's meta until the main app marks it
  // imported, then removes the um-react card via its own delete logic.
  function startPolling(stageId, btn, blobUrl) {
    var entry = { attempts: 0, timer: null };
    pending[stageId] = entry;
    entry.timer = setInterval(function () {
      entry.attempts++;
      fetch(STAGE_URL + "/" + encodeURIComponent(stageId))
        .then(function (res) {
          if (res.status === 404) {
            // Stage was deleted (user cancelled / expired) - give up, reset button.
            stopPolling(stageId);
            setBusy(btn, "导入", false);
            return null;
          }
          if (!res.ok) throw new Error("status " + res.status);
          return res.json();
        })
        .then(function (meta) {
          if (!meta) return;
          if (meta.imported) {
            stopPolling(stageId);
            removeCard(blobUrl);
            // Clean up the backend stage now that um-react's card is gone.
            fetch(STAGE_URL + "/" + encodeURIComponent(stageId), { method: "DELETE" }).catch(function () {});
          }
        })
        .catch(function () {
          // Transient error - keep polling until the cap below kicks in.
        });
      if (entry.attempts >= POLL_MAX_ATTEMPTS) {
        stopPolling(stageId);
        setBusy(btn, "导入", false);
      }
    }, POLL_INTERVAL);
  }

  function stopPolling(stageId) {
    var entry = pending[stageId];
    if (entry && entry.timer) clearInterval(entry.timer);
    delete pending[stageId];
  }

  // A blob URL identifies the exact card even when several tracks share a title.
  function removeCard(blobUrl) {
    var anchors = document.querySelectorAll('[data-testid="audio-download"]');
    for (var i = 0; i < anchors.length; i++) {
      if (anchors[i].getAttribute("href") === blobUrl) {
        var cardActions = anchors[i].parentElement;
        var delBtn = cardActions && cardActions.querySelector("button.btn-error");
        if (delBtn) {
          delBtn.click();
          return;
        }
      }
    }
  }

  function processRow(row) {
    var anchor = findDownloadAnchor(row);
    if (anchor && anchor.getAttribute("href")) createImportButton(anchor);
  }

  function scan(root) {
    var rows = root.querySelectorAll('[data-testid="file-row"]');
    rows.forEach(processRow);
  }

  function init() {
    scan(document.body);
    var observer = new MutationObserver(function (mutations) {
      for (var i = 0; i < mutations.length; i++) {
        var target = mutations[i].target;
        var row = target.closest && target.closest('[data-testid="file-row"]');
        if (row) processRow(row);
        var added = mutations[i].addedNodes;
        for (var j = 0; j < added.length; j++) {
          var node = added[j];
          if (node.nodeType !== 1) continue;
          if (node.matches && node.matches('[data-testid="file-row"]')) {
            processRow(node);
          } else if (node.querySelectorAll) {
            scan(node);
          }
        }
      }
    });
    observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ["href"] });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
