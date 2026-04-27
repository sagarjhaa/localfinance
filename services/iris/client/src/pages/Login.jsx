import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { authAPI, setAuthData } from '../api/client';
import { FONTS, COLORS, APP } from '../theme';

const s = {
  page: {
    minHeight: '100vh',
    background: '#ffffff',
    fontFamily: FONTS.body,
    color: '#2d3435',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
    position: 'relative',
  },
  texture: {
    position: 'fixed', inset: 0, zIndex: 0, opacity: 0.4, pointerEvents: 'none',
    backgroundColor: '#ffffff',
    backgroundImage: 'radial-gradient(#e5e7eb 0.5px, transparent 0.5px)',
    backgroundSize: '24px 24px',
  },
  blob1: {
    position: 'fixed', top: '-5%', right: '-2%', width: '45%', height: '45%',
    borderRadius: '50%', background: '#fafafa', filter: 'blur(140px)', opacity: 0.8, pointerEvents: 'none',
  },
  blob2: {
    position: 'fixed', bottom: '-10%', left: '-5%', width: '40%', height: '40%',
    borderRadius: '50%', background: '#f5f5f5', filter: 'blur(120px)', opacity: 0.6, pointerEvents: 'none',
  },
  header: {
    position: 'fixed', top: 0, width: '100%', display: 'flex', justifyContent: 'space-between',
    alignItems: 'center', padding: '32px 48px', zIndex: 50,
  },
  logo: {
    fontSize: 24, fontFamily: FONTS.headline, fontStyle: 'italic',
    color: COLORS.primary, letterSpacing: '-0.02em',
  },
  headerLabel: {
    fontSize: 12, letterSpacing: '0.15em', textTransform: 'uppercase', color: '#a0a0a0',
  },
  panel: {
    position: 'relative', zIndex: 10, width: '100%', maxWidth: 800, minHeight: 500,
    background: 'rgba(255, 255, 255, 0.6)', backdropFilter: 'blur(32px)',
    WebkitBackdropFilter: 'blur(32px)',
    borderTop: '1px solid rgba(255, 255, 255, 1)', borderLeft: '1px solid rgba(255, 255, 255, 1)',
    boxShadow: '0px 32px 100px rgba(0, 0, 0, 0.08)',
    borderRadius: 24, display: 'flex', flexDirection: 'column',
    alignItems: 'center', justifyContent: 'center', padding: '80px',
  },
  title: {
    fontFamily: FONTS.headline, fontSize: 56, lineHeight: 1.1,
    color: COLORS.primary, fontWeight: 400, letterSpacing: '-0.02em', textAlign: 'center',
  },
  subtitle: {
    fontSize: 16, color: '#8C8C8C', marginTop: 12, letterSpacing: '0.04em',
    fontWeight: 300, textAlign: 'center',
  },
  form: { width: '100%', maxWidth: 420, display: 'flex', flexDirection: 'column', gap: 48, marginTop: 64 },
  input: {
    width: '100%', background: 'transparent', border: 'none',
    borderBottom: '1px solid #D1D1D1', padding: '16px 0',
    fontFamily: FONTS.mono, fontSize: 12,
    letterSpacing: '0.1em', color: '#2d3435', outline: 'none', transition: 'border-color 0.2s',
  },
  btnWrap: {
    display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 32, marginTop: 16,
  },
  btn: {
    width: 240, height: 52, background: COLORS.primary, color: COLORS.white,
    fontFamily: FONTS.body, fontSize: 13, fontWeight: 600,
    letterSpacing: '0.15em', border: '1px solid #000', borderRadius: 2,
    cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
    boxShadow: 'inset 0 2px 4px rgba(0,0,0,0.4), inset 0 -1px 0 rgba(255,255,255,0.1), 0 1px 2px rgba(0,0,0,0.1)',
    transition: 'all 0.15s',
  },
  btnDisabled: { opacity: 0.6, cursor: 'not-allowed' },
  links: { display: 'flex', gap: 40 },
  link: {
    fontFamily: FONTS.mono, fontSize: 11, color: '#8C8C8C',
    textDecoration: 'none', borderBottom: '1px solid transparent', paddingBottom: 2,
    transition: 'all 0.2s',
  },
  hint: {
    marginTop: 12, marginBottom: 0,
    fontFamily: FONTS.mono, fontSize: 10, letterSpacing: '0.08em',
    color: '#a0a0a0', textAlign: 'left',
  },
  error: {
    background: COLORS.errorBg, border: '1px solid #fe8983', color: '#752121',
    padding: '12px 16px', borderRadius: 8, fontSize: 13, textAlign: 'center', width: '100%', maxWidth: 420,
  },
  footer: {
    position: 'fixed', bottom: 0, width: '100%', display: 'flex', justifyContent: 'space-between',
    alignItems: 'center', padding: '40px 48px', zIndex: 50,
  },
  footerText: {
    fontSize: 10, letterSpacing: '0.2em', textTransform: 'uppercase', color: '#a0a0a0', fontWeight: 500,
  },
};

