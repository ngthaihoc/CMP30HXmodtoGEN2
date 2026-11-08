const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const nodes = new Map(['specMrrsVal', 'throughputNumber', 'throughputSubtext', 'gaugeFill', 'boostTag']
  .map(id => [id, { style: {} }]));
const context = vm.createContext({ document: { getElementById: id => nodes.get(id) } });
vm.runInContext(fs.readFileSync(path.join(__dirname, '../frontend/js/telemetry.js'), 'utf8') + '\nthis.render = Telemetry.render;', context);
context.render({ gpuDetected: true, lanes: [] });
assert.equal(nodes.get('specMrrsVal').textContent, 'Chưa biết');
assert.equal(nodes.get('gaugeFill').style.width, '0%');
assert.match(nodes.get('throughputNumber').innerHTML, /^0 /);
context.render({ isGen2: true, linkSpeed: 1, linkWidth: 4, mrrs: 512, throughputMBs: 1000 });
assert.equal(nodes.get('boostTag').textContent, 'ĐÃ CẤU HÌNH GEN2');
assert.match(nodes.get('throughputSubtext').textContent, /Gen1 x4/);
console.log('Đạt: telemetry giữ rõ dữ liệu thiếu và link nhàn rỗi.');

async function checkLegacyStatusFailure() {
	const legacyNodes = new Map(['currentThroughput', 'gaugeBarFill', 'currentSpeedSub', 'boostRatio']
		.map(id => [id, { style: {} }]));
	const legacy = vm.createContext({
		document: {
			getElementById: id => legacyNodes.get(id),
			querySelectorAll: () => [], querySelector: () => null, addEventListener() {}
		},
		EventSource: class {}, fetch: async () => { throw new Error('Mất kết nối'); }
	});
	const source = fs.readFileSync(path.join(__dirname, '../../inst40hx/web/app.js'), 'utf8');
	vm.runInContext(source, legacy);
	await new Promise(resolve => setImmediate(resolve));
	assert.equal(legacyNodes.get('gaugeBarFill').style.width, '0%');
	assert.match(legacyNodes.get('currentThroughput').innerHTML, /^0 /);
	assert.equal(legacyNodes.get('currentSpeedSub').textContent, 'Chưa có dữ liệu GPU');
	console.log('Đạt: mất kết nối không hiển thị GPU hay Gen2 giả.');
}
checkLegacyStatusFailure().catch(error => { console.error(error); process.exitCode = 1; });
