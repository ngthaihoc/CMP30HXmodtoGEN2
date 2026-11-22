'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const path = require('node:path');
const { pathToFileURL } = require('node:url');

const workerModuleUrl = pathToFileURL(
  path.resolve(__dirname, '../src/index.js'),
).href;

let cachedPem = null;
async function getTestPrivateKeyPem() {
  if (cachedPem) return cachedPem;
  const pair = await crypto.webcrypto.subtle.generateKey(
    {
      name: 'RSASSA-PKCS1-v1_5',
      modulusLength: 2048,
      publicExponent: new Uint8Array([1, 0, 1]),
      hash: 'SHA-256',
    },
    true,
    ['sign', 'verify'],
  );
  const pkcs8 = await crypto.webcrypto.subtle.exportKey('pkcs8', pair.privateKey);
  const b64 = Buffer.from(pkcs8).toString('base64');
  const lines = b64.match(/.{1,64}/g).join('\n');
  cachedPem = `-----BEGIN PRIVATE KEY-----\n${lines}\n-----END PRIVATE KEY-----`;
  return cachedPem;
}

class MockKv {
  constructor() {
    this.store = new Map();
    this.reads = 0;
    this.writes = 0;
  }

  async get(key, type) {
    this.reads += 1;
    const entry = this.store.get(key);
    if (!entry) return null;
    if (type === 'json') return JSON.parse(entry.value);
    return entry.value;
  }

  async put(key, value, options = {}) {
    this.writes += 1;
    this.store.set(key, { value: String(value), options });
  }

  keys() {
    return Array.from(this.store.keys());
  }
}

function makeField(value, source = 'test', unit = null, status = 'ok') {
  return { value, source, unit, status };
}

function makeValidReport(overrides = {}) {
  const nvidiaFields = {
    name: makeField('NVIDIA GeForce RTX 2070', 'nvidia-smi'),
    driver_version: makeField('621.01', 'nvidia-smi'),
    vbios_version: makeField('90.06.18.80.06', 'nvidia-smi'),
    'pci.bus_id': makeField('0000:01:00.0', 'nvidia-smi'),
    'pci.device_id': makeField('0x1F0710DE', 'nvidia-smi'),
    'pci.sub_device_id': makeField('0x86701043', 'nvidia-smi'),
    'memory.total': makeField(8192, 'nvidia-smi', 'MiB'),
    compute_cap: makeField('7.5', 'nvidia-smi'),
    'pcie.link.gen.max': makeField(3, 'nvidia-smi', 'gen'),
    'pcie.link.gen.gpucurrent': makeField(3, 'nvidia-smi', 'gen'),
    'pcie.link.width.max': makeField(16, 'nvidia-smi', 'lanes'),
    'pcie.link.width.current': makeField(16, 'nvidia-smi', 'lanes'),
    'temperature.gpu': makeField(35.7, 'nvidia-smi', 'C'),
    'power.draw': makeField(40.04, 'nvidia-smi', 'W'),
    'clocks.sm': makeField(300, 'nvidia-smi', 'MHz'),
    'clocks.mem': makeField(405, 'nvidia-smi', 'MHz'),
    'clocks.max.sm': makeField(2175, 'nvidia-smi', 'MHz'),
    pstate: makeField('P8', 'nvidia-smi'),
  };

  return {
    schema_version: 1,
    collector_version: '1.0.0',
    report_id: '11111111-2222-4333-8444-555555555555',
    created_utc: '2026-11-21T07:00:00Z',
    experiment_id: 'aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee',
    consent: true,
    gpus: [
      {
        bdf: makeField('0000:01:00.0', 'pnp'),
        vendor_id: makeField('10DE', 'pnp'),
        device_id: makeField('1F07', 'pnp'),
        subsystem_id: makeField('86701043', 'pnp'),
        revision: makeField('A1', 'pnp'),
        name: makeField('NVIDIA GeForce RTX 2070', 'cim'),
        pnp_status: makeField('OK', 'pnp'),
        driver_version: makeField('32.0.16.2101', 'cim'),
      },
    ],
    nvidia: {
      complete: true,
      missing: [],
      fields: nvidiaFields,
    },
    pcie: {
      max_gen: makeField(3, 'pnp', 'gen'),
      current_gen: makeField(3, 'pnp', 'gen'),
      max_width: makeField(16, 'pnp', 'lanes'),
      current_width: makeField(16, 'pnp', 'lanes'),
    },
    platform: {
      cpu: makeField('AMD Ryzen 5 5600GT', 'cim'),
      mainboard_model: makeField('TUF GAMING B450M-PRO S', 'cim'),
      bios_version: makeField('4655', 'cim'),
      os_build: makeField('10.0.26100', 'cim'),
    },
    context: {
      mod_state: 'stock',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'baseline',
      notes: 'Clean test report',
    },
    ...overrides,
  };
}

