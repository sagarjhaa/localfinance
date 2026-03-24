const axios = require('axios');

const THESAURUS_URL = process.env.THESAURUS_URL || 'http://localhost:8001';

/**
 * Middleware to verify tokens via Thesaurus auth service
 */
const authenticateToken = async (req, res, next) => {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];

  if (!token) {
    return res.status(401).json({
      message: 'Access token required',
      code: 'NO_TOKEN'
    });
  }

  try {
    const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/validate`, {
      token
    }, { timeout: 5000 });

    if (response.data.valid && response.data.user) {
      req.user = response.data.user;
      next();
    } else {
      res.status(401).json({
        message: 'Invalid or expired token',
        code: 'INVALID_TOKEN'
      });
    }
  } catch (error) {
    console.error('Token validation error:', error.message);
    res.status(503).json({
      message: 'Auth service unavailable',
      code: 'AUTH_SERVICE_DOWN'
    });
  }
};

/**
 * Middleware for optional authentication (non-blocking)
 */
const optionalAuth = async (req, res, next) => {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];

  if (token) {
    try {
      const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/validate`, {
        token
      }, { timeout: 5000 });

      if (response.data.valid && response.data.user) {
        req.user = response.data.user;
      }
    } catch (error) {
      // Silently continue without auth
    }
  }

  next();
};

module.exports = {
  authenticateToken,
  optionalAuth
};
