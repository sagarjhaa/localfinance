import React, { useState, useEffect } from 'react';
import './App.css';

// API base URL
const API_BASE_URL = 'http://localhost:3000/api';

// Auth service
const AuthService = {
  login: async (email, password) => {
    const response = await fetch(`${API_BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    const data = await response.json();
    if (response.ok) {
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      return data;
    }
    throw new Error(data.error || 'Login failed');
  },

  register: async (email, password, firstName, lastName) => {
    const response = await fetch(`${API_BASE_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, first_name: firstName, last_name: lastName }),
    });
    const data = await response.json();
    if (response.ok) {
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      return data;
    }
    throw new Error(data.error || 'Registration failed');
  },

  logout: () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  },

  getToken: () => localStorage.getItem('token'),
  getUser: () => {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  },

  isAuthenticated: () => !!localStorage.getItem('token'),
};

// Login Form Component
function LoginForm({ onLogin, onToggleForm }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const data = await AuthService.login(email, password);
      onLogin(data.user);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-form">
        <h2>🏛️ LocalFinance Login</h2>
        {error && <div className="error-message">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Email:</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label>Password:</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={loading}
            />
          </div>
          <button type="submit" disabled={loading} className="primary-button">
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>
        <p className="toggle-form">
          Don't have an account? 
          <button onClick={onToggleForm} className="link-button">
            Register here
          </button>
        </p>
      </div>
    </div>
  );
}

// Register Form Component
function RegisterForm({ onLogin, onToggleForm }) {
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    firstName: '',
    lastName: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const data = await AuthService.register(
        formData.email,
        formData.password,
        formData.firstName,
        formData.lastName
      );
      onLogin(data.user);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-form">
        <h2>🏛️ LocalFinance Register</h2>
        {error && <div className="error-message">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>First Name:</label>
            <input
              type="text"
              name="firstName"
              value={formData.firstName}
              onChange={handleChange}
              required
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label>Last Name:</label>
            <input
              type="text"
              name="lastName"
              value={formData.lastName}
              onChange={handleChange}
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label>Email:</label>
            <input
              type="email"
              name="email"
              value={formData.email}
              onChange={handleChange}
              required
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label>Password:</label>
            <input
              type="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              required
              minLength="6"
              disabled={loading}
            />
          </div>
          <button type="submit" disabled={loading} className="primary-button">
            {loading ? 'Creating account...' : 'Register'}
          </button>
        </form>
        <p className="toggle-form">
          Already have an account? 
          <button onClick={onToggleForm} className="link-button">
            Login here
          </button>
        </p>
      </div>
    </div>
  );
}

// Dashboard Component
function Dashboard({ user, onLogout }) {
  const [services, setServices] = useState({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check service health
    const checkHealth = async () => {
      try {
        const response = await fetch('/health');
        const data = await response.json();
        setServices(data.services || {});
      } catch (err) {
        console.error('Health check failed:', err);
      } finally {
        setLoading(false);
      }
    };

    checkHealth();
  }, []);

  const handleLogout = () => {
    AuthService.logout();
    onLogout();
  };

  return (
    <div className="dashboard">
      <header className="dashboard-header">
        <h1>🏛️ LocalFinance Dashboard</h1>
        <div className="user-info">
          <span>Welcome, {user.first_name}!</span>
          <button onClick={handleLogout} className="logout-button">
            Logout
          </button>
        </div>
      </header>

      <main className="dashboard-content">
        <div className="service-grid">
          <div className="service-card">
            <h3>🏛️ Thesaurus</h3>
            <p>Database & CRUD Operations</p>
            <div className="service-status">
              Status: <span className="status-indicator">🟢 Healthy</span>
            </div>
          </div>

          <div className="service-card">
            <h3>🦉 Sophia</h3>
            <p>AI Financial Insights</p>
            <div className="service-status">
              Status: <span className="status-indicator">🟢 Healthy</span>
            </div>
          </div>

          <div className="service-card">
            <h3>📜 Logos</h3>
            <p>Document Processing</p>
            <div className="service-status">
              Status: <span className="status-indicator">🟢 Healthy</span>
            </div>
          </div>
        </div>

        <div className="quick-actions">
          <h3>Quick Actions</h3>
          <div className="action-buttons">
            <button className="action-button">📊 View Transactions</button>
            <button className="action-button">📝 Upload Document</button>
            <button className="action-button">💬 Ask AI Assistant</button>
            <button className="action-button">📋 Manage Budgets</button>
          </div>
        </div>

        <div className="recent-activity">
          <h3>Recent Activity</h3>
          <div className="activity-list">
            <div className="activity-item">
              <span className="activity-icon">📄</span>
              <span className="activity-text">Welcome to LocalFinance!</span>
              <span className="activity-time">Just now</span>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

// Main App Component
function App() {
  const [user, setUser] = useState(null);
  const [isLogin, setIsLogin] = useState(true);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check if user is already logged in
    const token = AuthService.getToken();
    const savedUser = AuthService.getUser();
    
    if (token && savedUser) {
      setUser(savedUser);
    }
    
    setLoading(false);
  }, []);

  const handleLogin = (userData) => {
    setUser(userData);
  };

  const handleLogout = () => {
    setUser(null);
  };

  const toggleForm = () => {
    setIsLogin(!isLogin);
  };

  if (loading) {
    return (
      <div className="loading-container">
        <h2>🏛️ LocalFinance</h2>
        <p>Loading...</p>
      </div>
    );
  }

  if (user) {
    return <Dashboard user={user} onLogout={handleLogout} />;
  }

  return isLogin ? (
    <LoginForm onLogin={handleLogin} onToggleForm={toggleForm} />
  ) : (
    <RegisterForm onLogin={handleLogin} onToggleForm={toggleForm} />
  );
}

export default App;