const TEST_KEY_K1 = Buffer.from('0123456789abcdef0123456789abcdef', 'utf8').toString('base64');
const TEST_KEY_K2 = Buffer.from('fedcba9876543210fedcba9876543210', 'utf8').toString('base64');
const FIXED_NOW = 1795244400; // 2026-11-21T07:00:00Z

function signBody(bodyString, keyId, keyBase64, timestamp) {
  const bodyHash = crypto.createHash('sha256').update(Buffer.from(bodyString, 'utf8')).digest('hex');
  const mac = crypto
    .createHmac('sha256', Buffer.from(keyBase64, 'base64'))
    .update(`${timestamp}.${bodyHash}`, 'utf8')
    .digest('hex');
  return {
    'X-Report-Key-Id': keyId,
    'X-Report-Timestamp': String(timestamp),
    'X-Report-Signature': mac,
  };
}

async function makeEnv(overrides = {}) {
  const privateKey = await getTestPrivateKeyPem();
  return {
    REPORTS: new MockKv(),
    TARGET_REPO: 'ngthaihoc/vga-report-sandbox',
    INTAKE_ENABLED: 'true',
    DAILY_CAP: '200',
    IP_LIMIT: '10',
    IP_SALT: 'test-ip-salt-value',
    GITHUB_APP_ID: '123456',
    GITHUB_APP_INSTALLATION_ID: '789012',
    GITHUB_APP_PRIVATE_KEY: privateKey,
    SIGNING_KEYS: JSON.stringify({ k1: TEST_KEY_K1 }),
    ...overrides,
  };
}

function makeSignedRequest(bodyString, {
  url = 'https://vga-reports.example.workers.dev/v1/reports',
  method = 'POST',
  contentType = 'application/json; charset=utf-8',
  ip = '203.0.113.10',
  keyId = 'k1',
  keyBase64 = TEST_KEY_K1,
  timestamp = FIXED_NOW,
  headersOverride = {},
} = {}) {
  const sigHeaders = keyId
    ? signBody(bodyString, keyId, keyBase64, timestamp)
    : {};
  const headers = {
    'Content-Type': contentType,
    'CF-Connecting-IP': ip,
    ...sigHeaders,
    ...headersOverride,
  };
  return new Request(url, {
    method,
    headers,
    body: method === 'GET' ? undefined : bodyString,
  });
}

function makeMockGithubFetch({ issueStatus = 201, tokenStatus = 201 } = {}) {
  const calls = [];
  let issueSeq = 100;
  const fetchFn = async (url, init = {}) => {
    calls.push({ url, init });
    if (String(url).includes('/access_tokens')) {
      if (tokenStatus !== 201) {
        return new Response(JSON.stringify({ message: 'token error' }), { status: tokenStatus });
      }
      return new Response(JSON.stringify({ token: 'ghs_test_installation_token' }), { status: 201 });
    }
    if (String(url).includes('/issues')) {
      if (issueStatus !== 201) {
        return new Response(JSON.stringify({ message: 'github error' }), { status: issueStatus });
      }
      issueSeq += 1;
      return new Response(
        JSON.stringify({
          number: issueSeq,
          html_url: `https://github.com/ngthaihoc/vga-report-sandbox/issues/${issueSeq}`,
        }),
        { status: 201 },
      );
    }
    throw new Error(`Unexpected fetch URL: ${url}`);
  };
  return { fetchFn, calls };
}

test('healthz_returns_200_ok_and_time', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const req = new Request('https://vga-reports.example.workers.dev/healthz', { method: 'GET' });
  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: async () => { throw new Error('no fetch'); } });
  assert.equal(res.status, 200);
  const data = await res.json();
  assert.deepEqual(data, { ok: true, time: FIXED_NOW });
});

test('post_valid_creates_issue_201', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport();
  const req = makeSignedRequest(JSON.stringify(report));

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 201);
  const data = await res.json();
  assert.equal(data.report_id, report.report_id);
  assert.equal(data.status, 'issue_created');
  assert.equal(data.issue_url, 'https://github.com/ngthaihoc/vga-report-sandbox/issues/101');
  assert.equal(calls.length, 2);

  const issuePayload = JSON.parse(calls[1].init.body);
  assert.deepEqual(issuePayload.labels, ['vga-report', 'hardware-sample', 'unverified']);
  assert.ok(issuePayload.body.includes(`<!-- vga-report:${report.report_id} -->`));
});

