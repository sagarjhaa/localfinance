export const FONTS = {
  headline: '"Newsreader", "Instrument Serif", serif',
  body: '"Manrope", "Hanken Grotesk", sans-serif',
  mono: '"IBM Plex Mono", "JetBrains Mono", "Courier New", monospace',
};

// Sarvam-inspired warm palette. Replaces the old stone-cold gray scale.
// Use sparingly — saffron is precious. Body text is ink, backgrounds are
// paper, dividers are rule. moss/ember only for amounts and status.
export const COLORS = {
  // Foundation
  paper:   '#f8f5ef', // warm sand background
  surface: '#ffffff', // crisp paper-white cards on top of paper
  ink:     '#1a1715', // warm black, not pure black — primary text
  ash:     '#6b6359', // muted warm gray — eyebrows, captions, secondary text
  rule:    'rgba(26,23,21,0.08)', // thin warm divider line

  // Accent
  saffron:    '#c87431', // THE accent. Buttons, links, focus rings, the ✦ glyph
  saffronBg:  '#fbeede', // soft tint for hovered cells, subtle highlights

  // Semantic
  moss:    '#5a7a4a', // muted sage — income / positive amounts only
  mossBg:  '#eef3ea',
  ember:   '#b04a30', // deep terracotta — overspend, anomalies, errors only
  emberBg: '#fbe6dd',

  // Backwards-compat shims so existing inline styles still resolve until
  // they're migrated. These map old tokens to the closest new ones.
  primary:  '#1a1715',           // -> ink
  white:    '#ffffff',           // -> surface
  stone900: '#1a1715',           // -> ink
  stone700: '#3d3733',           // -> dark ash
  stone500: '#6b6359',           // -> ash
  stone300: '#cdc6bd',           // -> faded paper edge
  stone200: '#e7e1d6',           // -> rule-strength
  stone100: '#f1ece2',           // -> paper tint
  stone50:  '#f8f5ef',           // -> paper
  error:    '#b04a30',           // -> ember
  errorBg:  '#fbe6dd',           // -> emberBg
  green:    '#5a7a4a',           // -> moss
  greenBg:  '#eef3ea',           // -> mossBg
};

export const APP = {
  name: 'LocalFinance',
  tagline: 'The Ethereal Vault',
  glyph: '✦', // recurring motif used by EyebrowHeading
};
