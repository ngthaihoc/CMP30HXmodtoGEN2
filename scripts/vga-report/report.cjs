'use strict';

const SCHEMA_VERSION = 2;
const SUPPORTED_SCHEMA_VERSIONS = new Set([1, 2]);
const MAX_PAYLOAD_BYTES = 2 * 1024 * 1024;
const MAX_GPUS = 16;
const MAX_STRING_LENGTH = 500;
const MAX_SUMMARY_LENGTH = 60000;

const VALID_STATUSES = new Set([
  'ok',
  'unsupported',
  'permission_denied',
  'timeout',
  'error',
  'skipped',
]);

const MOD_STATES = new Set(['stock', 'modded', 'unknown']);
const MOD_TYPE_STATES = new Set(['none', 'driver', 'efi', 'vbios', 'strap', 'software_aio', 'multi', 'unknown']);
const RISER_STATES = new Set(['direct', 'riser', 'unknown']);
const SLOT_STATES = new Set(['cpu_x16', 'chipset', 'other', 'unknown']);
const STAGE_STATES = new Set(['before_unlock', 'after_unlock', 'troubleshooting', 'baseline', 'unknown']);
const EVIDENCE_LEVELS = new Set(['reported', 'observed', 'compared', 'owner_verified']);
const COMPLETENESS_GROUPS = Object.freeze([
  'inventory',
  'topology',
  'link_idle',
  'link_load',
  'compute',
  'stability',
  'comparison',
]);
const COMPLETENESS_STATUSES = new Set(['ok', 'partial', 'skipped', 'unsupported', 'error']);
const SOURCE_TIMESTAMP_KEYS = Object.freeze(['pnp_utc', 'cim_utc', 'nvidia_smi_utc', 'security_utc']);
const SECURITY_KEYS = Object.freeze(['boot_mode', 'secure_boot', 'hvci_running', 'test_signing']);
const LEGACY_STATUS_KEYS = Object.freeze([
  'present',
  'source',
  'status_code',
  'error_code',
  'speed_current',
  'width_current',
  'tls_target',
  'device_id',
  'mrrs_bytes',
  'observed_utc',
]);
const LEGACY_SOURCES = new Set(['gen2_status', 'batch_status', 'none']);
const LEGACY_STATUS_CODES = new Set([
  'GEN2_SUCCESS',
  'GEN2_SKIPPED',
  'GEN1_STUCK',
  'GEN2_HARDWARE_LIMIT',
  'DRV_FAIL',
  'NO_GPU',
  'UNKNOWN',
  'NONE',
]);
const LEGACY_ERROR_CODE_RE = /^[A-Z0-9_]{1,64}$/;
const HEX_DEVICE_ID_RE = /^[0-9A-Fa-f]{4}$/;
const GPU_CATEGORIES = new Set(['cmp', 'mining', 'datacenter', 'consumer', 'unknown']);
const TENSOR_APPLICABLE_STATES = new Set(['yes', 'no', 'unverified', 'unknown']);
const PROFILE_KEYS = Object.freeze([
  'id',
  'name',
  'family',
  'category',
  'target_pcie_gen',
  'tensor_applicable',
  'identity_note',
]);
const DRIVER_KEYS = Object.freeze(['provider', 'date', 'inf_name', 'service', 'problem_code', 'location_path']);

const KNOWN_GPU_PROFILE_DETAILS = Object.freeze({
  '10DE:1F0B': Object.freeze({
    id: '10DE:1F0B',
    name: 'CMP 40HX (TU106)',
    family: 'TU106',
    category: 'cmp',
    target_pcie_gen: 2,
    tensor_applicable: 'unverified',
    identity_note: null,
  }),
  '10DE:2189': Object.freeze({
    id: '10DE:2189',
    name: 'CMP 30HX (TU116)',
    family: 'TU116',
    category: 'cmp',
    target_pcie_gen: 2,
    tensor_applicable: 'no',
    identity_note: null,
  }),
  '10DE:1C07': Object.freeze({
    id: '10DE:1C07',
    name: 'P106-100 (GP106)',
    family: 'GP106',
    category: 'mining',
    target_pcie_gen: null,
    tensor_applicable: 'no',
    identity_note: null,
  }),
  '10DE:1C09': Object.freeze({
    id: '10DE:1C09',
    name: 'P106-090 (GP106)',
    family: 'GP106',
    category: 'mining',
    target_pcie_gen: null,
    tensor_applicable: 'no',
    identity_note: null,
  }),
  '10DE:15F8': Object.freeze({
    id: '10DE:15F8',
    name: 'Tesla P100 PCIe 16GB (GP100)',
    family: 'GP100',
    category: 'datacenter',
    target_pcie_gen: 3,
    tensor_applicable: 'no',
    identity_note: 'Xác minh SUBSYS/dạng card trước khi áp định mức 250W',
  }),
  '10DE:15F7': Object.freeze({
    id: '10DE:15F7',
    name: 'Tesla P100 PCIe 12GB (GP100)',
    family: 'GP100',
    category: 'datacenter',
    target_pcie_gen: 3,
    tensor_applicable: 'no',
    identity_note: 'Xác minh SUBSYS/dạng card trước khi áp định mức 250W',
  }),
  '10DE:15F9': Object.freeze({
    id: '10DE:15F9',
    name: 'Tesla P100 SXM2 (GP100)',
    family: 'GP100',
    category: 'datacenter',
    target_pcie_gen: 3,
    tensor_applicable: 'no',
    identity_note: 'SXM2 dùng baseline công suất/kết nối riêng',
  }),
});

const KNOWN_GPU_PROFILES = Object.freeze(
  Object.fromEntries(
    Object.entries(KNOWN_GPU_PROFILE_DETAILS).map(([k, v]) => [k, v.name]),
  ),
);

function getGpuProfileDetails(vendorId, deviceId, reportedName = null) {
  const vid = typeof vendorId === 'string' ? vendorId.trim().toUpperCase() : '';
  const did = typeof deviceId === 'string' ? deviceId.trim().toUpperCase() : '';
  if (vid && did) {
    const key = `${vid}:${did}`;
    if (Object.prototype.hasOwnProperty.call(KNOWN_GPU_PROFILE_DETAILS, key)) {
      return { ...KNOWN_GPU_PROFILE_DETAILS[key] };
    }
  }
  const identityNote = typeof reportedName === 'string' && /\bP100\b/i.test(reportedName)
    ? 'Tên báo P100 nhưng PCI ID chưa khớp Tesla GP100; cần xác minh trước khi áp thông số Tesla'
    : null;
  return {
    id: null,
    name: null,
    family: null,
    category: 'unknown',
    target_pcie_gen: null,
    tensor_applicable: 'unknown',
    identity_note: identityNote,
  };
}

function identifyGpuProfile(vendorId, deviceId, reportedName = null) {
  return getGpuProfileDetails(vendorId, deviceId, reportedName).name;
}

function normalizeBdf(busId) {
  if (typeof busId !== 'string') {
    return null;
  }
  const t = busId.trim().toLowerCase();
  if (!t) {
    return null;
  }
  let match = t.match(/^0000([0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7])$/);
  if (match) {
    return match[1];
  }
  match = t.match(/^([0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7])$/);
  if (match) {
    return match[1];
  }
  match = t.match(/^([0-9a-f]{2}:[0-9a-f]{2}\.[0-7])$/);
  if (match) {
    return `0000:${match[1]}`;
  }
  return t;
}

const NVIDIA_REQUIRED_FIELDS = Object.freeze([
  'name',
  'driver_version',
  'vbios_version',
  'pci.bus_id',
  'pci.device_id',
  'pci.sub_device_id',
  'memory.total',
  'compute_cap',
  'pcie.link.gen.max',
  'pcie.link.gen.gpucurrent',
  'pcie.link.width.max',
  'pcie.link.width.current',
  'temperature.gpu',
  'power.draw',
  'clocks.sm',
  'clocks.mem',
  'clocks.max.sm',
  'pstate',
]);

const NVIDIA_EXTENDED_FIELDS = Object.freeze([
  'power.limit',
  'enforced.power.limit',
  'power.default_limit',
  'power.min_limit',
  'power.max_limit',
  'temperature.gpu.tlimit',
  'temperature.memory',
  'fan.speed',
  'utilization.gpu',
  'utilization.memory',
  'memory.used',
  'memory.free',
  'ecc.mode.current',
  'clocks_throttle_reasons.active',
  'clocks_throttle_reasons.sw_power_cap',
  'clocks_throttle_reasons.hw_slowdown',
  'clocks_throttle_reasons.hw_thermal_slowdown',
  'clocks_throttle_reasons.hw_power_brake_slowdown',
]);

