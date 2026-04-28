import React, { useState } from 'react';
import { transactionAPI } from '../api/client';
import { COLORS, FONTS } from '../theme';

// EvidenceList renders a "Why?" toggle. When opened it lazily resolves
// transaction IDs (or accepts pre-resolved sources) into a condensed
// table: date, description, amount, account. Used under insight cards
// and chat answers to show provenance — *which* transactions support
// the finding or answer.
//
// Props:
//   ids        — array of transaction UUIDs to fetch on first open
//   sources    — pre-resolved [{id, date, description, amount, category, type}]
//                from the chat path; if present, no lookup is fired
//   label      — toggle text (default "Why?")
export default function EvidenceList({ ids = [], sources = null, label = 'Why?' }) {
  const [open, setOpen] = useState(false);
  const [rows, setRows] = useState(sources);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState('');

  const total = sources ? sources.length : ids.length;
  if (total === 0) return null;

  const onToggle = async () => {
    const next = !open;
    setOpen(next);
    if (!next || rows) return;
    if (sources) { setRows(sources); return; }
    setLoading(true); setErr('');
    try {
      const res = await transactionAPI.lookup(ids);
      setRows(Array.isArray(res.data?.transactions) ? res.data.transactions : []);
    } catch (e) {
      setErr(e.message || 'Could not load transactions');
    } finally { setLoading(false); }
  };

  return (
    <div style={{ marginTop: 8 }}>
      <button
        type="button"
        onClick={onToggle}
        style={{
          background: 'transparent',
          border: 'none',
          color: COLORS.saffron,
          cursor: 'pointer',
          fontFamily: FONTS.body,
          fontSize: 12,
          fontWeight: 700,
          letterSpacing: '0.05em',
          textTransform: 'uppercase',
          padding: 0,
        }}
        aria-expanded={open}
      >
        {open ? '▾' : '▸'} {label} ({total})
      </button>
      {open && (
        <div style={{
          marginTop: 8,
          padding: '8px 10px',
          border: `1px solid ${COLORS.rule}`,
          borderRadius: 8,
          background: COLORS.paper,
          fontSize: 12,
          fontFamily: FONTS.body,
        }}>
          {loading && <div style={{ color: COLORS.ash }}>Loading…</div>}
          {err && <div style={{ color: COLORS.ember }}>{err}</div>}
          {!loading && !err && rows && rows.length === 0 && (
            <div style={{ color: COLORS.ash, fontStyle: 'italic' }}>No matching transactions found.</div>
          )}
          {!loading && rows && rows.length > 0 && (
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ color: COLORS.ash, textAlign: 'left' }}>
                  <th style={th}>Date</th>
                  <th style={th}>Account</th>
                  <th style={th}>Description</th>
                  <th style={{ ...th, textAlign: 'right' }}>Amount</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r, i) => (
                  <tr key={r.id || i} style={{ borderTop: `1px solid ${COLORS.rule}` }}>
                    <td style={td}>{fmtDate(r.date)}</td>
                    <td style={td}>{r.account || '—'}</td>
                    <td style={td}>{r.description}</td>
                    <td style={{ ...td, textAlign: 'right', fontFamily: FONTS.mono, fontVariantNumeric: 'tabular-nums', color: r.amount < 0 ? COLORS.moss : COLORS.ink }}>
                      {fmtMoney(r.amount)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}

const th = { padding: '4px 6px', fontSize: 11, fontWeight: 700, letterSpacing: '0.05em', textTransform: 'uppercase' };
const td = { padding: '4px 6px', verticalAlign: 'top' };

function fmtDate(s) {
  if (!s) return '';
  // Backend sends RFC3339 / ISO; chat sources send YYYY-MM-DD strings.
  const d = new Date(s);
  if (isNaN(d.getTime())) return s;
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}
function fmtMoney(n) {
  if (typeof n !== 'number') return '';
  const sign = n < 0 ? '-' : '';
  return sign + '$' + Math.abs(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}
