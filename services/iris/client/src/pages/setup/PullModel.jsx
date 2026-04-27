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
    display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 20,
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
    fontSize: 15, color: '#2d3435', textAlign: 'center', lineHeight: 1.6, margin: 0,
  },
  reason: {
    fontSize: 13, color: '#8C8C8C', textAlign: 'center', lineHeight: 1.5, margin: 0,
  },
  button: {
    padding: '14px 32px', fontSize: 15, fontWeight: 500,
    background: '#2d3435', color: '#fff', border: 'none', borderRadius: 12,
    cursor: 'pointer', letterSpacing: '0.02em',
  },
  buttonDisabled: { opacity: 0.5, cursor: 'not-allowed' },
  progressOuter: {
    width: '100%', height: 8, background: '#f0f0f0', borderRadius: 999, overflow: 'hidden',
  },
  progressInner: {
    height: '100%', background: '#2d3435', transition: 'width 0.3s ease',
  },
  status: {
    fontSize: 13, color: '#8C8C8C', textAlign: 'center', letterSpacing: '0.02em',
  },
  link: {
    fontSize: 13, color: '#5f5e5e', textDecoration: 'underline', cursor: 'pointer',
    background: 'none', border: 'none', padding: 0,
  },
  altList: {
    width: '100%', display: 'flex', flexDirection: 'column', gap: 8, marginTop: 8,
  },
  altRow: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '12px 16px', border: '1px solid #eee', borderRadius: 10, fontSize: 13,
  },
  altPull: {
    padding: '6px 14px', fontSize: 12, background: '#2d3435', color: '#fff',
    border: 'none', borderRadius: 8, cursor: 'pointer',
  },
  err: { color: '#b00020', fontSize: 13, textAlign: 'center' },
};

export default function PullModel() {
  const [recommended, setRecommended] = useState(null);
  const [model, setModel] = useState('');
  const [pulling, setPulling] = useState(false);
  const [progress, setProgress] = useState(0); // 0..1
  const [statusText, setStatusText] = useState('');
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [showAlts, setShowAlts] = useState(false);
  const [installedModels, setInstalledModels] = useState([]);
  const navigate = useNavigate();
  const abortRef = useRef(null);

  useEffect(() => {
    fetch('/api/setup/recommended')
      .then((r) => r.json())
      .then((data) => {
        setRecommended(data);
        setModel(data.model);
      })
      .catch(() => setError('Failed to load recommendation'));
    fetch('/api/setup/state')
      .then((r) => r.json())
      .then((data) => setInstalledModels(data.installed_models || []))
      .catch(() => {});
  }, []);

  const startPull = async (target) => {
    setPulling(true);
    setError('');
    setProgress(0);
    setStatusText('Connecting...');
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    try {
      const res = await fetch('/api/setup/pull-model', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: target }),
        signal: ctrl.signal,
      });
      if (!res.ok || !res.body) {
        throw new Error(`Pull failed (${res.status})`);
      }
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buf = '';
      while (true) {
        const { value, done: rdDone } = await reader.read();
        if (rdDone) break;
        buf += decoder.decode(value, { stream: true });
        // SSE frames are separated by \n\n; lines start with "data: " or "event: error"
        let idx;
        while ((idx = buf.indexOf('\n\n')) >= 0) {
          const frame = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          const lines = frame.split('\n');
          let isError = false;
          let dataPayload = '';
          for (const ln of lines) {
            if (ln.startsWith('event: error')) isError = true;
            else if (ln.startsWith('data: ')) dataPayload = ln.slice(6);
          }
          if (!dataPayload) continue;
          if (isError) {
            setError(dataPayload);
            continue;
          }
          try {
            const obj = JSON.parse(dataPayload);
            if (obj.status) setStatusText(obj.status);
            if (typeof obj.completed === 'number' && typeof obj.total === 'number' && obj.total > 0) {
              setProgress(obj.completed / obj.total);
            }
            if (obj.status === 'success') {
              setProgress(1);
              setDone(true);
              setTimeout(() => navigate('/login'), 1000);
            }
          } catch (_) {
            // skip non-JSON lines
          }
        }
      }
    } catch (e) {
      if (e.name !== 'AbortError') setError(e.message || 'Pull failed');
    } finally {
      setPulling(false);
    }
  };

  if (!recommended) {
    return (
      <div style={s.page}>
        <div style={s.panel}>
          <div style={s.brand}>LocalFinance</div>
          <p style={s.body}>Loading recommendation...</p>
        </div>
      </div>
    );
  }

  return (
    <div style={s.page}>
      <div style={s.panel}>
        <div style={s.brand}>LocalFinance</div>
        <h1 style={s.title}>Download Model</h1>
        <p style={s.body}>
          We'll download <strong>{model}</strong> (~{recommended.size_gb} GB).
          This takes 3-10 minutes depending on your connection.
        </p>
        <p style={s.reason}>{recommended.reason}</p>

        {!pulling && !done && (
          <button style={s.button} onClick={() => startPull(model)}>
            Download Model
          </button>
        )}

        {(pulling || done) && (
          <>
            <div style={s.progressOuter}>
              <div style={{ ...s.progressInner, width: `${Math.round(progress * 100)}%` }} />
            </div>
            <div style={s.status}>
              {done ? 'Download complete ✓' : statusText || 'Working...'}
              {progress > 0 && progress < 1 && ` — ${Math.round(progress * 100)}%`}
            </div>
          </>
        )}

        {error && <div style={s.err}>{error}</div>}

        {!pulling && !done && (
          <button style={s.link} onClick={() => setShowAlts((v) => !v)}>
            {showAlts ? 'Hide other models' : 'Choose different model'}
          </button>
        )}

        {showAlts && !pulling && !done && (
          <div style={s.altList}>
            {installedModels.length === 0 && (
              <div style={s.reason}>No other models installed yet.</div>
            )}
            {installedModels.map((m) => (
              <div key={m} style={s.altRow}>
                <span>{m}</span>
                <button style={s.altPull} onClick={() => { setModel(m); startPull(m); }}>
                  Pull
                </button>
              </div>
            ))}
            {/* Common picks the user might want even if not installed yet */}
            {['llama3.2:3b', 'gemma3:4b', 'qwen2.5:7b'].filter((m) => m !== model && !installedModels.includes(m)).map((m) => (
              <div key={m} style={s.altRow}>
                <span>{m}</span>
                <button style={s.altPull} onClick={() => { setModel(m); startPull(m); }}>
                  Pull
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