test('post_same_id_same_body_returns_same_issue_200', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());

  const res1 = await handleRequest(makeSignedRequest(body), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res1.status, 201);
  const data1 = await res1.json();

  const res2 = await handleRequest(makeSignedRequest(body), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res2.status, 200);
  const data2 = await res2.json();
  assert.equal(data2.issue_url, data1.issue_url);
  assert.equal(calls.length, 2, 'GitHub must not be called a second time');
});

test('post_same_id_different_body_409', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();

  const body1 = JSON.stringify(makeValidReport());
  const res1 = await handleRequest(makeSignedRequest(body1), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res1.status, 201);

  const modified = makeValidReport({
    context: {
      mod_state: 'modded',
      riser: 'riser',
      slot: 'chipset',
      stage: 'after_unlock',
      notes: 'Different body same report_id',
    },
  });
  const res2 = await handleRequest(makeSignedRequest(JSON.stringify(modified)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res2.status, 409);
});

test('post_pii_422_and_never_calls_github', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport({
    context: {
      mod_state: 'stock',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'baseline',
      notes: 'Saved in C:\\Users\\SecretPerson\\Desktop mac 00:11:22:33:44:55',
    },
  });

  const res = await handleRequest(makeSignedRequest(JSON.stringify(report)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 422);
  const text = await res.text();
  assert.equal(text.includes('SecretPerson'), false);
  assert.equal(text.includes('00:11:22:33:44:55'), false);
  assert.equal(calls.length, 0);
});

test('post_oversize_413', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const oversized = 'x'.repeat(2 * 1024 * 1024 + 10);
  const req = makeSignedRequest(oversized);

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 413);
  assert.equal(env.REPORTS.writes, 0);
});

test('post_wrong_content_type_415', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());
  const req = makeSignedRequest(body, { contentType: 'text/plain' });

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 415);
  assert.equal(env.REPORTS.writes, 0);
});

test('post_invalid_schema_400', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const badReport = makeValidReport({ extra_field: 'not_allowed' });

  const res = await handleRequest(makeSignedRequest(JSON.stringify(badReport)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 400);
});

test('quota_ip_limit_429_with_retry_after', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ IP_LIMIT: '2', DAILY_CAP: '200' });
  const { fetchFn } = makeMockGithubFetch();

  for (let i = 1; i <= 2; i += 1) {
    const report = makeValidReport({ report_id: `11111111-2222-4333-8444-55555555550${i}` });
    const res = await handleRequest(makeSignedRequest(JSON.stringify(report)), env, { now: () => FIXED_NOW, fetch: fetchFn });
    assert.equal(res.status, 201);
  }

  const report3 = makeValidReport({ report_id: '11111111-2222-4333-8444-555555555503' });
  const res3 = await handleRequest(makeSignedRequest(JSON.stringify(report3)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res3.status, 429);
  const retryAfter = res3.headers.get('Retry-After');
  assert.ok(retryAfter && Number(retryAfter) > 0);
});

test('quota_daily_cap_429', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ IP_LIMIT: '10', DAILY_CAP: '2' });
  const { fetchFn } = makeMockGithubFetch();

  for (let i = 1; i <= 2; i += 1) {
    const report = makeValidReport({ report_id: `11111111-2222-4333-8444-55555555560${i}` });
    const res = await handleRequest(
      makeSignedRequest(JSON.stringify(report), { ip: `203.0.113.1${i}` }),
      env,
      { now: () => FIXED_NOW, fetch: fetchFn },
    );
    assert.equal(res.status, 201);
  }

  const report3 = makeValidReport({ report_id: '11111111-2222-4333-8444-555555555603' });
  const res3 = await handleRequest(
    makeSignedRequest(JSON.stringify(report3), { ip: '203.0.113.99' }),
    env,
    { now: () => FIXED_NOW, fetch: fetchFn },
  );
  assert.equal(res3.status, 429);
});

test('intake_disabled_503', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ INTAKE_ENABLED: 'false' });
  const { fetchFn, calls } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());

  const res = await handleRequest(makeSignedRequest(body), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 503);
  assert.equal(calls.length, 0);
  assert.equal(env.REPORTS.writes, 0);
});

