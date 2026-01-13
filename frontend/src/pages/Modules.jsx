import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import SearchBar from '../components/SearchBar';
import { fetchModules } from '../services/api';

function Modules() {
  const [modules, setModules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    loadModules();
  }, [page]);

  const loadModules = async () => {
    try {
      setLoading(true);
      const data = await fetchModules({ page, limit: 20 });
      setModules(data.modules);
      setTotal(data.total);
    } catch (error) {
      console.error('Failed to load modules:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div className="mb-8">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">Modules</h1>
        <SearchBar placeholder="Search modules..." />
      </div>

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {modules.map((module) => (
              <ModuleCard key={module.id} module={module} />
            ))}
          </div>

          {modules.length === 0 && (
            <div className="text-center py-12">
              <p className="text-gray-500 text-lg">No modules found</p>
            </div>
          )}

          {total > 20 && (
            <div className="mt-8 flex justify-center gap-2">
              <button
                onClick={() => setPage(page - 1)}
                disabled={page === 1}
                className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Previous
              </button>
              <span className="px-4 py-2 text-sm text-gray-700">
                Page {page} of {Math.ceil(total / 20)}
              </span>
              <button
                onClick={() => setPage(page + 1)}
                disabled={page >= Math.ceil(total / 20)}
                className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function ModuleCard({ module }) {
  return (
    <Link
      to={`/modules/${module.namespace}/${module.name}/${module.provider}/${module.version}`}
      className="block bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow p-6"
    >
      <div className="flex items-start justify-between mb-3">
        <h3 className="text-xl font-semibold text-gray-900">
          {module.namespace}/{module.name}
        </h3>
        <span className="px-2 py-1 text-xs font-medium text-primary-700 bg-primary-50 rounded">
          v{module.version}
        </span>
      </div>
      <p className="text-sm text-gray-500 mb-2">Provider: {module.provider}</p>
      <p className="text-gray-600 text-sm mb-4 line-clamp-2">
        {module.description || 'No description available'}
      </p>
      <div className="flex items-center justify-between text-sm text-gray-500">
        <span>⬇️ {module.downloads || 0} downloads</span>
        <span>{new Date(module.published).toLocaleDateString()}</span>
      </div>
    </Link>
  );
}

export default Modules;
