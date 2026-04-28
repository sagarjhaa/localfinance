import React, { useEffect, useState, useCallback } from 'react';
import { insightsAPI } from '../api/client';
import { FONTS, COLORS, APP } from '../theme';

const PRIORITY_COLORS = {
  high:   { bg: '#fee2e2', fg: '#991b1b' },
  medium: { bg: '#fef3c7', fg: '#92400e' },
  low:    { bg: '#dcfce7', fg: '#166534' },
};

const InsightsPage = ({ user, onLogout }) => {
  const [insights, setInsights] = useState([]);
  const [narrative, setNarrative] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [dismissing, setDismissing] = useState({});

  const userId = String(user?.id || '');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await insightsAPI.list(userId);
      setInsights(Array.isArray(res.data?.insights) ? res.data.insights : []);
      setNarrative(res.data?.narrative || null);
    } catch (e) {
      setError(e.message || 'Failed to load insights');
    } finally {
      setLoading(false);
    }
  }, [userId]);

  useEffect(() => { if (userId) load(); }, [userId, load]);

  const dismiss = async (insight) => {
    // Defensive: if there's no key and no rule_id, the backend can't persist
    // a dismissal. Just hide it locally so the user isn't stuck looking at it.
    if (!insight.key && !insight.rule_id) {
      setInsights(prev => prev.filter(i => i !== insight));
      return;
    }
    const id = insight.key || insight.rule_id;
    setDismissing(prev => ({ ...prev, [id]: true }));
    try {
      await insightsAPI.dismiss(userId, { insight_key: insight.key, rule_id: insight.rule_id });
      setInsights(prev => prev.filter(i => (i.key || i.rule_id) !== id));
    } catch (e) {
      setError(e.message || 'Failed to dismiss insight');
    } finally {
      setDismissing(prev => { const n = { ...prev }; delete n[id]; return n; });
    }
  };

  return (
    <div style={S.page}>
      <aside style={S.sidebar}>
        <div style={{ padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>{APP.name}</h1>
          <p style={S.sidebarTier}>{APP.tagline}</p>
        </div>
        <nav>
          <div onClick={() => window.location.href = '/dashboard'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128196;</span>
            <span style={{ fontSize: 14 }}>Dashboard</span>
          </div>
          <div onClick={() => window.location.href = '/insights'} style={S.navActive}>
            <span style={{ fontSize: 20 }}>&#128161;</span>
            <span style={{ fontSize: 14, fontWeight: 700 }}>Insights</span>
          </div>
          <div onClick={() => window.location.href = '/month-review'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128197;</span>
            <span style={{ fontSize: 14 }}>This Month</span>
          </div>
          <div onClick={() => window.location.href = '/chat'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128172;</span>
            <span style={{ fontSize: 14 }}>Chat</span>
          </div>
        </nav>
        <div style={S.sidebarFooter}>
          <div onClick={() => window.location.href = '/profile'}
               style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer', flex: 1 }}>
            <div style={S.avatar}>{user?.first_name?.[0] || 'U'}</div>
            <div>
              <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>
                {user?.first_name || ''} {user?.last_name || ''}
              </p>
              <p style={{ fontSize: 10, color: COLORS.stone500, margin: 0 }}>View profile</p>
            </div>
          </div>
        </div>
      </aside>

      <main style={S.main}>
        <header style={S.topBar}>
          <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: 0 }}>Insights</h2>
        </header>

        <div style={S.content}>
          <header style={{ marginBottom: 32 }}>
            <h2 style={S.pageTitle}>What stood out.</h2>
            <p style={{ color: COLORS.stone500, maxWidth: 540 }}>
              Five rules quietly watched your last 90 days. Here's what they noticed.
            </p>
          </header>

          {error && <div style={S.error}>Something went sideways. Try again?</div>}

          {narrative?.overall && (
            <div style={S.narrativeBox}>
              <p style={{ margin: 0, fontFamily: FONTS.headline, fontSize: 16, lineHeight: 1.6 }}>
                {narrative.overall}
              </p>
            </div>
          )}

          {loading && <div style={S.empty}>Reading your last 90 days.</div>}

          {!loading && insights.length === 0 && !error && (
            <div style={S.empty}>
              No insights yet — upload a statement to see what stands out.
            </div>
          )}

          {!loading && insights.length > 0 && (
            <ul style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>
              {insights.map((ins, i) => {
                const id = ins.key || ins.rule_id || `i-${i}`;
                const palette = PRIORITY_COLORS[ins.priority] || PRIORITY_COLORS.medium;
                return (
                  <li key={id} style={S.card}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16 }}>
                      <div style={{ flex: 1 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                          <h3 style={{ fontFamily: FONTS.headline, fontSize: 18, margin: 0 }}>
                            {ins.title || 'Insight'}
                          </h3>
                          {ins.priority && (
                            <span style={{ ...S.badge, background: palette.bg, color: palette.fg }}>
                              {ins.priority}
                            </span>
                          )}
                        </div>
                        <p style={{ margin: '0 0 8px 0', color: COLORS.stone700, lineHeight: 1.5 }}>
                          {ins.description}
                        </p>
                        {ins.action_item && (
                          <p style={{ margin: 0, fontSize: 12, color: COLORS.stone500, fontStyle: 'italic' }}>
                            → {ins.action_item}
                          </p>
                        )}
                      </div>
                      <button
                        onClick={() => dismiss(ins)}
                        disabled={!!dismissing[id]}
                        style={{ ...S.dismissBtn, opacity: dismissing[id] ? 0.5 : 1 }}
                      >
                        {dismissing[id] ? 'Got it.' : 'Dismiss'}
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </main>
    </div>
  );
};

const S = {
  page: { display: 'flex', minHeight: '100vh', fontFamily: FONTS.body, color: COLORS.primary, background: COLORS.white },
  sidebar: { width: 256, height: '100vh', position: 'fixed', left: 0, top: 0, zIndex: 40, background: COLORS.white, borderRight: `1px solid ${COLORS.stone100}`, display: 'flex', flexDirection: 'column', paddingTop: 32, paddingBottom: 32 },
  sidebarLogo: { fontFamily: FONTS.headline, fontSize: 20, fontWeight: 700, margin: 0 },
  sidebarTier: { fontSize: 10, letterSpacing: '0.2em', textTransform: 'uppercase', color: COLORS.stone500, fontWeight: 700, marginTop: 4 },
  navActive: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: COLORS.primary, fontWeight: 700, background: COLORS.stone50, borderRight: `4px solid ${COLORS.primary}`, cursor: 'pointer' },
  navItem: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: COLORS.stone500, cursor: 'pointer' },
  sidebarFooter: { marginTop: 'auto', padding: '24px 32px', borderTop: `1px solid ${COLORS.stone100}` },
  avatar: { width: 40, height: 40, borderRadius: '50%', background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16, fontWeight: 600, textTransform: 'uppercase' },
  main: { flex: 1, marginLeft: 256, minHeight: '100vh', background: COLORS.white },
  topBar: { position: 'sticky', top: 0, zIndex: 30, background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(12px)', borderBottom: `1px solid ${COLORS.stone100}`, padding: '16px 48px' },
  content: { maxWidth: 880, margin: '0 auto', padding: '64px 48px 80px' },
  pageTitle: { fontFamily: FONTS.headline, fontSize: 40, fontWeight: 500, letterSpacing: '-0.02em', marginBottom: 8 },
  narrativeBox: { padding: '20px 24px', borderRadius: 16, background: COLORS.stone50, border: `1px solid ${COLORS.stone100}`, marginBottom: 24 },
  card: { padding: 20, borderRadius: 12, border: `1px solid ${COLORS.stone200}`, background: COLORS.white },
  badge: { fontSize: 10, letterSpacing: '0.1em', textTransform: 'uppercase', fontWeight: 700, padding: '2px 8px', borderRadius: 999 },
  dismissBtn: { padding: '6px 14px', background: COLORS.white, border: `1px solid ${COLORS.stone300}`, color: COLORS.stone700, fontSize: 12, fontWeight: 600, borderRadius: 6, cursor: 'pointer', whiteSpace: 'nowrap' },
  empty: { padding: 40, textAlign: 'center', color: COLORS.stone500, border: `1px dashed ${COLORS.stone200}`, borderRadius: 12 },
  error: { padding: '12px 20px', background: COLORS.errorBg, border: '1px solid #fe8983', color: '#752121', borderRadius: 12, fontSize: 14, marginBottom: 24 },
};

export default InsightsPage;
