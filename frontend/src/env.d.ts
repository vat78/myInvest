/// <reference types="vite/client" />
interface Window {
  go: {
    main: { App: typeof import('../wailsjs/go/main/App') };
    desktop: { API: typeof import('../wailsjs/go/desktop/API') };
  };
}
