// accessibility.ts — ARIA helpers, focus-trap, reduced-motion, high-contrast,
// and keyboard-shortcut screen-reader formatting. Referenced by every panel that
// renders interactive lists (background tasks, remote workers, etc.).
//
// All utilities are pure functions or stable hooks so they can be colocated
// without adding bundle tax.

import { useEffect, useRef, useState } from "react";

// ── Role attribute constants ─────────────────────────────────────────────────

/** Roles for interactive list regions (keyboard-navigable item collections). */
export const A11Y_ROLES = {
  LISTBOX: "listbox",
  OPTION: "option",
  TABLIST: "tablist",
  TAB: "tab",
  TABPANEL: "tabpanel",
  TOOLBAR: "toolbar",
  STATUS: "status",
  ALERT: "alert",
  LOG: "log",
  DIALOG: "dialog",
  ALERTDIALOG: "alertdialog",
  MENU: "menu",
  MENUITEM: "menuitem",
  GRID: "grid",
  ROW: "row",
  GRIDCELL: "gridcell",
} as const;

export type A11yRole = (typeof A11Y_ROLES)[keyof typeof A11Y_ROLES];

// ── ARIA label generation ────────────────────────────────────────────────────

/**
 * Build a descriptive ARIA label for a background task row.
 * Example output: "Subagent explore — running with 3 tools — latest: Read"
 */
export function taskRowLabel(params: {
  label: string;
  status: string;
  toolCount: number;
  latestTool: string;
}): string {
  const parts: string[] = [params.label];
  parts.push(`— ${params.status}`);
  if (params.toolCount > 0) {
    parts.push(`with ${params.toolCount} tool${params.toolCount !== 1 ? "s" : ""}`);
  }
  if (params.latestTool) {
    parts.push(`— latest: ${params.latestTool}`);
  }
  return parts.join(" ");
}

/**
 * Build a descriptive ARIA label for a remote worker row.
 * Example output: "Remote worker us-east-1 — online, 3 of 8 jobs active, latency 42ms"
 */
export function remoteRowLabel(params: {
  name: string;
  online: boolean;
  activeJobs: number;
  maxJobs: number;
  latencyMs: number;
  secure: boolean;
}): string {
  const parts: string[] = [`Remote worker ${params.name}`];
  parts.push(`— ${params.online ? "online" : "offline"}`);
  if (params.online) {
    parts.push(
      `${params.activeJobs} of ${params.maxJobs} jobs active`,
      `latency ${params.latencyMs}ms`,
    );
  }
  parts.push(params.secure ? "connection secure" : "connection not secure");
  return parts.join(", ");
}

/**
 * Build a live-region announcement for a panel state change.
 */
export function panelStateAnnouncement(panelName: string, state: string): string {
  return `${panelName}: ${state}`;
}

// ── Keyboard shortcut formatting for screen readers ──────────────────────────

export type ModifierKey = "Ctrl" | "Alt" | "Shift" | "Meta" | "Cmd";

export interface ShortcutDef {
  /** The visible key label shown in the UI. */
  key: string;
  /** OS-level modifiers (Cmd on macOS maps to Meta). */
  modifiers?: ModifierKey[];
}

const MAC_NAMES: Record<string, string> = {
  Meta: "Command",
  Ctrl: "Control",
  Alt: "Option",
  Shift: "Shift",
  Enter: "Return",
  Escape: "Escape",
  Backspace: "Delete",
  ArrowUp: "Up Arrow",
  ArrowDown: "Down Arrow",
  ArrowLeft: "Left Arrow",
  ArrowRight: "Right Arrow",
};

const WIN_NAMES: Record<string, string> = {
  Meta: "Windows",
  Ctrl: "Control",
  Alt: "Alt",
  Shift: "Shift",
  Enter: "Enter",
  Escape: "Escape",
  Backspace: "Backspace",
  ArrowUp: "Up Arrow",
  ArrowDown: "Down Arrow",
  ArrowLeft: "Left Arrow",
  ArrowRight: "Right Arrow",
};

/**
 * Format a shortcut for screen reader announcement.
 * On macOS, "Cmd+Enter" → "Command Return".
 * On Windows, "Ctrl+K" → "Control K".
 */
export function formatShortcutForScreenReader(
  shortcut: ShortcutDef,
  platform: "mac" | "win" = "win",
): string {
  const names = platform === "mac" ? MAC_NAMES : WIN_NAMES;
  const parts: string[] = [];
  if (shortcut.modifiers) {
    for (const mod of shortcut.modifiers) {
      parts.push(names[mod] ?? mod);
    }
  }
  parts.push(names[shortcut.key] ?? shortcut.key);
  return parts.join(" ");
}

/**
 * Announce a keyboard shortcut hint as hidden text for screen readers.
 * Returns an aria-label string suitable for a hint element.
 */
export function shortcutHintAriaLabel(
  shortcuts: ShortcutDef[],
  platform: "mac" | "win" = "win",
): string {
  const names = platform === "mac" ? MAC_NAMES : WIN_NAMES;
  return shortcuts
    .map((s) => {
      const parts: string[] = [];
      if (s.modifiers) {
        for (const mod of s.modifiers) parts.push(names[mod] ?? mod);
      }
      parts.push(names[s.key] ?? s.key);
      return parts.join("+");
    })
    .join(", ");
}

// ── Focus trap management ────────────────────────────────────────────────────

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * Trap focus within a container element. Returns focus to the previously
 * focused element when the trap is removed.
 *
 * Usage in a modal/panel:
 * ```tsx
 * const containerRef = useRef<HTMLDivElement>(null);
 * useFocusTrap(containerRef, isOpen);
 * ```
 */