test('github_5xx_returns_503_and_no_record_saved', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch({ issueStatus: 502 });
  const report = makeValidReport();

  const res = await handleRequest(makeSignedRequest(JSON.stringify(report)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 503);
  assert.equal(env.REPORTS.store.has(`r:${report.report_id}`), false);

  const { fetchFn: badSecretFetch } = makeMockGithubFetch({ tokenStatus: 401 });
  const resTokenFail = await handleRequest(
    makeSignedRequest(JSON.stringify(makeValidReport({ report_id: '11111111-2222-4333-8444-555555555599' }))),
    env,
    { now: () => FIXED_NOW, fetch: badSecretFetch },
  );
  assert.equal(resTokenFail.status, 503);
});

test('unknown_route_404', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const req = new Request('https://vga-reports.example.workers.dev/unknown', { method: 'GET' });
  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: async () => { throw new Error('no fetch'); } });
  assert.equal(res.status, 404);
});

test('issue_body_escapes_note_mention_and_expression', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport({
    context: {
      mod_state: 'modded',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'troubleshooting',
      notes: 'Hi @everyone <img src=x> ${{ env.SECRET }}',
    },
  });

  const res = await handleRequest(makeSignedRequest(JSON.stringify(report)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 201);
  const issuePayload = JSON.parse(calls[1].init.body);
  assert.equal(issuePayload.body.includes('@everyone'), false);
  assert.equal(issuePayload.body.includes('<img'), false);
  assert.equal(issuePayload.body.includes('${{'), false);
});

test('ip_is_hashed_with_salt_and_never_logged', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const rawIp = '198.51.100.77';
  const logged = [];
  const origLog = console.log;
  const origWarn = console.warn;
  const origErr = console.error;
  console.log = (...args) => logged.push(args.join(' '));
  console.warn = (...args) => logged.push(args.join(' '));
  console.error = (...args) => logged.push(args.join(' '));
  try {
    const body = JSON.stringify(makeValidReport());
    const res = await handleRequest(
      makeSignedRequest(body, { ip: rawIp }),
      env,
      { now: () => FIXED_NOW, fetch: fetchFn },
    );
    assert.equal(res.status, 201);
  } finally {
    console.log = origLog;
    console.warn = origWarn;
    console.error = origErr;
  }

  for (const key of env.REPORTS.keys()) {
    assert.equal(key.includes(rawIp), false, 'KV key must never store raw IP');
  }
  assert.ok(env.REPORTS.keys().some((k) => k.startsWith('q:2026-11-21:')));
  assert.equal(logged.join('\n').includes(rawIp), false, 'Logs must never contain raw IP');
});

test('unsigned_request_401_and_no_kv_write', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());
  const req = makeSignedRequest(body, { keyId: null });

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 401);
  assert.equal(env.REPORTS.reads, 0);
  assert.equal(env.REPORTS.writes, 0);
  assert.equal(calls.length, 0);
});

test('bad_signature_401_and_no_kv_write', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());
  const req = makeSignedRequest(body, {
    headersOverride: { 'X-Report-Signature': '0'.repeat(64) },
  });

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 401);
  const data = await res.json();
  assert.equal(data.reason, 'bad_signature');
  assert.equal(env.REPORTS.reads, 0);
  assert.equal(env.REPORTS.writes, 0);
});

test('unknown_key_id_401', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());
  const req = makeSignedRequest(body, { keyId: 'nonexistent', keyBase64: TEST_KEY_K1 });

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 401);
  const data = await res.json();
  assert.equal(data.reason, 'unknown_key');
  assert.equal(env.REPORTS.writes, 0);
});

test('revoked_key_id_401_while_new_key_accepted', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ SIGNING_KEYS: JSON.stringify({ k2: TEST_KEY_K2 }) });
  const { fetchFn } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());

  const oldReq = makeSignedRequest(body, { keyId: 'k1', keyBase64: TEST_KEY_K1 });
  const resOld = await handleRequest(oldReq, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(resOld.status, 401);

  const newReq = makeSignedRequest(body, { keyId: 'k2', keyBase64: TEST_KEY_K2 });
  const resNew = await handleRequest(newReq, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(resNew.status, 201);
});

test('tampered_body_after_signing_401', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const origBody = JSON.stringify(makeValidReport());
  const sigHeaders = signBody(origBody, 'k1', TEST_KEY_K1, FIXED_NOW);
  const tamperedBody = origBody.replace('Clean test report', 'Tampered test report');

  const req = new Request('https://vga-reports.example.workers.dev/v1/reports', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'CF-Connecting-IP': '203.0.113.10',
      ...sigHeaders,
    },
    body: tamperedBody,
  });

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 401);
  const data = await res.json();
  assert.equal(data.reason, 'bad_signature');
  assert.equal(env.REPORTS.writes, 0);
});

