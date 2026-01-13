import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import Layout from './components/Layout';
import Home from './pages/Home';
import Providers from './pages/Providers';
import Modules from './pages/Modules';
import ProviderDetail from './pages/ProviderDetail';
import ModuleDetail from './pages/ModuleDetail';
import Mirror from './pages/Mirror';
import Docs from './pages/Docs';
import Login from './pages/Login';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={
            <Layout>
              <Routes>
                <Route path="/" element={<Home />} />
                <Route path="/providers" element={<Providers />} />
                <Route path="/providers/:namespace/:name/:version" element={<ProviderDetail />} />
                <Route path="/modules" element={<Modules />} />
                <Route path="/modules/:namespace/:name/:provider/:version" element={<ModuleDetail />} />
                <Route path="/mirror" element={<Mirror />} />
                <Route path="/docs" element={<Docs />} />
              </Routes>
            </Layout>
          } />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
