const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
export const REGISTRY_HOST = import.meta.env.VITE_REGISTRY_HOST || window.location.host;

function getAuthToken() {
  return localStorage.getItem('token');
}

function getAuthHeaders() {
  const token = getAuthToken();
  return token ? { 'Authorization': `Bearer ${token}` } : {};
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(`${API_BASE_URL}${url}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
      ...options.headers,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || `HTTP error! status: ${response.status}`);
  }

  return response.json();
}

export async function fetchProviders({ page = 1, limit = 20, namespace = '', name = '' } = {}) {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });

  if (namespace) params.append('namespace', namespace);
  if (name) params.append('name', name);

  return fetchJSON(`/api/v1/providers?${params}`);
}

export async function fetchProvider(namespace, name, version) {
  return fetchJSON(`/api/v1/providers/${namespace}/${name}/${version}`);
}

export async function fetchProviderVersions(namespace, name) {
  return fetchJSON(`/v1/providers/${namespace}/${name}/versions`);
}

export async function fetchProviderVersionsDetail(namespace, name) {
  return fetchJSON(`/api/v1/mirror/providers/${namespace}/${name}`);
}

export async function fetchModules({ page = 1, limit = 20, namespace = '', name = '' } = {}) {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });

  if (namespace) params.append('namespace', namespace);
  if (name) params.append('name', name);

  return fetchJSON(`/api/v1/modules?${params}`);
}

export async function fetchModule(namespace, name, provider, version) {
  return fetchJSON(`/api/v1/modules/${namespace}/${name}/${provider}/${version}`);
}

export async function searchProviders(query) {
  return fetchJSON(`/api/v1/providers/search?q=${encodeURIComponent(query)}`);
}

// Mirror API
export async function fetchMirroredProviders({ page = 1, limit = 20, sourceType = '' } = {}) {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (sourceType) params.append('source_type', sourceType);
  return fetchJSON(`/api/v1/mirror/providers?${params}`);
}

export async function mirrorProvider(namespace, name, { version, os = 'all', arch = 'all', proxyUrl = '' } = {}) {
  const params = new URLSearchParams();
  if (version) params.append('version', version);
  params.append('os', os);
  params.append('arch', arch);
  if (proxyUrl) params.append('proxy_url', proxyUrl);
  
  return fetchJSON(`/api/v1/mirror/${namespace}/${name}?${params}`, {
    method: 'POST',
  });
}

// Mirror provider with SSE progress updates
export function mirrorProviderWithProgress(namespace, name, { version, os = 'all', arch = 'all', proxyUrl = '' } = {}, onProgress) {
  return new Promise((resolve, reject) => {
    const params = new URLSearchParams();
    if (version) params.append('version', version);
    params.append('os', os);
    params.append('arch', arch);
    if (proxyUrl) params.append('proxy_url', proxyUrl);

    const token = getAuthToken();
    const url = `${API_BASE_URL}/api/v1/mirror/${namespace}/${name}/stream?${params}${token ? '&token=' + token : ''}`;
    const eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (onProgress) {
          onProgress(data);
        }
        if (data.type === 'complete') {
          eventSource.close();
          resolve(data);
        } else if (data.type === 'error') {
          eventSource.close();
          reject(new Error(data.error));
        }
      } catch (e) {
        console.error('Failed to parse SSE data:', e);
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
      reject(new Error('Connection lost'));
    };
  });
}

export async function uploadProvider(formData) {
  const response = await fetch(`${API_BASE_URL}/api/v1/providers/upload`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: formData,
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || `HTTP error! status: ${response.status}`);
  }

  return response.json();
}

export async function deleteProvider(id) {
  return fetchJSON(`/api/v1/providers/${id}`, {
    method: 'DELETE',
  });
}

// Export provider as downloadable package
export function getExportProviderURL(id) {
  const token = getAuthToken();
  return `${API_BASE_URL}/api/v1/mirror/export/${id}${token ? '?token=' + token : ''}`;
}

export async function exportProvider(id) {
  const response = await fetch(`${API_BASE_URL}/api/v1/mirror/export/${id}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || `HTTP error! status: ${response.status}`);
  }
  return response;
}

// Import provider from exported package
export async function importProvider(formData) {
  const response = await fetch(`${API_BASE_URL}/api/v1/mirror/import`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || `HTTP error! status: ${response.status}`);
  }

  return response.json();
}

// Settings API
export async function fetchSettings() {
  return fetchJSON('/api/v1/settings');
}

export async function updateSettings(settings) {
  return fetchJSON('/api/v1/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });
}

// Sync schedules API
export async function fetchSyncSchedules() {
  return fetchJSON('/api/v1/sync/schedules');
}

export async function createSyncSchedule(schedule) {
  return fetchJSON('/api/v1/sync/schedules', {
    method: 'POST',
    body: JSON.stringify(schedule),
  });
}

export async function updateSyncSchedule(id, schedule) {
  return fetchJSON(`/api/v1/sync/schedules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(schedule),
  });
}

export async function deleteSyncSchedule(id) {
  return fetchJSON(`/api/v1/sync/schedules/${id}`, {
    method: 'DELETE',
  });
}

export async function runSyncScheduleNow(id) {
  return fetchJSON(`/api/v1/sync/schedules/${id}/run`, {
    method: 'POST',
  });
}

// Fetch upstream versions (requires auth)
export async function fetchUpstreamVersions(namespace, name) {
  return fetchJSON(`/api/v1/mirror/upstream/${namespace}/${name}`);
}