export function useFocusTrap(
  containerRef: React.RefObject<HTMLElement | null>,
  active: boolean,
): void {
  const previousFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!active) {
      // Restore focus to the element that had it before the trap was armed.
      previousFocusRef.current?.focus();
      previousFocusRef.current = null;
      return;
    }

    // Save current focus so we can restore it later.
    previousFocusRef.current = document.activeElement as HTMLElement | null;

    const container = containerRef.current;
    if (!container) return;

    // Focus the first focusable element inside the container.
    const first = container.querySelector<HTMLElement>(FOCUSABLE_SELECTOR);
    first?.focus();

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== "Tab") return;

      const focusable = Array.from(
        container!.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
      ).filter((el) => el.offsetParent !== null); // skip hidden elements

      if (focusable.length === 0) {
        e.preventDefault();
        return;
      }

      const firstEl = focusable[0];
      const lastEl = focusable[focusable.length - 1];
      const activeEl = document.activeElement;

      if (e.shiftKey) {
        if (activeEl === firstEl) {
          e.preventDefault();
          lastEl.focus();
        }
      } else {
        if (activeEl === lastEl) {
          e.preventDefault();
          firstEl.focus();
        }
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      previousFocusRef.current?.focus();
      previousFocusRef.current = null;
    };
  }, [active, containerRef]);
}

// ── Reduced motion media query hook ──────────────────────────────────────────

const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

/**
 * Returns true when the user has requested reduced motion via their OS
 * accessibility settings (prefers-reduced-motion: reduce).
 * Re-renders when the preference changes.
 */
export function usePrefersReducedMotion(): boolean {
  const [prefersReduced, setPrefersReduced] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.matchMedia(REDUCED_MOTION_QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    const mql = window.matchMedia(REDUCED_MOTION_QUERY);
    const handler = (e: MediaQueryListEvent) => setPrefersReduced(e.matches);
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, []);

  return prefersReduced;
}

// ── High contrast mode detection ─────────────────────────────────────────────

const HIGH_CONTRAST_QUERY = "(forced-colors: active)";
const PREFERS_CONTRAST_MORE_QUERY = "(prefers-contrast: more)";
const PREFERS_CONTRAST_LESS_QUERY = "(prefers-contrast: less)";

export interface ContrastPreference {
  /** Windows High Contrast Mode / forced-colors: active. */
  forcedColors: boolean;
  /** prefers-contrast: more (user wants higher contrast). */
  prefersMore: boolean;
  /** prefers-contrast: less (user wants lower contrast). */
  prefersLess: boolean;
}

/**
 * Returns the current OS-level contrast preferences.
 * Re-renders when any preference changes.
 */
export function useContrastPreference(): ContrastPreference {
  function read(): ContrastPreference {
    if (typeof window === "undefined")
      return { forcedColors: false, prefersMore: false, prefersLess: false };
    return {
      forcedColors: window.matchMedia(HIGH_CONTRAST_QUERY).matches,
      prefersMore: window.matchMedia(PREFERS_CONTRAST_MORE_QUERY).matches,
      prefersLess: window.matchMedia(PREFERS_CONTRAST_LESS_QUERY).matches,
    };
  }

  const [pref, setPref] = useState<ContrastPreference>(read);

  useEffect(() => {
    if (typeof window === "undefined") return;

    const hcMql = window.matchMedia(HIGH_CONTRAST_QUERY);
    const moreMql = window.matchMedia(PREFERS_CONTRAST_MORE_QUERY);
    const lessMql = window.matchMedia(PREFERS_CONTRAST_LESS_QUERY);

    const handler = () => setPref(read());
    hcMql.addEventListener("change", handler);
    moreMql.addEventListener("change", handler);
    lessMql.addEventListener("change", handler);
    return () => {
      hcMql.removeEventListener("change", handler);
      moreMql.removeEventListener("change", handler);
      lessMql.removeEventListener("change", handler);
    };
  }, []);

  return pref;
}

/**
 * Returns true when either forced-colors or prefers-contrast: more is active,
 * meaning the UI should apply high-contrast styling.
 */
export function useHighContrast(): boolean {
  const { forcedColors, prefersMore } = useContrastPreference();
  return forcedColors || prefersMore;
}

// ── Touch target helpers ─────────────────────────────────────────────────────

/** Minimum touch target size in px (WCAG 2.5.5 requires 44x44). */
export const MIN_TOUCH_TARGET = 44;

/**
 * Returns true if the user is likely on a touch device (coarse pointer).
 * Stable after first read — does not re-render on change.
 */
export function useIsTouchDevice(): boolean {
  const [isTouch, setIsTouch] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.matchMedia("(pointer: coarse)").matches;
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    const mql = window.matchMedia("(pointer: coarse)");
    const handler = (e: MediaQueryListEvent) => setIsTouch(e.matches);
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, []);

  return isTouch;
}

// ── ID generation for ARIA attributes ────────────────────────────────────────

let idCounter = 0;

/**
 * Generate a stable unique ID for aria-labelledby / aria-describedby
 * relationships. Avoids collisions with React's useId in concurrent trees.
 */
export function a11yId(prefix: string): string {
  idCounter += 1;
  return `a11y-${prefix}-${idCounter}`;
}

// ── Live region announcement helper ──────────────────────────────────────────

/**
 * Imperatively announce a message to screen readers via an aria-live region.
 * The container must exist in the DOM with aria-live="polite" or "assertive".
 */
export function announceToScreenReader(
  containerId: string,
  message: string,
): void {
  if (typeof document === "undefined") return;
  const region = document.getElementById(containerId);
  if (!region) return;
  // Clear and repopulate so the same message is announced twice if needed.
  region.textContent = "";
  // Force layout so the clear is processed before re-insertion.
  void region.offsetHeight;
  region.textContent = message;
}
