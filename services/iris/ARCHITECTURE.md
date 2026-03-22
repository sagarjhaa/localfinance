# Iris Service Architecture

**Service:** Iris (Greek goddess of the rainbow)  
**Port:** 3001  
**Stack:** React Frontend + Node.js/Express Backend  
**Purpose:** Main frontend interface for LocalFinance ecosystem

---

## Overview

Iris serves as the primary user-facing service for LocalFinance, providing a unified web interface that connects users to all backend services through a seamless React application backed by a Node.js proxy server.

```
┌─────────────────────────────────────────────────────────────┐
│                        IRIS (Port 3001)                    │
│                                                             │
│  ┌─────────────────┐    ┌───────────────────────────────┐  │
│  │  React Client   │    │     Node.js/Express Server   │  │
│  │                 │    │                               │  │
│  │ • Components    │◄──►│ • Authentication Routes       │  │
│  │ • Pages         │    │ • Service Proxy Routes        │  │
│  │ • State Mgmt    │    │ • Session Management          │  │
│  │ • API Client    │    │ • File Upload Handling        │  │
│  │ • Routing       │    │ • Static File Serving         │  │
│  └─────────────────┘    └───────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │      HERMES (Gateway)         │
                    │         Port 3000             │
                    └───────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              ┌──────────┐    ┌──────────┐    ┌──────────┐
              │ THESAURUS│    │  LOGOS   │    │  SOPHIA  │
              │Port 3002 │    │Port 3003 │    │Port 3004 │
              └──────────┘    └──────────┘    └──────────┘
```

---

## Directory Structure

```
services/iris/
├── package.json                 # Root package.json (workspace config)
├── Dockerfile                   # Multi-stage build for prod deployment
├── docker-compose.yml           # Local development setup
├── .env.example                 # Environment variables template
├── README.md                    # Service documentation
│
├── server/                      # Node.js/Express Backend
│   ├── package.json             # Server dependencies
│   ├── index.js                 # Server entry point
│   ├── config/                  # Configuration management
│   │   ├── database.js
│   │   ├── session.js
│   │   └── services.js
│   ├── middleware/              # Express middleware
│   │   ├── auth.js              # Authentication middleware
│   │   ├── cors.js              # CORS configuration
│   │   ├── logging.js           # Request logging
│   │   ├── errorHandler.js      # Error handling
│   │   └── fileUpload.js        # Multipart upload handling
│   ├── routes/                  # API route handlers
│   │   ├── auth/                # Authentication routes
│   │   │   ├── login.js
│   │   │   ├── logout.js
│   │   │   ├── register.js
│   │   │   └── profile.js
│   │   ├── proxy/               # Service proxy routes
│   │   │   ├── thesaurus.js     # Proxy to Thesaurus service
│   │   │   ├── logos.js         # Proxy to Logos service
│   │   │   └── sophia.js        # Proxy to Sophia service
│   │   ├── uploads/             # File upload routes
│   │   │   └── index.js
│   │   └── health/              # Health check routes
│   │       └── index.js
│   ├── services/                # External service clients
│   │   ├── hermesClient.js      # Gateway communication
│   │   ├── authService.js       # Authentication logic
│   │   └── fileService.js       # File handling logic
│   └── utils/                   # Utility functions
│       ├── logger.js
│       ├── validation.js
│       └── httpProxy.js
│
└── client/                      # React Frontend
    ├── package.json             # Client dependencies
    ├── public/                  # Static assets
    │   ├── index.html
    │   ├── favicon.ico
    │   └── manifest.json
    ├── src/                     # React source code
    │   ├── index.js             # React entry point
    │   ├── App.js               # Root component
    │   ├── components/          # Reusable UI components
    │   │   ├── common/          # Common components
    │   │   │   ├── Header/
    │   │   │   ├── Footer/
    │   │   │   ├── Navigation/
    │   │   │   ├── Loading/
    │   │   │   └── ErrorBoundary/
    │   │   ├── auth/            # Authentication components
    │   │   │   ├── LoginForm/
    │   │   │   ├── RegisterForm/
    │   │   │   └── UserProfile/
    │   │   ├── upload/          # File upload components
    │   │   │   ├── FileUploader/
    │   │   │   ├── ProgressBar/
    │   │   │   └── FileList/
    │   │   └── forms/           # Form components
    │   │       ├── Input/
    │   │       ├── Button/
    │   │       └── Select/
    │   ├── pages/               # Page components
    │   │   ├── HomePage/
    │   │   ├── LoginPage/
    │   │   ├── DashboardPage/
    │   │   ├── ProfilePage/
    │   │   └── NotFoundPage/
    │   ├── hooks/               # Custom React hooks
    │   │   ├── useAuth.js
    │   │   ├── useApi.js
    │   │   ├── useFileUpload.js
    │   │   └── useLocalStorage.js
    │   ├── api/                 # API client setup
    │   │   ├── client.js        # Axios configuration
    │   │   ├── auth.js          # Authentication API calls
    │   │   ├── thesaurus.js     # Thesaurus service calls
    │   │   ├── logos.js         # Logos service calls
    │   │   ├── sophia.js        # Sophia service calls
    │   │   └── uploads.js       # File upload API calls
    │   ├── store/               # State management
    │   │   ├── authStore.js     # Authentication state
    │   │   ├── uiStore.js       # UI state
    │   │   └── fileStore.js     # File upload state
    │   ├── utils/               # Utility functions
    │   │   ├── formatters.js
    │   │   ├── validators.js
    │   │   └── constants.js
    │   └── styles/              # Global styles
    │       ├── globals.css
    │       ├── variables.css
    │       └── components.css
    └── build/                   # Production build output (generated)
```

