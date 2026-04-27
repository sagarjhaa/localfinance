// Insights + Month-in-Review proxy routes
//
// Iris is a DUMB proxy. These routes forward auth headers and JSON bodies
// to Sophia (insights, month-review) and Thesaurus (dismiss). No business
// logic happens here.

const express = require('express');
const axios = require('axios');
const { authenticateToken } = require('../middleware/auth');

const router = express.Router();

const SOPHIA_URL = process.env.SOPHIA_URL || 'http://localhost:8002';
const THESAURUS_URL = process.env.THESAURUS_URL || 'http://localhost:8001';

// Sophia/Ollama needs a long timeout on Jetson; Thesaurus is fast.
const SOPHIA_TIMEOUT_MS = 300000;
const THESAURUS_TIMEOUT_MS = 30000;

function buildHeaders(req) {
  const headers = {
    'Content-Type': 'application/json',
    'User-Agent': 'Iris-Proxy/1.0',
  };
  if (req.headers.authorization) headers.Authorization = req.headers.authorization;
  if (req.correlationId) headers['X-Correlation-ID'] = req.correlationId;
  if (req.user) {
    headers['X-User-Id'] = req.user.id;
    if (req.user.username) headers['X-User-Name'] = req.user.username;
    if (req.user.role) headers['X-User-Role'] = req.user.role;
  }
  return headers;
}

function forwardError(res, req, target, error) {
  console.error(`[${req.correlationId}] insights proxy error -> ${target}:`, error.message);
  if (error.code === 'ECONNREFUSED') {
    return res.status(503).json({
      message: `Service ${target} is not available`,
      code: 'SERVICE_UNAVAILABLE',
      correlation_id: req.correlationId,
    });
  }
  if (error.code === 'ETIMEDOUT') {
    return res.status(504).json({
      message: `Request to ${target} timed out`,
      code: 'GATEWAY_TIMEOUT',
      correlation_id: req.correlationId,
    });
  }
  return res.status(502).json({
    message: 'Upstream proxy request failed',
    code: 'PROXY_ERROR',
    error: error.message,
    correlation_id: req.correlationId,
  });
}

async function forward(req, res, { method, url, data, params, timeout, target }) {
  try {
    const response = await axios({
      method,
      url,
      headers: buildHeaders(req),
      data,
      params,
      timeout,
      validateStatus: () => true,
    });
    res.status(response.status).json(response.data);
  } catch (err) {
    forwardError(res, req, target, err);
  }
}

// POST /api/v1/insights/  -> Sophia POST /api/v1/insights/
router.post('/insights/', authenticateToken, (req, res) =>
  forward(req, res, {
    method: 'post',
    url: `${SOPHIA_URL}/api/v1/insights/`,
    data: req.body,
    timeout: SOPHIA_TIMEOUT_MS,
    target: 'sophia',
  }),
);

// GET /api/v1/insights/:userId -> Sophia GET /api/v1/insights/:userId
router.get('/insights/:userId', authenticateToken, (req, res) =>
  forward(req, res, {
    method: 'get',
    url: `${SOPHIA_URL}/api/v1/insights/${encodeURIComponent(req.params.userId)}`,
    params: req.query,
    timeout: SOPHIA_TIMEOUT_MS,
    target: 'sophia',
  }),
);

// POST /api/v1/insights/:userId/dismiss
//   body: { insight_key, rule_id }
//   forwards to Thesaurus internal: POST /api/v1/internal/users/:userId/dismissed-insights
router.post('/insights/:userId/dismiss', authenticateToken, (req, res) => {
  const { insight_key, rule_id } = req.body || {};
  if (!insight_key && !rule_id) {
    return res.status(400).json({
      message: 'insight_key or rule_id is required',
      code: 'INVALID_REQUEST',
      correlation_id: req.correlationId,
    });
  }
  return forward(req, res, {
    method: 'post',
    url: `${THESAURUS_URL}/api/v1/internal/users/${encodeURIComponent(req.params.userId)}/dismissed-insights`,
    data: { insight_key, rule_id },
    timeout: THESAURUS_TIMEOUT_MS,
    target: 'thesaurus',
  });
});

// GET /api/v1/month-review/:period?user_id=... -> Sophia
router.get('/month-review/:period', authenticateToken, (req, res) =>
  forward(req, res, {
    method: 'get',
    url: `${SOPHIA_URL}/api/v1/month-review/${encodeURIComponent(req.params.period)}`,
    params: req.query,
    timeout: SOPHIA_TIMEOUT_MS,
    target: 'sophia',
  }),
);

// DELETE /api/v1/month-review/:period?user_id=... -> Sophia (cache invalidation)
router.delete('/month-review/:period', authenticateToken, (req, res) =>
  forward(req, res, {
    method: 'delete',
    url: `${SOPHIA_URL}/api/v1/month-review/${encodeURIComponent(req.params.period)}`,
    params: req.query,
    timeout: SOPHIA_TIMEOUT_MS,
    target: 'sophia',
  }),
);

module.exports = router;
