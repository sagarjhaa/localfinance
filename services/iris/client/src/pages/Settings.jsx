import React, { useState, useEffect, useCallback } from 'react';
import { proxyAPI } from '../api/client';
import { FONTS, COLORS, APP } from '../theme';

const CATEGORIES = [
  'Food', 'Transport', 'Shopping', 'Entertainment', 'Utilities',
  'Housing', 'Income', 'Transfer', 'Health', 'Cash', 'EMI', 'Education', 'Other',
];

const CATEGORY_COLORS = {
  Food:          { bg: '#f0fdf4', fg: '#15803d' },
  Transport:     { bg: '#eff6ff', fg: '#1d4ed8' },
  Shopping:      { bg: '#fefce8', fg: '#a16207' },
  Entertainment: { bg: '#fdf2f8', fg: '#be185d' },
  Utilities:     { bg: '#f0f9ff', fg: '#0369a1' },
  Housing:       { bg: '#faf5ff', fg: '#7e22ce' },
  Income:        { bg: '#ecfdf5', fg: '#047857' },
  Transfer:      { bg: '#f8fafc', fg: '#475569' },
  Health:        { bg: '#fff1f2', fg: '#be123c' },
  Cash:          { bg: '#fefce8', fg: '#854d0e' },
  EMI:           { bg: '#fef2f2', fg: '#b91c1c' },
  Education:     { bg: '#eef2ff', fg: '#4338ca' },
  Other:         { bg: '#f5f5f4', fg: '#44403c' },
};

const getCategoryColor = (category) => CATEGORY_COLORS[category] || CATEGORY_COLORS.Other;