test('clock_skew_401_returns_server_time', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn } = makeMockGithubFetch();
  const body = JSON.stringify(makeValidReport());
  const skewedTs = FIXED_NOW - 600;
  const req = makeSignedRequest(body, { timestamp: skewedTs });

  const res = await handleRequest(req, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 401);
  const data = await res.json();
  assert.equal(data.reason, 'clock_skew');
  assert.equal(data.server_time, FIXED_NOW);
  assert.equal(env.REPORTS.writes, 0);
});

test('deterministic_signature_test_vector', async () => {
  const { verifySignature } = await import(workerModuleUrl);
  const env = await makeEnv({
    SIGNING_KEYS: JSON.stringify({
      k1: 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=',
    }),
  });
  const body = '{"hello":"thế giới"}';
  const ts = 1700000000;
  const expectedSig = 'a1ef303751bfa35a80ffba19567a172abfdd3047b0c9d36ffd4edd5847ddb9a7';
  const expectedHeaders = signBody(body, 'k1', 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=', ts);
  assert.equal(expectedHeaders['X-Report-Signature'], expectedSig);

  const req = new Request('https://vga-reports.example.workers.dev/v1/reports', {
    method: 'POST',
    headers: expectedHeaders,
    body,
  });
  const res = await verifySignature(env, req, body, ts);
  assert.equal(res.ok, true);
});

test('invalid_payloads_do_not_touch_kv_or_github', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  for (const [body, expected] of [
    ['{', 400],
    [JSON.stringify({ bad: true }), 400],
    [JSON.stringify(makeValidReport({
      context: { mod_state: 'stock', riser: 'direct', slot: 'cpu_x16', stage: 'baseline', notes: 'MAC 00:11:22:33:44:55' },
    })), 422],
    [JSON.stringify(makeValidReport({
      context: { mod_state: 'stock', riser: 'direct', slot: 'cpu_x16', stage: 'baseline', notes: 'IPv6 2001:db8::1' },
    })), 422],
  ]) {
    const env = await makeEnv();
    const { fetchFn, calls } = makeMockGithubFetch();
    const res = await handleRequest(makeSignedRequest(body), env, { now: () => FIXED_NOW, fetch: fetchFn });
    assert.equal(res.status, expected);
    assert.equal(env.REPORTS.reads, 0);
    assert.equal(env.REPORTS.writes, 0);
    assert.equal(calls.length, 0);
  }
});

test('disabled_intake_precedes_json_validation_after_signature', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ INTAKE_ENABLED: 'false' });
  const { fetchFn, calls } = makeMockGithubFetch();
  const res = await handleRequest(makeSignedRequest('{'), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 503);
  assert.equal((await res.json()).reason, 'disabled');
  const unsigned = await handleRequest(makeSignedRequest('{', { keyId: null }), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(unsigned.status, 401);
  assert.equal(env.REPORTS.reads, 0);
  assert.equal(env.REPORTS.writes, 0);
  assert.equal(calls.length, 0);
});

test('missing_or_blank_ip_salt_fails_closed_without_kv', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  for (const IP_SALT of [undefined, '', '   ']) {
    const env = await makeEnv({ IP_SALT });
    const { fetchFn, calls } = makeMockGithubFetch();
    const res = await handleRequest(makeSignedRequest(JSON.stringify(makeValidReport())), env, { now: () => FIXED_NOW, fetch: fetchFn });
    assert.equal(res.status, 503);
    assert.equal((await res.json()).reason, 'configuration_error');
    assert.equal(env.REPORTS.reads, 0);
    assert.equal(env.REPORTS.writes, 0);
    assert.equal(calls.length, 0);
  }
});

