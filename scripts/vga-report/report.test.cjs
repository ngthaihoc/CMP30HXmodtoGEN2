'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const {
  validateReport,
  findPii,
  renderSummary,
  identifyGpuProfile,
  getGpuProfileDetails,
  NVIDIA_REQUIRED_FIELDS,
  NVIDIA_EXTENDED_FIELDS,
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

  const spacedReport = makeValidReport({
    context: {
      mod_state: 'unknown',
      riser: 'unknown',
      slot: 'unknown',
      stage: 'unknown',
      notes: 'UNC \\\\file server\\shared logs\\vga.txt',
    },
  });
  assert.ok(findPii(spacedReport).includes('path_user'));

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

test('findPii_flags_ipv6_without_leaking_values', () => {
  for (const address of [
    '2001:db8:1234:5678:90ab:cdef:1234:5678',
    '2001:db8::1', '::1', '::', 'fe80::1234%12',
    '::ffff:192.0.2.128', '[2001:db8::1]:443',
    'https://[2001:db8::1]/report',
  ]) {
    const report = makeValidReport();
    report.context.notes = `Địa chỉ ${address}, cần xóa.`;
    const findings = findPii(report);
    assert.ok(findings.includes('ipv6'), `Missing IPv6 category for ${address}`);
    assert.equal(JSON.stringify(findings).includes(address), false);
  }
});

test('findPii_does_not_confuse_pci_identifiers_or_versions_with_ipv6', () => {
  for (const notes of [
    '0000:01:00.0', '00000000:07:00.0', '10DE:2189',
    '90.06.18.80.06', '32.0.16.2101', 'abc:def:ghi',
  ]) {
    const report = makeValidReport();
    report.context.notes = notes;
    assert.deepEqual(findPii(report), [], notes);
  }
});

test('identifyGpuProfile_matches_40hxcore_and_leaves_1f06_unknown', () => {
  assert.equal(identifyGpuProfile('10DE', '1F0B'), 'CMP 40HX (TU106)');
  assert.equal(identifyGpuProfile('10de', '2189'), 'CMP 30HX (TU116)');
  assert.equal(identifyGpuProfile('10DE', '1F06'), null);
  assert.equal(identifyGpuProfile('10DE', '1F07'), null);
  assert.equal(identifyGpuProfile(null, '1F0B'), null);
});

test('renderSummary_marks_target_gpu_profile_and_unknown_1f06', () => {
  const report = makeValidReport({
    gpus: [
      makeGpu({
        bdf: makeField('0000:01:00.0', 'pnp'),
        vendor_id: makeField('10DE', 'pnp'),
        device_id: makeField('1F06', 'pnp'),
        name: makeField('NVIDIA GeForce RTX 2060 SUPER', 'cim'),
      }),
      makeGpu({
        bdf: makeField('0000:02:00.0', 'pnp'),
        vendor_id: makeField('10DE', 'pnp'),
        device_id: makeField('1F0B', 'pnp'),
        name: makeField('NVIDIA CMP 40HX', 'cim'),
      }),
    ],
    nvidia: makeNvidiaBlock({
      overrides: {
        name: makeField('NVIDIA CMP 40HX', 'nvidia-smi'),
        'pci.bus_id': makeField('0000:02:00.0', 'nvidia-smi'),
        'pci.device_id': makeField('0x1F0B10DE', 'nvidia-smi'),
      },
    }),
  });

  const summary = renderSummary(report);
  assert.ok(summary.includes('GPU mục tiêu:'));
  assert.ok(summary.includes('0000:02:00.0'));
  assert.ok(summary.includes('CMP 40HX &#40;TU106&#41;') || summary.includes('CMP 40HX (TU106)'));
  assert.ok(summary.includes('(mục tiêu)'));
  assert.ok(summary.includes('chưa nhận diện'));
});

