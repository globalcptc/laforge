/**
 * Kept out of theme-toggle.tsx on purpose: that file is 'use client', and a string exported from
 * a client module reaches a server component (the root layout) as a client reference, not text.
 */
export const THEME_STORAGE_KEY = 'theme';
export const DARK_QUERY = '(prefers-color-scheme: dark)';

/**
 * Runs inline in <head> before first paint so a dark-mode user never sees a light flash.
 * `localStorage.theme` holds an explicit choice; no entry means follow the OS.
 * Keep it dependency-free and tiny — it is inlined into every page.
 */
export const themeInitScript = `try{var t=localStorage.getItem('${THEME_STORAGE_KEY}');if(t==='dark'||(t!=='light'&&matchMedia('${DARK_QUERY}').matches))document.documentElement.dataset.theme='dark'}catch(e){}`;
