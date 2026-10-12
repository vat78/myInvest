/// <reference types="vite/client" />
interface Window {
  go: { main: { App: { GetAppInfo(): Promise<{ name: string }> } } };
}
