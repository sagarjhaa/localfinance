import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { LIGHT_PALETTE, DARK_PALETTE, applyPalette, COLORS, FONTS } from '../theme';

const STORAGE_KEY = 'lf:theme';

const ThemeContext = createContext({
  theme: 'light',
  toggle: () => {},
  setTheme: () => {},
});

export function ThemeProvider({ children }) {
  const [theme, setThemeState] = useState(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved === 'light' || saved === 'dark') return saved;
    } catch (_) { /* localStorage may be unavailable */ }
    return 'light';
  });

  // Apply palette + dataset on every theme change.
  useEffect(() => {
    applyPalette(theme === 'dark' ? DARK_PALETTE : LIGHT_PALETTE);
    document.documentElement.dataset.theme = theme;
    try { localStorage.setItem(STORAGE_KEY, theme); } catch (_) {}
  }, [theme]);

  const setTheme = useCallback((next) => {
    if (next === 'light' || next === 'dark') setThemeState(next);
  }, []);
  const toggle = useCallback(() => {
    setThemeState((t) => (t === 'dark' ? 'light' : 'dark'));
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, toggle, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}

// ThemeToggle is a small fixed-position button rendered globally for any
// authenticated page. It sits at the bottom-left, aligned with where the
// sidebar's user-profile area lives, so visually it reads as part of the
// sidebar without us having to edit each page's sidebar markup.
export function ThemeToggle() {
  const { theme, toggle } = useTheme();
  const isDark = theme === 'dark';
  return (
    <button
      type="button"
      onClick={toggle}
      title={isDark ? 'Switch to day view' : 'Switch to night view'}
      aria-label={isDark ? 'Switch to day view' : 'Switch to night view'}
      style={{
        position: 'fixed',
        left: 16,
        bottom: 16,
        zIndex: 50,
        width: 36,
        height: 36,
        borderRadius: '50%',
        border: `1px solid ${COLORS.rule}`,
        background: COLORS.surface,
        color: COLORS.ink,
        fontSize: 16,
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
        fontFamily: FONTS.body,
      }}
    >
      {isDark ? '☀' : '☾'}
    </button>
  );
}
