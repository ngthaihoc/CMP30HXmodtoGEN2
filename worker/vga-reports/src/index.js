import reportLib from '../../../scripts/vga-report/report.cjs';

const {
  validateReport,
  findPii,
  renderSummary,
  normalizeBdf,
  MAX_PAYLOAD_BYTES,
  MAX_SUMMARY_LENGTH,
} = reportLib;

const ENCODER = new TextEncoder();
const DECODER = new TextDecoder();
const CLOCK_SKEW_SECONDS = 300;
const QUOTA_TTL_SECONDS = 172800; // 2 days
const FIXED_ISSUE_LABELS = ['vga-report', 'hardware-sample', 'unverified'];
const REPO_NAME_RE = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;

const quotaLocks = new WeakMap();
let fallbackLock = Promise.resolve();

function withQuotaLock(kv, fn) {
  if (kv && typeof kv === 'object') {
    const prev = quotaLocks.get(kv) || Promise.resolve();
    const next = prev.then(fn, fn);
    quotaLocks.set(kv, next.catch(() => {}));
    return next;
  }
  const prev = fallbackLock;
  const next = prev.then(fn, fn);
  fallbackLock = next.catch(() => {});
  return next;
}

function jsonResponse(body, status, extraHeaders = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'no-store',
      ...extraHeaders,
    },
  });
}

async function readBoundedBody(request) {
  if (!request.body) return new Uint8Array();
  const reader = request.body.getReader();
  let bytes = new Uint8Array();
  let byteLength = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      const nextLength = byteLength + value.byteLength;
      if (nextLength > MAX_PAYLOAD_BYTES) {
        try { await reader.cancel(); } catch {}
        return null;
      }
      if (nextLength > bytes.byteLength) {
        const capacity = Math.min(MAX_PAYLOAD_BYTES, Math.max(nextLength, bytes.byteLength * 2, 1024));
        const grown = new Uint8Array(capacity);
        grown.set(bytes.subarray(0, byteLength));
        bytes = grown;
      }
      bytes.set(value, byteLength);
      byteLength = nextLength;
    }
  } finally {
    reader.releaseLock();
  }
  return bytes.subarray(0, byteLength);
}

function base64ToBytes(b64) {
  const clean = String(b64).trim();
  const bin = atob(clean);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i += 1) {
    bytes[i] = bin.charCodeAt(i);
  }
  return bytes;
}

function bytesToHex(bytes) {
  let out = '';
  for (let i = 0; i < bytes.length; i += 1) {
    out += bytes[i].toString(16).padStart(2, '0');
  }
  return out;
}

