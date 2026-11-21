import { registerSW } from 'virtual:pwa-register';

// The BCTools host serves and rolls back its own versioned assets.
if (window.BASE_URL === '/') {
  registerSW({
    onNeedRefresh() {},
    onOfflineReady() {},
  });
}