---

## Backend Architecture (Node.js/Express)

### Core Server Configuration

**Entry Point:** `server/index.js`
```javascript
const express = require('express');
const cors = require('cors');
const session = require('express-session');
const path = require('path');

// Middleware imports
const authMiddleware = require('./middleware/auth');
const loggingMiddleware = require('./middleware/logging');
const errorHandler = require('./middleware/errorHandler');

// Route imports
const authRoutes = require('./routes/auth');
const proxyRoutes = require('./routes/proxy');
const uploadRoutes = require('./routes/uploads');
const healthRoutes = require('./routes/health');

const app = express();
const PORT = process.env.PORT || 3001;

// Middleware setup
app.use(cors({ origin: true, credentials: true }));
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true, limit: '50mb' }));
app.use(loggingMiddleware);
app.use(session(sessionConfig));

// Routes
app.use('/api/auth', authRoutes);
app.use('/api/proxy', authMiddleware, proxyRoutes);
app.use('/api/uploads', authMiddleware, uploadRoutes);
app.use('/api/health', healthRoutes);

// Serve React static files
app.use(express.static(path.join(__dirname, '../client/build')));

// Catch-all handler for React Router
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, '../client/build/index.html'));
});

// Error handling
app.use(errorHandler);

app.listen(PORT, () => {
  console.log(`🌈 Iris server running on port ${PORT}`);
});
```

### Authentication Routes

**Route Prefix:** `/api/auth`

| Method | Endpoint | Description | Body | Response |
|--------|----------|-------------|------|----------|
| POST | `/login` | User login | `{email, password}` | `{user, token, sessionId}` |
| POST | `/logout` | User logout | - | `{success: true}` |
| POST | `/register` | User registration | `{name, email, password}` | `{user, token, sessionId}` |
| GET | `/profile` | Get user profile | - | `{user}` |
| PUT | `/profile` | Update profile | `{name, email, ...}` | `{user}` |
| GET | `/verify` | Verify token | - | `{valid: boolean, user}` |

### Service Proxy Routes

**Route Prefix:** `/api/proxy`

| Method | Endpoint | Description | Forwards To |
|--------|----------|-------------|-------------|
| ALL | `/thesaurus/*` | Proxy to Thesaurus service | `http://thesaurus:3002/*` |
| ALL | `/logos/*` | Proxy to Logos service | `http://logos:3003/*` |
| ALL | `/sophia/*` | Proxy to Sophia service | `http://sophia:3004/*` |