test('renderSummary_normalizes_gen_and_warns_only_on_source_conflict', () => {
  const conflictReport = makeValidReport({
    pcie: {
      max_gen: makeField(3, 'pnp', 'gen'),
      current_gen: makeField(1, 'pnp', 'gen'),
      max_width: makeField(16, 'pnp', 'lanes'),
      current_width: makeField(16, 'pnp', 'lanes'),
    },
  });
  const conflictSummary = renderSummary(conflictReport);
  assert.ok(conflictSummary.includes('Gen1 ×16'));
  assert.ok(conflictSummary.includes('Gen3 ×16'));
  assert.ok(conflictSummary.includes('Cảnh báo'));

  const alignedReport = makeValidReport();
  const alignedSummary = renderSummary(alignedReport);
  assert.ok(alignedSummary.includes('Gen3 ×16'));
  assert.equal(alignedSummary.includes('Cảnh báo:'), false);
});

test('renderSummary_shows_basic_nvidia_count_and_caveat', () => {
  const partialReport = makeValidReport({
    nvidia: makeNvidiaBlock({
      overrides: {
        compute_cap: makeUnsupportedField('nvidia-smi'),
      },
    }),
  });
  const summary = renderSummary(partialReport);
  assert.ok(summary.includes('17/18'));
  assert.ok(summary.includes('Trường NVIDIA cơ bản (nvidia-smi):'));
  assert.ok(summary.includes('không phải đủ dữ liệu nghiên cứu/mod'));
  assert.equal(validateReport(partialReport).ok, true);
});

function makeNvidiaExtendedBlock({ overrides = {} } = {}) {
  const sampleValues = {
    'power.limit': makeField(215, 'nvidia-smi', 'W'),
    'enforced.power.limit': makeField(140, 'nvidia-smi', 'W'),
    'power.default_limit': makeField(215, 'nvidia-smi', 'W'),
    'power.min_limit': makeField(105, 'nvidia-smi', 'W'),
    'power.max_limit': makeField(269, 'nvidia-smi', 'W'),
    'temperature.gpu.tlimit': makeField(45, 'nvidia-smi', 'C'),
    'temperature.memory': makeUnsupportedField('nvidia-smi', 'C'),
    'fan.speed': makeField(30, 'nvidia-smi', '%'),
    'utilization.gpu': makeField(12, 'nvidia-smi', '%'),
    'utilization.memory': makeField(8, 'nvidia-smi', '%'),
    'memory.used': makeField(512, 'nvidia-smi', 'MiB'),
    'memory.free': makeField(7680, 'nvidia-smi', 'MiB'),
    'ecc.mode.current': makeUnsupportedField('nvidia-smi'),
    'clocks_throttle_reasons.active': makeField('0x0000000000000001', 'nvidia-smi'),
    'clocks_throttle_reasons.sw_power_cap': makeField('Not Active', 'nvidia-smi'),
    'clocks_throttle_reasons.hw_slowdown': makeField('Not Active', 'nvidia-smi'),
    'clocks_throttle_reasons.hw_thermal_slowdown': makeField('Not Active', 'nvidia-smi'),
    'clocks_throttle_reasons.hw_power_brake_slowdown': makeField('Not Active', 'nvidia-smi'),
  };
  const extended = {};
  const extended_missing = [];
  for (const key of NVIDIA_EXTENDED_FIELDS) {
    extended[key] = overrides[key] || sampleValues[key];
    if (extended[key].status !== 'ok') {
      extended_missing.push(key);
    }
  }
  return { extended, extended_missing };
}