const UUID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
const ISO_UTC_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/;

const ROOT_KEYS = [
  'schema_version',
  'collector_version',
  'report_id',
  'created_utc',
  'experiment_id',
  'consent',
  'gpus',
  'nvidia',
  'pcie',
  'platform',
  'context',
];

const ROOT_KEYS_V2 = [
  ...ROOT_KEYS,
  'previous_report_id',
  'target_bdf',
  'collection_started_utc',
  'collection_finished_utc',
  'source_timestamps',
  'evidence_level',
  'security',
  'completeness',
  'legacy_status',
];

const FIELD_KEYS = ['value', 'source', 'unit', 'status'];
const PCIE_KEYS = ['max_gen', 'current_gen', 'max_width', 'current_width'];
const PLATFORM_KEYS = ['cpu', 'mainboard_model', 'bios_version', 'os_build'];
const CONTEXT_KEYS_V1 = ['mod_state', 'riser', 'slot', 'stage', 'notes'];
const CONTEXT_ALLOWED_V2 = new Set([...CONTEXT_KEYS_V1, 'mod_type']);
const GPU_REQUIRED_KEYS = [
  'bdf',
  'vendor_id',
  'device_id',
  'subsystem_id',
  'revision',
  'name',
  'pnp_status',
  'driver_version',
];
const GPU_ALLOWED_KEYS = new Set([...GPU_REQUIRED_KEYS, 'pcie', 'nvidia']);
const GPU_ALLOWED_KEYS_V2 = new Set([...GPU_ALLOWED_KEYS, 'profile', 'driver']);
const NVIDIA_KEYS = ['complete', 'missing', 'fields'];
const NVIDIA_ALLOWED_KEYS_V2 = new Set([...NVIDIA_KEYS, 'extended', 'extended_missing']);

const TEXT_ENCODER = new TextEncoder();

