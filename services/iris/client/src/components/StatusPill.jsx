import React, { useEffect, useState } from 'react';
import { FONTS, COLORS } from '../theme';

// StatusPill — small visibility chip in the sidebar showing the AI runtime
// state. Polls /api/setup/state + /api/setup/ollama-status every 30s so the
// dot reflects reality (green = ready, ember = needs setup, ash = unknown).
//
// Sits above the user pill at the bottom of the sidebar. Tiny, builds trust
// through visibility ("yes, AI is running on your Mac, here's the model").
export default function StatusPill({ style }) {
  const [state, setState] = useState('checking'); // 'ready' | 'pull_model' | 'install_ollama' | 'checking'
  const [model, setModel] = useState('');

  useEffect(() => {
    let cancelled = false;
    const tick = async () => {
      try {
        const res = await fetch('/api/setup/state');
        if (!res.ok) throw new Error('non-200');
        const data = await res.json();
        if (cancelled) return;
        setState(data.step || 'checking');
        // Try to surface installed model — fetch ollama-status for the
        // recommended pick (lightweight; same endpoint the wizard uses).
        const tagsRes = await fetch('/api/v1/models');
        if (tagsRes.ok) {
          const tags = await tagsRes.json();
          const sweet = (tags || []).find((m) => /:[3-9]b\b|:1[0-4]b\b/i.test(m.name));
          if (sweet && !cancelled) setModel(sweet.name);
        }
      } catch (_) {
        if (!cancelled) setState('checking');
      }
    };
    tick();
    const id = setInterval(tick, 30000);
    return () => { cancelled = true; clearInterval(id); };
  }, []);

  const dotColor = state === 'ready' ? COLORS.moss : state === 'checking' ? COLORS.ash : COLORS.ember;
  const label = state === 'ready'
    ? `running · ${model || 'local'}`
    : state === 'install_ollama' ? 'Ollama needed'
    : state === 'pull_model' ? 'Pull a model'
    : 'checking';

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      gap: 8,
      padding: '8px 12px',
      borderRadius: 8,
      background: COLORS.paper,
      border: `1px solid ${COLORS.rule}`,
      fontFamily: FONTS.body,
      fontSize: 11,
      color: COLORS.ash,
      ...style,
    }}>
      <span style={{
        display: 'inline-block',
        width: 8,
        height: 8,
        borderRadius: '50%',
        background: dotColor,
        boxShadow: state === 'ready' ? `0 0 0 3px ${COLORS.mossBg}` : 'none',
      }} />
      <span style={{ fontFamily: FONTS.mono, fontSize: 11, color: COLORS.ink }}>{label}</span>
    </div>
  );
}
