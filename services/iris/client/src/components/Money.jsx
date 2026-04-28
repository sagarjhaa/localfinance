import React, { useEffect, useRef, useState } from 'react';
import { FONTS, COLORS } from '../theme';

// Money renders a currency amount in IBM Plex Mono with subtle semantic
// coloring. Counts up from 0 to the target value on first mount (Stripe-
// style ticker). Pass `tone="auto"` (default) to color-by-sign, or override
// with "ink" / "moss" / "ember" / "ash" for explicit cases.
export default function Money({
  amount = 0,
  size = 14,
  tone = 'auto',
  weight = 600,
  animate = true,
  prefix = '$',
  className,
  style,
}) {
  const [shown, setShown] = useState(animate ? 0 : amount);
  const startedAt = useRef(0);
  const raf = useRef(null);

  useEffect(() => {
    if (!animate) { setShown(amount); return; }
    const target = Number(amount) || 0;
    const start = shown;
    const delta = target - start;
    const duration = 700;
    startedAt.current = performance.now();
    const tick = (now) => {
      const t = Math.min(1, (now - startedAt.current) / duration);
      // ease-out cubic
      const eased = 1 - Math.pow(1 - t, 3);
      setShown(start + delta * eased);
      if (t < 1) raf.current = requestAnimationFrame(tick);
    };
    raf.current = requestAnimationFrame(tick);
    return () => raf.current && cancelAnimationFrame(raf.current);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [amount, animate]);

  const abs = Math.abs(shown);
  const formatted = abs.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  const sign = shown < 0 ? '-' : '';

  let color = COLORS.ink;
  if (tone === 'auto') {
    if (amount > 0.01) color = COLORS.ink;        // expense — neutral
    else if (amount < -0.01) color = COLORS.moss; // income — sage (negative amounts in our convention are credits)
  } else if (tone === 'ink') color = COLORS.ink;
  else if (tone === 'ash') color = COLORS.ash;
  else if (tone === 'moss') color = COLORS.moss;
  else if (tone === 'ember') color = COLORS.ember;

  return (
    <span
      className={className}
      style={{
        fontFamily: FONTS.mono,
        fontSize: size,
        fontWeight: weight,
        fontVariantNumeric: 'tabular-nums',
        color,
        whiteSpace: 'nowrap',
        ...style,
      }}
    >
      {sign}{prefix}{formatted}
    </span>
  );
}
