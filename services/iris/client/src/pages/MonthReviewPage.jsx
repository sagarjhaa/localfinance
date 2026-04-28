import React, { useEffect, useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { monthReviewAPI } from '../api/client';
import { FONTS, COLORS, APP } from '../theme';
import { useIsNarrow, useIsMedium } from '../hooks/useMediaQuery';
import EvidenceList from '../components/EvidenceList';

// "2026-04" -> "April 2026". Falls back to the raw string if parsing fails.
function formatPeriod(period) {
  if (!period) return '';
  const m = /^(\d{4})-(\d{1,2})$/.exec(period);
  if (!m) return period;
  const year = parseInt(m[1], 10);
  const month = parseInt(m[2], 10);
  if (month < 1 || month > 12) return period;
  const d = new Date(Date.UTC(year, month - 1, 1));
  return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric', timeZone: 'UTC' });
}

const MonthReviewPage = ({ user, onLogout }) => {
  const isNarrow = useIsNarrow();
  const isMedium = useIsMedium();
  const S = styles(isNarrow, isMedium);
  const { period } = useParams();
  const navigate = useNavigate();
  const userId = String(user?.id || '');
  const [review, setReview] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [regenerating, setRegenerating] = useState(false);
  const [periods, setPeriods] = useState([]);

  // Fetch the list of months that have transactions. If the URL has no
  // period and there's at least one available, redirect to the newest.
  useEffect(() => {
    if (!userId) return;
    let cancelled = false;
    monthReviewAPI.periods(userId).then((res) => {
      if (cancelled) return;
      const list = Array.isArray(res.data?.periods) ? res.data.periods : [];
      setPeriods(list);
      if (!period && list.length > 0) {
        navigate(`/month-review/${list[0]}`, { replace: true });
      }
    }).catch(() => { /* non-fatal — page still works without dropdown */ });
    return () => { cancelled = true; };
  }, [userId, period, navigate]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await monthReviewAPI.get(period, userId);
      setReview(res.data || null);
    } catch (e) {
      setError(e.message || 'Failed to load month-in-review');
    } finally {
      setLoading(false);
    }
  }, [period, userId]);

  useEffect(() => { if (userId && period) load(); }, [userId, period, load]);

  const handlePeriodChange = (e) => {
    const next = e.target.value;
    if (next && next !== period) navigate(`/month-review/${next}`);
  };

  const regenerate = async () => {
    setRegenerating(true);
    setError('');
    try {
      await monthReviewAPI.invalidate(period, userId);
      // Re-fetch in place — don't toggle the page-level `loading` flag, so
      // the existing review stays on screen until the new one arrives.
      // Just swaps the review value when the GET resolves.
      const res = await monthReviewAPI.get(period, userId);
      setReview(res.data || null);
    } catch (e) {
      setError(e.message || 'Failed to regenerate');
    } finally {
      setRegenerating(false);
    }
  };

  const insights = Array.isArray(review?.insights) ? review.insights : [];
  const perInsight = review?.narrative?.per_insight || {};

  return (
    <div style={S.page}>
      <aside style={S.sidebar}>
        <div style={isNarrow ? { padding: 0, marginBottom: 0, marginRight: 8 } : { padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>{APP.name}</h1>
          {!isNarrow && <p style={S.sidebarTier}>{APP.tagline}</p>}
        </div>
        <nav style={isNarrow ? { display: 'flex', flexDirection: 'row', gap: 4, alignItems: 'center' } : {}}>
          <div onClick={() => window.location.href = '/dashboard'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128196;</span>
            <span style={{ fontSize: 14 }}>Dashboard</span>
          </div>
          <div onClick={() => window.location.href = '/insights'} style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128161;</span>
            <span style={{ fontSize: 14 }}>Insights</span>
          </div>
          <div onClick={() => window.location.href = '/month-review'} style={S.navActive}>
            <span style={{ fontSize: 20 }}>&#128197;</span>
            <span style={{ fontSize: 14, fontWeight: 700 }}>This Month</span>
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
          <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: 0 }}>This Month</h2>
        </header>

        <div style={S.content}>
          <header style={{ display: 'flex', flexDirection: isNarrow ? 'column' : 'row', gap: isNarrow ? 16 : 0, justifyContent: 'space-between', alignItems: isNarrow ? 'flex-start' : 'flex-end', marginBottom: 24 }}>
            <div>
              <p style={{ fontSize: 11, letterSpacing: '0.2em', textTransform: 'uppercase', color: COLORS.stone500, fontWeight: 700, margin: 0 }}>
                MONTH IN REVIEW
              </p>
              {periods.length > 0 ? (
                <select
                  value={period || periods[0]}
                  onChange={handlePeriodChange}
                  aria-label="Select month"
                  style={S.periodSelect}
                >
                  {periods.map((p) => (
                    <option key={p} value={p}>{formatPeriod(p)}</option>
                  ))}
                </select>
              ) : (
                <h2 style={S.pageTitle}>{formatPeriod(period)}</h2>
              )}
            </div>
            <button
              onClick={regenerate}
              disabled={regenerating || loading}
              style={{ ...S.regenerateBtn, opacity: (regenerating || loading) ? 0.5 : 1 }}
            >
              {regenerating ? 'Reading again.' : 'Read it again from scratch.'}
            </button>
          </header>

          {error && <div style={S.error}>Something went sideways. Try again?</div>}
          {loading && <div style={S.empty}>Pulling the month together.</div>}

          {!loading && !error && review && (
            <>
              {review.narrative?.overall && (
                <div style={S.narrativeBox}>
                  <p style={{ margin: 0, fontFamily: FONTS.headline, fontSize: 16, lineHeight: 1.6 }}>
                    {review.narrative.overall}
                  </p>
                </div>
              )}

              {insights.length === 0 && (
                <div style={{ ...S.empty, fontFamily: FONTS.headline, fontStyle: 'italic' }}>Nothing notable to flag — your spending looks steady.</div>
              )}

              {insights.map((ins, i) => {
                const id = ins.key || `ins-${i}`;
                const body = perInsight[ins.key] || ins.description;
                return (
                  <section key={id} style={S.card}>
                    <h3 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: '0 0 8px 0' }}>
                      {ins.title || 'Insight'}
                    </h3>
                    <p style={{ margin: 0, color: COLORS.stone700, lineHeight: 1.6 }}>
                      {body}
                    </p>
                    <EvidenceList ids={Array.isArray(ins.evidence_ids) ? ins.evidence_ids : []} label="Which transactions?" />
                  </section>
                );
              })}
            </>
          )}
        </div>
      </main>
    </div>
  );
};

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
  content: { maxWidth: 'clamp(320px, 80vw, 880px)', margin: '0 auto', padding: isNarrow ? '32px 20px 48px' : (isMedium ? '48px 32px 64px' : '64px 48px 80px') },
  pageTitle: { fontFamily: FONTS.headline, fontSize: isNarrow ? 28 : (isMedium ? 34 : 40), fontWeight: 500, letterSpacing: '-0.02em', margin: '8px 0 0 0' },
  periodSelect: {
    fontFamily: FONTS.headline,
    fontSize: isNarrow ? 28 : (isMedium ? 34 : 40),
    fontWeight: 500,
    letterSpacing: '-0.02em',
    margin: '8px 0 0 0',
    padding: '0 8px 0 0',
    background: 'transparent',
    color: COLORS.ink,
    border: 'none',
    outline: 'none',
    cursor: 'pointer',
    appearance: 'menulist',
    maxWidth: '100%',
  },
  narrativeBox: { padding: '20px 24px', borderRadius: 16, background: COLORS.stone50, border: `1px solid ${COLORS.stone100}`, marginBottom: 24 },
  card: { padding: 20, borderRadius: 12, border: `1px solid ${COLORS.stone200}`, background: COLORS.white, marginBottom: 12 },
  empty: { padding: 40, textAlign: 'center', color: COLORS.stone500, border: `1px dashed ${COLORS.stone200}`, borderRadius: 12 },
  error: { padding: '12px 20px', background: COLORS.errorBg, border: '1px solid #fe8983', color: '#752121', borderRadius: 12, fontSize: 14, marginBottom: 24 },
  regenerateBtn: { padding: '10px 20px', background: COLORS.primary, color: COLORS.white, fontSize: 13, fontWeight: 700, border: 'none', borderRadius: 8, cursor: 'pointer' },
});

export default MonthReviewPage;