function isPlainObject(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function checkKeys(obj, path, requiredKeys, allowedSet, errors) {
  const keys = Object.keys(obj);
  for (const key of requiredKeys) {
    if (!Object.prototype.hasOwnProperty.call(obj, key)) {
      errors.push(`${path}: missing required property "${key}"`);
    }
  }
  for (const key of keys) {
    if (!allowedSet.has(key)) {
      errors.push(`${path}: unexpected property "${key}"`);
    }
  }
}

function validateField(field, path, errors) {
  if (!isPlainObject(field)) {
    errors.push(`${path}: expected field object`);
    return;
  }
  checkKeys(field, path, FIELD_KEYS, new Set(FIELD_KEYS), errors);

  if (typeof field.source !== 'string' || field.source.length < 1 || field.source.length > 100) {
    errors.push(`${path}.source: must be a non-empty string (<= 100 chars)`);
  }
  if (field.unit !== null && (typeof field.unit !== 'string' || field.unit.length > 50)) {
    errors.push(`${path}.unit: must be null or string (<= 50 chars)`);
  }
  if (typeof field.status !== 'string' || !VALID_STATUSES.has(field.status)) {
    errors.push(`${path}.status: invalid status "${String(field.status)}"`);
    return;
  }

  const val = field.value;
  if (field.status === 'ok') {
    if (val === null || val === undefined) {
      errors.push(`${path}.value: must not be null when status is "ok"`);
    } else if (typeof val === 'string') {
      if (val.length > MAX_STRING_LENGTH) {
        errors.push(`${path}.value: string exceeds ${MAX_STRING_LENGTH} chars`);
      } else {
        const trimmed = val.trim();
        if (trimmed === '[N/A]' || trimmed === '[Not Supported]' || trimmed === 'N/A') {
          errors.push(`${path}.value: sentinel "${trimmed}" must use status "unsupported"`);
        }
      }
    } else if (typeof val === 'number') {
      if (!Number.isFinite(val)) {
        errors.push(`${path}.value: number must be finite`);
      }
    } else if (typeof val !== 'boolean') {
      errors.push(`${path}.value: invalid value type "${typeof val}"`);
    }
  } else if (val !== null) {
    errors.push(`${path}.value: must be null when status is "${field.status}" (do not substitute 0)`);
  }
}

function validatePcie(pcie, path, errors) {
  if (!isPlainObject(pcie)) {
    errors.push(`${path}: expected pcie object`);
    return;
  }
  checkKeys(pcie, path, PCIE_KEYS, new Set(PCIE_KEYS), errors);
  for (const key of PCIE_KEYS) {
    if (Object.prototype.hasOwnProperty.call(pcie, key)) {
      validateField(pcie[key], `${path}.${key}`, errors);
    }
  }
}

function isValidIsoUtc(value) {
  return typeof value === 'string' && ISO_UTC_RE.test(value) && !Number.isNaN(Date.parse(value));
}

function validateProfile(profile, path, errors) {
  if (!isPlainObject(profile)) {
    errors.push(`${path}: expected profile object`);
    return;
  }
  checkKeys(profile, path, PROFILE_KEYS, new Set(PROFILE_KEYS), errors);
  if (profile.id !== null && (typeof profile.id !== 'string' || profile.id.length > 32)) {
    errors.push(`${path}.id: must be null or string (<= 32 chars)`);
  }
  if (profile.name !== null && (typeof profile.name !== 'string' || profile.name.length > 100)) {
    errors.push(`${path}.name: must be null or string (<= 100 chars)`);
  }
  if (profile.family !== null && (typeof profile.family !== 'string' || profile.family.length > 32)) {
    errors.push(`${path}.family: must be null or string (<= 32 chars)`);
  }
  if (typeof profile.category !== 'string' || !GPU_CATEGORIES.has(profile.category)) {
    errors.push(`${path}.category: invalid category`);
  }
  if (
    profile.target_pcie_gen !== null
    && (!Number.isInteger(profile.target_pcie_gen) || profile.target_pcie_gen < 1 || profile.target_pcie_gen > 6)
  ) {
    errors.push(`${path}.target_pcie_gen: must be null or integer 1..6`);
  }
  if (typeof profile.tensor_applicable !== 'string' || !TENSOR_APPLICABLE_STATES.has(profile.tensor_applicable)) {
    errors.push(`${path}.tensor_applicable: invalid value`);
  }
  if (profile.identity_note !== null && (typeof profile.identity_note !== 'string' || profile.identity_note.length > 300)) {
    errors.push(`${path}.identity_note: must be null or string (<= 300 chars)`);
  }
}

function validateDriver(driver, path, errors) {
  if (!isPlainObject(driver)) {
    errors.push(`${path}: expected driver object`);
    return;
  }
  checkKeys(driver, path, DRIVER_KEYS, new Set(DRIVER_KEYS), errors);
  for (const key of DRIVER_KEYS) {
    if (Object.prototype.hasOwnProperty.call(driver, key)) {
      validateField(driver[key], `${path}.${key}`, errors);
    }
  }
  if (isPlainObject(driver.inf_name) && driver.inf_name.status === 'ok') {
    if (typeof driver.inf_name.value !== 'string' || /[\\/]/.test(driver.inf_name.value)) {
      errors.push(`${path}.inf_name.value: must be filename only without path separators`);
    }
  }
}

function validateSecurity(security, path, errors) {
  if (!isPlainObject(security)) {
    errors.push(`${path}: expected security object`);
    return;
  }
  checkKeys(security, path, SECURITY_KEYS, new Set(SECURITY_KEYS), errors);
  for (const key of SECURITY_KEYS) {
    if (Object.prototype.hasOwnProperty.call(security, key)) {
      validateField(security[key], `${path}.${key}`, errors);
    }
  }
}

function validateSourceTimestamps(tsObj, path, errors) {
  if (!isPlainObject(tsObj)) {
    errors.push(`${path}: expected object`);
    return;
  }
  checkKeys(tsObj, path, SOURCE_TIMESTAMP_KEYS, new Set(SOURCE_TIMESTAMP_KEYS), errors);
  for (const key of SOURCE_TIMESTAMP_KEYS) {
    if (Object.prototype.hasOwnProperty.call(tsObj, key)) {
      const v = tsObj[key];
      if (v !== null && !isValidIsoUtc(v)) {
        errors.push(`${path}.${key}: must be null or valid ISO-8601 UTC timestamp`);
      }
    }
  }
}

function validateCompleteness(comp, path, errors) {
  if (!isPlainObject(comp)) {
    errors.push(`${path}: expected object`);
    return;
  }
  checkKeys(comp, path, COMPLETENESS_GROUPS, new Set(COMPLETENESS_GROUPS), errors);
  const groupKeys = ['complete', 'status', 'missing', 'reason'];
  const groupKeySet = new Set(groupKeys);
  for (const groupName of COMPLETENESS_GROUPS) {
    if (!Object.prototype.hasOwnProperty.call(comp, groupName)) continue;
    const g = comp[groupName];
    const gPath = `${path}.${groupName}`;
    if (!isPlainObject(g)) {
      errors.push(`${gPath}: expected object`);
      continue;
    }
    checkKeys(g, gPath, groupKeys, groupKeySet, errors);
    if (typeof g.complete !== 'boolean') {
      errors.push(`${gPath}.complete: must be boolean`);
    }
    if (typeof g.status !== 'string' || !COMPLETENESS_STATUSES.has(g.status)) {
      errors.push(`${gPath}.status: invalid status`);
    }
    if (!Array.isArray(g.missing) || g.missing.length > 50) {
      errors.push(`${gPath}.missing: must be array of at most 50 items`);
    } else {
      for (let i = 0; i < g.missing.length; i += 1) {
        const m = g.missing[i];
        if (typeof m !== 'string' || m.length < 1 || m.length > 100) {
          errors.push(`${gPath}.missing[${i}]: must be non-empty string (<= 100 chars)`);
        }
      }
    }
    if (typeof g.reason !== 'string' || g.reason.length > 200) {
      errors.push(`${gPath}.reason: must be string (<= 200 chars)`);
    }
  }
}

function validateLegacyStatus(legacy, path, errors) {
  if (!isPlainObject(legacy)) {
    errors.push(`${path}: expected object`);
    return;
  }
  checkKeys(legacy, path, LEGACY_STATUS_KEYS, new Set(LEGACY_STATUS_KEYS), errors);
  if (typeof legacy.present !== 'boolean') {
    errors.push(`${path}.present: must be boolean`);
  }
  if (typeof legacy.source !== 'string' || !LEGACY_SOURCES.has(legacy.source)) {
    errors.push(`${path}.source: invalid source`);
  }
  if (typeof legacy.status_code !== 'string' || !LEGACY_STATUS_CODES.has(legacy.status_code)) {
    errors.push(`${path}.status_code: invalid status_code`);
  }
  if (typeof legacy.error_code !== 'string' || !LEGACY_ERROR_CODE_RE.test(legacy.error_code)) {
    errors.push(`${path}.error_code: must match allowlist ${LEGACY_ERROR_CODE_RE}`);
  }
  for (const intKey of ['speed_current', 'width_current', 'tls_target', 'mrrs_bytes']) {
    if (Object.prototype.hasOwnProperty.call(legacy, intKey)) {
      const iv = legacy[intKey];
      if (iv !== null && (!Number.isInteger(iv) || iv < 0)) {
        errors.push(`${path}.${intKey}: must be null or non-negative integer`);
      }
    }
  }
  if (legacy.device_id !== null && (typeof legacy.device_id !== 'string' || !HEX_DEVICE_ID_RE.test(legacy.device_id))) {
    errors.push(`${path}.device_id: must be null or 4-char hex string`);
  }
  if (legacy.observed_utc !== null && !isValidIsoUtc(legacy.observed_utc)) {
    errors.push(`${path}.observed_utc: must be null or valid ISO-8601 UTC timestamp`);
  }
}

function validateNvidia(nvidia, path, errors, isV2 = false) {
  if (!isPlainObject(nvidia)) {
    errors.push(`${path}: expected nvidia object`);
    return;
  }
  const allowedNvidiaKeys = isV2 ? NVIDIA_ALLOWED_KEYS_V2 : new Set(NVIDIA_KEYS);
  checkKeys(nvidia, path, NVIDIA_KEYS, allowedNvidiaKeys, errors);

  if (typeof nvidia.complete !== 'boolean') {
    errors.push(`${path}.complete: must be boolean`);
  }

  const seenMissing = new Set();
  if (!Array.isArray(nvidia.missing) || nvidia.missing.length > NVIDIA_REQUIRED_FIELDS.length) {
    errors.push(`${path}.missing: must be array of at most ${NVIDIA_REQUIRED_FIELDS.length} items`);
  } else {
    for (let i = 0; i < nvidia.missing.length; i += 1) {
      const item = nvidia.missing[i];
      if (typeof item !== 'string' || item.length < 1 || item.length > 100) {
        errors.push(`${path}.missing[${i}]: must be a non-empty string (<= 100 chars)`);
      } else if (seenMissing.has(item)) {
        errors.push(`${path}.missing[${i}]: duplicate item "${item}"`);
      } else {
        seenMissing.add(item);
      }
    }
  }

  if (!isPlainObject(nvidia.fields)) {
    errors.push(`${path}.fields: expected object`);
    return;
  }
  checkKeys(
    nvidia.fields,
    `${path}.fields`,
    NVIDIA_REQUIRED_FIELDS,
    new Set(NVIDIA_REQUIRED_FIELDS),
    errors,
  );

  let nonOkCount = 0;
  for (const key of NVIDIA_REQUIRED_FIELDS) {
    if (Object.prototype.hasOwnProperty.call(nvidia.fields, key)) {
      const f = nvidia.fields[key];
      validateField(f, `${path}.fields.${key}`, errors);
      if (isPlainObject(f) && f.status !== 'ok') {
        nonOkCount += 1;
        if (!seenMissing.has(key)) {
          errors.push(`${path}.missing: missing field "${key}" (status="${f.status}") must be listed`);
        }
      }
    }
  }

  if (typeof nvidia.complete === 'boolean') {
    const expectedComplete = nonOkCount === 0 && seenMissing.size === 0;
    if (nvidia.complete !== expectedComplete) {
      errors.push(`${path}.complete: expected ${expectedComplete} based on fields/missing`);
    }
  }

  const hasExtended = Object.prototype.hasOwnProperty.call(nvidia, 'extended');
  const hasExtendedMissing = Object.prototype.hasOwnProperty.call(nvidia, 'extended_missing');
  if (hasExtendedMissing && !hasExtended) {
    errors.push(`${path}.extended_missing: unexpected without extended`);
  }
  if (hasExtended) {
    if (!isPlainObject(nvidia.extended)) {
      errors.push(`${path}.extended: expected object`);
    } else {
      checkKeys(
        nvidia.extended,
        `${path}.extended`,
        NVIDIA_EXTENDED_FIELDS,
        new Set(NVIDIA_EXTENDED_FIELDS),
        errors,
      );
      const seenExtMissing = new Set();
      if (!hasExtendedMissing || !Array.isArray(nvidia.extended_missing) || nvidia.extended_missing.length > NVIDIA_EXTENDED_FIELDS.length) {
        errors.push(`${path}.extended_missing: must be array of at most ${NVIDIA_EXTENDED_FIELDS.length} items`);
      } else {
        for (let i = 0; i < nvidia.extended_missing.length; i += 1) {
          const item = nvidia.extended_missing[i];
          if (typeof item !== 'string' || item.length < 1 || item.length > 100) {
            errors.push(`${path}.extended_missing[${i}]: must be non-empty string (<= 100 chars)`);
          } else if (seenExtMissing.has(item)) {
            errors.push(`${path}.extended_missing[${i}]: duplicate item "${item}"`);
          } else {
            seenExtMissing.add(item);
          }
        }
      }
      let extNonOkCount = 0;
      for (const key of NVIDIA_EXTENDED_FIELDS) {
        if (Object.prototype.hasOwnProperty.call(nvidia.extended, key)) {
          const f = nvidia.extended[key];
          validateField(f, `${path}.extended.${key}`, errors);
          if (isPlainObject(f) && f.status !== 'ok') {
            extNonOkCount += 1;
            if (!seenExtMissing.has(key)) {
              errors.push(`${path}.extended_missing: missing field "${key}" (status="${f.status}") must be listed`);
            }
          }
        }
      }
      if (hasExtendedMissing && Array.isArray(nvidia.extended_missing) && seenExtMissing.size !== extNonOkCount) {
        errors.push(`${path}.extended_missing: count does not match non-ok extended fields`);
      }
    }
  }
}

function validateReport(report) {
  const errors = [];
  if (!isPlainObject(report)) {
    return { ok: false, errors: ['root: report must be a JSON object'] };
  }

  try {
    const byteLength = TEXT_ENCODER.encode(JSON.stringify(report)).length;
    if (byteLength > MAX_PAYLOAD_BYTES) {
      errors.push(`root: payload exceeds ${MAX_PAYLOAD_BYTES} bytes (${byteLength})`);
    }
  } catch (err) {
    errors.push(`root: report is not JSON-serializable (${err && err.message ? err.message : 'error'})`);
    return { ok: false, errors };
  }

  if (!SUPPORTED_SCHEMA_VERSIONS.has(report.schema_version)) {
    errors.push('root.schema_version: must equal 1 or 2');
  }
  const isV2 = report.schema_version === 2;
  const expectedRootKeys = isV2 ? ROOT_KEYS_V2 : ROOT_KEYS;
  checkKeys(report, 'root', expectedRootKeys, new Set(expectedRootKeys), errors);

  if (typeof report.collector_version !== 'string' || report.collector_version.length < 1 || report.collector_version.length > 50) {
    errors.push('root.collector_version: must be a non-empty string (<= 50 chars)');
  }
  if (typeof report.report_id !== 'string' || !UUID_RE.test(report.report_id)) {
    errors.push('root.report_id: must be a valid UUID');
  }
  if (!isValidIsoUtc(report.created_utc)) {
    errors.push('root.created_utc: must be a valid ISO-8601 UTC timestamp');
  }
  if (typeof report.experiment_id !== 'string' || !UUID_RE.test(report.experiment_id)) {
    errors.push('root.experiment_id: must be a valid UUID');
  }
  if (report.consent !== true) {
    errors.push('root.consent: must be true');
  }

  if (isV2) {
    if (report.previous_report_id !== null && (typeof report.previous_report_id !== 'string' || !UUID_RE.test(report.previous_report_id))) {
      errors.push('root.previous_report_id: must be null or a valid UUID');
    }
    if (report.target_bdf !== null && (typeof report.target_bdf !== 'string' || report.target_bdf.length < 1 || report.target_bdf.length > 32)) {
      errors.push('root.target_bdf: must be null or a non-empty string (<= 32 chars)');
    }
    if (Object.prototype.hasOwnProperty.call(report, 'collection_started_utc') && !isValidIsoUtc(report.collection_started_utc)) {
      errors.push('root.collection_started_utc: must be a valid ISO-8601 UTC timestamp');
    }
    if (Object.prototype.hasOwnProperty.call(report, 'collection_finished_utc') && !isValidIsoUtc(report.collection_finished_utc)) {
      errors.push('root.collection_finished_utc: must be a valid ISO-8601 UTC timestamp');
    }
    if (Object.prototype.hasOwnProperty.call(report, 'source_timestamps')) {
      validateSourceTimestamps(report.source_timestamps, 'root.source_timestamps', errors);
    }
    if (typeof report.evidence_level !== 'string' || !EVIDENCE_LEVELS.has(report.evidence_level)) {
      errors.push('root.evidence_level: invalid value');
    }
    if (Object.prototype.hasOwnProperty.call(report, 'security')) {
      validateSecurity(report.security, 'root.security', errors);
    }
    if (Object.prototype.hasOwnProperty.call(report, 'completeness')) {
      validateCompleteness(report.completeness, 'root.completeness', errors);
    }
    if (Object.prototype.hasOwnProperty.call(report, 'legacy_status')) {
      validateLegacyStatus(report.legacy_status, 'root.legacy_status', errors);
    }
  }

  if (!Array.isArray(report.gpus)) {
    errors.push('root.gpus: must be an array');
  } else {
    if (report.gpus.length < 1) {
      errors.push('root.gpus: must contain at least 1 GPU');
    }
    if (report.gpus.length > MAX_GPUS) {
      errors.push(`root.gpus: must contain at most ${MAX_GPUS} GPUs`);
    }
    const allowedGpuKeys = isV2 ? GPU_ALLOWED_KEYS_V2 : GPU_ALLOWED_KEYS;
    for (let i = 0; i < report.gpus.length; i += 1) {
      const gpu = report.gpus[i];
      const gpuPath = `root.gpus[${i}]`;
      if (!isPlainObject(gpu)) {
        errors.push(`${gpuPath}: expected object`);
        continue;
      }
      checkKeys(gpu, gpuPath, GPU_REQUIRED_KEYS, allowedGpuKeys, errors);
      for (const key of GPU_REQUIRED_KEYS) {
        if (Object.prototype.hasOwnProperty.call(gpu, key)) {
          validateField(gpu[key], `${gpuPath}.${key}`, errors);
        }
      }
      if (Object.prototype.hasOwnProperty.call(gpu, 'pcie')) {
        validatePcie(gpu.pcie, `${gpuPath}.pcie`, errors);
      }
      if (Object.prototype.hasOwnProperty.call(gpu, 'nvidia')) {
        validateNvidia(gpu.nvidia, `${gpuPath}.nvidia`, errors, isV2);
      }
      if (isV2 && Object.prototype.hasOwnProperty.call(gpu, 'profile')) {
        validateProfile(gpu.profile, `${gpuPath}.profile`, errors);
      }
      if (isV2 && Object.prototype.hasOwnProperty.call(gpu, 'driver')) {
        validateDriver(gpu.driver, `${gpuPath}.driver`, errors);
      }
    }
  }

  if (Object.prototype.hasOwnProperty.call(report, 'nvidia')) {
    validateNvidia(report.nvidia, 'root.nvidia', errors, isV2);
  }
  if (Object.prototype.hasOwnProperty.call(report, 'pcie')) {
    validatePcie(report.pcie, 'root.pcie', errors);
  }

  if (!isPlainObject(report.platform)) {
    errors.push('root.platform: expected object');
  } else {
    checkKeys(report.platform, 'root.platform', PLATFORM_KEYS, new Set(PLATFORM_KEYS), errors);
    for (const key of PLATFORM_KEYS) {
      if (Object.prototype.hasOwnProperty.call(report.platform, key)) {
        validateField(report.platform[key], `root.platform.${key}`, errors);
      }
    }
  }

  if (!isPlainObject(report.context)) {
    errors.push('root.context: expected object');
  } else {
    const allowedCtx = isV2 ? CONTEXT_ALLOWED_V2 : new Set(CONTEXT_KEYS_V1);
    checkKeys(report.context, 'root.context', CONTEXT_KEYS_V1, allowedCtx, errors);
    if (isV2 && Object.prototype.hasOwnProperty.call(report.context, 'mod_type')) {
      if (typeof report.context.mod_type !== 'string' || !MOD_TYPE_STATES.has(report.context.mod_type)) {
        errors.push('root.context.mod_type: invalid value');
      }
    }
    if (typeof report.context.mod_state !== 'string' || !MOD_STATES.has(report.context.mod_state)) {
      errors.push('root.context.mod_state: invalid value');
    }
    if (typeof report.context.riser !== 'string' || !RISER_STATES.has(report.context.riser)) {
      errors.push('root.context.riser: invalid value');
    }
    if (typeof report.context.slot !== 'string' || !SLOT_STATES.has(report.context.slot)) {
      errors.push('root.context.slot: invalid value');
    }
    if (typeof report.context.stage !== 'string' || !STAGE_STATES.has(report.context.stage)) {
      errors.push('root.context.stage: invalid value');
    }
    if (typeof report.context.notes !== 'string' || report.context.notes.length > MAX_STRING_LENGTH) {
      errors.push(`root.context.notes: must be a string (<= ${MAX_STRING_LENGTH} chars)`);
    }
  }

  return { ok: errors.length === 0, errors };
}

const PII_PATTERNS = [
  {
    name: 'path_user',
    regex: /(?:[A-Za-z]:\\Users\\[^\\\s]+|\\\\[^\\\s]+\\[^\\\s]+|\/(?:home|Users)\/[^/\s]+|%USERPROFILE%)/i,
  },
  {
    name: 'mac',
    regex: /(?<![0-9a-fA-F:-])(?:[0-9a-fA-F]{2}[:-]){5}[0-9a-fA-F]{2}(?![0-9a-fA-F:-])/,
  },
  {
    name: 'ipv4',
    regex: /(?<![.\dA-Za-z])(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]\d|\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]\d|\d)){3}(?![.\dA-Za-z])/,
  },
  {
    name: 'serial',
    regex: /\b(?:serial(?:\s*(?:number|no|#))?|s\/n|sn)\s*[:=]\s*[A-Za-z0-9_-]{4,}/i,
  },
  {
    name: 'uuid_gpu',
    regex: /\b(?:MIG-)?GPU-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b/i,
  },
];

function hasIpv6(text) {
  // Validate candidates with the platform URL parser; PCI BDFs are not IPs.
  const candidates = text.matchAll(/(?<![\w:.])(?:[a-fA-F0-9]*:){2,}[a-fA-F0-9:.]*(?:%[a-zA-Z0-9_.~-]+)?/g);
  for (const [candidate] of candidates) {
    const address = candidate.split('%')[0].replace(/\.+$/, '');
    try {
      new URL(`http://[${address}]/`);
      return true;
    } catch {
      // Not a valid IPv6 address (for example a PCI identifier).
    }
  }
  return false;
}

function scanPiiValue(value, path, found) {
  if (typeof value === 'string') {
    if (hasIpv6(value)) found.add('ipv6');
    for (const rule of PII_PATTERNS) {
      if (rule.regex.test(value)) {
        found.add(rule.name);
      }
    }
    return;
  }
  if (Array.isArray(value)) {
    for (let i = 0; i < value.length; i += 1) {
      scanPiiValue(value[i], `${path}[${i}]`, found);
    }
    return;
  }
  if (isPlainObject(value)) {
    for (const key of Object.keys(value)) {
      if (/serial/i.test(key) && value[key] !== null && value[key] !== '') {
        found.add('serial');
      }
      if (/gpu_?uuid/i.test(key) && value[key] !== null && value[key] !== '') {
        found.add('uuid_gpu');
      }
      scanPiiValue(value[key], `${path}.${key}`, found);
    }
  }
}

function findPii(report) {
  const found = new Set();
  scanPiiValue(report, 'root', found);
  return Array.from(found);
}

function escapeMarkdown(input) {
  const str = String(input === null || input === undefined ? '' : input)
    .replace(/\r?\n|\r/g, ' ')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/@/g, '&#64;')
    .replace(/\[/g, '&#91;')
    .replace(/\]/g, '&#93;')
    .replace(/\(/g, '&#40;')
    .replace(/\)/g, '&#41;')
    .replace(/`/g, '&#96;')
    .replace(/\|/g, '&#124;')
    .replace(/\$/g, '&#36;')
    .replace(/\{/g, '&#123;')
    .replace(/\}/g, '&#125;');
  return str;
}

function formatField(field) {
  if (!isPlainObject(field)) {
    return 'unsupported';
  }
  if (field.status !== 'ok') {
    return escapeMarkdown(field.status);
  }
  const unit = field.unit ? ` ${escapeMarkdown(field.unit)}` : '';
  return `${escapeMarkdown(field.value)}${unit}`;
}

function formatPcieLink(genField, widthField) {
  const genOk = isPlainObject(genField) && genField.status === 'ok' && genField.value !== null && genField.value !== undefined;
  const widthOk = isPlainObject(widthField) && widthField.status === 'ok' && widthField.value !== null && widthField.value !== undefined;
  const genText = genOk ? escapeMarkdown(genField.value) : '?';
  const widthText = widthOk ? escapeMarkdown(widthField.value) : '?';
  return `Gen${genText} ×${widthText}`;
}

function getPcieSourceWarnings(report) {
  const warnings = [];
  if (!isPlainObject(report) || !isPlainObject(report.pcie) || !isPlainObject(report.nvidia) || !isPlainObject(report.nvidia.fields)) {
    return warnings;
  }
  const pcie = report.pcie;
  const nvFields = report.nvidia.fields;

  const pnpCurGen = pcie.current_gen;
  const pnpCurW = pcie.current_width;
  const nvCurGen = nvFields['pcie.link.gen.gpucurrent'];
  const nvCurW = nvFields['pcie.link.width.current'];

  const curGenDiff = isPlainObject(pnpCurGen) && pnpCurGen.source === 'pnp' && pnpCurGen.status === 'ok'
    && isPlainObject(nvCurGen) && nvCurGen.status === 'ok'
    && String(pnpCurGen.value) !== String(nvCurGen.value);
  const curWDiff = isPlainObject(pnpCurW) && pnpCurW.source === 'pnp' && pnpCurW.status === 'ok'
    && isPlainObject(nvCurW) && nvCurW.status === 'ok'
    && String(pnpCurW.value) !== String(nvCurW.value);
  if (curGenDiff || curWDiff) {
    warnings.push(
      `Cảnh báo: PnP báo hiện tại ${formatPcieLink(pnpCurGen, pnpCurW)}, nvidia-smi báo ${formatPcieLink(nvCurGen, nvCurW)}. Hai nguồn đọc khác thời điểm; link có thể hạ tốc khi nhàn rỗi. Chưa kết luận nguồn nào sai.`,
    );
  }

  const pnpMaxGen = pcie.max_gen;
  const pnpMaxW = pcie.max_width;
  const nvMaxGen = nvFields['pcie.link.gen.max'];
  const nvMaxW = nvFields['pcie.link.width.max'];

  const maxGenDiff = isPlainObject(pnpMaxGen) && pnpMaxGen.source === 'pnp' && pnpMaxGen.status === 'ok'
    && isPlainObject(nvMaxGen) && nvMaxGen.status === 'ok'
    && String(pnpMaxGen.value) !== String(nvMaxGen.value);
  const maxWDiff = isPlainObject(pnpMaxW) && pnpMaxW.source === 'pnp' && pnpMaxW.status === 'ok'
    && isPlainObject(nvMaxW) && nvMaxW.status === 'ok'
    && String(pnpMaxW.value) !== String(nvMaxW.value);
  if (maxGenDiff || maxWDiff) {
    warnings.push(
      `Cảnh báo: PnP báo tối đa ${formatPcieLink(pnpMaxGen, pnpMaxW)}, nvidia-smi báo ${formatPcieLink(nvMaxGen, nvMaxW)}. Hai nguồn đọc khác thời điểm hoặc khác cách báo cáo khả năng khe/card. Chưa kết luận nguồn nào sai.`,
    );
  }

  return warnings;
}

function renderSummary(report) {
  if (!isPlainObject(report)) {
    return '';
  }

  const gpus = Array.isArray(report.gpus) ? report.gpus.slice(0, MAX_GPUS) : [];
  const nvidia = isPlainObject(report.nvidia) ? report.nvidia : {};
  const nvFields = isPlainObject(nvidia.fields) ? nvidia.fields : {};
  const nvExt = isPlainObject(nvidia.extended) ? nvidia.extended : null;
  const explicitTargetBdf = typeof report.target_bdf === 'string' ? normalizeBdf(report.target_bdf) : null;
  const nvBusField = nvFields['pci.bus_id'];
  const nvTargetBdf = isPlainObject(nvBusField) && nvBusField.status === 'ok'
    ? normalizeBdf(nvBusField.value)
    : null;
  const resolvedTargetBdf = explicitTargetBdf || nvTargetBdf;
  const targetGpu = resolvedTargetBdf
    ? gpus.find((gpu) => isPlainObject(gpu) && isPlainObject(gpu.bdf) && gpu.bdf.status === 'ok' && normalizeBdf(gpu.bdf.value) === resolvedTargetBdf)
    : null;
  const targetProfileDetails = targetGpu
    ? (isPlainObject(targetGpu.profile)
      ? targetGpu.profile
      : getGpuProfileDetails(
        isPlainObject(targetGpu.vendor_id) && targetGpu.vendor_id.status === 'ok' ? targetGpu.vendor_id.value : null,
        isPlainObject(targetGpu.device_id) && targetGpu.device_id.status === 'ok' ? targetGpu.device_id.value : null,
        isPlainObject(targetGpu.name) && targetGpu.name.status === 'ok' ? targetGpu.name.value : null,
      ))
    : null;

  let okCount = 0;
  for (const key of NVIDIA_REQUIRED_FIELDS) {
    if (isPlainObject(nvFields[key]) && nvFields[key].status === 'ok') {
      okCount += 1;
    }
  }
  let extOkCount = 0;
  if (nvExt) {
    for (const key of NVIDIA_EXTENDED_FIELDS) {
      if (isPlainObject(nvExt[key]) && nvExt[key].status === 'ok') {
        extOkCount += 1;
      }
    }
  }

  const pcie = isPlainObject(report.pcie) ? report.pcie : {};
  const pcieSource = isPlainObject(pcie.current_gen) && typeof pcie.current_gen.source === 'string'
    ? escapeMarkdown(pcie.current_gen.source)
    : 'unknown';
  const pnpMaxLink = formatPcieLink(pcie.max_gen, pcie.max_width);
  const pnpCurLink = formatPcieLink(pcie.current_gen, pcie.current_width);
  const nvMaxLink = formatPcieLink(nvFields['pcie.link.gen.max'], nvFields['pcie.link.width.max']);
  const nvCurLink = formatPcieLink(nvFields['pcie.link.gen.gpucurrent'], nvFields['pcie.link.width.current']);
  const pcieWarnings = getPcieSourceWarnings(report);
  const ctx = isPlainObject(report.context) ? report.context : {};

  const lines = [];
  lines.push('## Tóm tắt báo cáo chẩn đoán VGA');
  lines.push('');
  lines.push(`- **Mã báo cáo:** \`${escapeMarkdown(report.report_id)}\``);
  lines.push(`- **Mã chuỗi thử nghiệm (Experiment ID):** \`${escapeMarkdown(report.experiment_id || 'unknown')}\``);
  if (report.previous_report_id) {
    lines.push(`- **Báo cáo trước:** \`${escapeMarkdown(report.previous_report_id)}\``);
  } else if (report.schema_version === 2) {
    lines.push('- **Báo cáo trước:** `null` (báo cáo đầu tiên trong chuỗi thử nghiệm)');
  }
  lines.push(`- **Thời gian (UTC):** \`${escapeMarkdown(report.created_utc)}\``);
  if (report.collection_started_utc || report.collection_finished_utc) {
    lines.push(
      `- **Cửa sổ thu thập (UTC):** \`${escapeMarkdown(report.collection_started_utc || 'null')}\` → \`${escapeMarkdown(report.collection_finished_utc || 'null')}\``,
    );
  }
  lines.push(`- **Phiên bản collector:** \`${escapeMarkdown(report.collector_version)}\``);
  lines.push(`- **Schema:** \`v${escapeMarkdown(report.schema_version)}\``);
  if (report.evidence_level) {
    lines.push(`- **Mức bằng chứng:** \`${escapeMarkdown(report.evidence_level)}\``);
  }
  if (targetGpu) {
    const tProfileText = targetProfileDetails && targetProfileDetails.name
      ? escapeMarkdown(targetProfileDetails.name)
      : 'chưa nhận diện';
    const tFamilyText = targetProfileDetails && targetProfileDetails.family
      ? escapeMarkdown(targetProfileDetails.family)
      : 'unknown';
    const tCatText = targetProfileDetails && targetProfileDetails.category
      ? escapeMarkdown(targetProfileDetails.category)
      : 'unknown';
    const tGenText = targetProfileDetails && targetProfileDetails.target_pcie_gen
      ? `Gen${escapeMarkdown(targetProfileDetails.target_pcie_gen)}`
      : 'chưa đặt';
    const tTensorText = targetProfileDetails && targetProfileDetails.tensor_applicable
      ? escapeMarkdown(targetProfileDetails.tensor_applicable)
      : 'unknown';
    lines.push(
      `- **GPU mục tiêu:** \`${formatField(targetGpu.bdf)}\` ${formatField(targetGpu.name)} — hồ sơ: ${tProfileText} (kiến trúc: ${tFamilyText}, phân loại: ${tCatText}, mục tiêu PCIe: ${tGenText}, Tensor: ${tTensorText})`,
    );
    if (targetProfileDetails && targetProfileDetails.identity_note) {
      lines.push(`- **Lưu ý danh tính GPU mục tiêu:** ${escapeMarkdown(targetProfileDetails.identity_note)}`);
    }
  } else {
    lines.push('- **GPU mục tiêu:** chưa xác định');
  }
  if (isPlainObject(report.source_timestamps)) {
    const tsParts = SOURCE_TIMESTAMP_KEYS.map((k) => `${k}=\`${escapeMarkdown(report.source_timestamps[k] || 'null')}\``);
    lines.push(`- **Thời điểm thu thập theo nguồn:** ${tsParts.join(', ')}`);
  }
  lines.push('');

  lines.push('### Tóm tắt 4 lớp kết luận tách biệt (Nhận diện / Cấu hình / Phép đo / Bằng chứng)');
  lines.push('');
  lines.push('| Lớp phân tích | Dữ liệu quan sát được |');
  lines.push('| --- | --- |');
  const layer1Text = targetGpu
    ? `${formatField(targetGpu.name)} (\`${formatField(targetGpu.vendor_id)}:${formatField(targetGpu.device_id)}\`, Subsys \`${formatField(targetGpu.subsystem_id)}\`, Rev \`${formatField(targetGpu.revision)}\`) | Hồ sơ: ${targetProfileDetails && targetProfileDetails.name ? escapeMarkdown(targetProfileDetails.name) : 'chưa nhận diện'} | Kiến trúc: ${targetProfileDetails && targetProfileDetails.family ? escapeMarkdown(targetProfileDetails.family) : 'unknown'} | Loại: ${targetProfileDetails && targetProfileDetails.category ? escapeMarkdown(targetProfileDetails.category) : 'unknown'} | Mục tiêu: ${targetProfileDetails && targetProfileDetails.target_pcie_gen ? `Gen${escapeMarkdown(targetProfileDetails.target_pcie_gen)}` : 'chưa đặt'} | Tensor: ${targetProfileDetails && targetProfileDetails.tensor_applicable ? escapeMarkdown(targetProfileDetails.tensor_applicable) : 'unknown'}`
    : 'Chưa xác định GPU mục tiêu';
  const secSummary = isPlainObject(report.security)
    ? `Boot=${formatField(report.security.boot_mode)}, SecureBoot=${formatField(report.security.secure_boot)}, HVCI=${formatField(report.security.hvci_running)}, TestSigning=${formatField(report.security.test_signing)}`
    : 'Chưa thu thập (v1)';
  const drvSummary = targetGpu
    ? `Driver=${formatField(targetGpu.driver_version)}, PnP=${formatField(targetGpu.pnp_status)}, Code=${isPlainObject(targetGpu.driver) ? formatField(targetGpu.driver.problem_code) : 'unsupported'}, INF=${isPlainObject(targetGpu.driver) ? formatField(targetGpu.driver.inf_name) : 'unsupported'}`
    : 'n/a';
  const legacySummary = isPlainObject(report.legacy_status)
    ? (report.legacy_status.present
      ? `${escapeMarkdown(report.legacy_status.source)}:${escapeMarkdown(report.legacy_status.status_code)}`
      : 'không có log cục bộ')
    : 'n/a (v1)';
  const ctxSummary = `mod=${escapeMarkdown(ctx.mod_state || 'unknown')}/${escapeMarkdown(ctx.mod_type || 'unknown')}, riser=${escapeMarkdown(ctx.riser || 'unknown')}, slot=${escapeMarkdown(ctx.slot || 'unknown')}, stage=${escapeMarkdown(ctx.stage || 'unknown')}`;
  const layer2Text = `${secSummary} | ${drvSummary} | Legacy: ${legacySummary} | Ngữ cảnh: ${ctxSummary}`;
  const powerSnap = nvExt
    ? `${formatField(nvFields['power.draw'])} / Enforced ${formatField(nvExt['enforced.power.limit'])} / Default ${formatField(nvExt['power.default_limit'])} / Max ${formatField(nvExt['power.max_limit'])}`
    : formatField(nvFields['power.draw']);
  const clkThermalSnap = `P-State=${formatField(nvFields['pstate'])}, SM=${formatField(nvFields['clocks.sm'])}/${formatField(nvFields['clocks.max.sm'])}, Mem=${formatField(nvFields['clocks.mem'])}, Temp=${formatField(nvFields['temperature.gpu'])}${nvExt ? `, Fan=${formatField(nvExt['fan.speed'])}, Throttle=${formatField(nvExt['clocks_throttle_reasons.active'])}` : ''}`;
  const layer3Text = `PCIe (${pcieSource}): ${pnpCurLink} (max ${pnpMaxLink}) | nvidia-smi: ${nvCurLink} (max ${nvMaxLink}) | Cơ bản: ${okCount}/${NVIDIA_REQUIRED_FIELDS.length}${nvExt ? ` | Mở rộng: ${extOkCount}/${NVIDIA_EXTENDED_FIELDS.length}` : ''} | Power: ${powerSnap} | ${clkThermalSnap}`;
  let layer4Text = `Mức bằng chứng: \`${escapeMarkdown(report.evidence_level || 'reported')}\` (Schema v1 không có bảng completeness 7 nhóm)`;
  if (isPlainObject(report.completeness)) {
    let compOkCount = 0;
    const incompleteGroups = [];
    for (const groupName of COMPLETENESS_GROUPS) {
      const g = report.completeness[groupName];
      if (isPlainObject(g) && g.status === 'ok') {
        compOkCount += 1;
      } else {
        incompleteGroups.push(groupName);
      }
    }
    layer4Text = `Mức bằng chứng: \`${escapeMarkdown(report.evidence_level || 'reported')}\` | Nhóm đạt ok: \`${compOkCount}/${COMPLETENESS_GROUPS.length}\`${incompleteGroups.length > 0 ? ` | Chưa đủ/chờ: ${incompleteGroups.map((g) => `\`${escapeMarkdown(g)}\``).join(', ')}` : ''}`;
  }
  lines.push(`| **1. Nhận diện (Identity)** | ${layer1Text} |`);
  lines.push(`| **2. Cấu hình & Môi trường (Config)** | ${layer2Text} |`);
  lines.push(`| **3. Trạng thái đo (Measurement)** | ${layer3Text} |`);
  lines.push(`| **4. Workload & Bằng chứng (Evidence)** | ${layer4Text} |`);
  lines.push('');

  const platform = isPlainObject(report.platform) ? report.platform : {};
  lines.push('### Nền tảng hệ thống');
  lines.push('');
  lines.push('| Thành phần | Giá trị |');
  lines.push('| --- | --- |');
  lines.push(`| CPU | ${formatField(platform.cpu)} |`);
  lines.push(`| Bo mạch chủ | ${formatField(platform.mainboard_model)} |`);
  lines.push(`| BIOS | ${formatField(platform.bios_version)} |`);
  lines.push(`| Hệ điều hành | ${formatField(platform.os_build)} |`);
  lines.push('');

  if (isPlainObject(report.security)) {
    const sec = report.security;
    lines.push('### Bảo mật & khởi động hệ thống');
    lines.push('');
    lines.push('| Mục | Giá trị | Trạng thái | Nguồn |');
    lines.push('| --- | --- | --- | --- |');
    const secLabels = [
      ['Chế độ khởi động', sec.boot_mode],
      ['Secure Boot', sec.secure_boot],
      ['HVCI / Memory Integrity', sec.hvci_running],
      ['Test Signing', sec.test_signing],
    ];
    for (const [label, f] of secLabels) {
      const st = isPlainObject(f) ? escapeMarkdown(f.status) : 'unsupported';
      const src = isPlainObject(f) ? escapeMarkdown(f.source) : 'none';
      lines.push(`| ${label} | ${formatField(f)} | ${st} | ${src} |`);
    }
    lines.push('');
  }

  lines.push('### Ngữ cảnh thu thập');
  lines.push('');
  lines.push(`- **Trạng thái mod:** ${escapeMarkdown(ctx.mod_state || 'unknown')}`);
  if (ctx.mod_type) {
    lines.push(`- **Loại mod:** ${escapeMarkdown(ctx.mod_type)}`);
  }
  lines.push(`- **Riser:** ${escapeMarkdown(ctx.riser || 'unknown')}`);
  lines.push(`- **Khe cắm:** ${escapeMarkdown(ctx.slot || 'unknown')}`);
  lines.push(`- **Giai đoạn:** ${escapeMarkdown(ctx.stage || 'unknown')}`);
  lines.push(`- **Ghi chú:** ${ctx.notes ? escapeMarkdown(ctx.notes) : 'Không có'}`);
  const ctxKeysToCheck = report.schema_version === 2
    ? ['mod_state', 'mod_type', 'riser', 'slot', 'stage']
    : ['mod_state', 'riser', 'slot', 'stage'];
  if (ctxKeysToCheck.some((k) => !ctx[k] || ctx[k] === 'unknown')) {
    lines.push('- *Lưu ý tái lập phép thử: Một số trường ngữ cảnh vật lý đang là `unknown`; cần bổ sung thông tin phần cứng/môi trường (xem mục Mẫu bổ sung hồ sơ ở cuối báo cáo) trước khi kết luận đối chứng.*');
  }
  lines.push('');

  lines.push('### Danh sách GPU (PnP / CIM)');
  lines.push('');
  lines.push('| BDF | Tên GPU | VID:DID | Hồ sơ | Kiến trúc / Loại | PCIe PnP (Hiện tại / Tối đa) | Mục tiêu Gen / Tensor | Subsys | Rev | PnP | Driver |');
  lines.push('| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |');
  const gpuNotes = [];
  for (const gpu of gpus) {
    if (!isPlainObject(gpu)) continue;
    const vid = formatField(gpu.vendor_id);
    const did = formatField(gpu.device_id);
    const nameRaw = isPlainObject(gpu.name) && gpu.name.status === 'ok' ? gpu.name.value : null;
    const details = isPlainObject(gpu.profile)
      ? gpu.profile
      : getGpuProfileDetails(
        isPlainObject(gpu.vendor_id) && gpu.vendor_id.status === 'ok' ? gpu.vendor_id.value : null,
        isPlainObject(gpu.device_id) && gpu.device_id.status === 'ok' ? gpu.device_id.value : null,
        nameRaw,
      );
    const prof = details.name;
    const isTarget = Boolean(targetGpu && gpu === targetGpu);
    const bdfCell = `${formatField(gpu.bdf)}${isTarget ? ' (mục tiêu)' : ''}`;
    const profCell = prof ? escapeMarkdown(prof) : 'chưa nhận diện';
    const familyCatCell = `${details.family ? escapeMarkdown(details.family) : 'unknown'} / ${details.category ? escapeMarkdown(details.category) : 'unknown'}`;
    const gpuPcieCell = isPlainObject(gpu.pcie)
      ? `${formatPcieLink(gpu.pcie.current_gen, gpu.pcie.current_width)} / ${formatPcieLink(gpu.pcie.max_gen, gpu.pcie.max_width)}`
      : 'n/a';
    const targetGenCell = details.target_pcie_gen ? `Gen${escapeMarkdown(details.target_pcie_gen)}` : 'chưa đặt';
    const tensorCell = details.tensor_applicable ? escapeMarkdown(details.tensor_applicable) : 'unknown';
    const targetTensorCell = `${targetGenCell} / ${tensorCell}`;
    lines.push(
      `| ${bdfCell} | ${formatField(gpu.name)} | ${vid}:${did} | ${profCell} | ${familyCatCell} | ${gpuPcieCell} | ${targetTensorCell} | ${formatField(gpu.subsystem_id)} | ${formatField(gpu.revision)} | ${formatField(gpu.pnp_status)} | ${formatField(gpu.driver_version)} |`,
    );
    if (details.identity_note) {
      gpuNotes.push(`\`${formatField(gpu.bdf)}\`: ${escapeMarkdown(details.identity_note)}`);
    }
    if (isPlainObject(gpu.driver)) {
      const d = gpu.driver;
      gpuNotes.push(
        `Driver \`${formatField(gpu.bdf)}\`: Provider=${formatField(d.provider)}, Date=${formatField(d.date)}, INF=${formatField(d.inf_name)}, Service=${formatField(d.service)}, ProblemCode=${formatField(d.problem_code)}, Path=${formatField(d.location_path)}`,
      );
    }
  }
  for (const note of gpuNotes) {
    lines.push(`- ${note}`);
  }
  lines.push('');

  lines.push('### Liên kết PCIe (GPU mục tiêu / chính)');
  lines.push('');
  lines.push(`- **Tối đa (${pcieSource}):** ${pnpMaxLink}`);
  lines.push(`- **Hiện tại (${pcieSource}):** ${pnpCurLink}`);
  lines.push(`- **Tối đa (nvidia-smi):** ${nvMaxLink}`);
  lines.push(`- **Hiện tại (nvidia-smi):** ${nvCurLink}`);
  for (const warning of pcieWarnings) {
    lines.push(`- **${escapeMarkdown(warning)}**`);
  }
  lines.push('');

  const completeText = nvidia.complete === true ? 'true' : 'false';
  lines.push('### Dữ liệu `nvidia-smi`');
  lines.push('');
  lines.push(`- **Trường NVIDIA cơ bản (nvidia-smi):** \`${okCount}/${NVIDIA_REQUIRED_FIELDS.length}\` (đủ 18/18: \`${completeText}\`)`);
  lines.push('- *Lưu ý: 18/18 chỉ là nhóm trường NVIDIA cơ bản, không phải đủ dữ liệu nghiên cứu/mod.*');
  if (Array.isArray(nvidia.missing) && nvidia.missing.length > 0) {
    lines.push(`- **Trường thiếu/không hỗ trợ:** ${nvidia.missing.map((m) => `\`${escapeMarkdown(m)}\``).join(', ')}`);
  }
  lines.push('');
  lines.push('| Trường | Giá trị | Trạng thái | Nguồn |');
  lines.push('| --- | --- | --- | --- |');
  for (const key of NVIDIA_REQUIRED_FIELDS) {
    const field = nvFields[key];
    const valStr = formatField(field);
    const statusStr = isPlainObject(field) ? escapeMarkdown(field.status) : 'unsupported';
    const srcStr = isPlainObject(field) ? escapeMarkdown(field.source) : 'none';
    lines.push(`| \`${escapeMarkdown(key)}\` | ${valStr} | ${statusStr} | ${srcStr} |`);
  }

  if (nvExt) {
    lines.push('');
    lines.push('#### Công suất, nhiệt, tải & lý do giảm xung (`nvidia-smi` mở rộng)');
    lines.push('');
    lines.push(`- **Trường NVIDIA mở rộng (power/thermal/throttle):** \`${extOkCount}/${NVIDIA_EXTENDED_FIELDS.length}\``);
    if (Array.isArray(nvidia.extended_missing) && nvidia.extended_missing.length > 0) {
      lines.push(`- **Trường mở rộng thiếu/không hỗ trợ:** ${nvidia.extended_missing.map((m) => `\`${escapeMarkdown(m)}\``).join(', ')}`);
    }
    lines.push(
      `- **Tóm tắt Công suất (W):** draw=${formatField(nvFields['power.draw'])} | limit=${formatField(nvExt['power.limit'])} | enforced=${formatField(nvExt['enforced.power.limit'])} | default=${formatField(nvExt['power.default_limit'])} | min=${formatField(nvExt['power.min_limit'])} | max=${formatField(nvExt['power.max_limit'])}`,
    );
    lines.push(
      `- **Tóm tắt Xung, Nhiệt & VRAM:** P-State=${formatField(nvFields['pstate'])} | SM=${formatField(nvFields['clocks.sm'])}/${formatField(nvFields['clocks.max.sm'])} | MemClk=${formatField(nvFields['clocks.mem'])} | Temp=${formatField(nvFields['temperature.gpu'])} (TLimit=${formatField(nvExt['temperature.gpu.tlimit'])}, MemTemp=${formatField(nvExt['temperature.memory'])}) | Fan=${formatField(nvExt['fan.speed'])} | VRAM=${formatField(nvExt['memory.used'])}/${formatField(nvFields['memory.total'])} (Free=${formatField(nvExt['memory.free'])})`,
    );
    lines.push('');
    lines.push('| Trường | Giá trị | Trạng thái | Nguồn |');
    lines.push('| --- | --- | --- | --- |');
    for (const key of NVIDIA_EXTENDED_FIELDS) {
      const field = nvExt[key];
      const valStr = formatField(field);
      const statusStr = isPlainObject(field) ? escapeMarkdown(field.status) : 'unsupported';
      const srcStr = isPlainObject(field) ? escapeMarkdown(field.source) : 'none';
      lines.push(`| \`${escapeMarkdown(key)}\` | ${valStr} | ${statusStr} | ${srcStr} |`);
    }
  }

  if (isPlainObject(report.legacy_status)) {
    const leg = report.legacy_status;
    lines.push('');
    lines.push('### Trạng thái công cụ mở khóa cục bộ (allowlist)');
    lines.push('');
    if (leg.present) {
      lines.push(
        `- **Nguồn:** \`${escapeMarkdown(leg.source)}\` | **Mã trạng thái:** \`${escapeMarkdown(leg.status_code)}\` | **Lỗi:** \`${escapeMarkdown(leg.error_code)}\` | **Device ID:** \`${escapeMarkdown(leg.device_id || 'null')}\` | **Speed/Width/TLS:** Gen${leg.speed_current ?? '?'} ×${leg.width_current ?? '?'} (TLS Gen${leg.tls_target ?? '?'}) | **MRRS:** \`${leg.mrrs_bytes ?? 'null'}\` bytes | **Observed:** \`${escapeMarkdown(leg.observed_utc || 'null')}\``,
      );
    } else {
      lines.push(
        `- **Trạng thái:** Không phát hiện tệp trạng thái mở khóa cục bộ (\`present=false\`, nguồn: \`${escapeMarkdown(leg.source)}\`).`,
      );
    }
  }

  if (isPlainObject(report.completeness)) {
    lines.push('');
    lines.push('### Mức độ đầy đủ theo nhóm bằng chứng');
    lines.push('');
    lines.push('| Nhóm | Đủ | Trạng thái | Thiếu | Lý do |');
    lines.push('| --- | --- | --- | --- | --- |');
    for (const groupName of COMPLETENESS_GROUPS) {
      const g = report.completeness[groupName];
      if (!isPlainObject(g)) continue;
      const missingText = Array.isArray(g.missing) && g.missing.length > 0
        ? g.missing.map((m) => escapeMarkdown(m)).join(', ')
        : 'Không';
      lines.push(
        `| \`${escapeMarkdown(groupName)}\` | ${g.complete ? 'true' : 'false'} | ${escapeMarkdown(g.status)} | ${missingText} | ${g.reason ? escapeMarkdown(g.reason) : ''} |`,
      );
    }
  }

  lines.push('');
  lines.push('### Hồ sơ phần cứng & thử nghiệm cần chủ card bổ sung (modding-data-plan §7 & §9)');
  lines.push('');
  lines.push('<details>');
  lines.push('<summary>Mở rộng mẫu khai báo bổ sung (PSU, Riser, PCB/VRAM, Gói mod, Giả thuyết &amp; Kết quả sau Reboot/Sleep/Rollback)</summary>');
  lines.push('');
  lines.push('```text');
  lines.push('- Card/SKU thực tế & PCB revision / mã chip VRAM (từ ảnh hoặc nhãn PCB, che serial): ________');
  lines.push('- Khe cắm trên mainboard & Riser (cắm trực tiếp / model + chiều dài riser): ________');
  lines.push('- Nguồn PSU & cách cấp nguồn phụ cho card (1 dây/2 dây riêng, đầu 6/8-pin): ________');
  lines.push('- Can thiệp đã thử (strap / flash VBIOS / EFI / mod driver) & phiên bản hoặc SHA-256: ________');
  lines.push('- Mục tiêu thử nghiệm (TDP / PCIe Gen Speed / ReBAR / CUDA / Tensor / VRAM / ổn định): ________');
  lines.push('- Giả thuyết về nguồn giới hạn & dữ liệu còn thiếu cần đo thêm: ________');
  lines.push('- Kết quả sau thay đổi / sau reboot / sau sleep-wake / sau khôi phục (rollback): ________');
  lines.push('- Xác nhận của chủ phần cứng (chỉ bỏ nhãn unverified sau khi đối chứng): ________');
  lines.push('```');
  lines.push('');
  lines.push('</details>');

  let output = lines.join('\n');
  if (output.length > MAX_SUMMARY_LENGTH) {
    const suffix = '\n\n*(Đã cắt bớt để giữ giới hạn ký tự GitHub Issue)*';
    output = output.slice(0, MAX_SUMMARY_LENGTH - suffix.length) + suffix;
  }
  return output;
}

module.exports = {
  SCHEMA_VERSION,
  SUPPORTED_SCHEMA_VERSIONS,
  MAX_PAYLOAD_BYTES,
  MAX_GPUS,
  MAX_STRING_LENGTH,
  MAX_SUMMARY_LENGTH,
  NVIDIA_REQUIRED_FIELDS,
  NVIDIA_EXTENDED_FIELDS,
  KNOWN_GPU_PROFILES,
  KNOWN_GPU_PROFILE_DETAILS,
  identifyGpuProfile,
  getGpuProfileDetails,
  normalizeBdf,
  validateReport,
  findPii,
  renderSummary,
  escapeMarkdown,
};