**Proxy Implementation:**
```javascript
// routes/proxy/thesaurus.js
const express = require('express');
const httpProxy = require('../utils/httpProxy');
const router = express.Router();

const THESAURUS_URL = process.env.THESAURUS_URL || 'http://localhost:3002';

router.all('/*', async (req, res) => {
  try {
    const response = await httpProxy({
      method: req.method,
      url: `${THESAURUS_URL}${req.path}`,
      headers: {
        ...req.headers,
        'x-user-id': req.user?.id,
        'x-session-id': req.sessionID,
      },
      body: req.body,
      params: req.query,
    });
    
    res.status(response.status).json(response.data);
  } catch (error) {
    res.status(error.status || 500).json({ 
      error: 'Proxy request failed',
      message: error.message 
    });
  }
});

module.exports = router;
```

### Session Management

**Configuration:** `config/session.js`
```javascript
const session = require('express-session');
const MongoStore = require('connect-mongo');

const sessionConfig = {
  secret: process.env.SESSION_SECRET || 'iris-session-secret',
  resave: false,
  saveUninitialized: false,
  store: MongoStore.create({
    mongoUrl: process.env.MONGODB_URL || 'mongodb://localhost:27017/iris-sessions'
  }),
  cookie: {
    secure: process.env.NODE_ENV === 'production',
    httpOnly: true,
    maxAge: 1000 * 60 * 60 * 24, // 24 hours
  },
  name: 'iris.sid',
};

module.exports = sessionConfig;
```

### File Upload Handling

**Route:** `/api/uploads`
```javascript
const multer = require('multer');
const path = require('path');

const storage = multer.diskStorage({
  destination: (req, file, cb) => {
    cb(null, 'uploads/');
  },
  filename: (req, file, cb) => {
    const uniqueSuffix = Date.now() + '-' + Math.round(Math.random() * 1E9);
    cb(null, file.fieldname + '-' + uniqueSuffix + path.extname(file.originalname));
  }
});

const upload = multer({ 
  storage: storage,
  limits: { fileSize: 10 * 1024 * 1024 }, // 10MB limit
  fileFilter: (req, file, cb) => {
    // Validate file types
    const allowedTypes = /jpeg|jpg|png|gif|pdf|doc|docx|txt/;
    const extname = allowedTypes.test(path.extname(file.originalname).toLowerCase());
    const mimetype = allowedTypes.test(file.mimetype);
    
    if (mimetype && extname) {
      return cb(null, true);
    } else {
      cb(new Error('Invalid file type'));
    }
  }
});

// Routes
router.post('/single', upload.single('file'), async (req, res) => {
  try {
    const uploadedFile = {
      filename: req.file.filename,
      originalname: req.file.originalname,
      size: req.file.size,
      path: req.file.path,
      uploadedBy: req.user.id,
      uploadedAt: new Date(),
    };
    
    // Forward to Logos service for processing
    const logosResponse = await httpProxy({
      method: 'POST',
      url: `${LOGOS_URL}/api/files/process`,
      body: uploadedFile,
      headers: { 'x-user-id': req.user.id }
    });
    
    res.json(logosResponse.data);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

router.post('/multiple', upload.array('files', 5), async (req, res) => {
  // Handle multiple file uploads
});
```

---

## Frontend Architecture (React)

### State Management Strategy

**Zustand Store Configuration:**
```javascript
// store/authStore.js
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export const useAuthStore = create(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      
      login: async (credentials) => {
        try {
          const response = await authApi.login(credentials);
          set({
            user: response.user,
            token: response.token,
            isAuthenticated: true,
          });
          return response;
        } catch (error) {
          throw error;
        }
      },
      
      logout: () => {
        authApi.logout();
        set({
          user: null,
          token: null,
          isAuthenticated: false,
        });
      },
      
      updateProfile: (userData) => {
        set({ user: { ...get().user, ...userData } });
      },
    }),
    {
      name: 'iris-auth',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);
```

### Component Structure

