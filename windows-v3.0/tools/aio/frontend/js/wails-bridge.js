// Wails Web Bridge: Polyfills window.go and window.runtime when WebView2 is absent
(function() {
  if (window.runtime && window.go) return;

  const eventCallbacks = {};

  window.runtime = {
    EventsOn(event, callback) {
      if (!eventCallbacks[event]) eventCallbacks[event] = [];
      eventCallbacks[event].push(callback);
    },
    EventsEmit(event, data) {
      // client-to-client or future use
    }
  };

  function connectSSE() {
    try {
      const src = new EventSource('/api/events/stream');
      src.onmessage = (e) => {
        try {
          const payload = JSON.parse(e.data);
          const cbs = eventCallbacks[payload.event] || [];
          cbs.forEach(fn => fn(payload.data));
        } catch {}
      };
      src.onerror = () => {
        src.close();
        setTimeout(connectSSE, 2000);
      };
    } catch {}
  }
  connectSSE();

  async function rpc(endpoint, body) {
    const opts = { method: 'POST' };
    if (body !== undefined) {
      opts.headers = { 'Content-Type': 'application/json' };
      opts.body = JSON.stringify(body);
    }
    const res = await fetch('/api/' + endpoint, opts);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return await res.json();
  }

  window.go = {
    main: {
      App: {
        UnlockGen2Now: () => rpc('unlock-now'),
        ForceRootGen2: () => rpc('force-root-gen2'),
        RegisterAutostartTask: () => rpc('register-task'),
        UnregisterAutostartTask: () => rpc('unregister-task'),
        SavePolicy: (s, c, i, h) => rpc('save-policy', { strategy: s, retryCount: c, retryInterval: i, autoHard: h }),
        ScanEnvironment: () => rpc('scan'),
        CheckHVCIStatus: () => rpc('hvci'),
        OpenCoreIsolationSettings: () => rpc('open-core-isolation'),
        GetRiotStatus: () => rpc('riot-status'),
        RiotOptimize: () => rpc('riot-optimize'),
        RiotRevertOptimization: () => rpc('riot-revert'),
        RiotSignEFI: (phrase) => rpc('riot-sign-efi', { phrase }),
        RebootToBIOS: () => rpc('reboot-bios'),
        FixCSMError43: () => rpc('fix-csm'),
        SetupReBar: (sz) => rpc('setup-rebar', { barSizeGB: sz }),
        RevertReBar: () => rpc('revert-rebar'),
        UninstallAll: () => rpc('uninstall'),
        GetAppVersion: () => Promise.resolve("v3.0.0")
      }
    }
  };
})();
