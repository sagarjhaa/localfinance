const express = require('express');
const axios = require('axios');
const { authenticateToken, optionalAuth } = require('../middleware/auth');

const router = express.Router();

// Service endpoints configuration
const SERVICES = {
  hermes: {
    url: process.env.HERMES_URL || 'http://localhost:3000',
    name: 'Hermes Gateway'
  },
  thesaurus: {
    url: process.env.THESAURUS_URL || 'http://localhost:8001',
    name: 'Thesaurus Database'
  },
  logos: {
    url: process.env.LOGOS_URL || 'http://localhost:8003',
    name: 'Logos Document Processing'
  },
  sophia: {
    url: process.env.SOPHIA_URL || 'http://localhost:8002',
    name: 'Sophia AI'
  }
};

/**
 * GET /api/proxy/health
 * Check health of all services
 */
router.get('/health', optionalAuth, async (req, res) => {
  const healthChecks = await Promise.allSettled(
    Object.entries(SERVICES).map(async ([serviceName, config]) => {
      try {
        const response = await axios.get(`${config.url}/health`, {
          timeout: 5000,
          headers: req.user ? { 
            'Authorization': `Bearer ${req.headers.authorization ? req.headers.authorization.split(' ')[1] : ''}`
          } : {}
        });
        
        return {
          service: serviceName,
          name: config.name,
          status: 'healthy',
          url: config.url,
          response: response.data,
          responseTime: response.headers['x-response-time'] || 'unknown'
        };
      } catch (error) {
        return {
          service: serviceName,
          name: config.name,
          status: 'unhealthy',
          url: config.url,
          error: error.message,
          code: error.code
        };
      }
    })
  );

  const results = healthChecks.map(result => result.value);
  const allHealthy = results.every(result => result.status === 'healthy');

  res.json({
    overallStatus: allHealthy ? 'healthy' : 'degraded',
    services: results,
    timestamp: new Date().toISOString(),
    checkedBy: req.user ? req.user.username : 'anonymous'
  });
});

/**
 * Generic proxy middleware for forwarding requests to services
 */
const createProxyHandler = (serviceName) => {
  return async (req, res) => {
    try {
      const service = SERVICES[serviceName];
      if (!service) {
        return res.status(404).json({
          message: `Service ${serviceName} not found`,
          code: 'SERVICE_NOT_FOUND'
        });
      }

      // Construct target URL — req.originalUrl contains full path like /api/proxy/sophia/api/v1/chat/
      // Strip the /api/proxy/<service> prefix to get the downstream path
      const prefix = `/api/proxy/${serviceName}`;
      const targetPath = req.originalUrl.startsWith(prefix)
        ? req.originalUrl.slice(prefix.length) || '/'
        : req.path;
      const targetUrl = `${service.url}${targetPath}`;

      // Prepare headers
      const headers = {
        'Content-Type': req.headers['content-type'] || 'application/json',
        'User-Agent': 'Iris-Proxy/1.0'
      };

      // Forward authentication if present
      if (req.headers.authorization) {
        headers.Authorization = req.headers.authorization;
      }

      // Forward correlation ID
      if (req.correlationId) {
        headers['X-Correlation-ID'] = req.correlationId;
      }

      // Forward user information
      if (req.user) {
        headers['X-User-Id'] = req.user.id;
        headers['X-User-Name'] = req.user.username;
        headers['X-User-Role'] = req.user.role;
      }

      // Make the proxied request
      // Sophia/AI requests need longer timeout for Ollama inference on Jetson
      const timeoutMs = serviceName === 'sophia' ? 300000 : 30000;

      const axiosConfig = {
        method: req.method.toLowerCase(),
        url: targetUrl,
        headers,
        params: req.query,
        timeout: timeoutMs,
        validateStatus: () => true // Accept any status code
      };

      // Add body for non-GET requests
      if (['post', 'put', 'patch'].includes(axiosConfig.method)) {
        axiosConfig.data = req.body;
      }

      const response = await axios(axiosConfig);

      // Forward response headers
      const forwardHeaders = [
        'content-type',
        'content-length',
        'cache-control',
        'etag',
        'last-modified'
      ];

      forwardHeaders.forEach(header => {
        if (response.headers[header]) {
          res.set(header, response.headers[header]);
        }
      });

      // Add proxy information
      res.set('X-Proxied-By', 'Iris');
      res.set('X-Target-Service', serviceName);
      res.set('X-Target-Url', targetUrl);

      // Send response
      res.status(response.status).json(response.data);

    } catch (error) {
      console.error(`[${req.correlationId}] Proxy error for ${serviceName}:`, error.message);

      if (error.code === 'ECONNREFUSED') {
        return res.status(503).json({
          message: `Service ${serviceName} is not available`,
          code: 'SERVICE_UNAVAILABLE',
          service: serviceName,
          correlation_id: req.correlationId
        });
      }

      if (error.code === 'ETIMEDOUT') {
        return res.status(504).json({
          message: `Request to ${serviceName} timed out`,
          code: 'GATEWAY_TIMEOUT',
          service: serviceName,
          correlation_id: req.correlationId
        });
      }

      res.status(500).json({
        message: 'Proxy request failed',
        code: 'PROXY_ERROR',
        service: serviceName,
        error: error.message,
        correlation_id: req.correlationId
      });
    }
  };
};

