import React, { useState, useEffect } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import SearchBar from '../components/SearchBar';
import { fetchProviders, searchProviders } from '../services/api';

function Providers() {
  const [searchParams] = useSearchParams();
  const query = searchParams.get('q') || '';
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [onlineSearch, setOnlineSearch] = useState(false);

  useEffect(() => {
    setPage(1);
  }, [query]);

  useEffect(() => {
    loadProviders();
  }, [page, query]);

  const loadProviders = async () => {
    try {
      setLoading(true);
      if (query) {
        const data = await searchProviders(query);
        setProviders(data.providers || []);
        setTotal(data.total || 0);
        setOnlineSearch(data.online_search || false);
      } else {
        const data = await fetchProviders({ page, limit: 20 });
        setProviders(data.providers || []);
        setTotal(data.total || 0);
        setOnlineSearch(false);
      }
    } catch (error) {
      console.error('Failed to load providers:', error);
      setProviders([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div className="mb-8">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">Providers</h1>
        <SearchBar placeholder="Search providers..." initialQuery={query} />
        {query && (
          <div className="mt-4 flex items-center gap-2">
            <span className="text-gray-600">
              Search results for "<span className="font-medium">{query}</span>"
            </span>
            {onlineSearch && (
              <span className="px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded">
                Online search enabled
              </span>
            )}
            <Link to="/providers" className="text-blue-600 hover:text-blue-800 text-sm ml-2">
              Clear search
            </Link>
          </div>
        )}
      </div>

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {providers.map((provider, index) => (
              <ProviderCard key={provider.id || `${provider.namespace}-${provider.name}-${index}`} provider={provider} />
            ))}
          </div>

          {providers.length === 0 && (
            <div className="text-center py-12">
              <p className="text-gray-500 text-lg">No providers found</p>
              {query && !onlineSearch && (
                <p className="text-gray-400 text-sm mt-2">
                  Online search is disabled. Enable it in Settings to search upstream registry.
                </p>
              )}
            </div>
          )}

          {!query && total > 20 && (
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

function ProviderCard({ provider }) {
  // Always link to provider with its latest version
  const linkTo = provider.version 
    ? `/providers/${provider.namespace}/${provider.name}/${provider.version}`
    : `/mirror?namespace=${provider.namespace}&name=${provider.name}`;

  const isUpstream = provider.source === 'upstream';
  const isCached = provider.is_cached;
  const tier = provider.tier;
  const versionCount = provider.version_count;

  // Get tier badge styles
  const getTierBadge = () => {
    switch (tier) {
      case 'official':
        return { text: 'Official', className: 'text-purple-700 bg-purple-50' };
      case 'partner':
        return { text: 'Partner', className: 'text-indigo-700 bg-indigo-50' };
      default:
        return { text: 'Community', className: 'text-gray-700 bg-gray-100' };
    }
  };

  const tierBadge = getTierBadge();

  return (
    <Link
      to={linkTo}
      className="block bg-white rounded-xl shadow-sm hover:shadow-md transition-shadow p-6 border border-gray-100"
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <h3 className="text-lg font-semibold text-gray-900">
            {provider.namespace}/{provider.name}
          </h3>
          {tier && (
            <span className={`px-2 py-0.5 text-xs font-medium rounded ${tierBadge.className}`}>
              {tierBadge.text}
            </span>
          )}
        </div>
        <div className="flex gap-1 flex-shrink-0">
          {provider.version && (
            <span className="px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded">
              v{provider.version}
            </span>
          )}
          {versionCount > 1 && (
            <span className="px-2 py-1 text-xs font-medium text-gray-600 bg-gray-100 rounded">
              +{versionCount - 1} versions
            </span>
          )}
          {isUpstream && !isCached && (
            <span className="px-2 py-1 text-xs font-medium text-orange-700 bg-orange-50 rounded">
              Upstream
            </span>
          )}
          {isCached && (
            <span className="px-2 py-1 text-xs font-medium text-green-700 bg-green-50 rounded">
              Cached
            </span>
          )}
        </div>
      </div>
      <p className="text-gray-600 text-sm mb-4 line-clamp-2">
        {provider.description || 'No description available'}
      </p>
      <div className="flex items-center justify-between text-sm text-gray-500">
        <span>{(provider.downloads || 0).toLocaleString()} downloads</span>
        {provider.published && (
          <span>{new Date(provider.published).toLocaleDateString()}</span>
        )}
      </div>
    </Link>
  );
}

export default Providers;