function hexToBytes(hex) {
  if (typeof hex !== 'string' || !/^[0-9a-fA-F]{64}$/.test(hex)) {
    return null;
  }
  const out = new Uint8Array(32);
  for (let i = 0; i < 32; i += 1) {
    out[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

function base64UrlEncode(input) {
  const bytes = typeof input === 'string' ? ENCODER.encode(input) : new Uint8Array(input);
  let bin = '';
  for (let i = 0; i < bytes.length; i += 1) {
    bin += String.fromCharCode(bytes[i]);
  }
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

async function sha256Hex(rawBody) {
  const bytes = typeof rawBody === 'string' ? ENCODER.encode(rawBody) : rawBody;
  const digest = await crypto.subtle.digest('SHA-256', bytes);
  return bytesToHex(new Uint8Array(digest));
}

async function hmacSha256Hex(secretStr, messageStr) {
  const key = await crypto.subtle.importKey(
    'raw',
    ENCODER.encode(secretStr),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  );
  const sig = await crypto.subtle.sign('HMAC', key, ENCODER.encode(messageStr));
  return bytesToHex(new Uint8Array(sig));
}

function parseSigningKeys(env) {
  if (!env || !env.SIGNING_KEYS) return {};
  if (typeof env.SIGNING_KEYS === 'object') return env.SIGNING_KEYS;
  try {
    const parsed = JSON.parse(String(env.SIGNING_KEYS));
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch {
    return {};
  }
}

export async function verifySignature(env, request, rawBody, now) {
  const serverTime = Math.floor(typeof now === 'number' ? now : Date.now() / 1000);
  const keyId = request.headers.get('X-Report-Key-Id');
  const tsRaw = request.headers.get('X-Report-Timestamp');
  const sigHex = request.headers.get('X-Report-Signature');

  if (!keyId || !tsRaw || !sigHex || !/^\d+$/.test(tsRaw)) {
    return { ok: false, reason: 'missing', serverTime };
  }

  const keys = parseSigningKeys(env);
  if (!Object.prototype.hasOwnProperty.call(keys, keyId) || typeof keys[keyId] !== 'string') {
    return { ok: false, reason: 'unknown_key', serverTime };
  }

  const ts = Number.parseInt(tsRaw, 10);
  if (!Number.isFinite(ts) || Math.abs(serverTime - ts) > CLOCK_SKEW_SECONDS) {
    return { ok: false, reason: 'clock_skew', serverTime };
  }

  const sigBytes = hexToBytes(sigHex);
  if (!sigBytes) {
    return { ok: false, reason: 'bad_signature', serverTime };
  }

  try {
    const keyBytes = base64ToBytes(keys[keyId]);
    const cryptoKey = await crypto.subtle.importKey(
      'raw',
      keyBytes,
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['verify'],
    );
    const bodyHashHex = await sha256Hex(rawBody);
    const payloadBytes = ENCODER.encode(`${ts}.${bodyHashHex}`);
    const valid = await crypto.subtle.verify('HMAC', cryptoKey, sigBytes, payloadBytes);
    if (!valid) {
      return { ok: false, reason: 'bad_signature', serverTime };
    }
    return { ok: true, serverTime };
  } catch {
    return { ok: false, reason: 'bad_signature', serverTime };
  }
}

function secondsUntilNextUtcMidnight(nowSec) {
  const secOfDay = ((nowSec % 86400) + 86400) % 86400;
  return Math.max(1, 86400 - secOfDay);
}

export async function checkQuota(env, ip, now) {
  if (String(env.INTAKE_ENABLED || 'true').toLowerCase() === 'false') {
    return { allowed: false, reason: 'disabled' };
  }
  if (!env.REPORTS) {
    throw new Error('Missing REPORTS KV binding');
  }
  if (typeof env.IP_SALT !== 'string' || !env.IP_SALT.trim()) {
    throw new Error('Missing IP_SALT');
  }
  const nowSec = Math.floor(typeof now === 'number' ? now : Date.now() / 1000);
  const day = new Date(nowSec * 1000).toISOString().slice(0, 10);
  const dailyCap = Math.max(0, Number.parseInt(String(env.DAILY_CAP || '200'), 10) || 200);
  const ipLimit = Math.max(0, Number.parseInt(String(env.IP_LIMIT || '10'), 10) || 10);
  const ipHash = await hmacSha256Hex(env.IP_SALT, `${day}:${ip || 'unknown'}`);
  const ipKey = `q:${day}:${ipHash}`;
  const dailyKey = `d:${day}`;

  return withQuotaLock(env.REPORTS, async () => {
    const [rawDaily, rawIp] = await Promise.all([
      env.REPORTS.get(dailyKey),
      env.REPORTS.get(ipKey),
    ]);
    const dailyCount = rawDaily ? Number.parseInt(rawDaily, 10) || 0 : 0;
    const ipCount = rawIp ? Number.parseInt(rawIp, 10) || 0 : 0;

    if (dailyCount >= dailyCap) {
      return { allowed: false, reason: 'daily' };
    }
    if (ipCount >= ipLimit) {
      return { allowed: false, reason: 'ip' };
    }

    await Promise.all([
      env.REPORTS.put(dailyKey, String(dailyCount + 1), { expirationTtl: QUOTA_TTL_SECONDS }),
      env.REPORTS.put(ipKey, String(ipCount + 1), { expirationTtl: QUOTA_TTL_SECONDS }),
    ]);
    return { allowed: true };
  });
}

function pemToPkcs8Bytes(pem) {
  const stripped = String(pem || '')
    .replace(/-----BEGIN PRIVATE KEY-----/g, '')
    .replace(/-----END PRIVATE KEY-----/g, '')
    .replace(/\s+/g, '');
  if (!stripped) {
    throw new Error('Empty GitHub App private key');
  }
  return base64ToBytes(stripped);
}

export async function getInstallationToken(env, fetchFn, now) {
  const nowSec = Math.floor(typeof now === 'number' ? now : Date.now() / 1000);
  if (!env.GITHUB_APP_ID || !env.GITHUB_APP_INSTALLATION_ID || !env.GITHUB_APP_PRIVATE_KEY) {
    throw new Error('Missing GitHub App credentials');
  }

  const keyBytes = pemToPkcs8Bytes(env.GITHUB_APP_PRIVATE_KEY);
  const signingKey = await crypto.subtle.importKey(
    'pkcs8',
    keyBytes,
    { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' },
    false,
    ['sign'],
  );

  const headerPart = base64UrlEncode(JSON.stringify({ alg: 'RS256', typ: 'JWT' }));
  const payloadPart = base64UrlEncode(
    JSON.stringify({
      iat: nowSec - 60,
      exp: nowSec + 540,
      iss: String(env.GITHUB_APP_ID),
    }),
  );
  const unsigned = `${headerPart}.${payloadPart}`;
  const sig = await crypto.subtle.sign('RSASSA-PKCS1-v1_5', signingKey, ENCODER.encode(unsigned));
  const jwt = `${unsigned}.${base64UrlEncode(sig)}`;

  const res = await fetchFn(
    `https://api.github.com/app/installations/${encodeURIComponent(String(env.GITHUB_APP_INSTALLATION_ID))}/access_tokens`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${jwt}`,
        Accept: 'application/vnd.github+json',
        'User-Agent': 'cmp-vga-reports-worker/1.0',
        'X-GitHub-Api-Version': '2022-11-28',
      },
    },
  );

  if (!res.ok) {
    throw new Error(`GitHub access_tokens failed: ${res.status}`);
  }
  const data = await res.json();
  if (!data || typeof data.token !== 'string' || !data.token) {
    throw new Error('GitHub access_tokens returned no token');
  }
  return data.token;
}

function buildSafeIssueTitle(report) {
  const gpus = Array.isArray(report.gpus) ? report.gpus : [];
  const nvFields = report.nvidia && report.nvidia.fields ? report.nvidia.fields : {};
  const explicitBdf = typeof report.target_bdf === 'string' ? normalizeBdf(report.target_bdf) : null;
  const nvBusField = nvFields['pci.bus_id'];
  const nvBdf = nvBusField && nvBusField.status === 'ok' ? normalizeBdf(nvBusField.value) : null;
  const wantedBdf = explicitBdf || nvBdf;
  const matchedGpu = wantedBdf
    ? gpus.find((g) => g && g.bdf && g.bdf.status === 'ok' && normalizeBdf(g.bdf.value) === wantedBdf)
    : null;
  const chosenGpu = matchedGpu || gpus[0];
  const primaryName = chosenGpu && chosenGpu.name && chosenGpu.name.status === 'ok'
    ? String(chosenGpu.name.value)
    : 'Unknown GPU';
  const cleanName = primaryName.replace(/[\r\n@<>\[\]()`${}]/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 80);
  const stage = report.context && report.context.stage ? String(report.context.stage) : 'unknown';
  return `[VGA Report] ${cleanName || 'Unknown GPU'} (${stage})`;
}

export async function createIssue(env, token, report, fetchFn) {
  const repo = String(env.TARGET_REPO || '');
  if (!REPO_NAME_RE.test(repo)) {
    throw new Error('Invalid TARGET_REPO');
  }

  const title = buildSafeIssueTitle(report);
  const summary = renderSummary(report);
  const marker = `\n\n<!-- vga-report:${report.report_id} -->`;
  const body = summary.slice(0, MAX_SUMMARY_LENGTH - marker.length) + marker;

  const res = await fetchFn(`https://api.github.com/repos/${repo}/issues`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: 'application/vnd.github+json',
      'Content-Type': 'application/json',
      'User-Agent': 'cmp-vga-reports-worker/1.0',
      'X-GitHub-Api-Version': '2022-11-28',
    },
    body: JSON.stringify({
      title,
      body,
      labels: FIXED_ISSUE_LABELS,
    }),
  });

  if (!res.ok) {
    throw new Error(`GitHub create issue failed: ${res.status}`);
  }
  const data = await res.json();
  if (!data || typeof data.html_url !== 'string' || typeof data.number !== 'number') {
    throw new Error('GitHub create issue missing number or html_url');
  }
  return { number: data.number, url: data.html_url };
}

export async function handleRequest(request, env, deps = {}) {
  const nowFn = typeof deps.now === 'function' ? deps.now : () => Math.floor(Date.now() / 1000);
  const fetchFn = typeof deps.fetch === 'function' ? deps.fetch : fetch;
  const nowSec = Math.floor(nowFn());

  const url = new URL(request.url);
  if (request.method === 'GET' && url.pathname === '/healthz') {
    return jsonResponse({ ok: true, time: nowSec }, 200);
  }

  if (request.method !== 'POST' || url.pathname !== '/v1/reports') {
    return jsonResponse({ error: 'not_found' }, 404);
  }

  const contentType = String(request.headers.get('Content-Type') || '').toLowerCase();
  if (!contentType.startsWith('application/json')) {
    return jsonResponse({ error: 'unsupported_media_type' }, 415);
  }

  const declaredLen = request.headers.get('Content-Length');
  if (declaredLen !== null && Number(declaredLen) > MAX_PAYLOAD_BYTES) {
    return jsonResponse({ error: 'payload_too_large' }, 413);
  }

  let rawBytes;
  try {
    rawBytes = await readBoundedBody(request);
  } catch {
    return jsonResponse({ error: 'invalid_body' }, 400);
  }
  if (rawBytes === null) {
    return jsonResponse({ error: 'payload_too_large' }, 413);
  }

  const sig = await verifySignature(env, request, rawBytes, nowSec);
  if (!sig.ok) {
    return jsonResponse(
      {
        error: 'unauthorized',
        reason: sig.reason,
        server_time: sig.serverTime,
      },
      401,
    );
  }

  if (String(env.INTAKE_ENABLED || 'true').toLowerCase() === 'false') {
    return jsonResponse({ error: 'service_unavailable', reason: 'disabled' }, 503);
  }

  let report;
  try {
    report = JSON.parse(DECODER.decode(rawBytes));
  } catch {
    return jsonResponse({ error: 'invalid_json' }, 400);
  }

  const validation = validateReport(report);
  if (!validation.ok) {
    return jsonResponse({ error: 'invalid_schema', details: validation.errors }, 400);
  }

  const pii = findPii(report);
  if (pii.length > 0) {
    return jsonResponse({ error: 'pii_detected', categories: pii }, 422);
  }

  if (!env.REPORTS || typeof env.IP_SALT !== 'string' || !env.IP_SALT.trim()) {
    return jsonResponse({ error: 'service_unavailable', reason: 'configuration_error' }, 503);
  }

  const bodyHash = await sha256Hex(rawBytes);
  const dedupKey = `r:${report.report_id}`;
  let existing;
  try {
    existing = await env.REPORTS.get(dedupKey, 'json');
  } catch {
    return jsonResponse({ error: 'service_unavailable', reason: 'kv_error' }, 503);
  }
  if (existing) {
    if (existing.sha256_body === bodyHash) {
      return jsonResponse(
        {
          report_id: report.report_id,
          status: 'duplicate',
          issue_url: existing.issue_url,
        },
        200,
      );
    }
    return jsonResponse(
      {
        error: 'conflict',
        message: 'report_id already exists with different content',
      },
      409,
    );
  }

  const ip = request.headers.get('CF-Connecting-IP') || '0.0.0.0';
  let quota;
  try {
    quota = await checkQuota(env, ip, nowSec);
  } catch {
    return jsonResponse({ error: 'service_unavailable', reason: 'kv_error' }, 503);
  }
  if (!quota.allowed) {
    return jsonResponse(
      { error: 'rate_limited', reason: quota.reason },
      429,
      { 'Retry-After': String(secondsUntilNextUtcMidnight(nowSec)) },
    );
  }

  let issue;
  try {
    const token = await getInstallationToken(env, fetchFn, nowSec);
    issue = await createIssue(env, token, report, fetchFn);
  } catch {
    return jsonResponse({ error: 'service_unavailable', reason: 'github_error' }, 503);
  }
  try {
    await env.REPORTS.put(
      dedupKey,
      JSON.stringify({
        sha256_body: bodyHash,
        issue_url: issue.url,
        issue_number: issue.number,
        created_at: nowSec,
      }),
    );
  } catch {
    return jsonResponse({ error: 'service_unavailable', reason: 'kv_error' }, 503);
  }
  return jsonResponse(
    {
      report_id: report.report_id,
      status: 'issue_created',
      issue_url: issue.url,
    },
    201,
  );
}

export default {
  fetch(request, env) {
    return handleRequest(request, env);
  },
};