// Default credentials seeded by Thesaurus on first boot (single-user app).
// Pre-filled here so the user can sign in immediately on a fresh install,
// then change them in the Profile page.
const DEFAULT_EMAIL = 'local@localfinance.app';
const DEFAULT_PASSWORD = 'localfinance';

const Login = ({ onLogin }) => {
  const [formData, setFormData] = useState({ email: DEFAULT_EMAIL, password: DEFAULT_PASSWORD });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e) => {
    setFormData(prev => ({ ...prev, [e.target.name]: e.target.value }));
    if (error) setError('');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');
    try {
      const response = await authAPI.login(formData);
      const { user, token } = response.data;
      setAuthData(token, user);
      onLogin(user);
    } catch (err) {
      setError(err.message || 'Authentication failed.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div style={s.page}>
      <div style={s.texture} />
      <div style={s.blob1} />
      <div style={s.blob2} />

      <header style={s.header}>
        <div style={s.logo}>{APP.name}</div>
        <span style={s.headerLabel}>{APP.tagline}</span>
      </header>

      <main style={s.panel}>
        <div style={{ textAlign: 'center', marginBottom: 0 }}>
          <h1 style={s.title}>{APP.name} Sign In</h1>
          <p style={s.subtitle}>Seamless access to your financial world.</p>
        </div>

        {error && <div style={{ ...s.error, marginTop: 32 }}>{error}</div>}

        <form onSubmit={handleSubmit} style={s.form}>
          <div>
            <input
              name="email" type="email" required placeholder="ADDRESS@DOMAIN.COM"
              value={formData.email} onChange={handleChange} disabled={isLoading}
              style={s.input}
              onFocus={e => e.target.style.borderColor = COLORS.primary}
              onBlur={e => e.target.style.borderColor = '#D1D1D1'}
            />
          </div>
          <div>
            <input
              name="password" type="password" required placeholder="••••••••••••"
              value={formData.password} onChange={handleChange} disabled={isLoading}
              style={s.input}
              onFocus={e => e.target.style.borderColor = COLORS.primary}
              onBlur={e => e.target.style.borderColor = '#D1D1D1'}
            />
            <p style={s.hint}>
              Default credentials are pre-filled. Change them in Profile after sign-in.
            </p>
          </div>
          <div style={s.btnWrap}>
            <button type="submit" disabled={isLoading}
              style={{ ...s.btn, ...(isLoading ? s.btnDisabled : {}) }}
              onMouseDown={e => { if (!isLoading) e.currentTarget.style.transform = 'scale(0.98)'; }}
              onMouseUp={e => e.currentTarget.style.transform = 'scale(1)'}
              onMouseLeave={e => e.currentTarget.style.transform = 'scale(1)'}
            >
              {isLoading ? 'AUTHENTICATING...' : 'AUTHENTICATE'}
            </button>
            <div style={s.links}>
              <Link to="/register" style={s.link}
                onMouseOver={e => { e.target.style.color = COLORS.primary; e.target.style.borderColor = COLORS.primary; }}
                onMouseOut={e => { e.target.style.color = '#8C8C8C'; e.target.style.borderColor = 'transparent'; }}
              >Create Account</Link>
              <span style={{ ...s.link, cursor: 'default' }}>System Status</span>
            </div>
          </div>
        </form>
      </main>

      <footer style={s.footer}>
        <div style={s.footerText}>&copy; 2026 LocalFinance &mdash; Secure Infrastructure</div>
        <div style={{ display: 'flex', gap: 32 }}>
          <span style={s.footerText}>Legal</span>
          <span style={s.footerText}>Privacy</span>
        </div>
      </footer>
    </div>
  );
};

export default Login;
