import { useState, useEffect } from 'react';
import { fetchMirroredProviders, fetchUpstreamVersions, mirrorProviderWithProgress, uploadProvider, deleteProvider, getExportProviderURL, importProvider, fetchProviderVersionsDetail, REGISTRY_HOST } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import SyncSchedulesPanel from '../components/SyncSchedulesPanel';
import SettingsPanel from '../components/SettingsPanel';

export default function Mirror() {
  const { isAuthenticated } = useAuth();
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('mirrored');
  const [mirrorForm, setMirrorForm] = useState({ namespace: '', name: '', version: '', os: 'all', arch: 'all', proxyUrl: '' });
  const [uploadForm, setUploadForm] = useState({ namespace: '', name: '', version: '', os: 'linux', arch: 'amd64', description: '', file: null });
  const [importFile, setImportFile] = useState(null);
  const [importLoading, setImportLoading] = useState(false);
  const [upstreamVersions, setUpstreamVersions] = useState(null);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [mirrorLoading, setMirrorLoading] = useState(false);
  const [uploadLoading, setUploadLoading] = useState(false);
  const [message, setMessage] = useState(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  
  // Progress state
  const [mirrorProgress, setMirrorProgress] = useState(null);
  
  // Provider detail view state
  const [selectedProvider, setSelectedProvider] = useState(null);
  const [providerVersions, setProviderVersions] = useState([]);
  const [loadingVersions, setLoadingVersions] = useState(false);

  useEffect(() => {
    loadProviders();
  }, []);

  // Auto-fetch versions when namespace and name are both filled (only for authenticated users)
  useEffect(() => {
    if (!isAuthenticated || !mirrorForm.namespace || !mirrorForm.name) {
      setUpstreamVersions(null);
      return;
    }

    const timeoutId = setTimeout(async () => {
      try {
        setVersionsLoading(true);
        const data = await fetchUpstreamVersions(mirrorForm.namespace, mirrorForm.name);
        setUpstreamVersions(data);
      } catch {
        setUpstreamVersions(null);
      } finally {
        setVersionsLoading(false);
      }
    }, 500);

    return () => clearTimeout(timeoutId);
  }, [mirrorForm.namespace, mirrorForm.name, isAuthenticated]);

  async function loadProviders() {
    try {
      setLoading(true);
      const data = await fetchMirroredProviders();
      setProviders(data.providers || []);
    } catch (err) {
      setMessage({ type: 'error', text: `Failed to load providers: ${err.message}` });
    } finally {
      setLoading(false);
    }
  }

  async function handleViewVersions(provider) {
    try {
      setLoadingVersions(true);
      setSelectedProvider(provider);
      const data = await fetchProviderVersionsDetail(provider.namespace, provider.name);
      setProviderVersions(data.versions || []);
    } catch (err) {
      setMessage({ type: 'error', text: `Failed to load versions: ${err.message}` });
    } finally {
      setLoadingVersions(false);
    }
  }

  function handleBackToList() {
    setSelectedProvider(null);
    setProviderVersions([]);
  }

  async function handleMirror() {
    if (!mirrorForm.namespace || !mirrorForm.name) {
      setMessage({ type: 'error', text: 'Please enter namespace and provider name' });
      return;
    }
    try {
      setMirrorLoading(true);
      setMirrorProgress(null);
      
      await mirrorProviderWithProgress(
        mirrorForm.namespace, 
        mirrorForm.name, 
        {
          version: mirrorForm.version,
          os: mirrorForm.os,
          arch: mirrorForm.arch,
          proxyUrl: mirrorForm.proxyUrl,
        },
        (progress) => {
          setMirrorProgress(progress);
        }
      );
      
      setMessage({ type: 'success', text: 'Provider mirrored successfully!' });
      setMirrorProgress(null);
      loadProviders();
    } catch (err) {
      setMessage({ type: 'error', text: `Failed to mirror: ${err.message}` });
      setMirrorProgress(null);
    } finally {
      setMirrorLoading(false);
    }
  }

  async function handleUpload(e) {
    e.preventDefault();
    if (!uploadForm.namespace || !uploadForm.name || !uploadForm.version || !uploadForm.file) {
      setMessage({ type: 'error', text: 'Please fill all required fields' });
      return;
    }
    try {
      setUploadLoading(true);
      const formData = new FormData();
      formData.append('namespace', uploadForm.namespace);
      formData.append('name', uploadForm.name);
      formData.append('version', uploadForm.version);
      formData.append('os', uploadForm.os);
      formData.append('arch', uploadForm.arch);
      formData.append('description', uploadForm.description);
      formData.append('file', uploadForm.file);
      
      await uploadProvider(formData);
      setMessage({ type: 'success', text: 'Provider uploaded successfully!' });
      setUploadForm({ namespace: '', name: '', version: '', os: 'linux', arch: 'amd64', description: '', file: null });
      loadProviders();
    } catch (err) {
      setMessage({ type: 'error', text: `Failed to upload: ${err.message}` });
    } finally {
      setUploadLoading(false);
    }
  }

  async function handleDelete(id) {
    if (!confirm('Are you sure you want to delete this provider?')) return;
    try {
      await deleteProvider(id);
      setMessage({ type: 'success', text: 'Provider deleted successfully' });
      loadProviders();
    } catch (err) {
      setMessage({ type: 'error', text: `Failed to delete: ${err.message}` });
    }
  }

  function handleExport(id, namespace, name, version) {
    const url = getExportProviderURL(id);
    const link = document.createElement('a');
    link.href = url;
    link.download = `terraform-provider-${name}_${version}_${namespace}.zip`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    setMessage({ type: 'success', text: 'Download started! The package can be imported to another instance.' });
  }

  async function handleImport(e) {
    e.preventDefault();
    if (!importFile) {
      setMessage({ type: 'error', text: 'Please select a provider package file' });
      return;
    }
    try {
      setImportLoading(true);
      const formData = new FormData();
      formData.append('file', importFile);
      
      const result = await importProvider(formData);
      setMessage({ type: 'success', text: `Provider ${result.provider.Namespace}/${result.provider.Name} v${result.provider.Version} imported successfully!` });
      setImportFile(null);
      loadProviders();
    } catch (err) {
      setMessage({ type: 'error', text: `Failed to import: ${err.message}` });
    } finally {
      setImportLoading(false);
    }
  }

  const popularProviders = [
    { namespace: 'telmate', name: 'proxmox', description: 'Proxmox VE Provider' },
    { namespace: 'hashicorp', name: 'aws', description: 'AWS Provider' },
    { namespace: 'hashicorp', name: 'azurerm', description: 'Azure Provider' },
    { namespace: 'hashicorp', name: 'google', description: 'Google Cloud Provider' },
    { namespace: 'hashicorp', name: 'kubernetes', description: 'Kubernetes Provider' },
    { namespace: 'cloudflare', name: 'cloudflare', description: 'Cloudflare Provider' },
  ];

  // Format download speed
  function formatSpeed(bytesPerSecond) {
    if (bytesPerSecond >= 1024 * 1024) {
      return `${(bytesPerSecond / (1024 * 1024)).toFixed(1)} MB/s`;
    } else if (bytesPerSecond >= 1024) {
      return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`;
    }
    return `${bytesPerSecond} B/s`;
  }

  // Format ETA
  function formatETA(seconds) {
    if (seconds < 60) {
      return `${Math.ceil(seconds)}s`;
    } else if (seconds < 3600) {
      const mins = Math.floor(seconds / 60);
      const secs = Math.ceil(seconds % 60);
      return `${mins}m ${secs}s`;
    }
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${mins}m`;
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-semibold text-gray-900 tracking-tight">Provider Mirror</h1>
        <p className="mt-2 text-gray-500">
          {isAuthenticated 
            ? 'Mirror providers from upstream registries or upload custom providers' 
            : 'View cached providers. Sign in to mirror, upload, or import providers.'}
        </p>
      </div>

      {message && (
        <div className={`mb-6 p-4 rounded-xl ${message.type === 'error' ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700'}`}>
          {message.text}
          <button onClick={() => setMessage(null)} className="float-right font-bold">×</button>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 mb-8">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('mirrored')}
            className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
              activeTab === 'mirrored'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Cached Providers
          </button>
          {isAuthenticated && (
            <>
              <button
                onClick={() => setActiveTab('mirror')}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === 'mirror'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Mirror from Upstream
              </button>
              <button
                onClick={() => setActiveTab('upload')}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === 'upload'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Upload Provider
              </button>
              <button
                onClick={() => setActiveTab('import')}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === 'import'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Import Package
              </button>
              <button
                onClick={() => setActiveTab('schedules')}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === 'schedules'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Sync Schedules
              </button>
              <button
                onClick={() => setActiveTab('settings')}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === 'settings'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Settings
              </button>
            </>
          )}
        </nav>
      </div>

      {/* Mirrored Providers Tab */}
      {activeTab === 'mirrored' && (
        <div>
          {/* Provider Detail View */}
          {selectedProvider ? (
            <div>
              <button
                onClick={handleBackToList}
                className="flex items-center text-blue-600 hover:text-blue-800 mb-4"
              >
                <svg className="w-5 h-5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
                Back to Providers
              </button>
              
              <div className="bg-white rounded-2xl border border-gray-200 p-6 mb-6">
                <h2 className="text-2xl font-bold text-gray-900">
                  {selectedProvider.namespace}/{selectedProvider.name}
                </h2>
                <p className="text-gray-500 mt-1">
                  {selectedProvider.version_count} version{selectedProvider.version_count !== 1 ? 's' : ''} • 
                  {selectedProvider.platform_count} platform{selectedProvider.platform_count !== 1 ? 's' : ''} cached
                </p>
              </div>

              {loadingVersions ? (
                <div className="text-center py-12">
                  <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-500"></div>
                </div>
              ) : (
                <div className="space-y-4">
                  {providerVersions.map((version) => (
                    <div key={version.id} className="bg-white rounded-2xl border border-gray-200 p-6">
                      <div className="flex items-center justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-3">
                            <h3 className="text-lg font-semibold text-gray-900">
                              v{version.version}
                            </h3>
                            <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                              version.source_type === 'mirror' ? 'bg-blue-50 text-blue-700' : 'bg-green-50 text-green-700'
                            }`}>
                              {version.source_type === 'mirror' ? 'Mirrored' : 'Uploaded'}
                            </span>
                          </div>
                          <p className="text-sm text-gray-500 mt-1">
                            {version.downloads || 0} downloads • 
                            Published {version.published ? new Date(version.published).toLocaleDateString() : 'N/A'}
                          </p>
                          {version.platforms && version.platforms.length > 0 && (
                            <div className="flex flex-wrap gap-2 mt-3">
                              {version.platforms.map((p, i) => (
                                <span key={i} className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                                  {p.os}/{p.arch}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                        {isAuthenticated && (
                          <div className="flex items-center space-x-2">
                            <button
                              onClick={() => handleExport(version.id, version.namespace, version.name, version.version)}
                              className="inline-flex items-center px-3 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-xl hover:bg-blue-100 transition-colors"
                              title="Export this version"
                            >
                              <svg className="h-4 w-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                              </svg>
                              Export
                            </button>
                            <button
                              onClick={() => handleDelete(version.id)}
                              className="text-red-500 hover:text-red-700 p-2"
                              title="Delete this version"
                            >
                              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : loading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-500"></div>
            </div>
          ) : providers.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 rounded-2xl">
              <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
              </svg>
              <h3 className="mt-4 text-lg font-medium text-gray-900">No cached providers</h3>
              <p className="mt-2 text-gray-500">Mirror a provider from upstream or upload one manually</p>
            </div>
          ) : (
            <div className="grid gap-4">
              {providers.map((provider) => (
                <div 
                  key={`${provider.namespace}-${provider.name}`} 
                  className="bg-white rounded-2xl border border-gray-200 p-6 hover:shadow-lg transition-shadow cursor-pointer"
                  onClick={() => handleViewVersions(provider)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h3 className="text-lg font-semibold text-gray-900">
                        {provider.namespace}/{provider.name}
                      </h3>
                      <p className="text-sm text-gray-500 mt-1">
                        Latest: v{provider.version} • 
                        {provider.version_count > 1 && (
                          <span className="text-blue-600 font-medium ml-1">
                            {provider.version_count} versions
                          </span>
                        )}
                        {provider.version_count <= 1 && ' 1 version'}
                        {' • '}
                        {provider.platform_count} platform{provider.platform_count !== 1 ? 's' : ''}
                      </p>
                      <div className="flex items-center gap-3 mt-2">
                        <span className={`inline-flex items-center text-sm font-medium ${provider.source_type === 'mirror' ? 'text-blue-600' : 'text-green-600'}`}>
                          {provider.source_type === 'mirror' ? (
                            <><svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>Mirrored</>
                          ) : (
                            <><svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>Uploaded</>
                          )}
                        </span>
                        <span className="text-sm text-gray-500">
                          {(provider.downloads || 0).toLocaleString()} downloads
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center">
                      <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Mirror from Upstream Tab */}
      {activeTab === 'mirror' && isAuthenticated && (
        <div className="max-w-2xl">
          <div className="bg-white rounded-2xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Mirror Provider from registry.terraform.io</h3>
            
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
                  <input
                    type="text"
                    value={mirrorForm.namespace}
                    onChange={(e) => setMirrorForm({ ...mirrorForm, namespace: e.target.value })}
                    placeholder="e.g., telmate"
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Provider Name</label>
                  <input
                    type="text"
                    value={mirrorForm.name}
                    onChange={(e) => setMirrorForm({ ...mirrorForm, name: e.target.value })}
                    placeholder="e.g., proxmox"
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>

              {/* Version select with auto-loading */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Version
                  {versionsLoading && (
                    <span className="ml-2 inline-flex items-center">
                      <svg className="animate-spin h-4 w-4 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                    </span>
                  )}
                </label>
                <select
                  value={mirrorForm.version}
                  onChange={(e) => setMirrorForm({ ...mirrorForm, version: e.target.value })}
                  disabled={!upstreamVersions || versionsLoading}
                  className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-50 disabled:text-gray-400"
                >
                  <option value="">{upstreamVersions ? 'Latest' : 'Enter provider info above'}</option>
                  {upstreamVersions?.versions?.map((v) => (
                    <option key={v.version} value={v.version}>{v.version}</option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">OS (optional)</label>
                  <select
                    value={mirrorForm.os}
                    onChange={(e) => setMirrorForm({ ...mirrorForm, os: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="all">All Platforms</option>
                    <option value="linux">Linux</option>
                    <option value="darwin">macOS</option>
                    <option value="windows">Windows</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Architecture (optional)</label>
                  <select
                    value={mirrorForm.arch}
                    onChange={(e) => setMirrorForm({ ...mirrorForm, arch: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="all">All Architectures</option>
                    <option value="amd64">amd64</option>
                    <option value="arm64">arm64</option>
                    <option value="arm">arm</option>
                    <option value="386">386</option>
                  </select>
                </div>
              </div>

              {/* Advanced Options Toggle */}
              <button
                type="button"
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="flex items-center text-sm text-gray-500 hover:text-gray-700"
              >
                <svg 
                  className={`h-4 w-4 mr-1 transition-transform ${showAdvanced ? 'rotate-90' : ''}`} 
                  fill="none" 
                  viewBox="0 0 24 24" 
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
                Advanced Options
              </button>

              {/* Advanced Options Panel */}
              {showAdvanced && (
                <div className="p-4 bg-gray-50 rounded-xl space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      HTTP Proxy (optional)
                    </label>
                    <input
                      type="text"
                      value={mirrorForm.proxyUrl}
                      onChange={(e) => setMirrorForm({ ...mirrorForm, proxyUrl: e.target.value })}
                      placeholder="e.g., http://proxy.example.com:8080"
                      className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                    />
                    <p className="mt-1 text-xs text-gray-400">
                      Use a proxy server to download providers from upstream registry
                    </p>
                  </div>
                </div>
              )}

              <button
                onClick={handleMirror}
                disabled={mirrorLoading}
                className="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors disabled:opacity-50"
              >
                {mirrorLoading ? 'Mirroring...' : 'Mirror Provider'}
              </button>

              {/* Progress Bar */}
              {mirrorProgress && (
                <div className="mt-4 p-4 bg-gray-50 rounded-xl">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-gray-700">
                      {mirrorProgress.message}
                    </span>
                    <span className="text-sm text-gray-500">
                      {mirrorProgress.current}/{mirrorProgress.total}
                    </span>
                  </div>
                  
                  {/* Progress bar */}
                  <div className="w-full bg-gray-200 rounded-full h-3 mb-3">
                    <div 
                      className="bg-blue-600 h-3 rounded-full transition-all duration-300"
                      style={{ width: `${mirrorProgress.percent || 0}%` }}
                    ></div>
                  </div>
                  
                  {/* Stats */}
                  <div className="flex items-center justify-between text-xs text-gray-500">
                    <div className="flex items-center space-x-4">
                      {mirrorProgress.platform && (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-blue-100 text-blue-700">
                          {mirrorProgress.platform}
                        </span>
                      )}
                      {mirrorProgress.bytes_per_second > 0 && (
                        <span className="inline-flex items-center">
                          <svg className="w-3.5 h-3.5 mr-1 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                          </svg>
                          {formatSpeed(mirrorProgress.bytes_per_second)}
                        </span>
                      )}
                    </div>
                    {mirrorProgress.eta_seconds > 0 && (
                      <span className="inline-flex items-center">
                        <svg className="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        {formatETA(mirrorProgress.eta_seconds)} remaining
                      </span>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Popular Providers */}
          <div className="mt-8">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Popular Providers</h3>
            <div className="grid grid-cols-2 gap-3">
              {popularProviders.map((p) => (
                <button
                  key={`${p.namespace}/${p.name}`}
                  onClick={() => {
                    setMirrorForm({ ...mirrorForm, namespace: p.namespace, name: p.name });
                    setUpstreamVersions(null);
                  }}
                  className="p-4 bg-white rounded-xl border border-gray-200 hover:border-blue-500 hover:shadow-md transition-all text-left"
                >
                  <div className="font-medium text-gray-900">{p.namespace}/{p.name}</div>
                  <div className="text-sm text-gray-500">{p.description}</div>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Upload Provider Tab */}
      {activeTab === 'upload' && isAuthenticated && (
        <div className="max-w-2xl">
          <div className="bg-white rounded-2xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Upload Provider Binary</h3>
            
            <form onSubmit={handleUpload} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Namespace *</label>
                  <input
                    type="text"
                    value={uploadForm.namespace}
                    onChange={(e) => setUploadForm({ ...uploadForm, namespace: e.target.value })}
                    placeholder="e.g., myorg"
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Provider Name *</label>
                  <input
                    type="text"
                    value={uploadForm.name}
                    onChange={(e) => setUploadForm({ ...uploadForm, name: e.target.value })}
                    placeholder="e.g., myprovider"
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    required
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Version *</label>
                <input
                  type="text"
                  value={uploadForm.version}
                  onChange={(e) => setUploadForm({ ...uploadForm, version: e.target.value })}
                  placeholder="e.g., 1.0.0"
                  className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">OS *</label>
                  <select
                    value={uploadForm.os}
                    onChange={(e) => setUploadForm({ ...uploadForm, os: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    required
                  >
                    <option value="linux">Linux</option>
                    <option value="darwin">macOS</option>
                    <option value="windows">Windows</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Architecture *</label>
                  <select
                    value={uploadForm.arch}
                    onChange={(e) => setUploadForm({ ...uploadForm, arch: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    required
                  >
                    <option value="amd64">amd64</option>
                    <option value="arm64">arm64</option>
                    <option value="arm">arm</option>
                    <option value="386">386</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                <textarea
                  value={uploadForm.description}
                  onChange={(e) => setUploadForm({ ...uploadForm, description: e.target.value })}
                  placeholder="Optional description"
                  rows={2}
                  className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Provider Binary (zip) *</label>
                <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-xl hover:border-blue-400 transition-colors">
                  <div className="space-y-1 text-center">
                    <svg className="mx-auto h-12 w-12 text-gray-400" stroke="currentColor" fill="none" viewBox="0 0 48 48">
                      <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                    <div className="flex text-sm text-gray-600 justify-center">
                      <label className="relative cursor-pointer bg-white rounded-md font-medium text-blue-600 hover:text-blue-500 focus-within:outline-none">
                        <span>Upload a file</span>
                        <input
                          type="file"
                          accept=".zip"
                          onChange={(e) => setUploadForm({ ...uploadForm, file: e.target.files[0] })}
                          className="sr-only"
                        />
                      </label>
                      <p className="pl-1">or drag and drop</p>
                    </div>
                    <p className="text-xs text-gray-500">.zip file up to 100MB</p>
                    {uploadForm.file && (
                      <p className="text-sm text-blue-600 font-medium mt-2">
                        Selected: {uploadForm.file.name}
                      </p>
                    )}
                  </div>
                </div>
              </div>

              <button
                type="submit"
                disabled={uploadLoading}
                className="w-full py-3 bg-green-600 hover:bg-green-700 text-white rounded-xl font-medium transition-colors disabled:opacity-50"
              >
                {uploadLoading ? 'Uploading...' : 'Upload Provider'}
              </button>
            </form>
          </div>

          <div className="mt-6 p-4 bg-blue-50 rounded-xl">
            <h4 className="font-medium text-blue-900">Provider Binary Format</h4>
            <p className="text-sm text-blue-700 mt-1">
              The file should be a zip archive containing the provider plugin binary. 
              The naming convention is: <code className="bg-blue-100 px-1 rounded">terraform-provider-NAME_VERSION_OS_ARCH.zip</code>
            </p>
          </div>
        </div>
      )}

      {/* Import Package Tab */}
      {activeTab === 'import' && isAuthenticated && (
        <div className="max-w-2xl">
          <div className="bg-white rounded-2xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Import Provider Package</h3>
            <p className="text-gray-500 mb-6">
              Import a provider package that was exported from another VC Terraform Registry instance.
            </p>
            
            <form onSubmit={handleImport} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Provider Package (.zip) *</label>
                <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-xl hover:border-blue-400 transition-colors">
                  <div className="space-y-1 text-center">
                    <svg className="mx-auto h-12 w-12 text-gray-400" stroke="currentColor" fill="none" viewBox="0 0 48 48">
                      <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                    <div className="flex text-sm text-gray-600 justify-center">
                      <label className="relative cursor-pointer bg-white rounded-md font-medium text-blue-600 hover:text-blue-500 focus-within:outline-none">
                        <span>Select package file</span>
                        <input
                          type="file"
                          accept=".zip"
                          onChange={(e) => setImportFile(e.target.files[0])}
                          className="sr-only"
                        />
                      </label>
                      <p className="pl-1">or drag and drop</p>
                    </div>
                    <p className="text-xs text-gray-500">Exported .zip package file</p>
                    {importFile && (
                      <p className="text-sm text-blue-600 font-medium mt-2">
                        Selected: {importFile.name}
                      </p>
                    )}
                  </div>
                </div>
              </div>

              <button
                type="submit"
                disabled={importLoading || !importFile}
                className="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors disabled:opacity-50"
              >
                {importLoading ? 'Importing...' : 'Import Provider'}
              </button>
            </form>
          </div>

          <div className="mt-6 p-4 bg-blue-50 rounded-xl">
            <h4 className="font-medium text-blue-900">How to Transfer Providers</h4>
            <div className="text-sm text-blue-700 mt-2 space-y-2">
              <p><strong>1. Export:</strong> Go to "Cached Providers" tab, click the "Export" button on any provider</p>
              <p><strong>2. Transfer:</strong> Copy the downloaded .zip file to your target machine</p>
              <p><strong>3. Import:</strong> Upload the .zip file here to import all platform binaries</p>
            </div>
          </div>

          <div className="mt-4 p-4 bg-gray-50 rounded-xl">
            <h4 className="font-medium text-gray-900">Package Contents</h4>
            <p className="text-sm text-gray-600 mt-1">
              Exported packages include:
            </p>
            <ul className="text-sm text-gray-600 mt-2 list-disc list-inside space-y-1">
              <li><code className="bg-gray-200 px-1 rounded">manifest.json</code> - Provider metadata and checksums</li>
              <li><code className="bg-gray-200 px-1 rounded">linux/amd64/</code> - Platform-specific binaries</li>
              <li><code className="bg-gray-200 px-1 rounded">darwin/arm64/</code> - etc.</li>
            </ul>
          </div>
        </div>
      )}

      {/* Sync Schedules Tab */}
      {activeTab === 'schedules' && isAuthenticated && (
        <SyncSchedulesPanel onMessage={setMessage} />
      )}

      {/* Settings Tab */}
      {activeTab === 'settings' && isAuthenticated && (
        <SettingsPanel onMessage={setMessage} />
      )}

      {/* Usage Instructions */}
      <div className="mt-12 p-6 bg-gray-50 rounded-2xl">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Using This Registry</h3>
        <p className="text-gray-600 mb-4">Configure Terraform to use this registry as a mirror:</p>
        <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
          <p className="text-amber-800 text-sm">
            <strong>Note:</strong> HTTPS is required. Replace the port number with your actual HTTPS port (default: 3443).
          </p>
        </div>
        <pre className="bg-gray-900 text-gray-100 p-4 rounded-xl overflow-x-auto text-sm">
{`# ~/.terraformrc or terraform.rc

provider_installation {
  network_mirror {
    url = "https://${REGISTRY_HOST}/"
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}`}
        </pre>
        <p className="text-gray-600 mt-4">Or use explicit provider source:</p>
        <pre className="bg-gray-900 text-gray-100 p-4 rounded-xl overflow-x-auto text-sm">
{`terraform {
  required_providers {
    proxmox = {
      source  = "${REGISTRY_HOST}/telmate/proxmox"
      version = "~> 2.9"
    }
  }
}`}
        </pre>
      </div>
    </div>
  );
}
