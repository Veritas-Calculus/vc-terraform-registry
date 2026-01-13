import React, { useState, useRef, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

function Layout({ children }) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated, logout, loading } = useAuth();
  const [showUserMenu, setShowUserMenu] = useState(false);
  const menuRef = useRef(null);

  useEffect(() => {
    function handleClickOutside(event) {
      if (menuRef.current && !menuRef.current.contains(event.target)) {
        setShowUserMenu(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const isActive = (path) => location.pathname === path || location.pathname.startsWith(path + '/');

  const navLinkClass = (path) =>
    `inline-flex items-center px-1 pt-1 text-sm font-medium transition-colors ${
      isActive(path)
        ? 'text-blue-600 border-b-2 border-blue-600'
        : 'text-gray-900 hover:text-blue-600'
    }`;

  function handleLogout() {
    logout();
    setShowUserMenu(false);
    navigate('/');
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50">
      <header className="bg-white/80 backdrop-blur-xl border-b border-gray-200 sticky top-0 z-50">
        <nav className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              <Link to="/" className="flex items-center">
                <span className="text-xl font-semibold text-gray-900 tracking-tight">
                  VC Terraform Registry
                </span>
              </Link>
              <div className="hidden sm:ml-10 sm:flex sm:space-x-8">
                <Link to="/providers" className={navLinkClass('/providers')}>
                  Providers
                </Link>
                <Link to="/modules" className={navLinkClass('/modules')}>
                  Modules
                </Link>
                <Link to="/mirror" className={navLinkClass('/mirror')}>
                  <span className="flex items-center">
                    Mirror
                    <span className="ml-1.5 px-1.5 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 rounded">
                      NEW
                    </span>
                  </span>
                </Link>
              </div>
            </div>
            <div className="flex items-center space-x-4">
              <Link
                to="/docs"
                className="text-gray-500 hover:text-gray-700 transition-colors"
                title="Documentation"
              >
                <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                </svg>
              </Link>
              <a
                href="https://github.com/Veritas-Calculus/vc-terraform-registry"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-500 hover:text-gray-700 transition-colors"
              >
                <svg className="h-6 w-6" fill="currentColor" viewBox="0 0 24 24">
                  <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                </svg>
              </a>
              
              {!loading && (
                isAuthenticated ? (
                  <div className="relative" ref={menuRef}>
                    <button
                      type="button"
                      onClick={() => setShowUserMenu(!showUserMenu)}
                      className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 rounded-full hover:bg-gray-100 transition-colors"
                    >
                      <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center text-white font-medium text-sm">
                        {user?.username?.charAt(0).toUpperCase() || 'U'}
                      </div>
                      <span className="hidden sm:block">{user?.username}</span>
                      <svg className={`w-4 h-4 transition-transform ${showUserMenu ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    
                    {showUserMenu && (
                      <div className="absolute right-0 mt-2 w-56 rounded-xl bg-white shadow-lg border border-gray-200 py-2 z-50">
                        <div className="px-4 py-3 border-b border-gray-100">
                          <p className="text-sm font-medium text-gray-900">{user?.username}</p>
                          <p className="text-xs text-gray-500">{user?.email}</p>
                          {user?.role === 'admin' && (
                            <span className="inline-block mt-1 px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 rounded-full">
                              Admin
                            </span>
                          )}
                        </div>
                        <button
                          onClick={handleLogout}
                          className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 flex items-center gap-2"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                          </svg>
                          Sign Out
                        </button>
                      </div>
                    )}
                  </div>
                ) : (
                  <Link
                    to="/login"
                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-full hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-colors"
                  >
                    Sign In
                  </Link>
                )
              )}
            </div>
          </div>
        </nav>
      </header>

      <main className="flex-1">
        {children}
      </main>

      <footer className="bg-white border-t border-gray-200 mt-auto">
        <div className="max-w-7xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center">
            <p className="text-sm text-gray-500">
              © 2026 VC Terraform Registry. All rights reserved.
            </p>
            <div className="flex space-x-6">
              <a href="https://github.com/Veritas-Calculus/vc-terraform-registry" className="text-gray-400 hover:text-gray-500 transition-colors">
                GitHub
              </a>
              <a href="https://terraform.io/docs" className="text-gray-400 hover:text-gray-500 transition-colors">
                Terraform Docs
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default Layout;
