// Minimal ambient declarations for the Yandex Maps JS API 3.0.
// The API is loaded from a CDN, so a full type package would be overkill.
declare global {
  interface Window {
    ymaps3?: YandexMapsApi;
  }
}

export interface YandexMapsApi {
  ready: Promise<void>;
  YMap: new (element: HTMLElement, props: Record<string, unknown>) => YMapInstance;
  YMapDefaultSchemeLayer: new (props?: Record<string, unknown>) => unknown;
  YMapDefaultFeaturesLayer: new (props?: Record<string, unknown>) => unknown;
  YMapMarker: new (props: Record<string, unknown>, element: HTMLElement) => unknown;
}

export interface YMapInstance {
  addChild(child: unknown): void;
  removeChild(child: unknown): void;
  destroy(): void;
}

export {};
