import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { isAuthenticated, authAPI, clearAuthData } from './api/client';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import Chat from './pages/Chat';
import Profile from './pages/Profile';
import InsightsPage from './pages/InsightsPage';
import MonthReviewPage from './pages/MonthReviewPage';
import InstallOllama from './pages/setup/InstallOllama';
import PullModel from './pages/setup/PullModel';
import { ToastProvider } from './components/Toast';
import { ThemeProvider, ThemeToggle } from './components/ThemeContext';


function AppInner() {
  const [user, setUser] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [setupStep, setSetupStep] = useState(null); // 'install_ollama' | 'pull_model' | 'ready'

  useEffect(() => {
    const initialize = async () => {
      // First: check setup state. The wizard runs before auth.
      try {
        const res = await fetch('/api/setup/state');
        if (res.ok) {
          const data = await res.json();
          setSetupStep(data.step);
        } else {
          setSetupStep('ready'); // assume ready on error so we don't block forever
        }
      } catch (_) {
        setSetupStep('ready');
      }

      if (isAuthenticated()) {
        try {
          const response = await authAPI.getMe();
          setUser(response.data.user);
        } catch (error) {
          console.error('Auth initialization failed:', error);
          clearAuthData();
        }
      }
      setIsLoading(false);
    };
    initialize();
  }, []);

  const handleLogin = (userData) => {
    setUser(userData);
  };

  const handleLogout = async () => {
    try {
      await authAPI.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      clearAuthData();
      setUser(null);
    }
  };

  if (isLoading) {
    return (
      <div style={{
        minHeight: '100vh', background: '#fff', display: 'flex',
        alignItems: 'center', justifyContent: 'center', fontFamily: "'Manrope', sans-serif",
      }}>
        <div style={{ textAlign: 'center', color: '#2d3435' }}>
          <div style={{
            width: 40, height: 40, border: '3px solid #e5e7eb', borderTop: '3px solid #5f5e5e',
            borderRadius: '50%', animation: 'spin 1s linear infinite', margin: '0 auto 16px',
          }} />
          <h1 style={{ fontFamily: "'Instrument Serif', serif", fontSize: 28, fontStyle: 'italic', margin: 0 }}>
            LocalFinance
          </h1>
          <p style={{ fontSize: 13, color: '#8C8C8C', marginTop: 8 }}>Initializing...</p>
        </div>
        <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>
      </div>
    );
  }

  // Setup wizard takes precedence over auth — runs before login.
  if (setupStep === 'install_ollama' || setupStep === 'pull_model') {
    const target = setupStep === 'install_ollama' ? '/setup/install' : '/setup/pull';
    return (
      <Router>
        <Routes>
          <Route path="/setup/install" element={<InstallOllama />} />
          <Route path="/setup/pull" element={<PullModel />} />
          <Route path="*" element={<Navigate to={target} replace />} />
        </Routes>
      </Router>
    );
  }

  if (!user) {
    return (
      <Router>
        <Routes>
          <Route path="/setup/install" element={<InstallOllama />} />
          <Route path="/setup/pull" element={<PullModel />} />
          <Route path="/login" element={<Login onLogin={handleLogin} />} />
          <Route path="/register" element={<Register onLogin={handleLogin} />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </Router>
    );
  }

  return (
    <ToastProvider>
      <Router>
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard user={user} onLogout={handleLogout} />} />

          <Route path="/chat" element={<Chat user={user} onLogout={handleLogout} />} />
          <Route path="/profile" element={<Profile user={user} setUser={setUser} onLogout={handleLogout} />} />
          <Route path="/insights" element={<InsightsPage user={user} onLogout={handleLogout} />} />
          <Route path="/month-review" element={<MonthReviewPage user={user} onLogout={handleLogout} />} />
          <Route path="/month-review/:period" element={<MonthReviewPage user={user} onLogout={handleLogout} />} />

          <Route path="/setup/install" element={<InstallOllama />} />
          <Route path="/setup/pull" element={<PullModel />} />

          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </Router>
      <ThemeToggle />
    </ToastProvider>
  );
}

function App() {
  return (
    <ThemeProvider>
      <AppInner />
    </ThemeProvider>
  );
}

export default App;