test('duplicate_and_conflict_bypass_exhausted_quota_without_writes', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ IP_LIMIT: '1', DAILY_CAP: '1' });
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport();
  const body = JSON.stringify(report);
  const first = await handleRequest(makeSignedRequest(body), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(first.status, 201);
  const firstData = await first.json();
  const writes = env.REPORTS.writes;
  const duplicate = await handleRequest(makeSignedRequest(body), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(duplicate.status, 200);
  assert.equal((await duplicate.json()).issue_url, firstData.issue_url);
  const changed = JSON.stringify({ ...report, collector_version: '1.0.1' });
  const conflict = await handleRequest(makeSignedRequest(changed), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(conflict.status, 409);
  assert.equal(env.REPORTS.writes, writes);
  assert.equal(calls.length, 2);
});

test('streamed_oversize_body_is_cancelled_without_buffering_the_rest', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  for (const declaredLength of [null, '1']) {
    const env = await makeEnv();
    const { fetchFn, calls } = makeMockGithubFetch();
    let pulls = 0;
    let cancelled = false;
    const body = new ReadableStream({
      pull(controller) {
        pulls += 1;
        controller.enqueue(new Uint8Array(65536));
        if (pulls === 40) controller.close();
      },
      cancel() { cancelled = true; },
    }, { highWaterMark: 0 });
    const headers = { 'Content-Type': 'application/json' };
    if (declaredLength !== null) headers['Content-Length'] = declaredLength;
    const request = new Request('https://vga-reports.example.workers.dev/v1/reports', {
      method: 'POST', headers, body, duplex: 'half',
    });
    const res = await handleRequest(request, env, { now: () => FIXED_NOW, fetch: fetchFn });
    assert.equal(res.status, 413);
    assert.equal(cancelled, true);
    assert.equal(pulls, 33, 'stop immediately after the first chunk beyond 2 MiB');
    assert.equal(env.REPORTS.reads, 0);
    assert.equal(env.REPORTS.writes, 0);
    assert.equal(calls.length, 0);
  }
});

test('kv_read_and_write_failures_return_controlled_503', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  for (const stage of ['dedup_read', 'quota_read', 'quota_write', 'dedup_write']) {
    const kv = new MockKv();
    const get = kv.get.bind(kv);
    const put = kv.put.bind(kv);
    kv.get = async (key, type) => {
      if ((stage === 'dedup_read' && key.startsWith('r:')) || (stage === 'quota_read' && key.startsWith('d:'))) throw new Error('KV unavailable');
      return get(key, type);
    };
    kv.put = async (key, value, options) => {
      if ((stage === 'quota_write' && key.startsWith('d:')) || (stage === 'dedup_write' && key.startsWith('r:'))) throw new Error('KV unavailable');
      return put(key, value, options);
    };
    const env = await makeEnv({ REPORTS: kv });
    const { fetchFn, calls } = makeMockGithubFetch();
    const res = await handleRequest(makeSignedRequest(JSON.stringify(makeValidReport())), env, { now: () => FIXED_NOW, fetch: fetchFn });
    assert.equal(res.status, 503, stage);
    assert.equal((await res.json()).reason, 'kv_error', stage);
    assert.equal(calls.length, stage === 'dedup_write' ? 2 : 0, stage);
  }
});

test('same_key_kv_write_rate_failure_does_not_create_another_issue', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const kv = new MockKv();
  const put = kv.put.bind(kv);
  kv.put = async (key, value, options) => {
    if (key.startsWith('d:') && kv.store.has(key)) throw new Error('429: same key write rate exceeded');
    return put(key, value, options);
  };
  const env = await makeEnv({ REPORTS: kv });
  const { fetchFn, calls } = makeMockGithubFetch();
  for (let i = 0; i < 2; i += 1) {
    const report = makeValidReport({ report_id: `11111111-2222-4333-8444-55555555700${i}` });
    const res = await handleRequest(makeSignedRequest(JSON.stringify(report)), env, { now: () => FIXED_NOW, fetch: fetchFn });
    assert.equal(res.status, i === 0 ? 201 : 503);
    if (i === 1) assert.equal((await res.json()).reason, 'kv_error');
  }
  assert.equal(calls.length, 2);
});

test('issue_body_limit_includes_the_dedup_marker', async () => {
  const { createIssue } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport();
  const largeGpu = Object.fromEntries(Object.keys(report.gpus[0]).map((key) => [key, makeField('&'.repeat(500))]));
  const largeReport = { ...report, gpus: Array.from({ length: 16 }, () => largeGpu) };
  await createIssue(env, 'test-token', largeReport, fetchFn);
  const { body } = JSON.parse(calls[0].init.body);
  assert.equal(body.length, 60000);
  assert.ok(body.endsWith(`<!-- vga-report:${report.report_id} -->`));
});

