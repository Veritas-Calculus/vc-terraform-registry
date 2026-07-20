// Authentication context and hooks

import { createContext, useContext, useState, useEffect } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('token'));
  // Starts true only when a stored token still needs validating; every exit path
  // of the token effect's fetchCurrentUser clears it in its `finally`. Must stay
  // in sync with that effect -- see the comment there.
  const [loading, setLoading] = useState(!!token);
  const [authEnabled, setAuthEnabled] = useState(true);

  useEffect(() => {
    checkAuthStatus();
  }, []);

  // Early return, no else-branch: with no token, `loading` was already initialised
  // to false above. Reverting that initialiser without restoring an else-branch
  // would strand signed-out users at loading === true forever, hiding the Sign In
  // button (Layout gates the whole auth UI on !loading).
  //
  // fetchCurrentUser lives INSIDE the effect deliberately. Hoisting it back to
  // component scope makes exhaustive-deps demand it as a dep, and it is recreated
  // every render, so [token, fetchCurrentUser] would refetch /auth/me forever.
  // Deps stay [token] alone because logout() closes over nothing reactive (just
  // localStorage plus the stable setToken/setUser). If logout ever reads reactive
  // state, exhaustive-deps will start demanding it here -- add it, do not suppress.
  useEffect(() => {
    if (!token) return;

    // Guards against a superseded response landing on a newer session: without
    // it, a late 401 for an old token calls logout() over a session the user has
    // since signed into (/login is a top-level route outside Layout's !loading
    // gate, so that sequence is reachable), and a late 200 calls setUser() after
    // a logout, showing a signed-in UI with no token.
    let ignore = false;

    async function fetchCurrentUser() {
      try {
        const response = await fetch(`${API_BASE_URL}/api/v1/auth/me`, {
          headers: {
            'Authorization': `Bearer ${token}`,
          },
        });

        if (response.ok) {
          const data = await response.json();
          if (!ignore) setUser(data.user);
        } else if (!ignore) {
          // Token is invalid, clear it
          logout();
        }
      } catch (err) {
        console.error('Failed to fetch user:', err);
        if (!ignore) logout();
      } finally {
        // Unconditional on purpose. Guarding this with !ignore strands the user
        // at loading === true whenever the token clears mid-flight: the re-run
        // early-returns on the null token, so nothing else would ever clear it,
        // and Layout hides the whole auth block -- including Sign In -- while
        // loading is true.
        setLoading(false);
      }
    }

    fetchCurrentUser();

    return () => {
      ignore = true;
    };
  }, [token]);

  async function checkAuthStatus() {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/status`);
      const data = await response.json();
      setAuthEnabled(data.auth_enabled);
    } catch (err) {
      console.error('Failed to check auth status:', err);
    }
  }

  async function login(username, password) {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, password }),
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || 'Login failed');
    }

    localStorage.setItem('token', data.token);
    setToken(data.token);
    setUser(data.user);
    return data;
  }

  function logout() {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
  }

  function getAuthHeaders() {
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }

  const value = {
    user,
    token,
    loading,
    authEnabled,
    isAuthenticated: !!user,
    login,
    logout,
    getAuthHeaders,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
