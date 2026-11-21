'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const {
  validateReport,
  findPii,
  renderSummary,
  NVIDIA_REQUIRED_FIELDS,
} = require('./report.cjs');

function makeField(value, source = 'test', unit = null, status = 'ok') {
  return { value, source, unit, status };
}

function makeUnsupportedField(source = 'test', unit = null, status = 'unsupported') {
  return { value: null, source, unit, status };
}

function makeNvidiaBlock({ complete = true, overrides = {} } = {}) {
  const sampleValues = {
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

  const fields = {};
  const missing = [];
  for (const key of NVIDIA_REQUIRED_FIELDS) {
    fields[key] = overrides[key] || sampleValues[key];
    if (fields[key].status !== 'ok') {
      missing.push(key);
    }
  }

  return {
    complete: missing.length === 0 ? complete : false,
    missing,
    fields,
  };
}

function makeGpu(overrides = {}) {
  return {
    bdf: makeField('0000:01:00.0', 'pnp'),
    vendor_id: makeField('10DE', 'pnp'),
    device_id: makeField('1F07', 'pnp'),
    subsystem_id: makeField('86701043', 'pnp'),
    revision: makeField('A1', 'pnp'),
    name: makeField('NVIDIA GeForce RTX 2070', 'cim'),
    pnp_status: makeField('OK', 'pnp'),
    driver_version: makeField('32.0.16.2101', 'cim'),
    pcie: {
      max_gen: makeField(3, 'pnp', 'gen'),
      current_gen: makeField(3, 'pnp', 'gen'),
      max_width: makeField(16, 'pnp', 'lanes'),
      current_width: makeField(16, 'pnp', 'lanes'),
    },
    ...overrides,
  };
}

function makeValidReport(overrides = {}) {
  return {
    schema_version: 1,
    collector_version: '1.0.0',
    report_id: '11111111-2222-4333-8444-555555555555',
    created_utc: '2026-11-21T07:00:00Z',
    experiment_id: 'aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee',
    consent: true,
    gpus: [makeGpu()],
    nvidia: makeNvidiaBlock(),
    pcie: {
      max_gen: makeField(3, 'pnp', 'gen'),
      current_gen: makeField(3, 'pnp', 'gen'),
      max_width: makeField(16, 'pnp', 'lanes'),
      current_width: makeField(16, 'pnp', 'lanes'),
    },
    platform: {
      cpu: makeField('AMD Ryzen 5 5600GT', 'cim'),
      mainboard_model: makeField('ASUSTeK COMPUTER INC. TUF GAMING B450M-PRO S', 'cim'),
      bios_version: makeField('4655', 'cim'),
      os_build: makeField('10.0.26100', 'cim'),
    },
    context: {
      mod_state: 'stock',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'baseline',
      notes: 'Clean test sample',
    },
    ...overrides,
  };
}

test('validateReport_accepts_unsupported_fields_without_zero', () => {
  const report = makeValidReport({
    nvidia: makeNvidiaBlock({
      overrides: {
        compute_cap: makeUnsupportedField('nvidia-smi'),
        'power.draw': makeUnsupportedField('nvidia-smi', 'W'),
      },
    }),
    pcie: {
      max_gen: makeUnsupportedField('pnp', 'gen'),
      current_gen: makeUnsupportedField('pnp', 'gen'),
      max_width: makeUnsupportedField('pnp', 'lanes'),
      current_width: makeUnsupportedField('pnp', 'lanes'),
    },
  });

  const result = validateReport(report);
  assert.equal(result.ok, true, `Expected valid report, got errors: ${result.errors.join('; ')}`);
  assert.deepEqual(result.errors, []);

  // Using 0 when status is unsupported must be rejected
  const zeroReport = makeValidReport({
    nvidia: makeNvidiaBlock({
      overrides: {
        'power.draw': { value: 0, source: 'nvidia-smi', unit: 'W', status: 'unsupported' },
      },
    }),
  });
  const zeroRes = validateReport(zeroReport);
  assert.equal(zeroRes.ok, false);
});

test('validateReport_rejects_extra_property', () => {
  const report = makeValidReport({ unexpected_root: 'bad' });
  const resRoot = validateReport(report);
  assert.equal(resRoot.ok, false);
  assert.ok(resRoot.errors.some((e) => e.includes('unexpected_root')));

  const nestedExtra = makeValidReport({
    context: {
      mod_state: 'stock',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'baseline',
      notes: 'ok',
      extra_context: 123,
    },
  });
  const resNested = validateReport(nestedExtra);
  assert.equal(resNested.ok, false);
  assert.ok(resNested.errors.some((e) => e.includes('extra_context')));
});

test('validateReport_rejects_more_than_16_gpus', () => {
  const gpus = [];
  for (let i = 0; i < 17; i += 1) {
    const hexBus = i.toString(16).padStart(2, '0');
    gpus.push(makeGpu({ bdf: makeField(`0000:${hexBus}:00.0`, 'pnp') }));
  }
  const report = makeValidReport({ gpus });
  const result = validateReport(report);
  assert.equal(result.ok, false);
  assert.ok(result.errors.some((e) => e.includes('16') || e.includes('gpus')));
});

test('findPii_flags_user_path_mac_ipv4', () => {
  const report = makeValidReport({
    context: {
      mod_state: 'modded',
      riser: 'riser',
      slot: 'other',
      stage: 'troubleshooting',
      notes: 'Log at C:\\Users\\ngthaihoc\\Desktop\\log.txt mac 00:1A:2B:3C:4D:5E ip 192.168.1.45',
    },
  });

  const findings = findPii(report);
  assert.ok(findings.includes('path_user'));
  assert.ok(findings.includes('mac'));
  assert.ok(findings.includes('ipv4'));

  // Valid hardware versions like 90.06.18.80.06 or 32.0.16.2101 must NOT trigger ipv4
  const cleanReport = makeValidReport();
  assert.deepEqual(findPii(cleanReport), []);
});

test('findPii_returns_names_not_values', () => {
  const secretPath = 'C:\\Users\\AliceSecretUser\\Documents\\dump.txt';
  const secretMac = 'AA-BB-CC-DD-EE-FF';
  const secretIp = '10.24.56.78';
  const secretUuid = 'GPU-12345678-1234-1234-1234-123456789abc';
  const secretSerial = 'serial=SN123456789012';

  const report = makeValidReport({
    context: {
      mod_state: 'unknown',
      riser: 'unknown',
      slot: 'unknown',
      stage: 'unknown',
      notes: `${secretPath} ${secretMac} ${secretIp} ${secretUuid} ${secretSerial}`,
    },
  });

  const findings = findPii(report);
  const serialized = JSON.stringify(findings);
  for (const needle of ['AliceSecretUser', secretMac, secretIp, secretUuid, 'SN123456789012']) {
    assert.equal(serialized.includes(needle), false, `Must not echo leaked value: ${needle}`);
  }
  assert.deepEqual(
    findings.slice().sort(),
    ['ipv4', 'mac', 'path_user', 'serial', 'uuid_gpu'].sort(),
  );
});

test('renderSummary_escapes_mention_html_link_and_expression', () => {
  const report = makeValidReport({
    context: {
      mod_state: 'modded',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'after_unlock',
      notes: 'Ping @admin <script>alert(1)</script> [click](https://evil.example) ${{ secrets.GITHUB_TOKEN }}',
    },
  });

  const summary = renderSummary(report);
  assert.equal(summary.includes('@admin'), false);
  assert.equal(summary.includes('<script>'), false);
  assert.equal(summary.includes('[click](https://evil.example)'), false);
  assert.equal(summary.includes('${{'), false);
  assert.ok(summary.includes('&#64;admin'));
  assert.ok(summary.includes('&lt;script&gt;'));
  assert.ok(summary.includes('&#36;&#123;&#123;'));
});

test('renderSummary_handles_gpu_with_missing_driver', () => {
  const report = makeValidReport({
    gpus: [
      makeGpu({
        pnp_status: makeField('Error (Code 43)', 'pnp'),
        driver_version: makeUnsupportedField('cim'),
      }),
    ],
    nvidia: makeNvidiaBlock({
      overrides: Object.fromEntries(
        NVIDIA_REQUIRED_FIELDS.map((k) => [k, makeUnsupportedField('nvidia-smi', null, 'skipped')]),
      ),
    }),
  });

  const summary = renderSummary(report);
  assert.ok(summary.includes('unsupported'));
  assert.ok(summary.includes('skipped'));
  assert.ok(summary.includes('Error &#40;Code 43&#41;') || summary.includes('Error (Code 43)'));
});

test('renderSummary_handles_noncontiguous_bdf', () => {
  const report = makeValidReport({
    gpus: [
      makeGpu({
        bdf: makeField('0000:00:02.0', 'pnp'),
        vendor_id: makeField('1002', 'pnp'),
        device_id: makeField('1638', 'pnp'),
        name: makeField('AMD Radeon Graphics', 'cim'),
      }),
      makeGpu({
        bdf: makeField('0000:07:00.0', 'pnp'),
        vendor_id: makeField('10DE', 'pnp'),
        device_id: makeField('2189', 'pnp'),
        name: makeField('NVIDIA CMP 30HX', 'cim'),
      }),
    ],
  });

  const summary = renderSummary(report);
  assert.ok(summary.includes('0000:00:02.0'));
  assert.ok(summary.includes('0000:07:00.0'));
  assert.ok(summary.includes('NVIDIA CMP 30HX'));
});
