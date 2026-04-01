import React, { useState, useRef, useEffect } from 'react';
import { proxyAPI } from '../api/client';

const FONTS = {
  headline: '"Newsreader", "Instrument Serif", serif',
  body: '"Manrope", "Hanken Grotesk", sans-serif',
};

const COLORS = {
  stone900: '#1c1917',
  stone700: '#44403c',
  stone500: '#78716c',
  stone300: '#d6d3d1',
  stone200: '#e7e5e4',
  stone100: '#f5f5f4',
  error: '#dc2626',
  green: '#16a34a',
  white: '#ffffff',
};

const Chat = ({ user, onLogout }) => {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [ollamaStatus, setOllamaStatus] = useState('checking');
  const [modelName, setModelName] = useState('');
  const feedRef = useRef(null);
  const topInputRef = useRef(null);
  const bottomInputRef = useRef(null);

  // Check Ollama/Sophia health on mount
  useEffect(() => {
    proxyAPI.sophia
      .get('/health')
      .then((res) => {
        setOllamaStatus('active');
        setModelName(res.data.ai_model || '');
      })
      .catch(() => setOllamaStatus('offline'));
  }, []);

  // Auto-scroll feed on new messages
  useEffect(() => {
    if (feedRef.current) {
      feedRef.current.scrollTop = feedRef.current.scrollHeight;
    }
  }, [messages, loading]);

  const sendMessage = async (text) => {
    if (!text.trim() || loading) return;

    const userMsg = { role: 'user', content: text.trim(), timestamp: new Date() };
    setMessages((prev) => [...prev, userMsg]);
    setInput('');
    setLoading(true);

    try {
      const res = await proxyAPI.sophia.post('/api/v1/chat/', {
        user_id: String(user.id),
        question: text.trim(),
      });
      const aiMsg = {
        role: 'assistant',
        content: res.data.answer,
        confidence: res.data.confidence,
        sources: res.data.sources || [],
        insights: res.data.insights || [],
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, aiMsg]);
      setOllamaStatus('active');
    } catch (err) {
      const errMsg = {
        role: 'assistant',
        content:
          "I couldn't process that right now. Please check that Ollama is running on the Jetson.",
        error: true,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, errMsg]);
      setOllamaStatus('offline');
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage(input);
    }
  };

  const formatTime = (date) => {
    return new Date(date).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const statusColor =
    ollamaStatus === 'active'
      ? COLORS.green
      : ollamaStatus === 'offline'
      ? COLORS.error
      : COLORS.stone300;

  const statusLabel =
    ollamaStatus === 'active'
      ? `Ollama ${modelName || 'AI'} Active`
      : ollamaStatus === 'offline'
      ? 'Ollama Offline'
      : 'Checking Ollama...';

  const hasMessages = messages.length > 0;

  const renderInputRow = (ref, placement) => (
    <div style={placement === 'top' ? S.topInputRow : S.bottomBar}>
      <div style={S.inputContainer}>
        <input
          ref={ref}
          type="text"
          placeholder={
            placement === 'top' && !hasMessages
              ? 'Ask about your finances...'
              : 'Follow-up question...'
          }
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={loading}
          style={{
            ...S.input,
            ...(loading ? { opacity: 0.5 } : {}),
          }}
        />
        <button
          onClick={() => sendMessage(input)}
          disabled={loading || !input.trim()}
          style={{
            ...S.sendButton,
            ...(input.trim() && !loading
              ? { color: COLORS.stone900 }
              : { color: COLORS.stone300, cursor: 'default' }),
          }}
          aria-label="Send message"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="22" y1="2" x2="11" y2="13" />
            <polygon points="22 2 15 22 11 13 2 9 22 2" />
          </svg>
        </button>
      </div>
    </div>
  );

  const renderMessage = (msg, idx) => {
    if (msg.role === 'user') {
      return (
        <div key={idx} style={S.userRow}>
          <div style={S.userBubble}>
            <span>{msg.content}</span>
            <div style={{ fontSize: 11, color: COLORS.stone500, marginTop: 6, textAlign: 'right' }}>
              {formatTime(msg.timestamp)}
            </div>
          </div>
        </div>
      );
    }

    return (
      <div key={idx} style={S.aiRow}>
        <div style={S.aiAvatar}><span role="img" aria-label="AI">&#10022;</span></div>
        <div style={S.aiBubble}>
          <div style={msg.error ? S.aiError : S.aiContent}>{msg.content}</div>
          {msg.confidence != null && !msg.error && (
            <div style={S.confidenceBadge}>Confidence: {Math.round(msg.confidence * 100)}%</div>
          )}
          {msg.insights && msg.insights.length > 0 && (
            <div style={{ marginTop: 12 }}>
              {msg.insights.map((insight, i) => (
                <div key={i} style={{ background: 'rgba(28,25,23,0.03)', borderRadius: 10, padding: '10px 14px', marginBottom: 6, fontSize: 13, color: COLORS.stone700, lineHeight: '1.5' }}>
                  {typeof insight === 'string' ? insight : insight.description || insight.title || JSON.stringify(insight)}
                </div>
              ))}
            </div>
          )}
          <div style={{ fontSize: 11, color: COLORS.stone500, marginTop: 6 }}>{formatTime(msg.timestamp)}</div>
        </div>
      </div>
    );
  };

  return (
    <div style={S.page}>
      {/* Sidebar - same as Dashboard */}
      <aside style={S.sidebar}>
        <div style={{ padding: '0 32px', marginBottom: 16 }}>
          <h1 style={S.sidebarLogo}>LocalFinance</h1>
          <p style={S.sidebarTier}>The Ethereal Vault</p>
        </div>
        <nav>
          <a href="/dashboard" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128196;</span>
            <span style={{ fontSize: 14 }}>Statement Upload</span>
          </a>
          <a href="/dashboard" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#128274;</span>
            <span style={{ fontSize: 14 }}>The Vault</span>
          </a>
          <a href="/chat" style={S.navActive}>
            <span style={{ fontSize: 20 }}>&#128172;</span>
            <span style={{ fontSize: 14, fontWeight: 700 }}>Ollama Chat</span>
          </a>
          <a href="/settings" style={S.navItem}>
            <span style={{ fontSize: 20 }}>&#9881;</span>
            <span style={{ fontSize: 14 }}>Settings</span>
          </a>
        </nav>
        <div style={S.sidebarFooter}>
          <div style={S.avatarCircle}>{user?.first_name?.[0] || user?.username?.[0] || 'U'}</div>
          <div style={{ flex: 1 }}>
            <p style={{ fontSize: 12, fontWeight: 700, margin: 0 }}>{user?.first_name || user?.username} {user?.last_name || ''}</p>
            <button onClick={onLogout} style={S.logoutBtn}>Sign Out</button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main style={S.main}>
        {/* Top header */}
        <header style={S.topBar}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
            <h2 style={{ fontFamily: FONTS.headline, fontSize: 20, margin: 0, fontWeight: 700 }}>Ollama Chat</h2>
            <div style={{ height: 16, width: 1, background: '#e0e0e0' }} />
            <nav style={{ display: 'flex', gap: 24 }}>
              <span style={{ fontSize: 14, fontWeight: 700, borderBottom: '2px solid #1A1A1A', paddingBottom: 4 }}>Analysis</span>
              <span style={{ fontSize: 14, color: '#a0a0a0', cursor: 'pointer' }}>Archive</span>
            </nav>
          </div>
        </header>

        {/* Chat area */}
        <div style={S.chatArea}>
          <div style={S.glassPanel}>
            {/* Input at top */}
            {renderInputRow(topInputRef, 'top')}

            {/* Message feed */}
            <div ref={feedRef} style={S.feed}>
              {!hasMessages && (
                <div style={S.emptyState}>
                  <div style={S.emptyIcon}><span>&#10022;</span></div>
                  <h2 style={S.emptyTitle}>Ask me anything</h2>
                  <p style={S.emptyDesc}>
                    Query your transaction history, get spending insights, or ask
                    for category breakdowns. All processing stays on your device.
                  </p>
                </div>
              )}

              {messages.map(renderMessage)}

              {loading && (
                <div style={S.loadingRow}>
                  <div style={S.aiAvatar}><span role="img" aria-label="AI">&#10022;</span></div>
                  <div style={S.loadingBubble}>Querying Local LLM (Llama 3)...</div>
                </div>
              )}
            </div>

            {/* Bottom input */}
            {hasMessages && renderInputRow(bottomInputRef, 'bottom')}

            {/* Status bar */}
            <div style={S.statusBar}>
              <div style={S.statusDot(statusColor)} />
              <span style={S.statusText}>{statusLabel}</span>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};

const S = {
  // Layout (matches Dashboard)
  page: { display: 'flex', minHeight: '100vh', fontFamily: "'Manrope', sans-serif", color: '#1A1A1A', background: '#fff' },
  sidebar: { width: 256, height: '100vh', position: 'fixed', left: 0, top: 0, zIndex: 40, background: '#fff', borderRight: '1px solid #f5f5f5', display: 'flex', flexDirection: 'column', paddingTop: 32, paddingBottom: 32 },
  sidebarLogo: { fontFamily: "'Newsreader', serif", fontSize: 20, fontWeight: 700, margin: 0 },
  sidebarTier: { fontSize: 10, letterSpacing: '0.2em', textTransform: 'uppercase', color: '#a0a0a0', fontWeight: 700, marginTop: 4 },
  navActive: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: '#1A1A1A', fontWeight: 700, background: '#fafafa', borderRight: '4px solid #1A1A1A', textDecoration: 'none' },
  navItem: { display: 'flex', alignItems: 'center', gap: 16, padding: '12px 32px', color: '#a0a0a0', textDecoration: 'none' },
  sidebarFooter: { marginTop: 'auto', padding: '24px 32px', borderTop: '1px solid #f5f5f5', display: 'flex', alignItems: 'center', gap: 12 },
  avatarCircle: { width: 40, height: 40, borderRadius: '50%', background: '#f5f5f5', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16, fontWeight: 600, textTransform: 'uppercase', flexShrink: 0 },
  logoutBtn: { background: 'none', border: 'none', padding: 0, margin: '2px 0 0 0', fontSize: 10, color: '#a0a0a0', cursor: 'pointer', textDecoration: 'underline', fontFamily: "'Manrope', sans-serif" },
  main: { flex: 1, marginLeft: 256, minHeight: '100vh', background: 'radial-gradient(circle at 50% 50%, #ffffff 0%, #f2f4f4 100%)' },
  topBar: { position: 'sticky', top: 0, zIndex: 30, background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(12px)', borderBottom: '1px solid #f5f5f5', padding: '16px 48px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },

  // Chat area
  chatArea: { display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '32px 48px', minHeight: 'calc(100vh - 64px)' },
  glassPanel: {
    width: '100%', maxWidth: 800, height: 'calc(100vh - 128px)', maxHeight: 820,
    background: 'rgba(255,255,255,0.45)', backdropFilter: 'blur(32px)', WebkitBackdropFilter: 'blur(32px)',
    border: '1px solid rgba(255,255,255,0.9)', boxShadow: '0px 24px 80px rgba(0,0,0,0.08)',
    borderRadius: 16, display: 'flex', flexDirection: 'column', overflow: 'hidden', position: 'relative',
  },
  topInputRow: { padding: '20px 28px 12px 28px', flexShrink: 0 },
  inputContainer: {
    display: 'flex', alignItems: 'center', background: COLORS.white, borderRadius: 12,
    boxShadow: '0 1px 3px rgba(0,0,0,0.04), 0 0 0 1px rgba(0,0,0,0.06)', overflow: 'hidden',
  },
  input: { flex: 1, border: 'none', outline: 'none', padding: '14px 16px', fontSize: 15, fontFamily: FONTS.body, color: COLORS.stone900, background: 'transparent', lineHeight: '1.5' },
  sendButton: { padding: '10px 16px', background: 'transparent', border: 'none', cursor: 'pointer', color: COLORS.stone500, fontSize: 18, display: 'flex', alignItems: 'center', justifyContent: 'center', transition: 'color 0.15s ease', flexShrink: 0 },
  feed: { flex: 1, overflowY: 'auto', padding: '8px 28px 16px 28px', display: 'flex', flexDirection: 'column', gap: 20 },
  emptyState: { flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', textAlign: 'center', padding: '40px 20px' },
  emptyIcon: { width: 64, height: 64, borderRadius: '50%', background: COLORS.stone900, display: 'flex', alignItems: 'center', justifyContent: 'center', color: COLORS.white, fontSize: 24, marginBottom: 20 },
  emptyTitle: { fontFamily: FONTS.headline, fontSize: 22, fontWeight: 400, fontStyle: 'italic', color: COLORS.stone900, margin: '0 0 8px 0' },
  emptyDesc: { fontSize: 14, color: COLORS.stone500, maxWidth: 340, lineHeight: '1.6' },
  userRow: { display: 'flex', justifyContent: 'flex-end' },
  userBubble: { background: 'rgba(28,25,23,0.05)', borderRadius: '14px 14px 4px 14px', padding: '12px 16px', maxWidth: '80%', fontSize: 15, lineHeight: '1.55', color: COLORS.stone900, fontFamily: FONTS.body },
  aiRow: { display: 'flex', alignItems: 'flex-start', gap: 12 },
  aiAvatar: { width: 40, height: 40, borderRadius: '50%', background: COLORS.stone900, display: 'flex', alignItems: 'center', justifyContent: 'center', color: COLORS.white, fontSize: 16, flexShrink: 0, marginTop: 2 },
  aiBubble: { flex: 1, maxWidth: '85%' },
  aiContent: { fontSize: 15, lineHeight: '1.65', color: COLORS.stone700, fontFamily: FONTS.body, whiteSpace: 'pre-wrap' },
  aiError: { fontSize: 14, lineHeight: '1.55', color: COLORS.error, fontFamily: FONTS.body, fontStyle: 'italic' },
  confidenceBadge: { display: 'inline-block', marginTop: 8, fontSize: 11, color: COLORS.stone500, fontFamily: FONTS.body, fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.06em' },
  loadingRow: { display: 'flex', alignItems: 'flex-start', gap: 12 },
  loadingBubble: { fontSize: 14, color: COLORS.stone500, fontStyle: 'italic', fontFamily: FONTS.body, paddingTop: 10 },
  bottomBar: { padding: '12px 28px 16px 28px', flexShrink: 0, borderTop: '1px solid rgba(0,0,0,0.04)' },
  statusBar: { display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, padding: '10px 28px 14px 28px', flexShrink: 0 },
  statusDot: (color) => ({ width: 7, height: 7, borderRadius: '50%', background: color, boxShadow: color === COLORS.green ? `0 0 6px ${color}` : 'none', flexShrink: 0 }),
  statusText: { fontSize: 10, fontFamily: FONTS.body, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.12em', color: COLORS.stone500 },
};

export default Chat;