test('concurrent_50_mixed_requests_obey_quota_in_one_isolate', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv({ IP_LIMIT: '100', DAILY_CAP: '12' });
  const { fetchFn, calls } = makeMockGithubFetch();
  const requests = Array.from({ length: 50 }, (_, idx) => {
    const report = makeValidReport({ report_id: `11111111-2222-4333-8444-5555555560${String(idx + 1).padStart(2, '0')}` });
    const category = idx % 10;
    let body;
    if (category === 0) body = '{';
    else if (category === 1) body = JSON.stringify({ bad: true });
    else if (category === 2) body = JSON.stringify({ ...report, context: { ...report.context, notes: 'MAC 00:11:22:33:44:55' } });
    else body = JSON.stringify(report);
    const request = makeSignedRequest(body, { ip: `198.51.100.${idx + 1}`, keyId: category === 3 ? null : 'k1' });
    return handleRequest(request, env, { now: () => FIXED_NOW, fetch: fetchFn });
  });
  const responses = await Promise.all(requests);
  const count = (status) => responses.filter((res) => res.status === status).length;
  assert.equal(count(400), 10);
  assert.equal(count(422), 5);
  assert.equal(count(401), 5);
  assert.equal(count(201), 12);
  assert.equal(count(429), 18);
  assert.equal(env.REPORTS.writes, 36);
  assert.equal(env.REPORTS.store.get('d:2026-11-21').value, '12');
  assert.equal(calls.filter((call) => call.url.endsWith('/issues')).length, 12);
});

test('chunked_utf8_body_at_exact_limit_is_accepted_and_body_errors_are_controlled', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport();
  const json = JSON.stringify({ ...report, context: { ...report.context, notes: 'Báo cáo có dấu' } });
  const payload = ' '.repeat(2 * 1024 * 1024 - Buffer.byteLength(json)) + json;
  const bytes = new TextEncoder().encode(payload);
  let offset = 0;
  const body = new ReadableStream({
    pull(controller) {
      if (offset === bytes.length) { controller.close(); return; }
      controller.enqueue(bytes.subarray(offset, offset + 1019));
      offset = Math.min(bytes.length, offset + 1019);
    },
  });
  const request = new Request('https://vga-reports.example.workers.dev/v1/reports', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...signBody(payload, 'k1', TEST_KEY_K1, FIXED_NOW) },
    body,
    duplex: 'half',
  });
  const res = await handleRequest(request, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 201);
  assert.equal(calls.length, 2);
  const brokenRequest = new Request(request.url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: new ReadableStream({ start(controller) { controller.error(new Error('transport closed')); } }),
    duplex: 'half',
  });
  const broken = await handleRequest(brokenRequest, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(broken.status, 400);
  assert.equal((await broken.json()).error, 'invalid_body');
  assert.equal(calls.length, 2);
});

test('declared_oversize_body_is_rejected_before_reading', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  let reads = 0;
  const request = new Request('https://vga-reports.example.workers.dev/v1/reports', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Content-Length': String(2 * 1024 * 1024 + 1) },
    body: new ReadableStream({ pull(controller) { reads += 1; controller.close(); } }, { highWaterMark: 0 }),
    duplex: 'half',
  });
  const res = await handleRequest(request, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 413);
  assert.equal(reads, 0);
  assert.equal(env.REPORTS.reads, 0);
  assert.equal(env.REPORTS.writes, 0);
  assert.equal(calls.length, 0);
});

test('tiny_chunks_preserve_exact_utf8_signature_bytes', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const report = makeValidReport();
  const payload = JSON.stringify({ ...report, context: { ...report.context, notes: 'Báo cáo tiếng Việt' } });
  const bytes = new TextEncoder().encode(payload);
  let offset = 0;
  let chunkSize = 1;
  const request = new Request('https://vga-reports.example.workers.dev/v1/reports', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...signBody(payload, 'k1', TEST_KEY_K1, FIXED_NOW) },
    body: new ReadableStream({
      pull(controller) {
        if (offset === bytes.length) { controller.close(); return; }
        const end = Math.min(bytes.length, offset + chunkSize);
        controller.enqueue(bytes.subarray(offset, end));
        offset = end;
        chunkSize = chunkSize % 7 + 1;
      },
    }, { highWaterMark: 0 }),
    duplex: 'half',
  });
  const res = await handleRequest(request, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 201);
  assert.equal(calls.length, 2);
});

test('oversize_body_still_returns_413_when_stream_cancellation_rejects', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  let cancelled = false;
  const request = new Request('https://vga-reports.example.workers.dev/v1/reports', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: new ReadableStream({
      pull(controller) { controller.enqueue(new Uint8Array(2 * 1024 * 1024 + 1)); },
      cancel() { cancelled = true; throw new Error('transport already closed'); },
    }, { highWaterMark: 0 }),
    duplex: 'half',
  });
  const res = await handleRequest(request, env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 413);
  assert.equal(cancelled, true);
  assert.equal(env.REPORTS.reads, 0);
  assert.equal(env.REPORTS.writes, 0);
  assert.equal(calls.length, 0);
});

