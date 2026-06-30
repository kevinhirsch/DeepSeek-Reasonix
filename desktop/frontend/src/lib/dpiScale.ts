// dpiScale.ts — desktop zoom-factor helpers for the SettingsPanel display-zoom
// slider. Reads/writes a localStorage key so the zoom persists across restarts
// on all platforms; the Go backend applies the actual WebView zoom via the
// SetDesktopZoomFactor bound method.

export type ZoomLevel = number;

const STORAGE_KEY = "reasonix-display-zoom";

/** Snap zoom to the nearest 5 % step (0.05 increments). */
export function snapZoom(zoom: number): number {
  return Math.round(zoom * 20) / 20;
}

/** Convert a decimal zoom factor (1.0 = 100 %) to an integer percentage string. */
export function zoomToPercent(zoom: number): number {
  return Math.round(zoom * 100);
}

/** Persist the current zoom level for the next launch. */
export function saveRestartZoom(zoom: number): void {
  try {
    localStorage.setItem(STORAGE_KEY, String(zoom));
  } catch {
    // localStorage may be unavailable in some WebView configurations.
  }
}

/** Retrieve the persisted zoom level, defaulting to 1.0 (100 %). */
export function getRestartZoom(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === null) return 1.0;
    const n = parseFloat(raw);
    return Number.isFinite(n) && n > 0 ? n : 1.0;
  } catch {
    return 1.0;
  }
}