**App Component:**
```jsx
// App.js
import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { ErrorBoundary } from './components/common/ErrorBoundary';
import { Header } from './components/common/Header';
import { PrivateRoute } from './components/auth/PrivateRoute';

// Pages
import HomePage from './pages/HomePage';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import ProfilePage from './pages/ProfilePage';
import NotFoundPage from './pages/NotFoundPage';

function App() {
  return (
    <ErrorBoundary>
      <Router>
        <div className="app">
          <Header />
          <main className="main-content">
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/dashboard" element={
                <PrivateRoute>
                  <DashboardPage />
                </PrivateRoute>
              } />
              <Route path="/profile" element={
                <PrivateRoute>
                  <ProfilePage />
                </PrivateRoute>
              } />
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </main>
        </div>
      </Router>
    </ErrorBoundary>
  );
}

export default App;
```

### API Client Setup

**Base API Client:**
```javascript
// api/client.js
import axios from 'axios';
import { useAuthStore } from '../store/authStore';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:3001/api';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
apiClient.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout();
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default apiClient;
```

### Custom Hooks

**API Hook:**
```javascript
// hooks/useApi.js
import { useState, useEffect } from 'react';
import apiClient from '../api/client';

export const useApi = (url, options = {}) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const response = await apiClient.get(url, options);
        setData(response.data);
      } catch (err) {
        setError(err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [url]);

  return { data, loading, error, refetch: fetchData };
};
```

**File Upload Hook:**
```javascript
// hooks/useFileUpload.js
import { useState } from 'react';
import { uploadApi } from '../api/uploads';

export const useFileUpload = () => {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState(null);

  const uploadFile = async (file, onProgress) => {
    setUploading(true);
    setError(null);
    setProgress(0);

    try {
      const result = await uploadApi.uploadSingle(file, (progressEvent) => {
        const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
        setProgress(progress);
        onProgress?.(progress);
      });
      
      setUploading(false);
      return result;
    } catch (err) {
      setError(err.message);
      setUploading(false);
      throw err;
    }
  };

  return { uploadFile, uploading, progress, error };
};
```

---

## Service Communication

### Communication Patterns

```
┌─────────────────────────────────────────────────────────────┐
│                    IRIS Communication Flow                  │
└─────────────────────────────────────────────────────────────┘

React Client ──────► Node.js Server ──────► Hermes Gateway
     ▲                       │                      │
     │                       ▼                      ▼
     │               ┌─────────────────┐    ┌─────────────────┐
     │               │ Direct Service  │    │ Service Routing │
     │               │ Calls (bypass   │    │ & Load          │
     │               │ Hermes for      │    │ Balancing       │
     │               │ performance)    │    │                 │
     └───────────────┴─────────────────┘    └─────────────────┘
                             │                      │
                             ▼                      ▼
                    ┌─────────────┐        ┌─────────────┐
                    │  Service A  │        │  Service B  │
                    │ (Thesaurus) │        │ (Logos)     │
                    └─────────────┘        └─────────────┘
```

### Communication Strategy

**Dual-Path Approach:**
1. **Through Hermes (Default):** All requests route through Hermes gateway for service discovery, load balancing, and monitoring
2. **Direct Connection (Performance Critical):** File uploads and high-frequency operations bypass Hermes for reduced latency

### Authentication Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Authentication Flow                      │
└─────────────────────────────────────────────────────────────┘

1. User Login Request
React ──POST /api/auth/login──► Iris Server
                                     │
                                     ▼
2. Validate Credentials              │
Iris ────────────────────────────► User DB
                                     │
                                     ▼
3. Create Session & JWT              │
Iris ◄───────────────────────────── Session Store
                                     │
                                     ▼
4. Return Tokens                     │
React ◄──{user, token, sessionId}─── Iris Server

5. Subsequent Requests
React ──Authorization: Bearer <token>──► Iris Server
                                           │
                                           ▼
6. Token Validation                        │
Iris ──────────────────────────────────► JWT Verify
                                           │
                                           ▼
