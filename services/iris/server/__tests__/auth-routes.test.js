// Auth Routes Integration Tests
// Tests the Iris auth proxy routes against a mock Thesaurus

const http = require('http');

let passed = 0;
let failed = 0;

function assert(condition, message) {
  if (condition) { passed++; } else { failed++; console.error(`  FAIL: ${message}`); }
}

async function request(method, path, body = null, headers = {}) {
  return new Promise((resolve, reject) => {
    const url = new URL(`http://localhost:3099${path}`);
    const options = {
      hostname: url.hostname, port: url.port, path: url.pathname,
      method, headers: { 'Content-Type': 'application/json', ...headers },
      timeout: 5000,
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

async function runTests() {
  // Start a mock Thesaurus server
  const mockThesaurus = http.createServer((req, res) => {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', () => {
      res.setHeader('Content-Type', 'application/json');

      if (req.url === '/api/v1/auth/register' && req.method === 'POST') {
        const data = JSON.parse(body);
        if (!data.email || !data.password) {
          res.writeHead(400);
          res.end(JSON.stringify({ error: 'Missing fields' }));
        } else {
          res.writeHead(201);
          res.end(JSON.stringify({
            token: 'mock-token-123',
            expires_at: Date.now() + 86400000,
            user: { id: 'user-1', email: data.email, first_name: data.first_name, last_name: data.last_name, is_active: true },
          }));
        }
      } else if (req.url === '/api/v1/auth/login' && req.method === 'POST') {
        const data = JSON.parse(body);
        if (data.email === 'test@test.com' && data.password === 'password123') {
          res.writeHead(200);
          res.end(JSON.stringify({
            token: 'mock-token-456',
            expires_at: Date.now() + 86400000,
            user: { id: 'user-1', email: data.email, first_name: 'Test', last_name: 'User', is_active: true },
          }));
        } else {
          res.writeHead(401);
          res.end(JSON.stringify({ error: 'Invalid email or password' }));
        }
      } else if (req.url === '/api/v1/auth/validate' && req.method === 'POST') {
        const data = JSON.parse(body);
        if (data.token === 'mock-token-456') {
          res.writeHead(200);
          res.end(JSON.stringify({
            valid: true,
            user: { id: 'user-1', email: 'test@test.com', first_name: 'Test', last_name: 'User' },
          }));
        } else {
          res.writeHead(200);
          res.end(JSON.stringify({ valid: false }));
        }
      } else if (req.url === '/api/v1/auth/logout' && req.method === 'POST') {
        res.writeHead(200);
        res.end(JSON.stringify({ message: 'Logged out successfully' }));
      } else {
        res.writeHead(404);
        res.end(JSON.stringify({ error: 'Not found' }));
      }
    });
  });

  await new Promise(resolve => mockThesaurus.listen(18001, resolve));

  // Set env and start Iris server
  process.env.PORT = '3099';
  process.env.THESAURUS_URL = 'http://localhost:18001';
  process.env.NODE_ENV = 'test';
  const app = require('../index');
  await new Promise(resolve => setTimeout(resolve, 500));

  try {
    // --- Register ---
    console.log('Register:');
    const reg = await request('POST', '/api/auth/register', {
      email: 'new@test.com', password: 'pass123', first_name: 'New', last_name: 'User',
    });
    assert(reg.status === 201, `register status 201 (got ${reg.status})`);
    assert(reg.data.token === 'mock-token-123', 'register returns token');
    assert(reg.data.user?.email === 'new@test.com', 'register returns user');

    // Register missing fields
    const regBad = await request('POST', '/api/auth/register', { email: 'a@b.com' });
    assert(regBad.status === 400, `register missing fields → 400 (got ${regBad.status})`);

    // --- Login ---
    console.log('Login:');
    const login = await request('POST', '/api/auth/login', {
      email: 'test@test.com', password: 'password123',
    });
    assert(login.status === 200, `login success 200 (got ${login.status})`);
    assert(login.data.token === 'mock-token-456', 'login returns token');

    // Wrong password
    const loginBad = await request('POST', '/api/auth/login', {
      email: 'test@test.com', password: 'wrongpass',
    });
    assert(loginBad.status === 401, `login wrong password → 401 (got ${loginBad.status})`);

    // --- Get Me ---
    console.log('Auth /me:');
    const me = await request('GET', '/api/auth/me', null, {
      'Authorization': 'Bearer mock-token-456',
    });
    assert(me.status === 200, `me status 200 (got ${me.status})`);
    assert(me.data.user?.email === 'test@test.com', 'me returns correct user');

    // No token
    const meNoAuth = await request('GET', '/api/auth/me');
    assert(meNoAuth.status === 401, `me without token → 401 (got ${meNoAuth.status})`);

    // Invalid token
    const meBadToken = await request('GET', '/api/auth/me', null, {
      'Authorization': 'Bearer invalid-token',
    });
    assert(meBadToken.status === 401, `me with bad token → 401 (got ${meBadToken.status})`);

    // --- Health ---
    console.log('Health:');
    const health = await request('GET', '/api/health');
    assert(health.status === 200, `health 200 (got ${health.status})`);
    assert(health.data.service === 'iris', 'health returns iris');

  } finally {
    mockThesaurus.close();
    console.log(`\n${passed} passed, ${failed} failed`);
    process.exit(failed > 0 ? 1 : 0);
  }
}

runTests().catch(err => {
  console.error('Test error:', err);
  process.exit(1);
});
