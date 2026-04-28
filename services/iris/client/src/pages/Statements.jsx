import React, { useEffect, useState, useCallback } from 'react';
import { documentAPI } from '../api/client';
import { FONTS, COLORS, APP } from '../theme';
import { useIsNarrow, useIsMedium } from '../hooks/useMediaQuery';

// Statements page: a chronological list of every uploaded statement,
// grouped by the *transaction period* the statement covers (not the
// upload date). Click a row to expand its full transaction list.
//
// Backend contract:
//   GET /api/v1/documents/?user_id=X
//     → { documents: [{id, original_filename, status, error_message,
//                       created_at, txn_count, txn_period_start,
//                       txn_period_end, total_spend, total_income,
//                       account_name}, ...] }
//   GET /api/v1/transactions/by-document?document_id=X
//     → { transactions: [{...full Transaction with Account preloaded}], count }

const Statements = ({ user, onLogout }) => {
  const isNarrow = useIsNarrow();
  const isMedium = useIsMedium();
  const S = styles(isNarrow, isMedium);
  const userId = String(user?.id || '');

  const [documents, setDocuments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [openId, setOpenId] = useState(null);
  // documentId → transactions[] cache. Keeps already-fetched lists around
  // when toggling rows so the second open is instant.
  const [txByDoc, setTxByDoc] = useState({});
  const [txLoading, setTxLoading] = useState({});
  const [showFailed, setShowFailed] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setError('');
    try {
      const res = await documentAPI.list(userId);
      setDocuments(Array.isArray(res.data?.documents) ? res.data.documents : []);
    } catch (e) {
      setError(e.message || 'Failed to load statements');
    } finally { setLoading(false); }
  }, [userId]);

  useEffect(() => { if (userId) load(); }, [userId, load]);

  const toggle = async (docId) => {
    const next = openId === docId ? null : docId;
    setOpenId(next);
    if (!next || txByDoc[docId]) return;
    setTxLoading((s) => ({ ...s, [docId]: true }));
    try {
      const res = await documentAPI.getTransactions(docId);
      setTxByDoc((s) => ({ ...s, [docId]: Array.isArray(res.data?.transactions) ? res.data.transactions : [] }));
    } catch (e) {
      setTxByDoc((s) => ({ ...s, [docId]: [] }));
    } finally { setTxLoading((s) => ({ ...s, [docId]: false })); }
  };

  // Split successful from failed/in-flight. The main list shows only
  // documents that produced transactions; failures get a separate
  // collapsible section so they don't pollute the chronological view
  // but are still discoverable for re-upload / cleanup.
  const successful = documents.filter((d) => d.status === 'processed' && d.txn_count > 0);
  const failed = documents.filter((d) => d.status === 'error' || (d.status === 'processed' && d.txn_count === 0));
  const inflight = documents.filter((d) => d.status === 'processing' || d.status === 'uploaded');
  const groups = groupByMonth(successful);

  return (
    <div style={S.page}>
      <aside style={S.sidebar}>
        <div style={isNarrow ? { padding: 0, marginBottom: 0, marginRight: 8 } : { padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>{APP.name}</h1>
          {!isNarrow && <p style={S.sidebarTier}>{APP.tagline}</p>}
        </div>
        <nav style={isNarrow ? { display: 'flex', flexDirection: 'row', gap: 4, alignItems: 'center' } : {}}>
          <div onClick={() => window.location.href = '/dashboard'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128196;</span><span style={{ fontSize: 14 }}>Dashboard</span>
          </div>
          <div onClick={() => window.location.href = '/statements'} style={S.navActive}>
            <span style={{ fontSize: 20 }}>&#128203;</span><span style={{ fontSize: 14, fontWeight: 700 }}>Statements</span>
          </div>
          <div onClick={() => window.location.href = '/insights'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128161;</span><span style={{ fontSize: 14 }}>Insights</span>
          </div>
          <div onClick={() => window.location.href = '/month-review'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128197;</span><span style={{ fontSize: 14 }}>This Month</span>
          </div>
          <div onClick={() => window.location.href = '/chat'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128172;</span><span style={{ fontSize: 14 }}>Chat</span>
          </div>
        </nav>
        <div style={S.sidebarFooter}>
          <div onClick={() => window.location.href = '/profile'} style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer', flex: 1 }}>
            <div style={S.avatar}>{user?.first_name?.[0] || 'U'}</div>
            <div>
              <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>{user?.first_name || ''} {user?.last_name || ''}</p>
              <p style={{ fontSize: 10, color: COLORS.stone500, margin: 0 }}>View profile</p>
            </div>
          </div>
        </div>
      </aside>

      <main style={S.main}>
        <header style={S.topBar}>
          <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: 0 }}>Statements</h2>
        </header>

        <div style={S.content}>
          <header style={{ marginBottom: 32 }}>
            <h2 style={S.pageTitle}>Every statement you've uploaded.</h2>
            <p style={{ color: COLORS.stone500, maxWidth: 540 }}>
              Grouped by the month the statement covers. Click any row to see its transactions.
            </p>
          </header>

          {error && <div style={S.error}>{error}</div>}
          {loading && <div style={S.empty}>Loading…</div>}

          {!loading && groups.length === 0 && inflight.length === 0 && failed.length === 0 && (
            <div style={S.empty}>
              No statements uploaded yet. <a href="/dashboard" style={{ color: COLORS.saffron }}>Upload one</a>.
            </div>
          )}

          {inflight.length > 0 && (
            <section style={{ marginBottom: 24, padding: '12px 16px', background: COLORS.saffronBg, borderRadius: 12, border: `1px solid ${COLORS.rule}` }}>
              <p style={{ margin: 0, fontSize: 13, color: COLORS.stone700 }}>
                ⏳ {inflight.length} statement{inflight.length === 1 ? '' : 's'} still parsing in the background. The list will update when they're ready.
              </p>
            </section>
          )}

          {failed.length > 0 && (
            <section style={{ marginBottom: 32 }}>
              <button
                type="button"
                onClick={() => setShowFailed((s) => !s)}
                style={{ background: 'transparent', border: 'none', color: COLORS.ember, cursor: 'pointer', fontFamily: FONTS.body, fontSize: 13, fontWeight: 700, padding: 0, marginBottom: 8 }}
              >
                {showFailed ? '▾' : '▸'} {failed.length} failed upload{failed.length === 1 ? '' : 's'}
              </button>
              {showFailed && (
                <ul style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {failed.map((d) => (
                    <li key={d.id} style={{ ...S.card, borderColor: COLORS.emberBg, padding: '10px 14px' }}>
                      <div style={{ fontFamily: FONTS.headline, fontSize: 14, marginBottom: 2 }}>{d.original_filename || d.id}</div>
                      <div style={{ fontSize: 11, color: COLORS.stone500 }}>uploaded {fmtDate(d.created_at)}</div>
                      {d.error_message && (
                        <div style={{ marginTop: 6, fontSize: 12, color: COLORS.ember }}>{d.error_message}</div>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}

          {groups.map((g) => (
            <section key={g.month} style={{ marginBottom: 32 }}>
              <h3 style={S.monthHeading}>{g.label}</h3>
              <ul style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
                {g.docs.map((d) => {
                  const open = openId === d.id;
                  const txns = txByDoc[d.id];
                  return (
                    <li key={d.id} style={S.card}>
                      <div onClick={() => toggle(d.id)} style={S.cardHeader}>
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                            <span style={S.toggleArrow}>{open ? '▾' : '▸'}</span>
                            <strong style={{ fontFamily: FONTS.headline, fontSize: 16, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                              {d.original_filename || d.id}
                            </strong>
                            {d.status !== 'processed' && (
                              <span style={{ ...S.statusBadge, background: d.status === 'error' ? COLORS.emberBg : COLORS.stone100, color: d.status === 'error' ? COLORS.ember : COLORS.stone700 }}>
                                {d.status}
                              </span>
                            )}
                          </div>
                          <div style={{ fontSize: 12, color: COLORS.stone500, display: 'flex', flexWrap: 'wrap', gap: 12 }}>
                            {d.account_name && <span>{d.account_name}</span>}
                            {d.txn_period_start && d.txn_period_end && (
                              <span>{fmtDate(d.txn_period_start)} → {fmtDate(d.txn_period_end)}</span>
                            )}
                            <span>{d.txn_count} {d.txn_count === 1 ? 'transaction' : 'transactions'}</span>
                            <span>uploaded {fmtDate(d.created_at)}</span>
                          </div>
                        </div>
                        <div style={{ textAlign: 'right', fontFamily: FONTS.mono, fontVariantNumeric: 'tabular-nums', fontSize: 13 }}>
                          <div style={{ color: COLORS.ink }}>{fmtMoney(d.total_spend)}</div>
                          {d.total_income > 0 && (
                            <div style={{ color: COLORS.moss, fontSize: 11 }}>+{fmtMoney(d.total_income)}</div>
                          )}
                        </div>
                      </div>

                      {open && (
                        <div style={S.expanded}>
                          {d.error_message && (
                            <div style={S.errorInline}>{d.error_message}</div>
                          )}
                          {txLoading[d.id] && <div style={{ color: COLORS.stone500, fontSize: 13 }}>Loading transactions…</div>}
                          {!txLoading[d.id] && Array.isArray(txns) && txns.length === 0 && (
                            <div style={{ color: COLORS.stone500, fontSize: 13, fontStyle: 'italic' }}>No transactions for this statement.</div>
                          )}
                          {!txLoading[d.id] && Array.isArray(txns) && txns.length > 0 && (
                            <div style={{ overflowX: 'auto' }}>
                              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
                                <thead>
                                  <tr style={{ color: COLORS.stone500, textAlign: 'left' }}>
                                    <th style={th}>Date</th>
                                    <th style={th}>Description</th>
                                    <th style={th}>Category</th>
                                    <th style={th}>Account</th>
                                    <th style={{ ...th, textAlign: 'right' }}>Amount</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {txns.map((t) => (
                                    <tr key={t.id} style={{ borderTop: `1px solid ${COLORS.rule}` }}>
                                      <td style={td}>{fmtDate(t.date)}</td>
                                      <td style={td}>{t.description}</td>
                                      <td style={td}>{t.category || '—'}</td>
                                      <td style={td}>{t.account?.name || t.account?.institution || '—'}</td>
                                      <td style={{ ...td, textAlign: 'right', fontFamily: FONTS.mono, fontVariantNumeric: 'tabular-nums', color: t.amount < 0 ? COLORS.moss : COLORS.ink }}>
                                        {fmtMoney(t.amount)}
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          )}
                        </div>
                      )}
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </div>
      </main>
    </div>
  );
};

// groupByMonth produces a list of {month, label, docs} buckets in
// descending date order. month is YYYY-MM. label is e.g. "April 2026".
function groupByMonth(documents) {
  const buckets = {};
  for (const d of documents) {
    const stamp = d.txn_period_end || d.txn_period_start || d.created_at;
    const m = monthKey(stamp);
    if (!buckets[m]) buckets[m] = [];
    buckets[m].push(d);
  }
  const months = Object.keys(buckets).sort().reverse();
  return months.map((m) => ({ month: m, label: monthLabel(m), docs: buckets[m] }));
}

function monthKey(s) {
  if (!s) return 'unknown';
  const d = new Date(s);
  if (isNaN(d.getTime())) return 'unknown';
  return d.toISOString().slice(0, 7);
}

function monthLabel(m) {
  if (!m || m === 'unknown') return 'Date unknown';
  const [y, mm] = m.split('-').map(Number);
  if (!y || !mm) return m;
  const d = new Date(Date.UTC(y, mm - 1, 1));
  return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric', timeZone: 'UTC' });
}

function fmtDate(s) {
  if (!s) return '';
  const d = new Date(s);
  if (isNaN(d.getTime())) return s;
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function fmtMoney(n) {
  if (typeof n !== 'number') return '';
  const sign = n < 0 ? '-' : '';
  return sign + '$' + Math.abs(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

const th = { padding: '6px 8px', fontSize: 11, fontWeight: 700, letterSpacing: '0.05em', textTransform: 'uppercase' };
const td = { padding: '6px 8px', verticalAlign: 'top', color: COLORS.stone700 };

const styles = (isNarrow, isMedium) => ({
  page: { display: 'flex', flexDirection: isNarrow ? 'column' : 'row', minHeight: '100vh', fontFamily: FONTS.body, color: COLORS.primary, background: COLORS.white },
  sidebar: isNarrow
    ? { width: '100%', position: 'sticky', top: 0, zIndex: 40, background: COLORS.white, borderBottom: `1px solid ${COLORS.stone100}`, display: 'flex', flexDirection: 'row', alignItems: 'center', padding: '12px 16px', gap: 16, overflowX: 'auto' }
    : { width: isMedium ? 200 : 256, height: '100vh', position: 'fixed', left: 0, top: 0, zIndex: 40, background: COLORS.white, borderRight: `1px solid ${COLORS.stone100}`, display: 'flex', flexDirection: 'column', paddingTop: 32, paddingBottom: 32 },
  sidebarLogo: { fontFamily: FONTS.headline, fontSize: isNarrow ? 16 : 20, fontWeight: 700, margin: 0 },
  sidebarTier: { fontSize: 10, letterSpacing: '0.2em', textTransform: 'uppercase', color: COLORS.stone500, fontWeight: 700, marginTop: 4 },
  navActive: isNarrow
    ? { display: 'flex', alignItems: 'center', gap: 6, padding: '8px 12px', color: COLORS.primary, fontWeight: 700, background: COLORS.stone50, borderRadius: 8, cursor: 'pointer', whiteSpace: 'nowrap' }
    : { display: 'flex', alignItems: 'center', gap: 16, padding: isMedium ? '12px 20px' : '12px 32px', color: COLORS.primary, fontWeight: 700, background: COLORS.stone50, borderRight: `4px solid ${COLORS.primary}`, cursor: 'pointer' },
  navItem: isNarrow
    ? { display: 'flex', alignItems: 'center', gap: 6, padding: '8px 12px', color: COLORS.stone500, cursor: 'pointer', whiteSpace: 'nowrap' }
    : { display: 'flex', alignItems: 'center', gap: 16, padding: isMedium ? '12px 20px' : '12px 32px', color: COLORS.stone500, cursor: 'pointer' },
  sidebarFooter: isNarrow
    ? { marginLeft: 'auto', padding: 0 }
    : { marginTop: 'auto', padding: isMedium ? '24px 20px' : '24px 32px', borderTop: `1px solid ${COLORS.stone100}` },
  avatar: { width: isNarrow ? 32 : 40, height: isNarrow ? 32 : 40, borderRadius: '50%', background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, fontWeight: 600, textTransform: 'uppercase' },
  main: { flex: 1, marginLeft: isNarrow ? 0 : (isMedium ? 200 : 256), minHeight: '100vh', background: COLORS.white },
  topBar: { position: 'sticky', top: 0, zIndex: 30, background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(12px)', borderBottom: `1px solid ${COLORS.stone100}`, padding: isNarrow ? '12px 20px' : (isMedium ? '16px 32px' : '16px 48px') },
  content: { maxWidth: 'clamp(320px, 90vw, 1080px)', margin: '0 auto', padding: isNarrow ? '32px 20px 48px' : (isMedium ? '48px 32px 64px' : '64px 48px 80px') },
  pageTitle: { fontFamily: FONTS.headline, fontSize: isNarrow ? 28 : (isMedium ? 34 : 40), fontWeight: 500, letterSpacing: '-0.02em', marginBottom: 8 },
  monthHeading: { fontFamily: FONTS.headline, fontSize: 20, fontWeight: 500, margin: '0 0 12px 0', color: COLORS.stone700 },
  card: { borderRadius: 12, border: `1px solid ${COLORS.stone200}`, background: COLORS.white, overflow: 'hidden' },
  cardHeader: { display: 'flex', alignItems: 'center', gap: 16, padding: '14px 16px', cursor: 'pointer' },
  toggleArrow: { color: COLORS.saffron, fontSize: 14, fontWeight: 700, width: 12, display: 'inline-block' },
  statusBadge: { fontSize: 10, letterSpacing: '0.1em', textTransform: 'uppercase', fontWeight: 700, padding: '2px 8px', borderRadius: 999 },
  expanded: { borderTop: `1px solid ${COLORS.rule}`, padding: '12px 16px', background: COLORS.paper },
  errorInline: { padding: '8px 12px', background: COLORS.emberBg, color: COLORS.ember, borderRadius: 8, fontSize: 12, marginBottom: 12 },
  empty: { padding: 40, textAlign: 'center', color: COLORS.stone500, border: `1px dashed ${COLORS.stone200}`, borderRadius: 12 },
  error: { padding: '12px 20px', background: COLORS.errorBg, border: '1px solid #fe8983', color: '#752121', borderRadius: 12, fontSize: 14, marginBottom: 24 },
});

export default Statements;