7. Forward with User Context               │
Service ◄──x-user-id, x-session-id─────── Iris Server
```

### File Upload Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     File Upload Flow                        │
└─────────────────────────────────────────────────────────────┘

1. File Selection
React File Picker ──► FileUpload Component
                              │
                              ▼
2. Upload with Progress       │
React ──POST /api/uploads/single──► Iris Server
      └─multipart/form-data───┘         │
                                        ▼
3. File Processing                      │
Iris ──────────────────────────────► Local Storage
                                        │
                                        ▼
4. Forward to Logos                     │
Iris ──POST /api/files/process──────► Logos Service
     └─{file metadata}──────────┘         │
                                          ▼
5. File Analysis & Storage                │
Logos ─────────────────────────────────► File DB + S3
                                          │
                                          ▼
6. Response with File ID                  │
React ◄──{fileId, status, metadata}───── Logos Service
                                    (via Iris)
```

---

## API Contracts

### Authentication Endpoints

```yaml
POST /api/auth/login:
  body:
    email: string (required)
    password: string (required)
  response:
    user: User
    token: string (JWT)
    sessionId: string
    expiresAt: ISO8601

POST /api/auth/register:
  body:
    name: string (required)
    email: string (required)
    password: string (required, min 8 chars)
  response:
    user: User
    token: string
    sessionId: string

GET /api/auth/profile:
  headers:
    Authorization: Bearer <token>
  response:
    user: User

PUT /api/auth/profile:
  headers:
    Authorization: Bearer <token>
  body:
    name?: string
    email?: string
    settings?: object
  response:
    user: User (updated)
```

### Proxy Endpoints

```yaml
ALL /api/proxy/thesaurus/*:
  headers:
    Authorization: Bearer <token>
    x-user-id: string (auto-injected)
    x-session-id: string (auto-injected)
  forwards_to: http://thesaurus:3002/*
  
ALL /api/proxy/logos/*:
  headers:
    Authorization: Bearer <token>
    x-user-id: string (auto-injected)
    x-session-id: string (auto-injected)
  forwards_to: http://logos:3003/*

ALL /api/proxy/sophia/*:
  headers:
    Authorization: Bearer <token>
    x-user-id: string (auto-injected)
    x-session-id: string (auto-injected)
  forwards_to: http://sophia:3004/*
```

### Upload Endpoints

```yaml
POST /api/uploads/single:
  headers:
    Authorization: Bearer <token>
  body:
    file: File (multipart/form-data)
    description?: string
    tags?: string[]
  response:
    fileId: string
    filename: string
    originalName: string
    size: number
    mimeType: string
    uploadedAt: ISO8601
    processedBy: string (logos)

POST /api/uploads/multiple:
  headers:
    Authorization: Bearer <token>
  body:
    files: File[] (max 5 files)
    description?: string
    tags?: string[]
  response:
    uploads: UploadResult[]
    total: number
    successful: number
    failed: number
```

---

## Data Models

### User Model
```typescript
interface User {
  id: string;
  name: string;
  email: string;
  createdAt: Date;
  updatedAt: Date;
  settings: {
    theme: 'light' | 'dark';
    language: string;
    notifications: boolean;
  };
}
```

### Session Model
```typescript
interface Session {
  sessionId: string;
  userId: string;
  createdAt: Date;
  expiresAt: Date;
  ipAddress: string;
  userAgent: string;
  active: boolean;
}
```

### Upload Model
```typescript
interface Upload {
  fileId: string;
  filename: string;
  originalName: string;
  size: number;
  mimeType: string;
  uploadedBy: string;
  uploadedAt: Date;
  processedBy: string;
  tags: string[];
  description?: string;
}
```

---

## Environment Configuration

### Development (.env.development)
```env
NODE_ENV=development
PORT=3001
REACT_APP_API_URL=http://localhost:3001/api

# Session
SESSION_SECRET=dev-session-secret
MONGODB_URL=mongodb://localhost:27017/iris-sessions

# Service URLs
HERMES_URL=http://localhost:3000
THESAURUS_URL=http://localhost:3002
LOGOS_URL=http://localhost:3003
SOPHIA_URL=http://localhost:3004

# Auth
JWT_SECRET=dev-jwt-secret
JWT_EXPIRES_IN=24h

# Upload
UPLOAD_DIR=./uploads
MAX_FILE_SIZE=10485760
```