test('post_v2_report_creates_issue_201_with_target_gpu_title', async () => {
  const { handleRequest } = await import(workerModuleUrl);
  const { NVIDIA_EXTENDED_FIELDS } = require('../../../scripts/vga-report/report.cjs');
  const env = await makeEnv();
  const { fetchFn, calls } = makeMockGithubFetch();
  const v1Base = makeValidReport();
  const extended = Object.fromEntries(
    NVIDIA_EXTENDED_FIELDS.map((k) => [
      k,
      k === 'enforced.power.limit' ? makeField(125, 'nvidia-smi', 'W') : { value: null, source: 'nvidia-smi', unit: null, status: 'unsupported' },
    ]),
  );
  const extended_missing = NVIDIA_EXTENDED_FIELDS.filter((k) => k !== 'enforced.power.limit');
  const v2Report = {
    ...v1Base,
    schema_version: 2,
    collector_version: '1.1.0',
    previous_report_id: null,
    target_bdf: '0000:02:00.0',
    collection_started_utc: '2026-11-21T07:00:00Z',
    collection_finished_utc: '2026-11-21T07:00:03Z',
    source_timestamps: {
      pnp_utc: '2026-11-21T07:00:01Z',
      cim_utc: '2026-11-21T07:00:01Z',
      nvidia_smi_utc: '2026-11-21T07:00:02Z',
      security_utc: '2026-11-21T07:00:02Z',
    },
    evidence_level: 'reported',
    gpus: [
      {
        bdf: makeField('0000:00:02.0', 'pnp'),
        vendor_id: makeField('1002', 'pnp'),
        device_id: makeField('1638', 'pnp'),
        subsystem_id: makeField('88091043', 'pnp'),
        revision: makeField('C9', 'pnp'),
        name: makeField('AMD Radeon Graphics', 'cim'),
        pnp_status: makeField('OK', 'pnp'),
        driver_version: makeField('31.0.21921.1000', 'cim'),
      },
      {
        bdf: makeField('0000:02:00.0', 'pnp'),
        vendor_id: makeField('10DE', 'pnp'),
        device_id: makeField('1F0B', 'pnp'),
        subsystem_id: makeField('000010DE', 'pnp'),
        revision: makeField('A1', 'pnp'),
        name: makeField('NVIDIA CMP 40HX', 'cim'),
        pnp_status: makeField('OK', 'pnp'),
        driver_version: makeField('32.0.15.6094', 'cim'),
      },
    ],
    nvidia: {
      ...v1Base.nvidia,
      fields: {
        ...v1Base.nvidia.fields,
        name: makeField('NVIDIA CMP 40HX', 'nvidia-smi'),
        'pci.bus_id': makeField('0000:02:00.0', 'nvidia-smi'),
      },
      extended,
      extended_missing,
    },
    security: {
      boot_mode: makeField('UEFI', 'windows_env'),
      secure_boot: makeField(false, 'registry'),
      hvci_running: makeField(false, 'cim'),
      test_signing: makeField(false, 'registry'),
    },
    completeness: {
      inventory: { complete: true, status: 'ok', missing: [], reason: '' },
      topology: { complete: false, status: 'skipped', missing: ['upstream'], reason: 'P3' },
      link_idle: { complete: true, status: 'ok', missing: [], reason: '' },
      link_load: { complete: false, status: 'skipped', missing: ['load'], reason: 'P2' },
      compute: { complete: false, status: 'skipped', missing: ['cuda'], reason: 'P2' },
      stability: { complete: false, status: 'skipped', missing: ['whea'], reason: 'P2' },
      comparison: { complete: false, status: 'skipped', missing: ['prev'], reason: 'first run' },
    },
    legacy_status: {
      present: false,
      source: 'none',
      status_code: 'NONE',
      error_code: 'NONE',
      speed_current: null,
      width_current: null,
      tls_target: null,
      device_id: null,
      mrrs_bytes: null,
      observed_utc: null,
    },
    context: {
      ...v1Base.context,
      mod_type: 'none',
    },
  };

  const res = await handleRequest(makeSignedRequest(JSON.stringify(v2Report)), env, { now: () => FIXED_NOW, fetch: fetchFn });
  assert.equal(res.status, 201);
  assert.equal(calls.length, 2);
  const issuePayload = JSON.parse(calls[1].init.body);
  assert.ok(issuePayload.title.includes('NVIDIA CMP 40HX'), `Expected target GPU title, got: ${issuePayload.title}`);
  assert.equal(issuePayload.title.includes('AMD Radeon Graphics'), false);
  assert.ok(issuePayload.body.includes(`<!-- vga-report:${v2Report.report_id} -->`));
  assert.ok(issuePayload.body.includes('enforced.power.limit'));
});