/**
 * Service-specific proxy routes
 */
router.use('/hermes/*', authenticateToken, createProxyHandler('hermes'));
router.use('/thesaurus/*', authenticateToken, createProxyHandler('thesaurus'));
router.use('/logos/*', authenticateToken, createProxyHandler('logos'));
router.use('/sophia/*', authenticateToken, createProxyHandler('sophia'));

/**
 * GET /api/proxy/services
 * List available services
 */
router.get('/services', optionalAuth, (req, res) => {
  const servicesList = Object.entries(SERVICES).map(([key, config]) => ({
    name: key,
    displayName: config.name,
    url: config.url,
    proxyPath: `/api/proxy/${key}`
  }));

  res.json({
    services: servicesList,
    count: servicesList.length,
    timestamp: new Date().toISOString()
  });
});

/**
 * POST /api/proxy/batch
 * Execute multiple service requests in batch
 */
router.post('/batch', authenticateToken, async (req, res) => {
  try {
    const { requests } = req.body;

    if (!Array.isArray(requests) || requests.length === 0) {
      return res.status(400).json({
        message: 'Requests array is required',
        code: 'INVALID_BATCH_REQUEST'
      });
    }

    if (requests.length > 10) {
      return res.status(400).json({
        message: 'Maximum 10 requests allowed in batch',
        code: 'BATCH_LIMIT_EXCEEDED'
      });
    }

    const batchResults = await Promise.allSettled(
      requests.map(async (request, index) => {
        try {
          const { service, path, method = 'GET', data } = request;
          
          if (!SERVICES[service]) {
            throw new Error(`Unknown service: ${service}`);
          }

          const targetUrl = `${SERVICES[service].url}${path}`;
          const axiosConfig = {
            method: method.toLowerCase(),
            url: targetUrl,
            headers: {
              'Authorization': req.headers.authorization,
              'Content-Type': 'application/json',
              'X-User-Id': req.user.id,
              'X-User-Name': req.user.username,
              'X-Batch-Index': index.toString()
            },
            timeout: 15000
          };

          if (data && ['post', 'put', 'patch'].includes(axiosConfig.method)) {
            axiosConfig.data = data;
          }

          const response = await axios(axiosConfig);
          
          return {
            index,
            service,
            path,
            method,
            status: 'fulfilled',
            statusCode: response.status,
            data: response.data
          };

        } catch (error) {
          return {
            index,
            service: request.service,
            path: request.path,
            method: request.method,
            status: 'rejected',
            error: error.message,
            code: error.code
          };
        }
      })
    );

    const results = batchResults.map(result => result.value);

    res.json({
      message: 'Batch request completed',
      results,
      count: results.length,
      successful: results.filter(r => r.status === 'fulfilled').length,
      failed: results.filter(r => r.status === 'rejected').length,
      timestamp: new Date().toISOString()
    });

  } catch (error) {
    console.error('Batch proxy error:', error);
    res.status(500).json({
      message: 'Batch request failed',
      code: 'BATCH_ERROR'
    });
  }
});

module.exports = router;