function makeValidV2Report(overrides = {}) {
  const ext = makeNvidiaExtendedBlock();
  const base = makeValidReport({
    schema_version: 2,
    collector_version: '1.1.0',
    previous_report_id: '22222222-3333-4444-8555-666666666666',
    target_bdf: '0000:01:00.0',
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
      makeGpu({
        profile: {
          id: null,
          name: null,
          family: null,
          category: 'unknown',
          target_pcie_gen: null,
          tensor_applicable: 'unknown',
          identity_note: null,
        },
        driver: {
          provider: makeField('NVIDIA', 'pnp'),
          date: makeField('2026-10-19', 'pnp'),
          inf_name: makeField('oem2.inf', 'pnp'),
          service: makeField('nvlddmkm', 'pnp'),
          problem_code: makeField(0, 'pnp'),
          location_path: makeField('PCIROOT(0)#PCI(0101)#PCI(0000)', 'pnp'),
        },
      }),
    ],
    nvidia: {
      ...makeNvidiaBlock(),
      ...ext,
    },
    security: {
      boot_mode: makeField('UEFI', 'windows_env'),
      secure_boot: makeField(false, 'registry'),
      hvci_running: makeField(false, 'cim'),
      test_signing: makeField(false, 'registry'),
    },
    completeness: {
      inventory: { complete: true, status: 'ok', missing: [], reason: '' },
      topology: { complete: false, status: 'partial', missing: ['upstream_root_port', 'tls'], reason: 'P3 read-only PCIe topology helper not in P1' },
      link_idle: { complete: true, status: 'ok', missing: [], reason: '' },
      link_load: { complete: false, status: 'skipped', missing: ['h2d_d2h', 'load_timeseries'], reason: 'P2 load benchmark skipped in basic mode' },
      compute: { complete: false, status: 'skipped', missing: ['cuda_runtime', 'tensor_check'], reason: 'P2 CUDA helper not invoked' },
      stability: { complete: false, status: 'skipped', missing: ['whea_events', 'tdr_events'], reason: 'Stability window requires load mode' },
      comparison: { complete: false, status: 'partial', missing: ['paired_diff'], reason: 'Previous report linked without P2 baseline metrics' },
    },
    legacy_status: {
      present: true,
      source: 'gen2_status',
      status_code: 'GEN2_SUCCESS',
      error_code: 'NONE',
      speed_current: 2,
      width_current: 16,
      tls_target: 2,
      device_id: '2189',
      mrrs_bytes: 512,
      observed_utc: '2026-11-21T06:50:00Z',
    },
    context: {
      mod_state: 'modded',
      mod_type: 'efi',
      riser: 'direct',
      slot: 'cpu_x16',
      stage: 'after_unlock',
      notes: 'Clean P1 v2 sample',
    },
  });
  return { ...base, ...overrides };
}

test('getGpuProfileDetails_covers_p106_p100_cmp_and_p100_name_ambiguity', () => {
  const cmp40 = getGpuProfileDetails('10DE', '1F0B');
  assert.equal(cmp40.name, 'CMP 40HX (TU106)');
  assert.equal(cmp40.target_pcie_gen, 2);
  assert.equal(cmp40.tensor_applicable, 'unverified');

  const cmp30 = getGpuProfileDetails('10DE', '2189');
  assert.equal(cmp30.name, 'CMP 30HX (TU116)');
  assert.equal(cmp30.tensor_applicable, 'no');

  const p106_100 = getGpuProfileDetails('10DE', '1C07');
  assert.equal(p106_100.name, 'P106-100 (GP106)');
  assert.equal(p106_100.category, 'mining');
  assert.equal(p106_100.tensor_applicable, 'no');
  assert.equal(identifyGpuProfile('10DE', '1C07'), 'P106-100 (GP106)');

  const p106_090 = getGpuProfileDetails('10DE', '1C09');
  assert.equal(p106_090.name, 'P106-090 (GP106)');
  assert.equal(p106_090.tensor_applicable, 'no');

  const p100_16 = getGpuProfileDetails('10DE', '15F8');
  assert.equal(p100_16.name, 'Tesla P100 PCIe 16GB (GP100)');
  assert.equal(p100_16.target_pcie_gen, 3);
  assert.equal(p100_16.tensor_applicable, 'no');

  assert.equal(getGpuProfileDetails('10DE', '15F7').name, 'Tesla P100 PCIe 12GB (GP100)');
  assert.equal(getGpuProfileDetails('10DE', '15F9').name, 'Tesla P100 SXM2 (GP100)');

  const ambiguous = getGpuProfileDetails('10DE', '9999', 'NVIDIA P100 Modded');
  assert.equal(ambiguous.name, null);
  assert.equal(ambiguous.category, 'unknown');
  assert.ok(ambiguous.identity_note && ambiguous.identity_note.includes('P100'));
});