### Production (.env.production)
```env
NODE_ENV=production
PORT=3001
REACT_APP_API_URL=https://iris.localfinance.com/api

# Session
SESSION_SECRET=${SESSION_SECRET}
MONGODB_URL=${MONGODB_URL}

# Service URLs (Docker internal)
HERMES_URL=http://hermes:3000
THESAURUS_URL=http://thesaurus:3002
LOGOS_URL=http://logos:3003
SOPHIA_URL=http://sophia:3004

# Auth
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRES_IN=24h

# Upload
UPLOAD_DIR=/app/uploads
MAX_FILE_SIZE=52428800
```

---

## Performance Considerations

### Optimization Strategies

1. **Code Splitting:**
   ```javascript
   const DashboardPage = lazy(() => import('./pages/DashboardPage'));
   const ProfilePage = lazy(() => import('./pages/ProfilePage'));
   ```

2. **API Response Caching:**
   ```javascript
   // React Query for server state caching
   const { data, isLoading } = useQuery(
     ['user', userId],
     () => userApi.getProfile(userId),
     { staleTime: 5 * 60 * 1000 } // 5 minutes
   );
   ```

3. **Service Proxy Optimization:**
   - Connection pooling for service calls
   - Request batching where applicable
   - Response caching for read-heavy operations

4. **File Upload Optimization:**
   - Chunked uploads for large files
   - Progress tracking
   - Resume capability

### Monitoring

**Health Check Endpoint:** `/api/health`
```json
{
  "status": "healthy",
  "timestamp": "2024-03-15T10:30:00Z",
  "services": {
    "hermes": "connected",
    "thesaurus": "connected",
    "logos": "connected",
    "sophia": "connected",
    "database": "connected"
  },
  "uptime": 3600,
  "version": "1.0.0"
}
```

---

## Security Considerations

### Input Validation
- All API inputs validated using Joi schemas
- File upload type restrictions
- Size limits enforced
- SQL injection prevention

### Authentication Security
- JWT tokens with expiration
- Secure session configuration
- CORS properly configured
- Rate limiting on auth endpoints

### File Upload Security
- MIME type validation
- File extension whitelist
- Virus scanning (production)
- Secure file storage

---

## Deployment

### Docker Configuration

**Dockerfile:**
```dockerfile
# Multi-stage build
FROM node:18-alpine AS builder

# Build client
WORKDIR /app/client
COPY client/package*.json ./
RUN npm ci --only=production
COPY client/ ./
RUN npm run build

# Build server
WORKDIR /app/server
COPY server/package*.json ./
RUN npm ci --only=production
COPY server/ ./

# Production stage
FROM node:18-alpine AS production
WORKDIR /app

# Copy server
COPY --from=builder /app/server ./server
# Copy built client
COPY --from=builder /app/client/build ./client/build

WORKDIR /app/server
EXPOSE 3001

CMD ["node", "index.js"]
```

### Docker Compose (Development)
```yaml
version: '3.8'
services:
  iris:
    build: .
    ports:
      - "3001:3001"
    environment:
      - NODE_ENV=development
      - MONGODB_URL=mongodb://mongo:27017/iris-sessions
    depends_on:
      - mongo
      - hermes
    volumes:
      - ./uploads:/app/uploads

  mongo:
    image: mongo:5
    ports:
      - "27017:27017"
    volumes:
      - iris_mongo_data:/data/db

volumes:
  iris_mongo_data:
```

---

## Summary

Iris serves as the unified frontend gateway for LocalFinance, providing:

- **Seamless User Experience:** React SPA with server-side rendering support
- **Secure Authentication:** JWT-based auth with session management
- **Service Integration:** Smart proxy routing to all backend services
- **File Handling:** Robust upload system with progress tracking
- **Performance:** Optimized caching, code splitting, and monitoring
- **Scalability:** Docker deployment ready with environment-specific configs

The architecture balances simplicity with functionality, providing a solid foundation for the LocalFinance ecosystem while maintaining flexibility for future enhancements.
