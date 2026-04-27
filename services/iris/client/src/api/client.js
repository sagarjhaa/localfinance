import axios from 'axios';

// Create axios instance with base configuration
const apiClient = axios.create({
  baseURL: process.env.REACT_APP_API_URL || '/api',
  timeout: 300000, // 300s for Ollama inference on Jetson (AI parse can take 2-3min)
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

// Auth API
export const authAPI = {
  login: (credentials) => apiClient.post('/auth/login', credentials),
  register: (data) => apiClient.post('/auth/register', data),
  logout: () => apiClient.post('/auth/logout'),
  getMe: () => apiClient.get('/auth/me'),
  refresh: () => apiClient.post('/auth/refresh'),
  validate: (token) => apiClient.post('/auth/validate', { token }),
};

// Document API (polling for processing status)
export const documentAPI = {
  getStatus: (documentId) => apiClient.get(`/upload/status/${documentId}`),
  getTransactions: (documentId) => apiClient.get('/upload/transactions', { params: { document_id: documentId } }),
};

// Upload API
export const uploadAPI = {
  single: (file, onProgress) => {
    const formData = new FormData();
    formData.append('file', file);
    
    return apiClient.post('/upload/single', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: onProgress,
    });
  },
  
  multiple: (files, onProgress) => {
    const formData = new FormData();
    files.forEach((file) => {
      formData.append('files', file);
    });
    
    return apiClient.post('/upload/multiple', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: onProgress,
    });
  },
  
  getFiles: () => apiClient.get('/upload/files'),
  deleteFile: (filename) => apiClient.delete(`/upload/files/${filename}`),
};

// Proxy API
export const proxyAPI = {
  health: () => apiClient.get('/proxy/health'),
  services: () => apiClient.get('/proxy/services'),
  batch: (requests) => apiClient.post('/proxy/batch', { requests }),
  
  // Service-specific methods
  hermes: {
    get: (path, params) => apiClient.get(`/proxy/hermes${path}`, { params }),
    post: (path, data) => apiClient.post(`/proxy/hermes${path}`, data),
    put: (path, data) => apiClient.put(`/proxy/hermes${path}`, data),
    delete: (path) => apiClient.delete(`/proxy/hermes${path}`),
  },
  
  thesaurus: {
    get: (path, params, config) => apiClient.get(`/proxy/thesaurus${path}`, { params, ...config }),
    post: (path, data, config) => apiClient.post(`/proxy/thesaurus${path}`, data, config),
    put: (path, data, config) => apiClient.put(`/proxy/thesaurus${path}`, data, config),
    delete: (path, config) => apiClient.delete(`/proxy/thesaurus${path}`, config),
  },
  
  logos: {
    get: (path, params) => apiClient.get(`/proxy/logos${path}`, { params }),
    post: (path, data) => apiClient.post(`/proxy/logos${path}`, data),
    put: (path, data) => apiClient.put(`/proxy/logos${path}`, data),
    delete: (path) => apiClient.delete(`/proxy/logos${path}`),
  },
  
  sophia: {
    get: (path, params) => apiClient.get(`/proxy/sophia${path}`, { params }),
    post: (path, data) => apiClient.post(`/proxy/sophia${path}`, data),
    put: (path, data) => apiClient.put(`/proxy/sophia${path}`, data),
    delete: (path) => apiClient.delete(`/proxy/sophia${path}`),
  },
};

// Insights / Month-in-Review API (Iris -> Sophia/Thesaurus)
export const insightsAPI = {
  list: (userId) => apiClient.get(`/v1/insights/${encodeURIComponent(userId)}`),
  generate: (userId, period) => apiClient.post('/v1/insights/', { user_id: userId, period }),
  dismiss: (userId, { insight_key, rule_id }) =>
    apiClient.post(`/v1/insights/${encodeURIComponent(userId)}/dismiss`, { insight_key, rule_id }),
};

export const monthReviewAPI = {
  get: (period, userId) =>
    apiClient.get(`/v1/month-review/${encodeURIComponent(period)}`, { params: { user_id: userId } }),
  invalidate: (period, userId) =>
    apiClient.delete(`/v1/month-review/${encodeURIComponent(period)}`, { params: { user_id: userId } }),
};

// Health check
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
      error: error.message || errorMessage 
    };
  }
};

export default apiClient;