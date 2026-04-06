const express = require('express');
const crypto = require('crypto');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const path = require('path');
require('dotenv').config();

const authRoutes = require('./routes/auth');
const uploadRoutes = require('./routes/upload');
const proxyRoutes = require('./routes/proxy');

const app = express();
const PORT = process.env.PORT || 3001;

// Security middleware
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
      fontSrc: ["'self'", "https://fonts.gstatic.com"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", "http://localhost:*", "http://10.0.0.*:*"],
      upgradeInsecureRequests: null
    }
  },
  crossOriginResourcePolicy: { policy: "cross-origin" }
}));

// Compression middleware
app.use(compression());

// CORS configuration — allow local network access
app.use(cors({
  origin: true,
  credentials: true,
  optionsSuccessStatus: 200
}));

// Body parsing middleware
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Correlation ID middleware
app.use((req, res, next) => {
  const correlationId = req.headers['x-correlation-id'] ||
    'lf_' + (crypto.randomUUID ? crypto.randomUUID().replace(/-/g, '') : Date.now().toString(36) + Math.random().toString(36).substring(2));
  req.correlationId = correlationId;
  res.set('X-Correlation-ID', correlationId);
  next();
});

// Request logging middleware
app.use((req, res, next) => {
  console.log(`${new Date().toISOString()} [${req.correlationId}] ${req.method} ${req.path}`);
  next();
});

// API routes
app.use('/api/auth', authRoutes);
app.use('/api/upload', uploadRoutes);
app.use('/api/proxy', proxyRoutes);

// Health check endpoint
app.get('/api/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'iris',
    timestamp: new Date().toISOString(),
    version: '1.0.0'
  });
});

// Serve static files from React app
const buildPath = process.env.BUILD_PATH || path.join(__dirname, '../client/build');
app.use(express.static(buildPath));

// Catch all handler: send back React's index.html for any non-API routes
app.get('*', (req, res) => {
  res.sendFile(path.join(buildPath, 'index.html'));
});

// Error handling middleware
app.use((error, req, res, next) => {
  console.error(`[${req.correlationId}] Error:`, error);
  res.status(error.status || 500).json({
    message: error.message || 'Internal server error',
    correlation_id: req.correlationId,
    ...(process.env.NODE_ENV === 'development' && { stack: error.stack })
  });
});

// 404 handler
app.use('*', (req, res) => {
  res.status(404).json({ message: 'Route not found' });
});

app.listen(PORT, () => {
  console.log(`🌈 Iris server running on port ${PORT}`);
  console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
  console.log(`Time: ${new Date().toISOString()}`);
});

module.exports = app;