test('validateReport_accepts_v2_full_contract_and_rejects_unallowlisted_v2_fields', () => {
  const validV2 = makeValidV2Report();
  const resValid = validateReport(validV2);
  assert.equal(resValid.ok, true, `Expected valid v2 report, got: ${resValid.errors.join('; ')}`);
  assert.deepEqual(findPii(validV2), []);

  const extraSecurity = makeValidV2Report({
    security: { ...validV2.security, extra_sec: makeField('bad') },
  });
  assert.equal(validateReport(extraSecurity).ok, false);

  const extraTimestamps = makeValidV2Report({
    source_timestamps: { ...validV2.source_timestamps, extra_ts: '2026-11-21T07:00:00Z' },
  });
  assert.equal(validateReport(extraTimestamps).ok, false);

  const badEvidence = makeValidV2Report({ evidence_level: 'magical' });
  assert.equal(validateReport(badEvidence).ok, false);

  const missingCompleteness = makeValidV2Report();
  delete missingCompleteness.completeness;
  assert.equal(validateReport(missingCompleteness).ok, false);

  const v1WithV2Field = makeValidReport({ security: validV2.security });
  assert.equal(validateReport(v1WithV2Field).ok, false);
});

test('validateReport_rejects_driver_inf_with_path_or_unallowlisted_legacy_error_code', () => {
  const validV2 = makeValidV2Report();

  const winPathInf = makeValidV2Report({
    gpus: [
      {
        ...validV2.gpus[0],
        driver: {
          ...validV2.gpus[0].driver,
          inf_name: makeField('C:\\Windows\\INF\\oem2.inf', 'pnp'),
        },
      },
    ],
  });
  assert.equal(validateReport(winPathInf).ok, false);

  const slashInf = makeValidV2Report({
    gpus: [
      {
        ...validV2.gpus[0],
        driver: {
          ...validV2.gpus[0].driver,
          inf_name: makeField('dir/oem2.inf', 'pnp'),
        },
      },
    ],
  });
  assert.equal(validateReport(slashInf).ok, false);

  const badLegacyErr = makeValidV2Report({
    legacy_status: {
      ...validV2.legacy_status,
      error_code: 'free text with spaces',
    },
  });
  assert.equal(validateReport(badLegacyErr).ok, false);

  const badLegacyCode = makeValidV2Report({
    legacy_status: {
      ...validV2.legacy_status,
      status_code: 'ARBITRARY_TEXT',
    },
  });
  assert.equal(validateReport(badLegacyCode).ok, false);
});

test('renderSummary_v2_includes_timestamps_security_extended_power_completeness_and_legacy', () => {
  const report = makeValidV2Report();
  const summary = renderSummary(report);
  assert.ok(summary.includes('22222222-3333-4444-8555-666666666666'));
  assert.ok(summary.includes('reported'));
  assert.ok(summary.includes('pnp_utc'));
  assert.ok(summary.includes('nvidia_smi_utc'));
  assert.ok(summary.includes('Secure Boot'));
  assert.ok(summary.includes('HVCI'));
  assert.ok(summary.includes('Test Signing'));
  assert.ok(summary.includes('oem2.inf'));
  assert.ok(summary.includes('enforced.power.limit'));
  assert.ok(summary.includes('140 W'));
  assert.ok(summary.includes('GEN2_SUCCESS'));
  assert.ok(summary.includes('link_load'));
});

