const express = require('express');
const { generateToken, verifyToken, authenticateToken } = require('../middleware/auth');

const router = express.Router();

// Mock user database (in production, this would be a real database)
const MOCK_USERS = [
  {
    id: 1,
    username: 'demo',
    password: 'demo123', // In production, this would be hashed
    email: 'demo@localfinance.dev',
    role: 'user'
  },
  {
    id: 2,
    username: 'admin',
    password: 'admin123', // In production, this would be hashed
    email: 'admin@localfinance.dev',
    role: 'admin'
  }
];

/**
 * POST /api/auth/login
 * Authenticate user and return JWT token
 */
router.post('/login', async (req, res) => {
  try {
    const { username, password } = req.body;

    if (!username || !password) {
      return res.status(400).json({
        message: 'Username and password are required',
        code: 'MISSING_CREDENTIALS'
      });
    }

    // Find user (in production, query database with hashed password comparison)
    const user = MOCK_USERS.find(u => 
      u.username === username && u.password === password
    );

    if (!user) {
      return res.status(401).json({
        message: 'Invalid credentials',
        code: 'INVALID_CREDENTIALS'
      });
    }

    // Generate JWT token
    const token = generateToken({
      id: user.id,
      username: user.username,
      email: user.email,
      role: user.role
    });

    // Return user info and token
    res.json({
      message: 'Login successful',
      user: {
        id: user.id,
        username: user.username,
        email: user.email,
        role: user.role
      },
      token,
      expiresIn: '24h'
    });

  } catch (error) {
    console.error('Login error:', error);
    res.status(500).json({
      message: 'Login failed',
      code: 'LOGIN_ERROR'
    });
  }
});

/**
 * POST /api/auth/logout
 * Logout user (token invalidation would be handled client-side or via blacklist)
 */
router.post('/logout', authenticateToken, (req, res) => {
  // In production, you might want to blacklist the token
  res.json({
    message: 'Logout successful'
  });
});

/**
 * GET /api/auth/me
 * Get current user information
 */
router.get('/me', authenticateToken, (req, res) => {
  // req.user is set by authenticateToken middleware
  res.json({
    user: req.user
  });
});

/**
 * POST /api/auth/refresh
 * Refresh JWT token
 */
router.post('/refresh', authenticateToken, (req, res) => {
  try {
    // Generate new token with current user data
    const token = generateToken({
      id: req.user.id,
      username: req.user.username,
      email: req.user.email,
      role: req.user.role
    });

    res.json({
      message: 'Token refreshed successfully',
      token,
      expiresIn: '24h'
    });

  } catch (error) {
    console.error('Token refresh error:', error);
    res.status(500).json({
      message: 'Token refresh failed',
      code: 'REFRESH_ERROR'
    });
  }
});

/**
 * POST /api/auth/validate
 * Validate token (useful for other services)
 */
router.post('/validate', (req, res) => {
  try {
    const { token } = req.body;

    if (!token) {
      return res.status(400).json({
        message: 'Token is required',
        code: 'NO_TOKEN'
      });
    }

    const decoded = verifyToken(token);
    
    res.json({
      valid: true,
      user: decoded
    });

  } catch (error) {
    res.status(401).json({
      valid: false,
      message: 'Invalid token',
      code: 'INVALID_TOKEN'
    });
  }
});

module.exports = router;