export const FONTS = {
  headline: '"Newsreader", "Instrument Serif", serif',
  body: '"Manrope", "Hanken Grotesk", sans-serif',
  mono: '"IBM Plex Mono", "JetBrains Mono", "Courier New", monospace',
};

// COLORS values are CSS custom-property references. The actual hex values
// live in LIGHT_PALETTE and DARK_PALETTE below and are written to :root by
// ThemeContext at app boot. Anywhere a component does
// `style={{ background: COLORS.paper }}` it now compiles to
// `background: var(--c-paper)`, which the browser resolves against the
// active palette without any re-render.
const v = (name) => `var(--c-${name})`;

export const COLORS = {
  // Foundation
  paper:   v('paper'),
  surface: v('surface'),
  ink:     v('ink'),
  ash:     v('ash'),
  rule:    v('rule'),

  // Accent
  saffron:    v('saffron'),
  saffronBg:  v('saffronBg'),

  // Semantic
  moss:    v('moss'),
  mossBg:  v('mossBg'),
  ember:   v('ember'),
  emberBg: v('emberBg'),

  // Backwards-compat shims so existing inline styles still resolve.
  primary:  v('ink'),
  white:    v('surface'),
  stone900: v('ink'),
  stone700: v('stone700'),
  stone500: v('ash'),
  stone300: v('stone300'),
  stone200: v('rule-strength'),
  stone100: v('stone100'),
  stone50:  v('paper'),
  error:    v('ember'),
  errorBg:  v('emberBg'),
  green:    v('moss'),
  greenBg:  v('mossBg'),
};

// LIGHT_PALETTE — the warm Sarvam-inspired daytime look.
export const LIGHT_PALETTE = {
  paper:           '#f8f5ef',
  surface:         '#ffffff',
  ink:             '#1a1715',
  ash:             '#6b6359',
  rule:            'rgba(26,23,21,0.08)',
  'rule-strength': '#e7e1d6',
  saffron:         '#c87431',
  saffronBg:       '#fbeede',
  moss:            '#5a7a4a',
  mossBg:          '#eef3ea',
  ember:           '#b04a30',
  emberBg:         '#fbe6dd',
  stone700:        '#3d3733',
  stone300:        '#cdc6bd',
  stone100:        '#f1ece2',
};

// DARK_PALETTE — same vibe at night. Warm dark instead of pure black so
// long sessions don't feel cold; saffron stays readable on warm dark.
export const DARK_PALETTE = {
  paper:           '#1a1614',
  surface:         '#231e1a',
  ink:             '#f1ece2',
  ash:             '#a89f93',
  rule:            'rgba(241,236,226,0.10)',
  'rule-strength': '#3a342f',
  saffron:         '#e08a4a',
  saffronBg:       '#3d2818',
  moss:            '#9bb585',
  mossBg:          '#283220',
  ember:           '#e07458',
  emberBg:         '#3d2018',
  stone700:        '#d6cdbf',
  stone300:        '#5a5147',
  stone100:        '#2a2421',
};

// applyPalette writes a palette's hex values to :root as --c-* custom
// properties. Called by ThemeContext on mount and when the user toggles.
export function applyPalette(palette) {
  const root = document.documentElement;
  for (const [k, val] of Object.entries(palette)) {
    root.style.setProperty(`--c-${k}`, val);
  }
}

export const APP = {
  name: 'LocalFinance',
  tagline: 'The Ethereal Vault',
  glyph: '✦',
};
