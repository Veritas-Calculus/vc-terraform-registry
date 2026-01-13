import { useState, useEffect } from 'react';
import { fetchSettings, updateSettings } from '../services/api';

export default function SettingsPanel({ onMessage }) {
  const [settings, setSettings] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [proxyForm, setProxyForm] = useState({
    proxy_url: '',
    proxy_type: 'http'
  });

  useEffect(() => {
    loadSettings();
  }, []);

  useEffect(() => {
    if (settings) {
      setProxyForm({
        proxy_url: settings.proxy_url || '',
        proxy_type: settings.proxy_type || 'http'
      });
    }
  }, [settings]);

  const loadSettings = async () => {
    try {
      setLoading(true);
      const data = await fetchSettings();
      setSettings(data);
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to load settings: ' + err.message });
    } finally {
      setLoading(false);
    }
  };

  const handleToggleOnlineSearch = async () => {
    if (!settings) return;
    
    const newValue = !settings.allow_online_search;
    try {
      setSaving(true);
      const updated = await updateSettings({ allow_online_search: newValue });
      setSettings(updated);
      onMessage({
        type: 'success',
        text: newValue ? 'Online search enabled' : 'Online search disabled'
      });
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to update settings: ' + err.message });
    } finally {
      setSaving(false);
    }
  };

  const handleToggleProxy = async () => {
    if (!settings) return;
    
    const newValue = !settings.proxy_enabled;
    try {
      setSaving(true);
      const updated = await updateSettings({ proxy_enabled: newValue });
      setSettings(updated);
      onMessage({
        type: 'success',
        text: newValue ? 'Proxy enabled' : 'Proxy disabled'
      });
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to update settings: ' + err.message });
    } finally {
      setSaving(false);
    }
  };

  const handleSaveProxy = async () => {
    if (!settings) return;
    
    try {
      setSaving(true);
      const updated = await updateSettings({
        proxy_url: proxyForm.proxy_url,
        proxy_type: proxyForm.proxy_type
      });
      setSettings(updated);
      onMessage({ type: 'success', text: 'Proxy settings saved' });
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to save proxy settings: ' + err.message });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!settings) {
    return (
      <div className="text-center py-12 text-gray-500">
        Failed to load settings
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900">Registry Settings</h2>
        <p className="text-sm text-gray-500 mt-1">
          Configure how the registry behaves
        </p>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-200">
        {/* Online Search Toggle */}
        <div className="p-4 flex items-center justify-between">
          <div className="flex-1">
            <h3 className="font-medium text-gray-900">Allow Online Search</h3>
            <p className="text-sm text-gray-500 mt-1">
              When enabled, provider searches will query the upstream Terraform Registry
              for providers not found locally.
            </p>
          </div>
          <button
            onClick={handleToggleOnlineSearch}
            disabled={saving}
            className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
              settings.allow_online_search ? 'bg-blue-600' : 'bg-gray-200'
            } ${saving ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <span
              className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                settings.allow_online_search ? 'translate-x-5' : 'translate-x-0'
              }`}
            />
          </button>
        </div>

        {/* Default Upstream URL (read-only display) */}
        <div className="p-4">
          <h3 className="font-medium text-gray-900">Default Upstream URL</h3>
          <p className="text-sm text-gray-500 mt-1">
            The upstream registry used for provider lookups and mirroring
          </p>
          <div className="mt-2 font-mono text-sm bg-gray-50 px-3 py-2 rounded-lg text-gray-700">
            {settings.default_upstream_url || 'https://registry.terraform.io'}
          </div>
        </div>

        {/* Registry URL for Terraform Config */}
        <div className="p-4">
          <h3 className="font-medium text-gray-900">Registry URL</h3>
          <p className="text-sm text-gray-500 mt-1">
            The URL that will be shown in Terraform configuration examples (e.g., registry.example.com)
          </p>
          <div className="mt-2 flex gap-2">
            <input
              type="text"
              value={settings.registry_url || ''}
              onChange={(e) => setSettings({ ...settings, registry_url: e.target.value })}
              placeholder="registry.example.com"
              className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
            />
            <button
              onClick={async () => {
                try {
                  setSaving(true);
                  const updated = await updateSettings({ registry_url: settings.registry_url });
                  setSettings(updated);
                  onMessage({ type: 'success', text: 'Registry URL saved' });
                } catch (err) {
                  onMessage({ type: 'error', text: 'Failed to save: ' + err.message });
                } finally {
                  setSaving(false);
                }
              }}
              disabled={saving}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              Save
            </button>
          </div>
          <p className="mt-1 text-xs text-gray-500">
            Leave empty to use the current domain automatically
          </p>
        </div>
      </div>

      {/* Proxy Settings */}
      <div className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-200">
        <div className="p-4 flex items-center justify-between">
          <div className="flex-1">
            <h3 className="font-medium text-gray-900">Enable Proxy</h3>
            <p className="text-sm text-gray-500 mt-1">
              Use a proxy server for upstream registry requests (supports HTTP and SOCKS5)
            </p>
          </div>
          <button
            onClick={handleToggleProxy}
            disabled={saving}
            className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
              settings.proxy_enabled ? 'bg-blue-600' : 'bg-gray-200'
            } ${saving ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <span
              className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                settings.proxy_enabled ? 'translate-x-5' : 'translate-x-0'
              }`}
            />
          </button>
        </div>

        {/* Proxy Configuration */}
        <div className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Proxy Type
            </label>
            <select
              value={proxyForm.proxy_type}
              onChange={(e) => setProxyForm({ ...proxyForm, proxy_type: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="http">HTTP / HTTPS</option>
              <option value="socks5">SOCKS5</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Proxy URL
            </label>
            <input
              type="text"
              value={proxyForm.proxy_url}
              onChange={(e) => setProxyForm({ ...proxyForm, proxy_url: e.target.value })}
              placeholder={proxyForm.proxy_type === 'socks5' ? 'socks5://127.0.0.1:1080' : 'http://127.0.0.1:8080'}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
            />
            <p className="mt-1 text-xs text-gray-500">
              {proxyForm.proxy_type === 'socks5' 
                ? 'Format: socks5://host:port or host:port'
                : 'Format: http://host:port or https://host:port'}
            </p>
          </div>

          <div className="flex justify-end">
            <button
              onClick={handleSaveProxy}
              disabled={saving}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? 'Saving...' : 'Save Proxy Settings'}
            </button>
          </div>
        </div>
      </div>

      {/* Info Box */}
      <div className="bg-blue-50 border border-blue-200 rounded-xl p-4">
        <div className="flex">
          <svg className="h-5 w-5 text-blue-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-blue-800">About Proxy Settings</h3>
            <p className="mt-1 text-sm text-blue-700">
              When a proxy is enabled, all requests to the upstream Terraform Registry
              will be routed through the configured proxy server. This is useful for
              environments with restricted network access or when you need to route
              traffic through a specific network path.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
