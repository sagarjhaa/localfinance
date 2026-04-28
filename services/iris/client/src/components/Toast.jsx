import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { FONTS, COLORS } from '../theme';

// Tiny toast system. Bottom-right, slide-in, auto-dismiss after 2.5s.
// Use:
//   const { toast } = useToast();
//   toast('Saved.');                 // info
//   toast('Insight dismissed.', 'success');
//   toast("Couldn't save.", 'error');
//
// Wrap App with <ToastProvider>...</ToastProvider> exactly once (in App.jsx).

const ToastCtx = createContext({ toast: () => {} });
export const useToast = () => useContext(ToastCtx);

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([]);

  const toast = useCallback((message, kind = 'info', durationMs = 2500) => {
    const id = Math.random().toString(36).slice(2);
    setToasts((prev) => [...prev, { id, message, kind }]);
    if (durationMs > 0) {
      setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
      }, durationMs);
    }
    return id;
  }, []);

  return (
    <ToastCtx.Provider value={{ toast }}>
      {children}
      <div style={{
        position: 'fixed',
        bottom: 24,
        right: 24,
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        pointerEvents: 'none',
      }}>
        {toasts.map((t) => <ToastItem key={t.id} toast={t} />)}
      </div>
    </ToastCtx.Provider>
  );
}

function ToastItem({ toast }) {
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const id = requestAnimationFrame(() => setVisible(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const accent = toast.kind === 'success' ? COLORS.moss
    : toast.kind === 'error' ? COLORS.ember
    : COLORS.saffron;

  return (
    <div style={{
      pointerEvents: 'auto',
      background: COLORS.surface,
      border: `1px solid ${COLORS.rule}`,
      borderLeft: `3px solid ${accent}`,
      borderRadius: 8,
      padding: '10px 16px',
      fontFamily: FONTS.body,
      fontSize: 13,
      color: COLORS.ink,
      boxShadow: '0 8px 24px -8px rgba(0,0,0,0.15)',
      minWidth: 220,
      maxWidth: 360,
      transform: visible ? 'translateX(0)' : 'translateX(120%)',
      opacity: visible ? 1 : 0,
      transition: 'transform 220ms ease-out, opacity 220ms ease-out',
    }}>
      {toast.message}
    </div>
  );
}
