import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { FONTS, COLORS } from '../../theme';

const s = {
  page: {
    minHeight: '100vh', background: '#ffffff', fontFamily: FONTS.body, color: '#2d3435',
    display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
  },
  panel: {
    width: '100%', maxWidth: 640, padding: '64px 56px', borderRadius: 24,
    background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(32px)',
    boxShadow: '0px 32px 100px rgba(0,0,0,0.08)',
    display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 24,
  },
  brand: {
    fontFamily: FONTS.headline, fontSize: 28, fontStyle: 'italic',
    color: COLORS.primary, letterSpacing: '-0.02em',
  },
  title: {
    fontFamily: FONTS.headline, fontSize: 36, lineHeight: 1.2,
    fontWeight: 400, letterSpacing: '-0.02em', textAlign: 'center', margin: 0,
  },
  body: {
    fontSize: 15, color: '#5f5e5e', textAlign: 'center', lineHeight: 1.6, margin: 0,
  },
  button: {
    padding: '14px 32px', fontSize: 15, fontWeight: 500,
    background: '#2d3435', color: '#fff', border: 'none', borderRadius: 12,
    cursor: 'pointer', letterSpacing: '0.02em',
  },
  status: {
    fontSize: 13, color: '#8C8C8C', textAlign: 'center', letterSpacing: '0.04em',
  },
  statusGood: { color: '#0a7d3a', fontWeight: 500 },
};

const DOWNLOAD_URL = 'https://ollama.com/download/Ollama-darwin.zip';

export default function InstallOllama() {
  const [installed, setInstalled] = useState(false);
  const [version, setVersion] = useState('');
  const navigate = useNavigate();
  const navigatedRef = useRef(false);

  useEffect(() => {
    let alive = true;
    const poll = async () => {
      try {
        const res = await fetch('/api/setup/ollama-status');
        const data = await res.json();
        if (!alive) return;
        if (data.installed) {
          setInstalled(true);
          setVersion(data.version || '');
          if (!navigatedRef.current) {
            navigatedRef.current = true;
            setTimeout(() => navigate('/setup/pull'), 1000);
          }
        }
      } catch (_) {
        // network error — keep polling
      }
    };
    poll();
    const id = setInterval(poll, 2000);
    return () => { alive = false; clearInterval(id); };
  }, [navigate]);

  return (
    <div style={s.page}>
      <div style={s.panel}>
        <div style={s.brand}>LocalFinance</div>
        <h1 style={s.title}>Install Ollama</h1>
        <p style={s.body}>
          LocalFinance needs Ollama to run AI on your Mac. It's free and open-source.
        </p>
        <button style={s.button} onClick={() => window.open(DOWNLOAD_URL, '_blank')}>
          Download Ollama
        </button>
        <div style={{ ...s.status, ...(installed ? s.statusGood : {}) }}>
          {installed
            ? `Ollama detected ${version ? `(v${version}) ` : ''}✓`
            : 'Waiting for Ollama...'}
        </div>
      </div>
    </div>
  );
}
