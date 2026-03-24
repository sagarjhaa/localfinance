const express = require('express');
const axios = require('axios');
const { authenticateToken } = require('../middleware/auth');

const router = express.Router();

const THESAURUS_URL = process.env.THESAURUS_URL || 'http://localhost:8001';

/**
 * POST /api/auth/register
 * Proxy registration to Thesaurus
 */
router.post('/register', async (req, res) => {
  try {
    const { email, password, first_name, last_name } = req.body;

    if (!email || !password || !first_name || !last_name) {
      return res.status(400).json({
        message: 'Email, password, first name, and last name are required',
        code: 'MISSING_FIELDS'
      });
    }

    const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/register`, {
      email,
      password,
      first_name,
      last_name
    }, { timeout: 10000 });

    res.status(201).json(response.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Registration error:', error.message);
    res.status(503).json({
      message: 'Auth service unavailable',
      code: 'SERVICE_UNAVAILABLE'
    });
  }
});

/**
 * POST /api/auth/login
 * Proxy login to Thesaurus
 */
router.post('/login', async (req, res) => {
  try {
    const { email, password } = req.body;

    if (!email || !password) {
      return res.status(400).json({
        message: 'Email and password are required',
        code: 'MISSING_CREDENTIALS'
      });
    }

    const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/login`, {
      email,
      password
    }, { timeout: 10000 });

    res.json(response.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Login error:', error.message);
    res.status(503).json({
      message: 'Auth service unavailable',
      code: 'SERVICE_UNAVAILABLE'
    });
  }
});

/**
 * POST /api/auth/logout
 * Proxy logout to Thesaurus
 */
router.post('/logout', authenticateToken, async (req, res) => {
  try {
    const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/logout`, {}, {
      timeout: 10000,
      headers: { 'Authorization': req.headers.authorization }
    });
    res.json(response.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    res.status(503).json({ message: 'Auth service unavailable' });
  }
});

/**
 * GET /api/auth/me
 * Validate token via Thesaurus and return user info
 */
router.get('/me', async (req, res) => {
  try {
    const authHeader = req.headers['authorization'];
    const token = authHeader && authHeader.split(' ')[1];

    if (!token) {
      return res.status(401).json({ message: 'Access token required' });
    }

    const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/validate`, {
      token
    }, { timeout: 10000 });

    if (response.data.valid) {
      res.json({ user: response.data.user });
    } else {
      res.status(401).json({ message: 'Invalid token' });
    }
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    res.status(503).json({ message: 'Auth service unavailable' });
  }
});

/**
 * POST /api/auth/refresh
 * Proxy token refresh to Thesaurus
 */
router.post('/refresh', authenticateToken, async (req, res) => {
  try {
    const response = await axios.post(`${THESAURUS_URL}/api/v1/auth/refresh`, {}, {
      timeout: 10000,
      headers: { 'Authorization': req.headers.authorization }
    });
    res.json(response.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    res.status(503).json({ message: 'Auth service unavailable' });
  }
});

module.exports = router;
