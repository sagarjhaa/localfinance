// Insights / Month-in-Review Proxy Route Tests
//
// Spins up mock Sophia + Thesaurus servers and exercises the new /api/v1/*
// routes added in services/iris/server/routes/insights.js. Mirrors the style
// of auth-routes.test.js — no test framework, just node http.

const http = require('http');

let passed = 0;
let failed = 0;

function assert(condition, message) {
  if (condition) { passed++; }
  else { failed++; console.error(`  FAIL: ${message}`); }
}

async function request(method, path, body = null, headers = {}) {
  return new Promise((resolve, reject) => {
    const url = new URL(`http://localhost:3098${path}`);
    const options = {
      hostname: url.hostname, port: url.port,
      path: url.pathname + url.search,
      method,
      headers: { 'Content-Type': 'application/json', ...headers },
      timeout: 10000,
    };
    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try { resolve({ status: res.statusCode, data: JSON.parse(data) }); }
        catch { resolve({ status: res.statusCode, data }); }
      });
    });
    req.on('error', reject);
    if (body) req.write(JSON.stringify(body));
    req.end();
  });
}

const VALID_TOKEN = 'mock-token-iris';

async function runTests() {
  // --- Mock Thesaurus (auth validate + dismiss) ---
  const sophiaCalls = [];
  const thesaurusCalls = [];

  const mockThesaurus = http.createServer((req, res) => {
    let body = '';
    req.on('data', c => body += c);
    req.on('end', () => {
      thesaurusCalls.push({ method: req.method, url: req.url, body });
      res.setHeader('Content-Type', 'application/json');

      if (req.url === '/api/v1/auth/validate' && req.method === 'POST') {
        const data = JSON.parse(body || '{}');
        if (data.token === VALID_TOKEN) {
          res.writeHead(200);
          return res.end(JSON.stringify({
            valid: true,
            user: { id: 'user-42', username: 'tester', email: 't@t.com', role: 'user' },
          }));
        }
        res.writeHead(200);
        return res.end(JSON.stringify({ valid: false }));
      }

      // Dismiss: POST /api/v1/internal/users/:userId/dismissed-insights
      const dismissMatch = /^\/api\/v1\/internal\/users\/([^/]+)\/dismissed-insights$/.exec(req.url || '');
      if (dismissMatch && req.method === 'POST') {
        const data = JSON.parse(body || '{}');
        if (!data.insight_key) {
          res.writeHead(400);
          return res.end(JSON.stringify({ error: 'insight_key required' }));
        }
        res.writeHead(201);
        return res.end(JSON.stringify({ user_id: dismissMatch[1], insight_key: data.insight_key, dismissed: true }));
      }

      res.writeHead(404);
      res.end(JSON.stringify({ error: 'Not found' }));
    });
  });

  // --- Mock Sophia (insights + month-review) ---
  const mockSophia = http.createServer((req, res) => {
    let body = '';
    req.on('data', c => body += c);
    req.on('end', () => {
      sophiaCalls.push({ method: req.method, url: req.url, body });
      res.setHeader('Content-Type', 'application/json');

      const insightsGet = /^\/api\/v1\/insights\/([^/]+)$/.exec(req.url || '');
      if (insightsGet && req.method === 'GET') {
        res.writeHead(200);
        return res.end(JSON.stringify({
          user_id: insightsGet[1],
          insights: [
            { type: 'spending_insight', title: 'Eating out', description: 'You ate out 14x',
              priority: 'medium', key: 'abc123' },
          ],
          narrative: { overall: 'Solid month overall.' },
        }));
      }

      if (req.url === '/api/v1/insights/' && req.method === 'POST') {
        const data = JSON.parse(body || '{}');
        if (!data.user_id) {
          res.writeHead(400);
          return res.end(JSON.stringify({ error: 'user_id required' }));
        }
        res.writeHead(200);
        return res.end(JSON.stringify({ user_id: data.user_id, insights: [], period: data.period || 'monthly' }));
      }

      const monthMatch = /^\/api\/v1\/month-review\/([^/?]+)/.exec(req.url || '');
      if (monthMatch && req.method === 'GET') {
        res.writeHead(200);
        return res.end(JSON.stringify({
          user_id: 'user-42', period: monthMatch[1],
          insights: [{ key: 'k1', title: 'T', description: 'd' }],
          narrative: { overall: 'ok', per_insight: { k1: 'detailed' } },
        }));
      }
      if (monthMatch && req.method === 'DELETE') {
        res.writeHead(200);
        return res.end(JSON.stringify({ invalidated: true, period: monthMatch[1] }));
      }

      res.writeHead(404);
      res.end(JSON.stringify({ error: 'Not found' }));
    });
  });

  await new Promise(r => mockThesaurus.listen(18002, r));
  await new Promise(r => mockSophia.listen(18003, r));

  process.env.PORT = '3098';
  process.env.THESAURUS_URL = 'http://localhost:18002';
  process.env.SOPHIA_URL = 'http://localhost:18003';
  process.env.NODE_ENV = 'test';
  // Force a fresh require so env vars are picked up if a prior test loaded the module.
  delete require.cache[require.resolve('../index')];
  delete require.cache[require.resolve('../routes/insights')];
  delete require.cache[require.resolve('../middleware/auth')];
  delete require.cache[require.resolve('../routes/auth')];
  delete require.cache[require.resolve('../routes/upload')];
  delete require.cache[require.resolve('../routes/proxy')];

  require('../index');
  await new Promise(r => setTimeout(r, 500));

  const authHeader = { Authorization: `Bearer ${VALID_TOKEN}` };

  try {
    // GET insights — happy
    console.log('GET /api/v1/insights/:userId:');
    const r1 = await request('GET', '/api/v1/insights/user-42', null, authHeader);
    assert(r1.status === 200, `insights 200 (got ${r1.status})`);
    assert(Array.isArray(r1.data.insights), 'insights array returned');
    assert(r1.data.narrative?.overall === 'Solid month overall.', 'narrative forwarded');

    // GET insights — no auth -> 401
    const r1b = await request('GET', '/api/v1/insights/user-42');
    assert(r1b.status === 401, `insights no-auth 401 (got ${r1b.status})`);

    // POST insights — happy
    console.log('POST /api/v1/insights/:');
    const r2 = await request('POST', '/api/v1/insights/', { user_id: 'user-42', period: 'monthly' }, authHeader);
    assert(r2.status === 200, `insights POST 200 (got ${r2.status})`);
    assert(r2.data.user_id === 'user-42', 'user_id forwarded');

    // POST insights — error path (missing user_id, upstream 400)
    const r2b = await request('POST', '/api/v1/insights/', {}, authHeader);
    assert(r2b.status === 400, `insights POST missing user_id forwarded as 400 (got ${r2b.status})`);

    // POST dismiss — happy
    console.log('POST /api/v1/insights/:userId/dismiss:');
    const r3 = await request('POST', '/api/v1/insights/user-42/dismiss',
      { insight_key: 'abc123', rule_id: '' }, authHeader);
    assert(r3.status === 201, `dismiss 201 (got ${r3.status})`);
    assert(r3.data.dismissed === true, 'dismiss forwarded to thesaurus');
    const lastTh = thesaurusCalls[thesaurusCalls.length - 1];
    assert(lastTh.url === '/api/v1/internal/users/user-42/dismissed-insights',
      `dismiss hit thesaurus path (got ${lastTh.url})`);

    // POST dismiss — missing both keys -> 400 from Iris
    const r3b = await request('POST', '/api/v1/insights/user-42/dismiss', {}, authHeader);
    assert(r3b.status === 400, `dismiss missing keys 400 (got ${r3b.status})`);

    // GET month-review
    console.log('GET /api/v1/month-review/:period:');
    const r4 = await request('GET', '/api/v1/month-review/2026-04?user_id=user-42', null, authHeader);
    assert(r4.status === 200, `month-review GET 200 (got ${r4.status})`);
    assert(r4.data.period === '2026-04', 'period forwarded');
    assert(r4.data.narrative?.per_insight?.k1 === 'detailed', 'per_insight passed through');

    // DELETE month-review
    console.log('DELETE /api/v1/month-review/:period:');
    const r5 = await request('DELETE', '/api/v1/month-review/2026-04?user_id=user-42', null, authHeader);
    assert(r5.status === 200, `month-review DELETE 200 (got ${r5.status})`);
    assert(r5.data.invalidated === true, 'invalidate forwarded');

    // DELETE no auth
    const r5b = await request('DELETE', '/api/v1/month-review/2026-04?user_id=user-42');
    assert(r5b.status === 401, `month-review DELETE no-auth 401 (got ${r5b.status})`);
  } finally {
    mockThesaurus.close();
    mockSophia.close();
    console.log(`\n${passed} passed, ${failed} failed`);
    process.exit(failed > 0 ? 1 : 0);
  }
}

runTests().catch(err => {
  console.error('Test error:', err);
  process.exit(1);
});
