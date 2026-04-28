import axios from 'axios';

// After the consolidation (Phase 1.5), every endpoint lives on the same
// Go binary at /api/v1/* (with a few unversioned ones at /api/setup/* and
// /health). The legacy Iris Express proxy that used to map /api/auth/*,
// /api/upload/*, and /api/proxy/<svc>/* into Thesaurus/Sophia/Logos paths
// is gone. baseURL is empty so every call below uses full absolute paths
// matching the server.
const apiClient = axios.create({
  baseURL: process.env.REACT_APP_API_URL || '',
  timeout: 300000, // 300s for Ollama inference (AI parse can take 2-3min)
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('authToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle common errors
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    // Handle 401 Unauthorized — only logout on explicit auth failures, not background fetches
    if (error.response?.status === 401 && !error.config?.skipLogoutOn401) {
      localStorage.removeItem('authToken');
      localStorage.removeItem('user');
      window.location.href = '/login';
      return Promise.reject(new Error('Session expired. Please login again.'));
    }

    // Handle network errors
    if (!error.response) {
      return Promise.reject(new Error('Network error. Please check your connection.'));
    }

    // Handle other HTTP errors
    const message = error.response.data?.message || `HTTP ${error.response.status} Error`;
    return Promise.reject(new Error(message));
  }
);

// Auth API — server: /api/v1/auth/*
export const authAPI = {
  login: (credentials) => apiClient.post('/api/v1/auth/login', credentials),
  register: (data) => apiClient.post('/api/v1/auth/register', data),
  logout: () => apiClient.post('/api/v1/auth/logout'),
  getMe: () => apiClient.get('/api/v1/auth/me'),
  refresh: () => apiClient.post('/api/v1/auth/refresh'),
  validate: (token) => apiClient.post('/api/v1/auth/validate', { token }),
  changePassword: ({ current_password, new_password }) =>
    apiClient.post('/api/v1/auth/change-password', { current_password, new_password }),
};

// Document API — server: /api/v1/documents/* and /api/v1/transactions/by-document
export const documentAPI = {
  getStatus: (documentId) =>
    apiClient.get(`/api/v1/documents/${encodeURIComponent(documentId)}`),
  getTransactions: (documentId) =>
    apiClient.get('/api/v1/transactions/by-document', { params: { document_id: documentId } }),
};

// Upload API — server: POST /api/v1/upload (single file, multipart). The old
// /upload/multiple, /upload/files, /upload/files/:name endpoints don't exist
// on the consolidated binary; multiple-file flows just call single() in a loop.
export const uploadAPI = {
  single: (file, onProgress) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.post('/api/v1/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: onProgress,
    });
  },
};

// proxyAPI shim — kept so existing pages (Profile, Dashboard polling, etc.)
// keep working. Each method routes the caller's already-fully-qualified path
// straight to the server. Pre-consolidation, /api/proxy/<svc> would hop through
// Iris; now there's no hop.
export const proxyAPI = {
  thesaurus: {
    get: (path, params, config) => apiClient.get(path, { params, ...config }),
    post: (path, data, config) => apiClient.post(path, data, config),
    put: (path, data, config) => apiClient.put(path, data, config),
    delete: (path, config) => apiClient.delete(path, config),
  },
  sophia: {
    get: (path, params) => apiClient.get(path, { params }),
    post: (path, data) => apiClient.post(path, data),
    put: (path, data) => apiClient.put(path, data),
    delete: (path) => apiClient.delete(path),
  },
  // Hermes was a thin gateway service that's now gone. Logos got folded into
  // the parse pipeline. Both kept here as no-op shims so existing imports
  // don't blow up — but pages should stop using them.
  hermes: {
    get: (path, params) => apiClient.get(path, { params }),
    post: (path, data) => apiClient.post(path, data),
    put: (path, data) => apiClient.put(path, data),
    delete: (path) => apiClient.delete(path),
  },
  logos: {
    get: (path, params) => apiClient.get(path, { params }),
    post: (path, data) => apiClient.post(path, data),
    put: (path, data) => apiClient.put(path, data),
    delete: (path) => apiClient.delete(path),
  },
};

// Insights / Month-in-Review API — server: /api/v1/insights/*, /api/v1/month-review/*
export const insightsAPI = {
  list: (userId) => apiClient.get(`/api/v1/insights/${encodeURIComponent(userId)}`),
  generate: (userId, period) => apiClient.post('/api/v1/insights/', { user_id: userId, period }),
  dismiss: (userId, { insight_key, rule_id }) =>
    apiClient.post(`/api/v1/insights/${encodeURIComponent(userId)}/dismiss`, { insight_key, rule_id }),
};

export const monthReviewAPI = {
  get: (period, userId) =>
    apiClient.get(`/api/v1/month-review/${encodeURIComponent(period)}`, { params: { user_id: userId } }),
  invalidate: (period, userId) =>
    apiClient.delete(`/api/v1/month-review/${encodeURIComponent(period)}`, { params: { user_id: userId } }),
  // Returns { periods: ["2026-04", "2026-03", ...] } sorted newest-first.
  // Only includes months for which at least one transaction exists.
  periods: (userId) =>
    apiClient.get('/api/v1/month-review/periods', { params: { user_id: userId } }),
};

export const transactionAPI = {
  // Resolve transaction IDs to a condensed view (date, description,
  // amount, category, account name). Used by insight cards + chat
  // answers to show provenance.
  lookup: (ids) =>
    apiClient.post('/api/v1/transactions/lookup', { ids }),
};

// Health check — unversioned, served at root
export const healthAPI = {
  check: () => apiClient.get('/health'),
};

// Utility functions
export const isAuthenticated = () => {
  const token = localStorage.getItem('authToken');
  const user = localStorage.getItem('user');
  return !!(token && user);
};

export const getUser = () => {
  try {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  } catch (error) {
    console.error('Error parsing user data:', error);
    return null;
  }
};

export const setAuthData = (token, user) => {
  localStorage.setItem('authToken', token);
  localStorage.setItem('user', JSON.stringify(user));
};

export const clearAuthData = () => {
  localStorage.removeItem('authToken');
  localStorage.removeItem('user');
};

// Format file size
export const formatFileSize = (bytes) => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

// Format date
export const formatDate = (dateString) => {
  const date = new Date(dateString);
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
};

// Handle async operations with error handling
export const handleAsync = async (asyncFn, errorMessage = 'Operation failed') => {
  try {
    const result = await asyncFn();
    return { data: result.data, error: null };
  } catch (error) {
    console.error(errorMessage, error);
    return {
      data: null,
      error: error.message || errorMessage,
    };
  }
};

export default apiClient;