const Settings = ({ user, onLogout }) => {
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [pattern, setPattern] = useState('');
  const [category, setCategory] = useState('Food');
  const [saving, setSaving] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [editPattern, setEditPattern] = useState('');
  const [editCategory, setEditCategory] = useState('');
  const [models, setModels] = useState([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [savedModel, setSavedModel] = useState('');
  const [modelSaving, setModelSaving] = useState(false);
  const [modelMessage, setModelMessage] = useState('');

  const fetchRules = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await proxyAPI.thesaurus.get('/api/v1/category-rules');
      setRules(res.data.rules || res.data || []);
    } catch (err) {
      setError(err.message || 'Failed to load rules');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  useEffect(() => {
    // Fetch available models
    proxyAPI.sophia.get('/api/v1/models')
      .then((res) => setModels(res.data || []))
      .catch(() => {});

    // Fetch current preference
    proxyAPI.thesaurus.get('/api/v1/preferences', undefined, { skipLogoutOn401: true })
      .then((res) => {
        const model = res.data?.chat_model || 'llama3.2:1b';
        setSelectedModel(model);
        setSavedModel(model);
      })
      .catch(() => {
        setSelectedModel('llama3.2:1b');
        setSavedModel('llama3.2:1b');
      });
  }, []);

  const saveModel = async () => {
    setModelSaving(true);
    setModelMessage('');
    try {
      await proxyAPI.thesaurus.put('/api/v1/preferences', { chat_model: selectedModel });
      setSavedModel(selectedModel);
      setModelMessage('Saved.');
      setTimeout(() => setModelMessage(''), 3000);
    } catch (err) {
      setModelMessage('Failed to save model');
    } finally {
      setModelSaving(false);
    }
  };

  const handleAddRule = async (e) => {
    e.preventDefault();
    if (!pattern.trim() || saving) return;
    setSaving(true);
    try {
      const res = await proxyAPI.thesaurus.post('/api/v1/category-rules', {
        pattern: pattern.trim(),
        category,
      });
      setRules((prev) => [...prev, res.data.rule || res.data]);
      setPattern('');
      setCategory('Food');
    } catch (err) {
      setError(err.message || 'Failed to add rule');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id) => {
    const prev = rules;
    setRules((r) => r.filter((rule) => rule.id !== id));
    try {
      await proxyAPI.thesaurus.delete('/api/v1/category-rules/' + id);
    } catch (err) {
      setRules(prev);
      setError(err.message || 'Failed to delete rule');
    }
  };

  const startEdit = (rule) => {
    setEditingId(rule.id);
    setEditPattern(rule.pattern);
    setEditCategory(rule.category);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditPattern('');
    setEditCategory('');
  };

  const handleUpdate = async (id) => {
    if (!editPattern.trim()) return;
    const prev = rules;
    setRules((r) =>
      r.map((rule) =>
        rule.id === id ? { ...rule, pattern: editPattern.trim(), category: editCategory } : rule
      )
    );
    setEditingId(null);
    try {
      await proxyAPI.thesaurus.put('/api/v1/category-rules/' + id, {
        pattern: editPattern.trim(),
        category: editCategory,
      });
    } catch (err) {
      setRules(prev);
      setError(err.message || 'Failed to update rule');
    }
  };

  const CategoryBadge = ({ cat }) => {
    const c = getCategoryColor(cat);
    return (
      <span
        style={{
          display: 'inline-block',
          padding: '4px 12px',
          borderRadius: 9999,
          fontSize: 12,
          fontWeight: 600,
          background: c.bg,
          color: c.fg,
          letterSpacing: '0.02em',
        }}
      >
        {cat}
      </span>
    );
  };

  return (
    <div style={S.page}>
      {/* Sidebar */}
      <aside style={S.sidebar}>
        <div style={{ padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>{APP.name}</h1>
          <p style={S.sidebarTier}>{APP.tagline}</p>
        </div>
        <nav>
          <a href="/dashboard" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128196;</span>
            <span style={{ fontSize: 14 }}>Dashboard</span>
          </a>
          <a href="/insights" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128161;</span>
            <span style={{ fontSize: 14 }}>Insights</span>
          </a>
          <a href="/month-review" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128197;</span>
            <span style={{ fontSize: 14 }}>This Month</span>
          </a>
          <a href="/chat" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128172;</span>
            <span style={{ fontSize: 14 }}>Chat</span>
          </a>
          <a href="/settings" style={S.navActive}>
            <span style={{ fontSize: 20 }}>&#9881;</span>
            <span style={{ fontSize: 14, fontWeight: 700 }}>Settings</span>
          </a>
        </nav>
        <div style={S.sidebarFooter}>
          <div style={S.avatarCircle}>
            {user?.first_name?.[0] || user?.username?.[0] || 'U'}
          </div>
          <div style={{ flex: 1 }}>
            <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>
              {user?.first_name || user?.username} {user?.last_name || ''}
            </p>
            <button onClick={onLogout} style={S.logoutBtn}>Sign out.</button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main style={S.main}>
        <header style={S.topBar}>
          <div>
            <h2 style={S.pageTitle}>Settings</h2>
            <p style={S.pageSubtitle}>How I sort what comes in.</p>
          </div>
        </header>

        <div style={S.content}>
          {/* Chat Model Section */}
          <div style={{ marginBottom: 40 }}>
            <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, fontWeight: 700, marginBottom: 16 }}>
              Which brain to use.
            </h2>
            <p style={{ fontFamily: FONTS.body, fontSize: 13, color: COLORS.stone500, marginBottom: 16 }}>
              Smaller is quicker. Larger thinks harder.
            </p>
            <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
              <select
                value={selectedModel}
                onChange={(e) => setSelectedModel(e.target.value)}
                style={{
                  flex: 1,
                  padding: '12px 16px',
                  borderRadius: 10,
                  border: '1px solid ' + COLORS.stone200,
                  fontFamily: FONTS.body,
                  fontSize: 14,
                  color: COLORS.stone900 || '#1c1917',
                  background: COLORS.white || '#fff',
                  outline: 'none',
                  cursor: 'pointer',
                }}
              >
                {models.map((m) => (
                  <option key={m.name} value={m.name}>
                    {m.name} ({m.size})
                  </option>
                ))}
                {models.length === 0 && (
                  <option value={selectedModel}>{selectedModel}</option>
                )}
              </select>
              <button
                onClick={saveModel}
                disabled={modelSaving || selectedModel === savedModel}
                style={{
                  padding: '12px 24px',
                  background: selectedModel !== savedModel ? COLORS.primary || '#1A1A1A' : COLORS.stone300 || '#d6d3d1',
                  color: selectedModel !== savedModel ? COLORS.white || '#fff' : COLORS.stone500 || '#78716c',
                  border: 'none',
                  borderRadius: 10,
                  fontFamily: FONTS.body,
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: selectedModel !== savedModel ? 'pointer' : 'default',
                }}
              >
                {modelSaving ? 'Saving.' : 'Save.'}
              </button>
            </div>
            {modelMessage && (
              <div style={{
                marginTop: 8,
                fontFamily: FONTS.body,
                fontSize: 12,
                color: modelMessage.includes('success') ? (COLORS.green || '#16a34a') : (COLORS.error || '#dc2626'),
              }}>
                {modelMessage}
              </div>
            )}
          </div>

          {/* Add rule form */}
          <form onSubmit={handleAddRule} style={S.addForm}>
            <div style={S.formRow}>
              <input
                type="text"
                placeholder="e.g., Safeway, Trader Joe"
                value={pattern}
                onChange={(e) => setPattern(e.target.value)}
                style={S.input}
                disabled={saving}
              />
              <select
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                style={S.select}
                disabled={saving}
              >
                {CATEGORIES.map((cat) => (
                  <option key={cat} value={cat}>{cat}</option>
                ))}
              </select>
              <button
                type="submit"
                disabled={!pattern.trim() || saving}
                style={{
                  ...S.addBtn,
                  ...(!pattern.trim() || saving ? { opacity: 0.4, cursor: 'default' } : {}),
                }}
              >
                {saving ? 'Adding...' : 'Add Rule'}
              </button>
            </div>
          </form>

          {/* Error banner */}
          {error && (
            <div style={S.errorBanner}>
              <span>{error}</span>
              <button onClick={() => setError(null)} style={S.errorDismiss}>
                &#10005;
              </button>
            </div>
          )}

          {/* Rules list */}
          <div style={S.rulesSection}>
            {loading ? (
              <div style={S.emptyState}>
                <p style={S.emptyText}>Loading rules...</p>
              </div>
            ) : rules.length === 0 ? (
              <div style={S.emptyState}>
                <div style={S.emptyIcon}>&#10022;</div>
                <p style={S.emptyTitle}>No custom rules yet</p>
                <p style={S.emptyText}>
                  AI will categorize all transactions automatically.
                </p>
              </div>
            ) : (
              <div style={S.rulesList}>
                {rules.map((rule) => (
                  <div key={rule.id} style={S.ruleRow}>
                    {editingId === rule.id ? (
                      <div style={S.editRow}>
                        <input
                          type="text"
                          value={editPattern}
                          onChange={(e) => setEditPattern(e.target.value)}
                          style={S.editInput}
                          autoFocus
                        />
                        <select
                          value={editCategory}
                          onChange={(e) => setEditCategory(e.target.value)}
                          style={S.editSelect}
                        >
                          {CATEGORIES.map((cat) => (
                            <option key={cat} value={cat}>{cat}</option>
                          ))}
                        </select>
                        <button onClick={() => handleUpdate(rule.id)} style={S.saveBtn}>
                          Save
                        </button>
                        <button onClick={cancelEdit} style={S.cancelBtn}>
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <>
                        <div style={S.ruleInfo}>
                          <span style={S.rulePattern}>{rule.pattern}</span>
                          <CategoryBadge cat={rule.category} />
                        </div>
                        <div style={S.ruleActions}>
                          <button onClick={() => startEdit(rule)} style={S.editBtn}>
                            Edit
                          </button>
                          <button onClick={() => handleDelete(rule.id)} style={S.deleteBtn}>
                            Delete
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Info section */}
          <div style={S.infoSection}>
            <div style={S.infoIcon}>&#9432;</div>
            <p style={S.infoText}>
              When you upload statements, transactions matching your rules are categorized first.
              Everything else is categorized by AI.
            </p>
          </div>
        </div>
      </main>
    </div>
  );
};

const S = {
  // Layout (matches Dashboard / Chat)
  page: { display: 'flex', minHeight: '100vh', fontFamily: FONTS.body, color: COLORS.primary, background: COLORS.white },
  sidebar: { width: 256, height: '100vh', position: 'fixed', left: 0, top: 0, zIndex: 40, background: COLORS.white, borderRight: `1px solid ${COLORS.stone100}`, display: 'flex', flexDirection: 'column', paddingTop: 32, paddingBottom: 32 },
  sidebarLogo: { fontFamily: FONTS.headline, fontSize: 20, fontWeight: 700, margin: 0 },
  sidebarTier: { fontSize: 10, letterSpacing: '0.2em', textTransform: 'uppercase', color: COLORS.stone500, fontWeight: 700, marginTop: 4 },
  navActive: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: COLORS.primary, fontWeight: 700, background: COLORS.stone50, borderRight: `4px solid ${COLORS.primary}`, textDecoration: 'none' },
  navItem: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: COLORS.stone500, textDecoration: 'none' },
  sidebarFooter: { marginTop: 'auto', padding: '24px 32px', borderTop: `1px solid ${COLORS.stone100}`, display: 'flex', alignItems: 'center', gap: 12 },
  avatarCircle: { width: 40, height: 40, borderRadius: '50%', background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16, fontWeight: 600, textTransform: 'uppercase', flexShrink: 0 },
  logoutBtn: { background: 'none', border: 'none', padding: 0, margin: '2px 0 0 0', fontSize: 10, color: COLORS.stone500, cursor: 'pointer', textDecoration: 'underline', fontFamily: FONTS.body },
  main: { flex: 1, marginLeft: 256, minHeight: '100vh', background: 'radial-gradient(circle at 50% 50%, #ffffff 0%, #f2f4f4 100%)' },
  topBar: { position: 'sticky', top: 0, zIndex: 30, background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(12px)', borderBottom: `1px solid ${COLORS.stone100}`, padding: '16px 48px' },
  pageTitle: { fontFamily: FONTS.headline, fontSize: 20, margin: 0, fontWeight: 700 },
  pageSubtitle: { fontSize: 13, color: COLORS.stone500, margin: '4px 0 0 0', fontFamily: FONTS.body },

  // Content area
  content: { padding: '32px 48px', maxWidth: 800 },

  // Add rule form
  addForm: { marginBottom: 24 },
  formRow: { display: 'flex', gap: 12, alignItems: 'center' },
  input: {
    flex: 1, padding: '12px 16px', fontSize: 14, fontFamily: FONTS.body,
    border: `1px solid ${COLORS.stone200}`, borderRadius: 10, outline: 'none', color: COLORS.primary,
    background: COLORS.white, transition: 'border-color 0.15s ease',
  },
  select: {
    padding: '12px 16px', fontSize: 14, fontFamily: FONTS.body,
    border: `1px solid ${COLORS.stone200}`, borderRadius: 10, outline: 'none', color: COLORS.primary,
    background: COLORS.white, cursor: 'pointer', minWidth: 140,
  },
  addBtn: {
    padding: '12px 24px', fontSize: 14, fontWeight: 600, fontFamily: FONTS.body,
    border: 'none', borderRadius: 10, background: COLORS.primary, color: COLORS.white,
    cursor: 'pointer', whiteSpace: 'nowrap', transition: 'opacity 0.15s ease',
  },

  // Error
  errorBanner: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '12px 16px', background: '#fef2f2', border: '1px solid #fecaca',
    borderRadius: 10, marginBottom: 24, fontSize: 14, color: '#b91c1c',
  },
  errorDismiss: {
    background: 'none', border: 'none', cursor: 'pointer', fontSize: 14,
    color: '#b91c1c', padding: '0 4px', fontFamily: FONTS.body,
  },

  // Rules list
  rulesSection: { marginBottom: 32 },
  rulesList: { display: 'flex', flexDirection: 'column', gap: 1 },
  ruleRow: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '16px 20px', background: COLORS.white, borderRadius: 10,
    border: `1px solid ${COLORS.stone100}`, marginBottom: 8,
  },
  ruleInfo: { display: 'flex', alignItems: 'center', gap: 16 },
  rulePattern: { fontSize: 15, fontWeight: 600, color: COLORS.stone900 },
  ruleActions: { display: 'flex', gap: 8 },
  editBtn: {
    padding: '6px 14px', fontSize: 12, fontWeight: 600, fontFamily: FONTS.body,
    border: `1px solid ${COLORS.stone200}`, borderRadius: 8, background: COLORS.white, color: COLORS.stone700,
    cursor: 'pointer',
  },
  deleteBtn: {
    padding: '6px 14px', fontSize: 12, fontWeight: 600, fontFamily: FONTS.body,
    border: '1px solid #fecaca', borderRadius: 8, background: COLORS.white, color: '#b91c1c',
    cursor: 'pointer',
  },

  // Edit row
  editRow: { display: 'flex', gap: 10, alignItems: 'center', width: '100%' },
  editInput: {
    flex: 1, padding: '8px 12px', fontSize: 14, fontFamily: FONTS.body,
    border: `1px solid ${COLORS.stone200}`, borderRadius: 8, outline: 'none', color: COLORS.primary,
  },
  editSelect: {
    padding: '8px 12px', fontSize: 14, fontFamily: FONTS.body,
    border: `1px solid ${COLORS.stone200}`, borderRadius: 8, outline: 'none', color: COLORS.primary,
    background: COLORS.white, cursor: 'pointer', minWidth: 120,
  },
  saveBtn: {
    padding: '8px 16px', fontSize: 12, fontWeight: 600, fontFamily: FONTS.body,
    border: 'none', borderRadius: 8, background: COLORS.primary, color: COLORS.white, cursor: 'pointer',
  },
  cancelBtn: {
    padding: '8px 16px', fontSize: 12, fontWeight: 600, fontFamily: FONTS.body,
    border: `1px solid ${COLORS.stone200}`, borderRadius: 8, background: COLORS.white, color: COLORS.stone700,
    cursor: 'pointer',
  },

  // Empty state
  emptyState: { display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '48px 20px', textAlign: 'center' },
  emptyIcon: { fontSize: 32, color: COLORS.stone300, marginBottom: 16 },
  emptyTitle: { fontFamily: FONTS.headline, fontSize: 18, fontWeight: 400, fontStyle: 'italic', color: COLORS.stone900, margin: '0 0 8px 0' },
  emptyText: { fontSize: 14, color: COLORS.stone500, lineHeight: '1.6', maxWidth: 360, margin: 0 },

  // Info section
  infoSection: {
    display: 'flex', alignItems: 'flex-start', gap: 12, padding: '20px 24px',
    background: '#f8fafc', borderRadius: 12, border: '1px solid #f1f5f9',
  },
  infoIcon: { fontSize: 18, color: '#64748b', flexShrink: 0, marginTop: 1 },
  infoText: { fontSize: 13, color: '#64748b', lineHeight: '1.6', margin: 0, fontFamily: FONTS.body },
};

export default Settings;
