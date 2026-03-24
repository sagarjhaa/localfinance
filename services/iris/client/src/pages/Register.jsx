import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { authAPI, setAuthData } from '../api/client';

const styles = {
  page: {
    minHeight: '100vh',
    background: '#ffffff',
    fontFamily: "'Manrope', sans-serif",
    color: '#2d3435',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
    position: 'relative',
  },
  bgDot: {
    position: 'absolute',
    inset: 0,
    zIndex: 0,
    opacity: 0.05,
    pointerEvents: 'none',
    backgroundImage: 'radial-gradient(#1A1A1A 0.5px, transparent 0.5px)',
    backgroundSize: '24px 24px',
  },
  bgBlob1: {
    position: 'absolute',
    top: '-10%',
    right: '-5%',
    width: 600,
    height: 600,
    background: '#ebeeef',
    borderRadius: '50%',
    filter: 'blur(80px)',
    opacity: 0.3,
    pointerEvents: 'none',
  },
  bgBlob2: {
    position: 'absolute',
    bottom: '-10%',
    left: '-5%',
    width: 500,
    height: 500,
    background: '#e4e2e1',
    borderRadius: '50%',
    filter: 'blur(80px)',
    opacity: 0.2,
    pointerEvents: 'none',
  },
  panel: {
    position: 'relative',
    zIndex: 10,
    width: '100%',
    maxWidth: 800,
    minHeight: 600,
    background: 'rgba(255, 255, 255, 0.45)',
    backdropFilter: 'blur(32px)',
    WebkitBackdropFilter: 'blur(32px)',
    borderTop: '1px solid rgba(255, 255, 255, 0.9)',
    borderLeft: '1px solid rgba(255, 255, 255, 0.9)',
    boxShadow: '0 48px 100px -24px rgba(0, 0, 0, 0.08)',
    borderRadius: 24,
    display: 'flex',
    flexDirection: 'column',
    padding: '64px',
  },
  logo: {
    fontFamily: "'Instrument Serif', serif",
    fontSize: 32,
    fontWeight: 600,
    color: '#1A1A1A',
    letterSpacing: '-0.04em',
    lineHeight: 1,
    marginBottom: 48,
  },
  title: {
    fontFamily: "'Instrument Serif', serif",
    fontSize: 48,
    color: '#1A1A1A',
    lineHeight: 1.1,
    marginBottom: 48,
    letterSpacing: '-0.02em',
    fontWeight: 400,
  },
  form: {
    maxWidth: 480,
  },
  fieldGroup: {
    marginBottom: 40,
  },
  label: {
    display: 'block',
    fontSize: 10,
    textTransform: 'uppercase',
    letterSpacing: '0.15em',
    color: '#8C8C8C',
    marginBottom: 4,
    fontWeight: 600,
  },
  input: {
    width: '100%',
    padding: '12px 0',
    fontSize: 15,
    fontFamily: "'Courier New', Courier, monospace",
    color: '#1A1A1A',
    background: 'transparent',
    border: 'none',
    borderBottom: '1px solid #8C8C8C',
    borderRadius: 0,
    outline: 'none',
    transition: 'border-color 0.2s',
  },
  nameRow: {
    display: 'flex',
    gap: 32,
  },
  nameField: {
    flex: 1,
  },
  button: {
    height: 48,
    padding: '0 40px',
    background: '#1A1A1A',
    color: '#ffffff',
    fontFamily: "'Manrope', sans-serif",
    fontWeight: 500,
    fontSize: 14,
    border: 'none',
    borderRadius: 8,
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 12,
    transition: 'opacity 0.2s, transform 0.1s',
    marginTop: 24,
  },
  buttonDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed',
  },
  signInLink: {
    marginTop: 24,
    fontSize: 14,
    color: '#8C8C8C',
  },
  signInAnchor: {
    color: '#1A1A1A',
    fontWeight: 600,
    textDecoration: 'underline',
    textUnderlineOffset: 4,
  },
  error: {
    background: '#fff7f6',
    border: '1px solid #fe8983',
    color: '#752121',
    padding: '12px 16px',
    borderRadius: 8,
    fontSize: 14,
    marginBottom: 24,
  },
  footer: {
    marginTop: 'auto',
    paddingTop: 32,
    borderTop: '1px solid rgba(0,0,0,0.05)',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  footerMeta: {
    display: 'flex',
    gap: 32,
    alignItems: 'center',
  },
  footerLabel: {
    fontSize: 10,
    textTransform: 'uppercase',
    letterSpacing: '0.2em',
    color: '#8C8C8C',
    fontWeight: 700,
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  statusDot: {
    width: 6,
    height: 6,
    borderRadius: '50%',
    background: '#10b981',
  },
};

const Register = ({ onLogin }) => {
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    first_name: '',
    last_name: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    if (error) setError('');
  };

  const handleFocus = (e) => {
    e.target.style.borderColor = '#1A1A1A';
  };

  const handleBlur = (e) => {
    e.target.style.borderColor = '#8C8C8C';
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      const response = await authAPI.register(formData);
      const { user, token } = response.data;
      setAuthData(token, user);
      onLogin(user);
    } catch (err) {
      setError(err.message || 'Registration failed. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div style={styles.page}>
      <div style={styles.bgDot} />
      <div style={styles.bgBlob1} />
      <div style={styles.bgBlob2} />

      <main style={styles.panel}>
        <header>
          <h1 style={styles.logo}>LocalFinance</h1>
        </header>

        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
          <h2 style={styles.title}>Create Your Edge Account</h2>

          <form onSubmit={handleSubmit} style={styles.form}>
            {error && <div style={styles.error}>{error}</div>}

            <div style={{ ...styles.fieldGroup, ...styles.nameRow }}>
              <div style={styles.nameField}>
                <label style={styles.label}>First Name</label>
                <input
                  name="first_name"
                  type="text"
                  placeholder="First Name"
                  required
                  value={formData.first_name}
                  onChange={handleChange}
                  onFocus={handleFocus}
                  onBlur={handleBlur}
                  disabled={isLoading}
                  style={styles.input}
                />
              </div>
              <div style={styles.nameField}>
                <label style={styles.label}>Last Name</label>
                <input
                  name="last_name"
                  type="text"
                  placeholder="Last Name"
                  required
                  value={formData.last_name}
                  onChange={handleChange}
                  onFocus={handleFocus}
                  onBlur={handleBlur}
                  disabled={isLoading}
                  style={styles.input}
                />
              </div>
            </div>

            <div style={styles.fieldGroup}>
              <label style={styles.label}>Node Identifier</label>
              <input
                name="email"
                type="email"
                placeholder="Email Address"
                required
                value={formData.email}
                onChange={handleChange}
                onFocus={handleFocus}
                onBlur={handleBlur}
                disabled={isLoading}
                style={styles.input}
              />
            </div>

            <div style={styles.fieldGroup}>
              <label style={styles.label}>Security Protocol</label>
              <input
                name="password"
                type="password"
                placeholder="Secure Passphrase"
                required
                minLength={6}
                value={formData.password}
                onChange={handleChange}
                onFocus={handleFocus}
                onBlur={handleBlur}
                disabled={isLoading}
                style={styles.input}
              />
            </div>

            <div>
              <button
                type="submit"
                disabled={isLoading}
                style={{
                  ...styles.button,
                  ...(isLoading ? styles.buttonDisabled : {}),
                }}
                onMouseDown={(e) => { if (!isLoading) e.currentTarget.style.transform = 'scale(0.98)'; }}
                onMouseUp={(e) => { e.currentTarget.style.transform = 'scale(1)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.transform = 'scale(1)'; }}
              >
                {isLoading ? 'Registering...' : 'Register Node'}
                {!isLoading && <span style={{ fontSize: 18 }}>&#8594;</span>}
              </button>

              <p style={styles.signInLink}>
                Already have an account?{' '}
                <Link to="/login" style={styles.signInAnchor}>Sign In</Link>
              </p>
            </div>
          </form>
        </div>

        <footer style={styles.footer}>
          <div style={styles.footerMeta}>
            <div style={styles.footerLabel}>
              <span style={styles.statusDot} />
              Network Status: Optimal
            </div>
            <div style={{ ...styles.footerLabel, gap: 0 }}>
              v4.2.0-Editorial
            </div>
          </div>
        </footer>
      </main>
    </div>
  );
};

export default Register;