test('renderSummary_v2_includes_4_layer_summary_experiment_window_profile_details_and_owner_checklist', () => {
  const cmpProfile = getGpuProfileDetails('10DE', '1F0B');
  const report = makeValidV2Report({
    previous_report_id: null,
    target_bdf: '0000:02:00.0',
    gpus: [
      makeGpu({
        bdf: makeField('0000:00:02.0', 'pnp'),
        vendor_id: makeField('1002', 'pnp'),
        device_id: makeField('1638', 'pnp'),
        name: makeField('AMD Radeon Graphics', 'cim'),
        pcie: {
          max_gen: makeField(3, 'pnp', 'gen'),
          current_gen: makeField(3, 'pnp', 'gen'),
          max_width: makeField(16, 'pnp', 'lanes'),
          current_width: makeField(16, 'pnp', 'lanes'),
        },
        profile: getGpuProfileDetails('1002', '1638', 'AMD Radeon Graphics'),
        driver: {
          provider: makeField('AMD', 'pnp'),
          date: makeField('2026-09-01', 'pnp'),
          inf_name: makeField('oem1.inf', 'pnp'),
          service: makeField('amdkmdag', 'pnp'),
          problem_code: makeField(0, 'pnp'),
          location_path: makeField('PCIROOT(0)#PCI(0801)#PCI(0000)', 'pnp'),
        },
      }),
      makeGpu({
        bdf: makeField('0000:02:00.0', 'pnp'),
        vendor_id: makeField('10DE', 'pnp'),
        device_id: makeField('1F0B', 'pnp'),
        name: makeField('NVIDIA CMP 40HX', 'cim'),
        pcie: {
          max_gen: makeField(3, 'pnp', 'gen'),
          current_gen: makeField(2, 'pnp', 'gen'),
          max_width: makeField(16, 'pnp', 'lanes'),
          current_width: makeField(16, 'pnp', 'lanes'),
        },
        profile: cmpProfile,
        driver: {
          provider: makeField('NVIDIA', 'pnp'),
          date: makeField('2026-10-19', 'pnp'),
          inf_name: makeField('oem2.inf', 'pnp'),
          service: makeField('nvlddmkm', 'pnp'),
          problem_code: makeField(0, 'pnp'),
          location_path: makeField('PCIROOT(0)#PCI(0101)#PCI(0000)', 'pnp'),
        },
      }),
    ],
    nvidia: {
      ...makeNvidiaBlock({
        overrides: {
          name: makeField('NVIDIA CMP 40HX', 'nvidia-smi'),
          'pci.bus_id': makeField('0000:02:00.0', 'nvidia-smi'),
          'pci.device_id': makeField('0x1F0B10DE', 'nvidia-smi'),
        },
      }),
      ...makeNvidiaExtendedBlock(),
    },
  });

  const summary = renderSummary(report);
  assert.ok(summary.includes(report.experiment_id));
  assert.ok(summary.includes('báo cáo đầu tiên trong chuỗi thử nghiệm'));
  assert.ok(summary.includes('2026-11-21T07:00:00Z'));
  assert.ok(summary.includes('2026-11-21T07:00:03Z'));
  assert.ok(summary.includes('Tóm tắt 4 lớp kết luận tách biệt'));
  for (const layerNum of ['1. Nhận diện', '2. Cấu hình', '3. Trạng thái đo', '4. Workload']) {
    const row = summary.split('\n').find((l) => l.includes(layerNum));
    assert.ok(row, `Missing row for ${layerNum}`);
    assert.equal(row.split('|').length - 1, 3, `Row must keep 2 GFM columns (3 pipes): ${row}`);
  }
  assert.ok(summary.includes('TU106'));
  assert.ok(summary.includes('cmp'));
  assert.ok(summary.includes('Gen2'));
  assert.ok(summary.includes('unverified'));
  assert.ok(summary.includes('Gen2 ×16 / Gen3 ×16'));
  assert.ok(summary.includes('Tối đa (nvidia-smi):'));
  assert.ok(summary.includes('Hiện tại (nvidia-smi):'));
  assert.ok(summary.includes('16/18'));
  assert.ok(summary.includes('temperature.memory'));
  assert.ok(summary.includes('ecc.mode.current'));
  assert.ok(summary.includes('Tóm tắt Công suất'));
  assert.ok(summary.includes('Tóm tắt Xung, Nhiệt & VRAM'));
  assert.ok(summary.includes('2189'));
  assert.ok(summary.includes('512'));
  assert.ok(summary.includes('modding-data-plan §7 & §9'));
  assert.ok(summary.includes('<details>'));
  assert.equal(summary.includes('Lưu ý tái lập phép thử'), false);
  assert.equal(summary.includes('Cảnh báo:'), false);

  const reportNoLegacyAndUnknownCtx = makeValidV2Report({
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
      mod_state: 'unknown',
      mod_type: 'unknown',
      riser: 'unknown',
      slot: 'unknown',
      stage: 'baseline',
      notes: '',
    },
  });
  const summary2 = renderSummary(reportNoLegacyAndUnknownCtx);
  assert.ok(summary2.includes('present=false'));
  assert.ok(summary2.includes('Lưu ý tái lập phép thử'));
  assert.equal(summary2.includes('Cảnh báo:'), false);
});
