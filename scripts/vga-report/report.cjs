'use strict';

const SCHEMA_VERSION = 1;
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
const RISER_STATES = new Set(['direct', 'riser', 'unknown']);
const SLOT_STATES = new Set(['cpu_x16', 'chipset', 'other', 'unknown']);
const STAGE_STATES = new Set(['before_unlock', 'after_unlock', 'troubleshooting', 'baseline', 'unknown']);

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

const FIELD_KEYS = ['value', 'source', 'unit', 'status'];
const PCIE_KEYS = ['max_gen', 'current_gen', 'max_width', 'current_width'];
const PLATFORM_KEYS = ['cpu', 'mainboard_model', 'bios_version', 'os_build'];
const CONTEXT_KEYS = ['mod_state', 'riser', 'slot', 'stage', 'notes'];
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
const NVIDIA_KEYS = ['complete', 'missing', 'fields'];

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

function validateNvidia(nvidia, path, errors) {
  if (!isPlainObject(nvidia)) {
    errors.push(`${path}: expected nvidia object`);
    return;
  }
  checkKeys(nvidia, path, NVIDIA_KEYS, new Set(NVIDIA_KEYS), errors);

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

  checkKeys(report, 'root', ROOT_KEYS, new Set(ROOT_KEYS), errors);

  if (report.schema_version !== SCHEMA_VERSION) {
    errors.push(`root.schema_version: must equal ${SCHEMA_VERSION}`);
  }
  if (typeof report.collector_version !== 'string' || report.collector_version.length < 1 || report.collector_version.length > 50) {
    errors.push('root.collector_version: must be a non-empty string (<= 50 chars)');
  }
  if (typeof report.report_id !== 'string' || !UUID_RE.test(report.report_id)) {
    errors.push('root.report_id: must be a valid UUID');
  }
  if (typeof report.created_utc !== 'string' || !ISO_UTC_RE.test(report.created_utc) || Number.isNaN(Date.parse(report.created_utc))) {
    errors.push('root.created_utc: must be a valid ISO-8601 UTC timestamp');
  }
  if (typeof report.experiment_id !== 'string' || !UUID_RE.test(report.experiment_id)) {
    errors.push('root.experiment_id: must be a valid UUID');
  }
  if (report.consent !== true) {
    errors.push('root.consent: must be true');
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
    for (let i = 0; i < report.gpus.length; i += 1) {
      const gpu = report.gpus[i];
      const gpuPath = `root.gpus[${i}]`;
      if (!isPlainObject(gpu)) {
        errors.push(`${gpuPath}: expected object`);
        continue;
      }
      checkKeys(gpu, gpuPath, GPU_REQUIRED_KEYS, GPU_ALLOWED_KEYS, errors);
      for (const key of GPU_REQUIRED_KEYS) {
        if (Object.prototype.hasOwnProperty.call(gpu, key)) {
          validateField(gpu[key], `${gpuPath}.${key}`, errors);
        }
      }
      if (Object.prototype.hasOwnProperty.call(gpu, 'pcie')) {
        validatePcie(gpu.pcie, `${gpuPath}.pcie`, errors);
      }
      if (Object.prototype.hasOwnProperty.call(gpu, 'nvidia')) {
        validateNvidia(gpu.nvidia, `${gpuPath}.nvidia`, errors);
      }
    }
  }

  if (Object.prototype.hasOwnProperty.call(report, 'nvidia')) {
    validateNvidia(report.nvidia, 'root.nvidia', errors);
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
    checkKeys(report.context, 'root.context', CONTEXT_KEYS, new Set(CONTEXT_KEYS), errors);
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

function scanPiiValue(value, path, found) {
  if (typeof value === 'string') {
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

function renderSummary(report) {
  if (!isPlainObject(report)) {
    return '';
  }

  const lines = [];
  lines.push('## Tóm tắt báo cáo chẩn đoán VGA');
  lines.push('');
  lines.push(`- **Mã báo cáo:** \`${escapeMarkdown(report.report_id)}\``);
  lines.push(`- **Thời gian (UTC):** \`${escapeMarkdown(report.created_utc)}\``);
  lines.push(`- **Phiên bản collector:** \`${escapeMarkdown(report.collector_version)}\``);
  lines.push(`- **Schema:** \`v${escapeMarkdown(report.schema_version)}\``);
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

  const ctx = isPlainObject(report.context) ? report.context : {};
  lines.push('### Ngữ cảnh thu thập');
  lines.push('');
  lines.push(`- **Trạng thái mod:** ${escapeMarkdown(ctx.mod_state || 'unknown')}`);
  lines.push(`- **Riser:** ${escapeMarkdown(ctx.riser || 'unknown')}`);
  lines.push(`- **Khe cắm:** ${escapeMarkdown(ctx.slot || 'unknown')}`);
  lines.push(`- **Giai đoạn:** ${escapeMarkdown(ctx.stage || 'unknown')}`);
  lines.push(`- **Ghi chú:** ${ctx.notes ? escapeMarkdown(ctx.notes) : 'Không có'}`);
  lines.push('');

  lines.push('### Danh sách GPU (PnP / CIM)');
  lines.push('');
  lines.push('| BDF | Tên GPU | VID:DID | Subsys | Rev | PnP | Driver |');
  lines.push('| --- | --- | --- | --- | --- | --- | --- |');
  const gpus = Array.isArray(report.gpus) ? report.gpus.slice(0, MAX_GPUS) : [];
  for (const gpu of gpus) {
    if (!isPlainObject(gpu)) continue;
    const vid = formatField(gpu.vendor_id);
    const did = formatField(gpu.device_id);
    lines.push(
      `| ${formatField(gpu.bdf)} | ${formatField(gpu.name)} | ${vid}:${did} | ${formatField(gpu.subsystem_id)} | ${formatField(gpu.revision)} | ${formatField(gpu.pnp_status)} | ${formatField(gpu.driver_version)} |`,
    );
  }
  lines.push('');

  const pcie = isPlainObject(report.pcie) ? report.pcie : {};
  lines.push('### Liên kết PCIe (GPU mục tiêu / chính)');
  lines.push('');
  lines.push(`- **Tối đa:** Gen ${formatField(pcie.max_gen)} x${formatField(pcie.max_width)}`);
  lines.push(`- **Hiện tại:** Gen ${formatField(pcie.current_gen)} x${formatField(pcie.current_width)}`);
  lines.push('');

  const nvidia = isPlainObject(report.nvidia) ? report.nvidia : {};
  const completeText = nvidia.complete === true ? 'true (18/18)' : 'false';
  lines.push('### Dữ liệu `nvidia-smi`');
  lines.push('');
  lines.push(`- **Đủ 18/18 trường:** \`${completeText}\``);
  if (Array.isArray(nvidia.missing) && nvidia.missing.length > 0) {
    lines.push(`- **Trường thiếu/không hỗ trợ:** ${nvidia.missing.map((m) => `\`${escapeMarkdown(m)}\``).join(', ')}`);
  }
  lines.push('');
  lines.push('| Trường | Giá trị | Trạng thái | Nguồn |');
  lines.push('| --- | --- | --- | --- |');
  const nvFields = isPlainObject(nvidia.fields) ? nvidia.fields : {};
  for (const key of NVIDIA_REQUIRED_FIELDS) {
    const field = nvFields[key];
    const valStr = formatField(field);
    const statusStr = isPlainObject(field) ? escapeMarkdown(field.status) : 'unsupported';
    const srcStr = isPlainObject(field) ? escapeMarkdown(field.source) : 'none';
    lines.push(`| \`${escapeMarkdown(key)}\` | ${valStr} | ${statusStr} | ${srcStr} |`);
  }

  let output = lines.join('\n');
  if (output.length > MAX_SUMMARY_LENGTH) {
    const suffix = '\n\n*(Đã cắt bớt để giữ giới hạn ký tự GitHub Issue)*';
    output = output.slice(0, MAX_SUMMARY_LENGTH - suffix.length) + suffix;
  }
  return output;
}

module.exports = {
  SCHEMA_VERSION,
  MAX_PAYLOAD_BYTES,
  MAX_GPUS,
  MAX_STRING_LENGTH,
  MAX_SUMMARY_LENGTH,
  NVIDIA_REQUIRED_FIELDS,
  validateReport,
  findPii,
  renderSummary,
  escapeMarkdown,
};
