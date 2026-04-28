import { useEffect, useState } from 'react';

// Returns true if the given media query matches. Re-renders when it
// changes. SSR-safe (returns false on first render before window exists).
export function useMediaQuery(query) {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return false;
    return window.matchMedia(query).matches;
  });
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia(query);
    const onChange = (e) => setMatches(e.matches);
    mql.addEventListener('change', onChange);
    setMatches(mql.matches);
    return () => mql.removeEventListener('change', onChange);
  }, [query]);
  return matches;
}

// Convenience helpers — call them to know which layout bucket we're in.
export const useIsNarrow = () => useMediaQuery('(max-width: 1023px)');
export const useIsMedium = () => useMediaQuery('(min-width: 1024px) and (max-width: 1279px)');
export const useIsWide = () => useMediaQuery('(min-width: 1280px)');
