import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { fetchProvider, fetchSettings, fetchProviderVersions } from '../services/api';

function ProviderDetail() {
  const { namespace, name, version } = useParams();
  const navigate = useNavigate();
  const [provider, setProvider] = useState(null);
  const [versions, setVersions] = useState([]);
  const [registryUrl, setRegistryUrl] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [namespace, name, version]);

  const loadData = async () => {
    try {
      setLoading(true);
      
      // Load settings for registry URL
      try {
        const settings = await fetchSettings();
        setRegistryUrl(settings.registry_url || window.location.host);
      } catch (e) {
        setRegistryUrl(window.location.host);
      }
      
      // Load all versions for this provider
      try {
        const versionsData = await fetchProviderVersions(namespace, name);
        setVersions(versionsData.versions || []);
      } catch (e) {
        setVersions([]);
      }
      
      // Load specific version or latest
      const data = await fetchProvider(namespace, name, version);
      setProvider(data);
    } catch (error) {
      console.error('Failed to load provider:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleVersionChange = (newVersion) => {
    navigate(`/providers/${namespace}/${name}/${newVersion}`);
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (!provider) {
    return (
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Provider Not Found</h2>
          <p className="text-gray-600">The requested provider could not be found.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      {/* Breadcrumb */}
      <div className="mb-4">
        <Link to="/providers" className="text-blue-600 hover:text-blue-800">
          ← Back to Providers
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow-sm p-8">
        <div className="border-b border-gray-200 pb-6 mb-6">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            {provider.namespace}/{provider.name}
          </h1>
          <p className="text-gray-600">{provider.description}</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          {/* Version Selector */}
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-500 mb-1">Version</div>
            {versions.length > 1 ? (
              <select
                value={provider.version}
                onChange={(e) => handleVersionChange(e.target.value)}
                className="w-full text-lg font-semibold text-gray-900 bg-white border border-gray-300 rounded-lg px-2 py-1 focus:ring-2 focus:ring-blue-500"
              >
                {versions.map((v) => (
                  <option key={v.version} value={v.version}>
                    v{v.version}
                  </option>
                ))}
              </select>
            ) : (
              <div className="text-2xl font-semibold text-gray-900">{provider.version}</div>
            )}
          </div>
          <InfoCard title="Downloads" value={(provider.downloads || 0).toLocaleString()} />
          <InfoCard 
            title="Published" 
            value={new Date(provider.published).toLocaleDateString()} 
          />
          <InfoCard title="Available Versions" value={versions.length || 1} />
        </div>

        <div className="bg-gray-50 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Usage Example</h3>
          <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 overflow-x-auto">
            <code>{`terraform {
  required_providers {
    ${name} = {
      source  = "${registryUrl}/${namespace}/${name}"
      version = "~> ${provider.version}"
    }
  }
}

provider "${name}" {
  # Configuration options
}`}</code>
          </pre>
        </div>
      </div>
    </div>
  );
}

function InfoCard({ title, value }) {
  return (
    <div className="bg-gray-50 rounded-lg p-4">
      <div className="text-sm text-gray-500 mb-1">{title}</div>
      <div className="text-2xl font-semibold text-gray-900">{value}</div>
    </div>
  );
}

export default ProviderDetail;
