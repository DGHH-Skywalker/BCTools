/**
 * um-react bridge (non-destructive injection)
 *
 * Adds an "导入" (Import) button next to the "下载" (Download) button on each
 * decrypted-audio card in um-react. Clicking it uploads the decrypted audio to
 * the BCTools backend and opens the main app's song-import page IN A SEPARATE
 * TAB, leaving um-react open so the user can import more songs.
 *
 * Cross-page communication goes through the BACKEND, not window.opener/postMessage:
 * the um-react tab and the main app may run on different devices on the LAN (e.g.
 * a phone decrypting and a PC importing), and window.opener only works between
 * tabs on a single browser on one device. Routing the hand-off through the
 * backend makes it work everywhere, including mobile.
 *
 * Import-success feedback: after the main app imports the staged file it POSTs
 * /api/decrypt/stage/{id}/imported. This script polls that flag; once it sees
 * imported=true it removes the decrypted card using um-react's OWN delete logic
 * (it finds the card's native 删除 button and clicks it), then deletes the stage.
 * um-react is never closed or navigated away.
 *
 * This script is injected into um-react's index.html by build.py and does NOT
 * modify any um-react source, so updating um-react from upstream won't break it.
 */
(function () {
  "use strict";

  var STAGE_URL = "/api/decrypt/stage";
  var IMPORT_URL = "/#/dorm/manage?stage=";
  var LOGO_URL = "/logo.png";
  var IMPORT_TAB_NAME = "bctools_import";
  var POLL_INTERVAL = 1500; // ms
  var POLL_MAX_ATTEMPTS = 200; // ~5 min before giving up

  // stageId -> { btn, fileName, timer, attempts }
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
    // The download attribute carries the decrypted filename; we match on it later
    // to find this exact card again and click its native 删除 button.
    var fileName = downloadAnchor.getAttribute("download") || "decrypted.mp3";
    if (!blobUrl) {
      alert("未找到解密后的音频，请稍后再试。");
      return;
    }
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
        // Open the main app's import page in its own tab; um-react stays open.
        // A named tab is reused for successive imports so no tab accumulates.
        var opened = window.open(IMPORT_URL + encodeURIComponent(data.stageId), IMPORT_TAB_NAME);
        if (!opened) {
          alert("无法打开导入页面，请允许本站的弹出式窗口后重试。");
          setBusy(btn, "导入", false);
          return;
        }
        setBusy(btn, "等待导入…", true);
        startPolling(data.stageId, btn, fileName);
      })
      .catch(function (err) {
        console.error("[bctools-bridge] import failed:", err);
        setBusy(btn, "导入", false);
        alert("导入失败：" + (err && err.message ? err.message : "未知错误"));
      });
  }

  // startPolling watches the staged file's meta until the main app marks it
  // imported, then removes the um-react card via its own delete logic.
  function startPolling(stageId, btn, fileName) {
    var entry = { btn: btn, fileName: fileName, attempts: 0, timer: null };
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
            removeCardByFileName(fileName);
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

  // removeCardByFileName finds the decrypted card whose download filename matches
  // and clicks um-react's own 删除 button, reusing um-react's native delete logic
  // (Redux deleteFile + blob URL revocation). Matching by filename is robust to
  // React re-renders, which would stale a stored element reference.
  function removeCardByFileName(fileName) {
    var anchors = document.querySelectorAll('[data-testid="audio-download"]');
    for (var i = 0; i < anchors.length; i++) {
      if (anchors[i].getAttribute("download") === fileName) {
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
    if (anchor) createImportButton(anchor);
  }

  function scan(root) {
    var rows = root.querySelectorAll('[data-testid="file-row"]');
    rows.forEach(processRow);
  }

  function init() {
    scan(document.body);
    var observer = new MutationObserver(function (mutations) {
      for (var i = 0; i < mutations.length; i++) {
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
    observer.observe(document.body, { childList: true, subtree: true });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
