import React from 'react';
import { FONTS, COLORS, APP } from '../theme';

// EyebrowHeading is the page-hero pattern: a saffron ✦ glyph + tracked
// uppercase eyebrow + serif headline + optional sub. Drop in at the top
// of any page so the heading hierarchy stays consistent.
//
// Example:
//   <EyebrowHeading
//     eyebrow="MONTH IN REVIEW"
//     title={`This is what ${monthName} looked like.`}
//     sub="Hand me your statement and I'll show you where it went."
//   />
export default function EyebrowHeading({
  eyebrow,
  title,
  sub,
  align = 'left',
  size = 'lg',
  glyph = APP.glyph,
  style,
}) {
  const titleSize = size === 'sm' ? 28 : size === 'md' ? 36 : 48;
  return (
    <header style={{ marginBottom: 32, textAlign: align, ...style }}>
      {eyebrow && (
        <div style={{
          fontFamily: FONTS.body,
          fontSize: 11,
          fontWeight: 600,
          letterSpacing: '0.18em',
          textTransform: 'uppercase',
          color: COLORS.ash,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          justifyContent: align === 'center' ? 'center' : 'flex-start',
          marginBottom: 14,
        }}>
          <span style={{ color: COLORS.saffron, fontSize: 13 }}>{glyph}</span>
          <span>{eyebrow}</span>
        </div>
      )}
      <h1 style={{
        fontFamily: FONTS.headline,
        fontSize: titleSize,
        lineHeight: 1.05,
        fontWeight: 400,
        margin: 0,
        color: COLORS.ink,
        letterSpacing: '-0.01em',
      }}>
        {title}
      </h1>
      {sub && (
        <p style={{
          fontFamily: FONTS.body,
          fontSize: 15,
          color: COLORS.ash,
          margin: '12px 0 0 0',
          maxWidth: 520,
          marginLeft: align === 'center' ? 'auto' : 0,
          marginRight: align === 'center' ? 'auto' : 0,
          lineHeight: 1.5,
        }}>
          {sub}
        </p>
      )}
    </header>
  );
}
