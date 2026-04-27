import React, { useState, useCallback, useEffect, useRef } from 'react';
import { uploadAPI, documentAPI, proxyAPI } from '../api/client';
import { FONTS, COLORS, APP } from '../theme';

const categoryIcons = {
  Food: '🍽️', Transport: '✈️', Shopping: '🛍️', Entertainment: '🎬',
  Utilities: '⚡', Housing: '🏠', Income: '💰', Transfer: '↗️',
  Health: '💊', Cash: '🏧', EMI: '📋', Education: '📚', Other: '⚙️',
};

const Dashboard = ({ user, onLogout }) => {
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [processing, setProcessing] = useState(false);
  const [documentId, setDocumentId] = useState(null);
  const [allTransactions, setAllTransactions] = useState([]);
  const [visibleTransactions, setVisibleTransactions] = useState([]);
  const [currentTicker, setCurrentTicker] = useState(null);
  const [revealing, setRevealing] = useState(false);
  const [error, setError] = useState('');
  // The Static-vs-AI dual parse view was removed once Logos started routing every
  // upload through Sophia/Ollama — the "static" column was always identical to
  // the saved transactions, just labeled differently.
  const pollRef = useRef(null);
  const revealRef = useRef(null);

  // Poll for document status
  useEffect(() => {
    if (!documentId || !processing) return;
    const poll = async () => {
      try {
        const res = await documentAPI.getStatus(documentId);
        if (res.data.status === 'processed') {
          setProcessing(false);
          const txnRes = await documentAPI.getTransactions(documentId);
          const txns = txnRes.data.transactions || [];
          setAllTransactions(txns);
          if (txns.length > 0) {
            setRevealing(true);
            revealTransactionsOneByOne(txns);
          }
        } else if (res.data.status === 'error') {
          setProcessing(false);
          setError(res.data.error_message || 'Processing failed');
        }
      } catch (e) { /* keep polling */ }
    };
    pollRef.current = setInterval(poll, 2000);
    poll();
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [documentId, processing]);

  // Reveal transactions one by one with ticker animation
  const revealTransactionsOneByOne = (txns) => {
    let idx = 0;
    const reveal = () => {
      if (idx >= txns.length) {
        setCurrentTicker(null);
        setRevealing(false);
        return;
      }
      const txn = txns[idx];
      setCurrentTicker(txn);
      setVisibleTransactions(prev => [txn, ...prev]);
      idx++;
      revealRef.current = setTimeout(reveal, 600);
    };
    reveal();
  };

  useEffect(() => {
    return () => { if (revealRef.current) clearTimeout(revealRef.current); };
  }, []);

  const handleDragOver = useCallback((e) => { e.preventDefault(); setIsDragging(true); }, []);
  const handleDragLeave = useCallback(() => setIsDragging(false), []);
  const handleDrop = useCallback(async (e) => {
    e.preventDefault(); setIsDragging(false);
    const files = Array.from(e.dataTransfer.files);
    if (files.length > 0) await uploadFile(files[0]);
  }, []);

  const handleBrowse = () => {
    const input = document.createElement('input');
    input.type = 'file'; input.accept = '.pdf,.csv,.xlsx,.xls,.txt';
    input.onchange = (e) => {
      if (e.target.files.length > 0) uploadFile(e.target.files[0]);
    };
    input.click();
  };

  const uploadFile = async (file) => {
    setUploading(true); setError(''); setAllTransactions([]); setVisibleTransactions([]);
    setCurrentTicker(null); setDocumentId(null); setRevealing(false);
    try {
      const res = await uploadAPI.single(file);
      if (res.data.document_id) {
        setDocumentId(res.data.document_id);
        setProcessing(true);
      } else { setError('No document ID returned'); }
    } catch (err) { setError(err.message || 'Upload failed'); }
    finally { setUploading(false); }
  };

  const reset = () => {
    setAllTransactions([]); setVisibleTransactions([]); setCurrentTicker(null);
    setDocumentId(null); setError(''); setProcessing(false); setRevealing(false);
    if (pollRef.current) clearInterval(pollRef.current);
    if (revealRef.current) clearTimeout(revealRef.current);
  };

  const fmtAmt = (a) => {
    const abs = Math.abs(a);
    const s = abs.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    return a < 0 ? `-$${s}` : `$${s}`;
  };
  const fmtDate = (d) => {
    try { return new Date(d).toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: 'numeric' }); }
    catch { return d; }
  };

  const isActive = uploading || processing || revealing;
  const hasResults = visibleTransactions.length > 0;

  // Infer the dominant period (YYYY-MM) from parsed transactions so we can link
  // to /month-review/<period>. Picks the most common YYYY-MM among valid dates.
  const inferredPeriod = (() => {
    if (!allTransactions.length) return null;
    const counts = {};
    for (const t of allTransactions) {
      if (!t?.date) continue;
      const d = new Date(t.date);
      if (isNaN(d.getTime())) continue;
      const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
      counts[key] = (counts[key] || 0) + 1;
    }
    const entries = Object.entries(counts);
    if (!entries.length) return null;
    entries.sort((a, b) => b[1] - a[1]);
    return entries[0][0];
  })();

  return (
    <div style={S.page}>
      {/* Sidebar */}
      <aside style={S.sidebar}>
        <div style={{ padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>{APP.name}</h1>
          <p style={S.sidebarTier}>{APP.tagline}</p>
        </div>
        <nav>
          <div
            onClick={() => window.location.href = '/dashboard'}
            style={S.navActive}
          >
            <span style={{ fontSize: 20 }}>&#128196;</span>
            <span style={{ fontSize: 14, fontWeight: 700 }}>Statement Upload</span>
          </div>
          <div
            onClick={() => window.location.href = '/dashboard'}
            style={S.navItem}
          >
            <span style={{ fontSize: 20 }}>&#128274;</span>
            <span style={{ fontSize: 14 }}>The Vault</span>
          </div>
          <div
            onClick={() => window.location.href = '/insights'}
            style={S.navItem}
          >
            <span style={{ fontSize: 20 }}>&#128161;</span>
            <span style={{ fontSize: 14 }}>Insights</span>
          </div>
          <div
            onClick={() => window.location.href = '/month-review'}
            style={S.navItem}
          >
            <span style={{ fontSize: 20 }}>&#128197;</span>
            <span style={{ fontSize: 14 }}>Month in Review</span>
          </div>
          <div
            onClick={() => window.location.href = '/chat'}
            style={S.navItem}
          >
            <span style={{ fontSize: 20 }}>&#128172;</span>
            <span style={{ fontSize: 14 }}>Ollama Chat</span>
          </div>
        </nav>
        <div style={S.sidebarFooter}>
          <div
            onClick={() => window.location.href = '/profile'}
            style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer', flex: 1 }}
          >
            <div style={S.avatar}>{user?.first_name?.[0] || 'U'}</div>
            <div>
              <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>{user?.first_name || ''} {user?.last_name || ''}</p>
              <p style={{ fontSize: 10, color: COLORS.stone500, margin: 0 }}>View profile</p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main */}
      <main style={S.main}>
        {/* Top header */}
        <header style={S.topBar}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
            <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: 0 }}>Dashboard</h2>
            <nav style={{ display: 'flex', gap: 24 }}>
              <span style={{ fontSize: 14, fontWeight: 700, borderBottom: `2px solid ${COLORS.primary}`, paddingBottom: 4 }}>Overview</span>
              <span style={{ fontSize: 14, color: COLORS.stone500 }}>Analytics</span>
              <span style={{ fontSize: 14, color: COLORS.stone500 }}>Reports</span>
            </nav>
          </div>
        </header>

        <div style={S.content}>
          {/* Title */}
          <header style={{ marginBottom: 48 }}>
            <h2 style={S.pageTitle}>Statement Ingestion Zone</h2>
            <p style={{ color: '#737373', maxWidth: 480 }}>Upload your financial records for immediate neural processing and institutional-grade ledgering.</p>
          </header>

          {error && <div style={S.error}>{error}</div>}

          {/* Rolling Transaction Ticker */}
          {(isActive || currentTicker) && (
            <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 24 }}>
              <div style={S.ticker}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={S.pulseDot} />
                  <span style={S.tickerLabel}>Real-time Processing</span>
                </div>
                {currentTicker && (<>
                  <div style={{ width: 1, height: 16, background: '#e0e0e0' }} />
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <div style={S.tickerIcon}>{categoryIcons[currentTicker.category] || '⚙️'}</div>
                    <div>
                      <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>{currentTicker.description}</p>
                      <p style={{ fontSize: 10, color: COLORS.stone500, margin: 0 }}>{currentTicker.category || 'Processing...'}</p>
                    </div>
                    <span style={{ fontFamily: FONTS.headline, fontSize: 14, fontWeight: 500, marginLeft: 16 }}>{fmtAmt(currentTicker.amount)}</span>
                  </div>
                </>)}
                {!currentTicker && isActive && (
                  <span style={{ fontSize: 12, color: COLORS.stone500, marginLeft: 16 }}>Waiting for Logos to parse...</span>
                )}
              </div>
            </div>
          )}

          {/* Drop Zone */}
          <section style={S.uploadSection}>
            <div style={{ padding: 16 }}>
              <div
                style={{ ...S.dropZone, ...(isDragging ? S.dropZoneActive : {}), ...(isActive ? { opacity: 0.5, pointerEvents: 'none' } : {}) }}
                onDragOver={handleDragOver} onDragLeave={handleDragLeave} onDrop={handleDrop}
              >
                <div style={S.uploadIcon}>
                  {isActive
                    ? <div style={{ width: 36, height: 36, border: '3px solid #e5e5e5', borderTop: `3px solid ${COLORS.primary}`, borderRadius: '50%', animation: 'spin 1s linear infinite' }} />
                    : <span style={{ fontSize: 36 }}>&#9729;</span>}
                </div>
                <h3 style={S.dropTitle}>{isActive ? 'Processing Statement...' : 'Drop PDF or CSV Statements'}</h3>
                <p style={{ color: '#737373', textAlign: 'center', maxWidth: 380 }}>
                  {isActive ? 'Logos is extracting and classifying your transactions.' : 'Drag and drop institutional records here for immediate ingestion and neural classification.'}
                </p>
                {!isActive && <button onClick={handleBrowse} style={S.browseBtn}>Browse Files</button>}
              </div>
            </div>
          </section>

          {/* Processed Ledger Header */}
          {hasResults && (
            <section style={{ marginTop: 80 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 24 }}>
                <div>
                  <h3 style={S.ledgerTitle}>Processed Ledger</h3>
                  <p style={{ fontSize: 14, color: COLORS.stone500, marginTop: 4 }}>Institutional records post-enrichment</p>
                </div>
                <button onClick={reset} style={S.exportBtn}>+ New Upload</button>
              </div>

              {/* Upload-success CTA: link to month-in-review for the inferred period */}
              {!processing && !revealing && (
                <a
                  href={inferredPeriod ? `/month-review/${inferredPeriod}` : '/insights'}
                  style={{
                    display: 'inline-flex', alignItems: 'center', gap: 10,
                    padding: '12px 20px', marginBottom: 24,
                    background: COLORS.stone50, border: `1px solid ${COLORS.stone200}`,
                    color: COLORS.primary, textDecoration: 'none', borderRadius: 999,
                    fontSize: 13, fontWeight: 600,
                  }}
                >
                  <span style={{ fontSize: 16 }}>&#128161;</span>
                  {inferredPeriod
                    ? `View Month in Review →`
                    : `View Insights →`}
                </a>
              )}
            </section>
          )}

          {/* Parsed Transactions */}
          {allTransactions.length > 0 && (
            <div style={{ marginTop: hasResults ? 0 : 24 }}>
              <div style={{ background: COLORS.white, borderRadius: 12, border: '1px solid ' + COLORS.stone200, overflow: 'hidden' }}>
                <div style={{ padding: '16px 20px', borderBottom: '1px solid ' + COLORS.stone200, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <h3 style={{ fontFamily: FONTS.headline, fontSize: 16, fontWeight: 700, margin: 0 }}>Parsed Transactions</h3>
                    <p style={{ fontFamily: FONTS.body, fontSize: 11, color: COLORS.stone500, margin: '4px 0 0 0' }}>{allTransactions.length} transactions extracted by the local AI model</p>
                  </div>
                  <span style={{ fontSize: 10, fontWeight: 600, color: COLORS.green, background: COLORS.stone100, padding: '4px 8px', borderRadius: 4 }}>PERSISTED</span>
                </div>
                <div style={{ maxHeight: 500, overflowY: 'auto' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse', fontFamily: FONTS.body, fontSize: 12 }}>
                    <thead>
                      <tr style={{ background: COLORS.stone50 }}>
                        <th style={{ textAlign: 'left', padding: '8px 12px', fontWeight: 600, color: COLORS.stone500, fontSize: 10, textTransform: 'uppercase' }}>Date</th>
                        <th style={{ textAlign: 'left', padding: '8px 12px', fontWeight: 600, color: COLORS.stone500, fontSize: 10, textTransform: 'uppercase' }}>Description</th>
                        <th style={{ textAlign: 'left', padding: '8px 12px', fontWeight: 600, color: COLORS.stone500, fontSize: 10, textTransform: 'uppercase' }}>Category</th>
                        <th style={{ textAlign: 'right', padding: '8px 12px', fontWeight: 600, color: COLORS.stone500, fontSize: 10, textTransform: 'uppercase' }}>Amount</th>
                      </tr>
                    </thead>
                    <tbody>
                      {allTransactions.map((t, i) => (
                        <tr key={i} style={{ borderBottom: '1px solid ' + COLORS.stone100 }}>
                          <td style={{ padding: '8px 12px', whiteSpace: 'nowrap', color: COLORS.stone500 }}>{t.date ? new Date(t.date).toLocaleDateString() : ''}</td>
                          <td style={{ padding: '8px 12px', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{t.description}</td>
                          <td style={{ padding: '8px 12px' }}>
                            <span style={{ fontSize: 10, padding: '2px 6px', borderRadius: 4, background: COLORS.stone100, color: COLORS.stone700 }}>
                              {categoryIcons[t.category] || '⚙️'} {t.category || 'Other'}
                            </span>
                          </td>
                          <td style={{ padding: '8px 12px', textAlign: 'right', fontWeight: 600, color: t.amount < 0 ? COLORS.green : COLORS.stone900 }}>
                            {fmtAmt(t.amount)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

            </div>
          )}
        </div>
      </main>
      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: translateY(0); } }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
      `}</style>
    </div>
  );
};

const S = {
  page: { display: 'flex', minHeight: '100vh', fontFamily: FONTS.body, color: COLORS.primary, background: COLORS.white },
  sidebar: { width: 256, height: '100vh', position: 'fixed', left: 0, top: 0, zIndex: 40, background: COLORS.white, borderRight: `1px solid ${COLORS.stone100}`, display: 'flex', flexDirection: 'column', paddingTop: 32, paddingBottom: 32 },
  sidebarLogo: { fontFamily: FONTS.headline, fontSize: 20, fontWeight: 700, margin: 0 },
  sidebarTier: { fontSize: 10, letterSpacing: '0.2em', textTransform: 'uppercase', color: COLORS.stone500, fontWeight: 700, marginTop: 4 },
  navActive: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: COLORS.primary, fontWeight: 700, background: COLORS.stone50, borderRight: `4px solid ${COLORS.primary}`, textDecoration: 'none' },
  navItem: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: COLORS.stone500, textDecoration: 'none' },
  sidebarFooter: { marginTop: 'auto', padding: '24px 32px', borderTop: `1px solid ${COLORS.stone100}`, display: 'flex', alignItems: 'center', gap: 12 },
  avatar: { width: 40, height: 40, borderRadius: '50%', background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16, fontWeight: 600, textTransform: 'uppercase' },
  main: { flex: 1, marginLeft: 256, minHeight: '100vh', background: COLORS.white },
  topBar: { position: 'sticky', top: 0, zIndex: 30, background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(12px)', borderBottom: `1px solid ${COLORS.stone100}`, padding: '16px 48px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  content: { maxWidth: 1024, margin: '0 auto', padding: '96px 48px 80px' },
  pageTitle: { fontFamily: FONTS.headline, fontSize: 48, fontWeight: 500, letterSpacing: '-0.02em', marginBottom: 8 },
  error: { padding: '12px 20px', background: COLORS.errorBg, border: '1px solid #fe8983', color: '#752121', borderRadius: 12, fontSize: 14, marginBottom: 24 },
  ticker: { display: 'inline-flex', alignItems: 'center', gap: 24, padding: '16px 32px', background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(24px)', border: '1px solid #fff', boxShadow: '0 24px 60px -12px rgba(0,0,0,0.12), inset 0 1px 1px rgba(255,255,255,0.8)', borderRadius: 999 },
  pulseDot: { width: 8, height: 8, borderRadius: '50%', background: '#22c55e', animation: 'pulse 2s infinite' },
  tickerLabel: { fontSize: 10, letterSpacing: '0.15em', textTransform: 'uppercase', fontWeight: 700, color: COLORS.stone500 },
  tickerIcon: { width: 32, height: 32, borderRadius: 8, background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 },
  uploadSection: { background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(24px)', border: '1px solid #fff', boxShadow: '0 24px 60px -12px rgba(0,0,0,0.12), inset 0 1px 1px rgba(255,255,255,0.8)', borderRadius: 24, overflow: 'hidden', marginBottom: 0 },
  dropZone: { border: '2px dashed #e0e0e0', borderRadius: 12, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '96px 48px', background: 'rgba(255,255,255,0.4)', cursor: 'pointer', transition: 'all 0.3s' },
  dropZoneActive: { borderColor: COLORS.primary, background: 'rgba(245,245,245,0.6)' },
  uploadIcon: { width: 80, height: 80, borderRadius: '50%', background: '#fff', boxShadow: '0 10px 30px rgba(0,0,0,0.1), inset 0 0 0 1px rgba(255,255,255,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 32 },
  dropTitle: { fontFamily: FONTS.headline, fontSize: 28, marginBottom: 16 },
  browseBtn: { marginTop: 40, padding: '12px 32px', background: COLORS.primary, color: COLORS.white, fontSize: 14, fontWeight: 700, border: 'none', borderRadius: 8, cursor: 'pointer', boxShadow: '0 4px 12px rgba(0,0,0,0.15)' },
  ledgerTitle: { fontFamily: FONTS.headline, fontSize: 28, fontWeight: 500 },
  exportBtn: { padding: '10px 24px', background: COLORS.primary, color: COLORS.white, fontSize: 14, fontWeight: 700, border: 'none', borderRadius: 8, cursor: 'pointer', boxShadow: '0 4px 12px rgba(0,0,0,0.15)', display: 'flex', alignItems: 'center', gap: 8 },
  tableContainer: { background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(24px)', borderRadius: 24, border: `1px solid ${COLORS.stone100}`, overflow: 'hidden' },
  table: { width: '100%', borderCollapse: 'collapse' },
  th: { padding: '20px 32px', fontSize: 10, letterSpacing: '0.15em', textTransform: 'uppercase', fontWeight: 700, color: COLORS.stone500, borderBottom: `1px solid ${COLORS.stone100}`, textAlign: 'left' },
  tdDate: { padding: '16px 32px', fontSize: 12, fontWeight: 500, color: '#737373', borderBottom: `1px solid ${COLORS.stone100}` },
  tdDesc: { padding: '16px 32px', borderBottom: `1px solid ${COLORS.stone100}` },
  txnIcon: { width: 32, height: 32, borderRadius: '50%', background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 },
  tdAmount: { padding: '16px 32px', textAlign: 'right', borderBottom: `1px solid ${COLORS.stone100}`, fontSize: 14 },
  tdStatus: { padding: '16px 32px', borderBottom: `1px solid ${COLORS.stone100}` },
  verifiedBadge: { display: 'inline-block', padding: '4px 8px', background: COLORS.greenBg, color: '#15803d', fontSize: 9, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.08em', borderRadius: 999 },
  tableFooter: { padding: '24px 32px', background: 'rgba(250,250,250,0.3)', borderTop: `1px solid ${COLORS.stone100}` },
};

export default Dashboard;
