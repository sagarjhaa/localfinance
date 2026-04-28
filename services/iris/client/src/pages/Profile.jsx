import React, { useState, useEffect } from 'react';
import { FONTS, COLORS, APP } from '../theme';
import { proxyAPI, authAPI } from '../api/client';
import { useIsNarrow, useIsMedium } from '../hooks/useMediaQuery';

const Profile = ({ user, onLogout }) => {
  const isNarrow = useIsNarrow();
  const isMedium = useIsMedium();
  const S = styles(isNarrow, isMedium);
  const [firstName, setFirstName] = useState(user?.first_name || '');
  const [lastName, setLastName] = useState(user?.last_name || '');
  const [displayName, setDisplayName] = useState('');
  const [chatModel, setChatModel] = useState('llama3.2:1b');
  const [models, setModels] = useState([]);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState('');

  // Password change form state
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmNewPassword, setConfirmNewPassword] = useState('');
  const [pwSaving, setPwSaving] = useState(false);
  const [pwMessage, setPwMessage] = useState('');
  const [pwMessageType, setPwMessageType] = useState('');

  useEffect(() => {
    // Fetch available models
    proxyAPI.sophia.get('/api/v1/models')
      .then((res) => setModels(res.data || []))
      .catch(() => {});

    // Fetch user preferences
    proxyAPI.thesaurus.get('/api/v1/preferences', undefined, { skipLogoutOn401: true })
      .then((res) => {
        const prefs = res.data || {};
        if (prefs.chat_model) setChatModel(prefs.chat_model);
        if (prefs.display_name) setDisplayName(prefs.display_name);
      })
      .catch(() => {});
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setMessage('');
    try {
      await proxyAPI.thesaurus.put('/api/v1/preferences', {
        chat_model: chatModel,
        display_name: displayName,
        first_name: firstName,
        last_name: lastName,
      });
      setMessage('Saved.');
      setMessageType('success');
      setTimeout(() => setMessage(''), 3000);
    } catch (err) {
      setMessage("Couldn't save that. Try again?");
      setMessageType('error');
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async () => {
    setPwMessage('');

    if (!currentPassword || !newPassword || !confirmNewPassword) {
      setPwMessage('Fill in all three fields.');
      setPwMessageType('error');
      return;
    }
    if (newPassword !== confirmNewPassword) {
      setPwMessage("Those don't match.");
      setPwMessageType('error');
      return;
    }
    if (newPassword.length < 8) {
      setPwMessage('Eight characters or more, please.');
      setPwMessageType('error');
      return;
    }

    setPwSaving(true);
    try {
      await authAPI.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      });
      setPwMessage('Saved.');
      setPwMessageType('success');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmNewPassword('');
      setTimeout(() => setPwMessage(''), 3000);
    } catch (err) {
      if (err?.message) console.error('[profile] password change failed:', err.message);
      setPwMessage("Couldn't save that. Try again?");
      setPwMessageType('error');
    } finally {
      setPwSaving(false);
    }
  };

  return (
    <div style={S.page}>
      {/* Sidebar */}
      <aside style={S.sidebar}>
        <div style={isNarrow ? { padding: 0, marginBottom: 0, marginRight: 8 } : { padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>{APP.name}</h1>
          {!isNarrow && <p style={S.sidebarTier}>{APP.tagline}</p>}
        </div>
        <nav style={isNarrow ? { display: 'flex', flexDirection: 'row', gap: 4, alignItems: 'center' } : {}}>
          <div
            onClick={() => window.location.href = '/dashboard'}
            style={S.navItem}
          >
            <span style={{ fontSize: 20 }}>&#128196;</span>
            <span style={{ fontSize: 14 }}>Dashboard</span>
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
            <span style={{ fontSize: 14 }}>This Month</span>
          </div>
          <div
            onClick={() => window.location.href = '/chat'}
            style={S.navItem}
          >
            <span style={{ fontSize: 20 }}>&#128172;</span>
            <span style={{ fontSize: 14 }}>Chat</span>
          </div>
        </nav>
        <div style={S.sidebarFooter}>
          <div
            onClick={() => window.location.href = '/profile'}
            style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer', flex: 1 }}
          >
            <div style={S.avatarCircle}>{user?.first_name?.[0] || 'U'}</div>
            <div>
              <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>{user?.first_name || ''} {user?.last_name || ''}</p>
              <p style={{ fontSize: 10, color: COLORS.stone500, margin: 0 }}>View profile</p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main style={S.main}>
        <header style={S.topBar}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
            <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: 0, fontWeight: 700 }}>Profile</h2>
          </div>
        </header>

        <div style={S.content}>
          <header style={{ marginBottom: 48 }}>
            <h2 style={S.pageTitle}>Your Profile</h2>
            <p style={{ color: COLORS.stone500, maxWidth: 480 }}>Your details and what I should call you.</p>
          </header>

          <div style={S.formCard}>
            {/* Personal Information */}
            <div style={S.section}>
              <h3 style={S.sectionTitle}>Personal Information</h3>

              <div style={S.fieldGroup}>
                <label style={S.label}>First Name</label>
                <input
                  type="text"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  style={S.input}
                  placeholder="First name"
                />
              </div>

              <div style={S.fieldGroup}>
                <label style={S.label}>Last Name</label>
                <input
                  type="text"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  style={S.input}
                  placeholder="Last name"
                />
              </div>

              <div style={S.fieldGroup}>
                <label style={S.label}>Display Name</label>
                <input
                  type="text"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  style={S.input}
                  placeholder="What should we call you?"
                />
                <p style={S.fieldHint}>This is how you'll be addressed in the app.</p>
              </div>
            </div>

            {/* AI Preferences */}
            <div style={S.section}>
              <h3 style={S.sectionTitle}>AI Preferences</h3>

              <div style={S.fieldGroup}>
                <label style={S.label}>Chat Model</label>
                <select
                  value={chatModel}
                  onChange={(e) => setChatModel(e.target.value)}
                  style={S.select}
                >
                  {models.map((m) => (
                    <option key={m.name} value={m.name}>{m.name} ({m.size})</option>
                  ))}
                  {models.length === 0 && <option value={chatModel}>{chatModel}</option>}
                </select>
                <p style={S.fieldHint}>Select the Ollama model used for chat conversations.</p>
              </div>
            </div>

            {/* Actions */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, paddingTop: 16 }}>
              <button
                onClick={handleSave}
                disabled={saving}
                style={{
                  ...S.saveButton,
                  ...(saving ? { opacity: 0.6, cursor: 'default' } : {}),
                }}
              >
                {saving ? 'Saving.' : 'Save.'}
              </button>
              {message && (
                <span style={{
                  fontSize: 13,
                  fontFamily: FONTS.body,
                  color: messageType === 'success' ? COLORS.green : COLORS.error,
                  fontWeight: 600,
                }}>
                  {message}
                </span>
              )}
            </div>

            {/* Change Password */}
            <div style={{ ...S.section, borderTop: `1px solid ${COLORS.stone100}`, marginTop: 32, paddingTop: 32 }}>
              <h3 style={S.sectionTitle}>Set a new lock.</h3>

              <div style={S.fieldGroup}>
                <label style={S.label}>Current Password</label>
                <input
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  style={S.input}
                  placeholder="Current password"
                  autoComplete="current-password"
                />
              </div>

              <div style={S.fieldGroup}>
                <label style={S.label}>New Password</label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  style={S.input}
                  placeholder="At least 8 characters"
                  autoComplete="new-password"
                />
              </div>

              <div style={S.fieldGroup}>
                <label style={S.label}>Confirm New Password</label>
                <input
                  type="password"
                  value={confirmNewPassword}
                  onChange={(e) => setConfirmNewPassword(e.target.value)}
                  style={S.input}
                  placeholder="Re-enter new password"
                  autoComplete="new-password"
                />
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: 16, paddingTop: 8 }}>
                <button
                  onClick={handleChangePassword}
                  disabled={pwSaving}
                  style={{
                    ...S.saveButton,
                    ...(pwSaving ? { opacity: 0.6, cursor: 'default' } : {}),
                  }}
                >
                  {pwSaving ? 'Saving.' : 'Save.'}
                </button>
                {pwMessage && (
                  <span style={{
                    fontSize: 13,
                    fontFamily: FONTS.body,
                    color: pwMessageType === 'success' ? COLORS.green : COLORS.error,
                    fontWeight: 600,
                  }}>
                    {pwMessage}
                  </span>
                )}
              </div>
            </div>

            {/* Sign Out */}
            <div style={{ borderTop: `1px solid ${COLORS.stone100}`, marginTop: 32, paddingTop: 24 }}>
              <button
                onClick={onLogout}
                style={S.signOutButton}
              >
                Sign Out
              </button>
            </div>
          </div>
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
  navItem: isNarrow
    ? { display: 'flex', alignItems: 'center', gap: 6, padding: '8px 12px', color: COLORS.stone500, textDecoration: 'none', cursor: 'pointer', whiteSpace: 'nowrap' }
    : { display: 'flex', alignItems: 'center', gap: 16, padding: isMedium ? '12px 20px' : '12px 32px', color: COLORS.stone500, textDecoration: 'none', cursor: 'pointer' },
  sidebarFooter: isNarrow
    ? { marginLeft: 'auto', padding: 0, display: 'flex', alignItems: 'center', gap: 12 }
    : { marginTop: 'auto', padding: isMedium ? '24px 20px' : '24px 32px', borderTop: `1px solid ${COLORS.stone100}`, display: 'flex', alignItems: 'center', gap: 12 },
  avatarCircle: { width: isNarrow ? 32 : 40, height: isNarrow ? 32 : 40, borderRadius: '50%', background: COLORS.stone100, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, fontWeight: 600, textTransform: 'uppercase', flexShrink: 0 },
  main: { flex: 1, marginLeft: isNarrow ? 0 : (isMedium ? 200 : 256), minHeight: '100vh', background: COLORS.white },
  topBar: { position: 'sticky', top: 0, zIndex: 30, background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(12px)', borderBottom: `1px solid ${COLORS.stone100}`, padding: isNarrow ? '12px 20px' : (isMedium ? '16px 32px' : '16px 48px'), display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  content: { maxWidth: 'clamp(320px, 80vw, 640px)', margin: '0 auto', padding: isNarrow ? '32px 20px 48px' : (isMedium ? '64px 32px 64px' : '96px 48px 80px') },
  pageTitle: { fontFamily: FONTS.headline, fontSize: isNarrow ? 32 : (isMedium ? 40 : 48), fontWeight: 500, letterSpacing: '-0.02em', marginBottom: 8 },
  formCard: {
    background: 'rgba(255,255,255,0.7)',
    backdropFilter: 'blur(24px)',
    border: `1px solid ${COLORS.stone100}`,
    borderRadius: 24,
    padding: isNarrow ? '24px 20px 24px' : (isMedium ? '32px 32px 28px' : '40px 40px 32px'),
  },
  section: { marginBottom: 32 },
  sectionTitle: {
    fontFamily: FONTS.headline,
    fontSize: 18,
    fontWeight: 500,
    marginBottom: 20,
    marginTop: 0,
    paddingBottom: 12,
    borderBottom: `1px solid ${COLORS.stone100}`,
  },
  fieldGroup: { marginBottom: 20 },
  label: {
    display: 'block',
    fontSize: 11,
    fontWeight: 700,
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    color: COLORS.stone500,
    marginBottom: 8,
    fontFamily: FONTS.body,
  },
  input: {
    width: '100%',
    padding: '12px 16px',
    borderRadius: 10,
    border: `1px solid ${COLORS.stone200}`,
    fontFamily: FONTS.body,
    fontSize: 14,
    color: COLORS.primary,
    background: COLORS.white,
    outline: 'none',
    boxSizing: 'border-box',
    transition: 'border-color 0.15s ease',
  },
  select: {
    width: '100%',
    padding: '12px 16px',
    borderRadius: 10,
    border: `1px solid ${COLORS.stone200}`,
    fontFamily: FONTS.body,
    fontSize: 14,
    color: COLORS.primary,
    background: COLORS.white,
    outline: 'none',
    cursor: 'pointer',
    boxSizing: 'border-box',
  },
  fieldHint: {
    fontSize: 11,
    color: COLORS.stone500,
    marginTop: 6,
    marginBottom: 0,
    fontFamily: FONTS.body,
  },
  saveButton: {
    padding: '12px 32px',
    background: COLORS.primary,
    color: COLORS.white,
    fontSize: 14,
    fontWeight: 700,
    border: 'none',
    borderRadius: 8,
    cursor: 'pointer',
    fontFamily: FONTS.body,
    boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
  },
  signOutButton: {
    background: 'none',
    border: 'none',
    padding: 0,
    fontSize: 13,
    color: COLORS.error,
    cursor: 'pointer',
    fontFamily: FONTS.body,
    textDecoration: 'underline',
    fontWeight: 500,
  },
});

export default Profile;
