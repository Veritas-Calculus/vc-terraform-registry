import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { fetchModule } from '../services/api';

function ModuleDetail() {
  const { namespace, name, provider, version } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadModule();
  }, [namespace, name, provider, version]);

  const loadModule = async () => {
    try {
      setLoading(true);
      const data = await fetchModule(namespace, name, provider, version);
      setModule(data);
    } catch (error) {
      console.error('Failed to load module:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (!module) {
    return (
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Module Not Found</h2>
          <p className="text-gray-600">The requested module could not be found.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div className="bg-white rounded-lg shadow-sm p-8">
        <div className="border-b border-gray-200 pb-6 mb-6">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            {module.namespace}/{module.name}
          </h1>
          <p className="text-gray-600">{module.description}</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <InfoCard title="Version" value={module.version} />
          <InfoCard title="Provider" value={module.provider} />
          <InfoCard title="Downloads" value={module.downloads || 0} />
          <InfoCard 
            title="Published" 
            value={new Date(module.published).toLocaleDateString()} 
          />
        </div>

        <div className="bg-gray-50 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Usage Example</h3>
          <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 overflow-x-auto">
            <code>{`module "${name}" {
  source  = "registry.example.com/${namespace}/${name}/${provider}"
  version = "~> ${module.version}"

  # Module inputs
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

export default ModuleDetail;
