import { useState } from 'react';
import { Link } from 'react-router-dom';

const sections = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
    ),
  },
  {
    id: 'configuration',
    title: 'Configuration',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
  {
    id: 'mirror',
    title: 'Provider Mirror',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
    ),
  },
  {
    id: 'upload',
    title: 'Upload Provider',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
      </svg>
    ),
  },
  {
    id: 'sync',
    title: 'Sync Schedules',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
  {
    id: 'import-export',
    title: 'Import / Export',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
      </svg>
    ),
  },
  {
    id: 'api',
    title: 'API Reference',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
      </svg>
    ),
  },
  {
    id: 'troubleshooting',
    title: 'Troubleshooting',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
  },
];

export default function Docs() {
  const [activeSection, setActiveSection] = useState('getting-started');

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex gap-8">
        {/* Sidebar */}
        <nav className="w-64 flex-shrink-0">
          <div className="sticky top-24">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Documentation</h2>
            <ul className="space-y-1">
              {sections.map((section) => (
                <li key={section.id}>
                  <button
                    onClick={() => setActiveSection(section.id)}
                    className={`w-full flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
                      activeSection === section.id
                        ? 'bg-blue-50 text-blue-700'
                        : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                    }`}
                  >
                    {section.icon}
                    {section.title}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </nav>

        {/* Content */}
        <main className="flex-1 min-w-0">
          <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
            {activeSection === 'getting-started' && <GettingStarted />}
            {activeSection === 'configuration' && <Configuration />}
            {activeSection === 'mirror' && <MirrorDocs />}
            {activeSection === 'upload' && <UploadDocs />}
            {activeSection === 'sync' && <SyncDocs />}
            {activeSection === 'import-export' && <ImportExportDocs />}
            {activeSection === 'api' && <ApiReference />}
            {activeSection === 'troubleshooting' && <Troubleshooting />}
          </div>
        </main>
      </div>
    </div>
  );
}

function SectionTitle({ children }) {
  return <h1 className="text-2xl font-bold text-gray-900 mb-6">{children}</h1>;
}

function SubSection({ title, children }) {
  return (
    <div className="mb-8">
      <h2 className="text-lg font-semibold text-gray-900 mb-3">{title}</h2>
      {children}
    </div>
  );
}

function CodeBlock({ children, title }) {
  return (
    <div className="mb-4">
      {title && <p className="text-sm text-gray-500 mb-2">{title}</p>}
      <pre className="bg-gray-900 text-gray-100 p-4 rounded-xl overflow-x-auto text-sm">
        <code>{children}</code>
      </pre>
    </div>
  );
}

function Note({ type = 'info', children }) {
  const styles = {
    info: 'bg-blue-50 border-blue-200 text-blue-800',
    warning: 'bg-amber-50 border-amber-200 text-amber-800',
    success: 'bg-green-50 border-green-200 text-green-800',
  };
  return (
    <div className={`p-4 rounded-lg border mb-4 ${styles[type]}`}>
      <p className="text-sm">{children}</p>
    </div>
  );
}

function GettingStarted() {
  return (
    <div>
      <SectionTitle>Getting Started</SectionTitle>
      
      <SubSection title="Overview">
        <p className="text-gray-600 mb-4">
          VC Terraform Registry is a private Terraform provider registry that allows you to host and manage 
          Terraform providers in your own infrastructure. It supports provider mirroring from the official 
          Terraform Registry, manual uploads, and offline deployments.
        </p>
      </SubSection>

      <SubSection title="Quick Start">
        <p className="text-gray-600 mb-4">1. Clone the repository:</p>
        <CodeBlock>git clone https://github.com/Veritas-Calculus/vc-terraform-registry.git &&
cd vc-terraform-registry</CodeBlock>

        <p className="text-gray-600 mb-4">2. Generate SSL certificates (required for Terraform):</p>
        <CodeBlock>{`mkdir -p certs
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \\
  -keyout certs/server.key -out certs/server.crt \\
  -subj "/CN=your-registry-host"`}</CodeBlock>

        <p className="text-gray-600 mb-4">3. Configure environment variables:</p>
        <CodeBlock>{`# .env
JWT_SECRET=your-secret-key
ADMIN_PASSWORD=your-admin-password
REGISTRY_HOST=your-registry-host`}</CodeBlock>

        <p className="text-gray-600 mb-4">4. Start the services:</p>
        <CodeBlock>docker compose up -d</CodeBlock>

        <p className="text-gray-600 mb-4">5. Access the registry at <code className="bg-gray-100 px-1.5 py-0.5 rounded">https://your-registry-host:3443</code></p>
      </SubSection>

      <SubSection title="Default Credentials">
        <Note type="warning">
          <strong>Important:</strong> Change the default admin password immediately after first login.
        </Note>
        <ul className="list-disc list-inside text-gray-600 space-y-1">
          <li>Username: <code className="bg-gray-100 px-1.5 py-0.5 rounded">admin</code></li>
          <li>Password: Configured via <code className="bg-gray-100 px-1.5 py-0.5 rounded">ADMIN_PASSWORD</code> environment variable</li>
        </ul>
      </SubSection>
    </div>
  );
}

function Configuration() {
  return (
    <div>
      <SectionTitle>Configuration</SectionTitle>

      <SubSection title="Environment Variables">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="text-left py-3 px-4 font-semibold text-gray-900">Variable</th>
                <th className="text-left py-3 px-4 font-semibold text-gray-900">Description</th>
                <th className="text-left py-3 px-4 font-semibold text-gray-900">Default</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              <tr>
                <td className="py-3 px-4"><code className="bg-gray-100 px-1.5 py-0.5 rounded">JWT_SECRET</code></td>
                <td className="py-3 px-4 text-gray-600">Secret key for JWT token signing</td>
                <td className="py-3 px-4 text-gray-500">Required</td>
              </tr>
              <tr>
                <td className="py-3 px-4"><code className="bg-gray-100 px-1.5 py-0.5 rounded">ADMIN_PASSWORD</code></td>
                <td className="py-3 px-4 text-gray-600">Initial admin password</td>
                <td className="py-3 px-4 text-gray-500">admin123</td>
              </tr>
              <tr>
                <td className="py-3 px-4"><code className="bg-gray-100 px-1.5 py-0.5 rounded">REGISTRY_HOST</code></td>
                <td className="py-3 px-4 text-gray-600">Public hostname of the registry</td>
                <td className="py-3 px-4 text-gray-500">localhost</td>
              </tr>
              <tr>
                <td className="py-3 px-4"><code className="bg-gray-100 px-1.5 py-0.5 rounded">UPSTREAM_URL</code></td>
                <td className="py-3 px-4 text-gray-600">Upstream registry URL</td>
                <td className="py-3 px-4 text-gray-500">https://registry.terraform.io</td>
              </tr>
              <tr>
                <td className="py-3 px-4"><code className="bg-gray-100 px-1.5 py-0.5 rounded">STORAGE_PATH</code></td>
                <td className="py-3 px-4 text-gray-600">Path to store provider files</td>
                <td className="py-3 px-4 text-gray-500">/data/providers</td>
              </tr>
            </tbody>
          </table>
        </div>
      </SubSection>

      <SubSection title="Terraform Client Configuration">
        <p className="text-gray-600 mb-4">Configure Terraform to use this registry as a network mirror:</p>
        <CodeBlock title="~/.terraformrc or terraform.rc">{`provider_installation {
  network_mirror {
    url = "https://your-registry-host:3443/"
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}`}</CodeBlock>

        <Note type="info">
          <strong>HTTPS Required:</strong> Terraform requires HTTPS for network mirrors. Make sure your registry has valid SSL certificates configured.
        </Note>
      </SubSection>

      <SubSection title="Using Provider Source">
        <p className="text-gray-600 mb-4">Alternatively, specify the provider source directly:</p>
        <CodeBlock>{`terraform {
  required_providers {
    aws = {
      source  = "your-registry-host:3443/hashicorp/aws"
      version = "~> 5.0"
    }
  }
}`}</CodeBlock>
      </SubSection>
    </div>
  );
}

function MirrorDocs() {
  return (
    <div>
      <SectionTitle>Provider Mirror</SectionTitle>

      <SubSection title="Overview">
        <p className="text-gray-600 mb-4">
          The mirror feature allows you to download providers from the official Terraform Registry 
          and store them locally. This is useful for offline environments or to ensure provider 
          availability.
        </p>
      </SubSection>

      <SubSection title="How to Mirror a Provider">
        <ol className="list-decimal list-inside text-gray-600 space-y-3 mb-4">
          <li>Navigate to the <Link to="/mirror" className="text-blue-600 hover:underline">Mirror</Link> page</li>
          <li>Sign in with your admin account</li>
          <li>Enter the provider namespace (e.g., <code className="bg-gray-100 px-1.5 py-0.5 rounded">hashicorp</code>)</li>
          <li>Enter the provider name (e.g., <code className="bg-gray-100 px-1.5 py-0.5 rounded">aws</code>)</li>
          <li>Optionally specify version, OS, and architecture filters</li>
          <li>Click "Mirror Provider"</li>
        </ol>
        <Note type="info">
          Leave the version field empty to mirror the latest version. Use "all" for OS/arch to mirror all platforms.
        </Note>
      </SubSection>

      <SubSection title="Using a Proxy">
        <p className="text-gray-600 mb-4">
          If your environment requires a proxy to access the internet, expand the "Advanced Options" 
          section and enter your proxy URL:
        </p>
        <CodeBlock>http://proxy.example.com:8080</CodeBlock>
      </SubSection>

      <SubSection title="Progress Tracking">
        <p className="text-gray-600 mb-4">
          When mirroring providers, you'll see real-time progress including:
        </p>
        <ul className="list-disc list-inside text-gray-600 space-y-1">
          <li>Current file being downloaded</li>
          <li>Download progress (bytes transferred)</li>
          <li>Estimated time remaining</li>
          <li>Overall completion percentage</li>
        </ul>
      </SubSection>
    </div>
  );
}

function UploadDocs() {
  return (
    <div>
      <SectionTitle>Upload Provider</SectionTitle>

      <SubSection title="Overview">
        <p className="text-gray-600 mb-4">
          You can manually upload provider binaries for custom providers or providers that 
          are not available in the official registry.
        </p>
      </SubSection>

      <SubSection title="File Requirements">
        <ul className="list-disc list-inside text-gray-600 space-y-1 mb-4">
          <li>File must be a ZIP archive</li>
          <li>ZIP should contain the provider binary</li>
          <li>Binary should follow Terraform provider naming convention</li>
        </ul>
        <Note type="info">
          The expected binary format is: <code className="bg-gray-100 px-1.5 py-0.5 rounded">terraform-provider-NAME_vVERSION</code>
        </Note>
      </SubSection>

      <SubSection title="Upload Steps">
        <ol className="list-decimal list-inside text-gray-600 space-y-3">
          <li>Go to the <Link to="/mirror" className="text-blue-600 hover:underline">Mirror</Link> page</li>
          <li>Sign in with your admin account</li>
          <li>Switch to the "Upload" tab</li>
          <li>Fill in the provider details:
            <ul className="list-disc list-inside ml-6 mt-2 space-y-1">
              <li>Namespace (your organization name)</li>
              <li>Provider name</li>
              <li>Version (semver format, e.g., 1.0.0)</li>
              <li>Target OS (linux, darwin, windows)</li>
              <li>Architecture (amd64, arm64)</li>
            </ul>
          </li>
          <li>Select the ZIP file</li>
          <li>Click "Upload"</li>
        </ol>
      </SubSection>
    </div>
  );
}

function SyncDocs() {
  return (
    <div>
      <SectionTitle>Sync Schedules</SectionTitle>

      <SubSection title="Overview">
        <p className="text-gray-600 mb-4">
          Sync schedules allow you to automatically keep your mirrored providers up-to-date 
          with the upstream registry. When a new version is released, the registry will 
          automatically download it.
        </p>
      </SubSection>

      <SubSection title="Creating a Schedule">
        <ol className="list-decimal list-inside text-gray-600 space-y-3">
          <li>Go to the <Link to="/mirror" className="text-blue-600 hover:underline">Mirror</Link> page</li>
          <li>Sign in with your admin account</li>
          <li>Switch to the "Schedules" tab</li>
          <li>Enter the provider namespace and name</li>
          <li>Select the sync interval:
            <ul className="list-disc list-inside ml-6 mt-2 space-y-1">
              <li>Hourly - Check every hour</li>
              <li>Daily - Check once per day</li>
              <li>Weekly - Check once per week</li>
            </ul>
          </li>
          <li>Click "Add Schedule"</li>
        </ol>
      </SubSection>

      <SubSection title="Managing Schedules">
        <p className="text-gray-600 mb-4">
          Existing schedules are displayed in a table. You can:
        </p>
        <ul className="list-disc list-inside text-gray-600 space-y-1">
          <li>View the last sync time and status</li>
          <li>See the next scheduled sync time</li>
          <li>Delete schedules you no longer need</li>
        </ul>
      </SubSection>
    </div>
  );
}

function ImportExportDocs() {
  return (
    <div>
      <SectionTitle>Import / Export</SectionTitle>

      <SubSection title="Export Providers">
        <p className="text-gray-600 mb-4">
          Export allows you to download a provider with all its platform binaries as a single 
          ZIP archive. This is useful for:
        </p>
        <ul className="list-disc list-inside text-gray-600 space-y-1 mb-4">
          <li>Transferring providers to air-gapped environments</li>
          <li>Backing up your provider collection</li>
          <li>Sharing providers between registries</li>
        </ul>
        <p className="text-gray-600 mb-4">To export:</p>
        <ol className="list-decimal list-inside text-gray-600 space-y-2">
          <li>Go to the Mirror page and view the provider details</li>
          <li>Click the "Export" button next to the version you want</li>
          <li>A ZIP file will be downloaded containing all platform binaries</li>
        </ol>
      </SubSection>

      <SubSection title="Import Providers">
        <p className="text-gray-600 mb-4">
          Import allows you to upload a previously exported provider package. The registry 
          will automatically extract and register all platform binaries.
        </p>
        <ol className="list-decimal list-inside text-gray-600 space-y-2">
          <li>Go to the <Link to="/mirror" className="text-blue-600 hover:underline">Mirror</Link> page</li>
          <li>Sign in with your admin account</li>
          <li>Use the "Import Provider" section</li>
          <li>Select the exported ZIP file</li>
          <li>Click "Import"</li>
        </ol>
        <Note type="info">
          The import feature expects a ZIP file in the same format as the export output. 
          It includes a manifest.json file with provider metadata.
        </Note>
      </SubSection>
    </div>
  );
}

function ApiReference() {
  return (
    <div>
      <SectionTitle>API Reference</SectionTitle>

      <SubSection title="Terraform Registry Protocol">
        <p className="text-gray-600 mb-4">
          This registry implements the Terraform Registry Protocol v1. The following endpoints are available:
        </p>
      </SubSection>

      <SubSection title="Service Discovery">
        <CodeBlock title="GET /.well-known/terraform.json">{`{
  "providers.v1": "/v1/providers/"
}`}</CodeBlock>
      </SubSection>

      <SubSection title="List Provider Versions">
        <CodeBlock title="GET /v1/providers/:namespace/:name/versions">{`{
  "versions": [
    {
      "version": "5.0.0",
      "protocols": ["5.0"],
      "platforms": [
        { "os": "linux", "arch": "amd64" },
        { "os": "darwin", "arch": "arm64" }
      ]
    }
  ]
}`}</CodeBlock>
      </SubSection>

      <SubSection title="Download Provider">
        <CodeBlock title="GET /v1/providers/:namespace/:name/:version/download/:os/:arch">{`{
  "protocols": ["5.0"],
  "os": "linux",
  "arch": "amd64",
  "filename": "terraform-provider-aws_5.0.0_linux_amd64.zip",
  "download_url": "https://registry/download/...",
  "shasums_url": "https://registry/shasums/...",
  "shasum": "abc123..."
}`}</CodeBlock>
      </SubSection>

      <SubSection title="Authentication">
        <p className="text-gray-600 mb-4">
          Administrative endpoints require JWT authentication. Obtain a token via:
        </p>
        <CodeBlock title="POST /api/auth/login">{`// Request
{
  "username": "admin",
  "password": "your-password"
}

// Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "username": "admin",
    "role": "admin"
  }
}`}</CodeBlock>
        <p className="text-gray-600 mt-4">
          Include the token in subsequent requests:
        </p>
        <CodeBlock>Authorization: Bearer eyJhbGciOiJIUzI1NiIs...</CodeBlock>
      </SubSection>
    </div>
  );
}

function Troubleshooting() {
  return (
    <div>
      <SectionTitle>Troubleshooting</SectionTitle>

      <SubSection title="SSL Certificate Issues">
        <p className="text-gray-600 mb-4">
          <strong>Problem:</strong> Terraform fails with certificate errors
        </p>
        <p className="text-gray-600 mb-4">
          <strong>Solution:</strong> For self-signed certificates, add the CA to your system trust store:
        </p>
        <CodeBlock title="Linux">{`sudo cp certs/server.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates`}</CodeBlock>
        <CodeBlock title="macOS">{`sudo security add-trusted-cert -d -r trustRoot \\
  -k /Library/Keychains/System.keychain certs/server.crt`}</CodeBlock>
      </SubSection>

      <SubSection title="Provider Not Found">
        <p className="text-gray-600 mb-4">
          <strong>Problem:</strong> Terraform cannot find a provider in the registry
        </p>
        <p className="text-gray-600 mb-4">
          <strong>Checklist:</strong>
        </p>
        <ul className="list-disc list-inside text-gray-600 space-y-1">
          <li>Verify the provider is mirrored (check the Mirror page)</li>
          <li>Confirm the version exists for your OS/architecture</li>
          <li>Check the terraformrc configuration is correct</li>
          <li>Ensure HTTPS is being used (not HTTP)</li>
        </ul>
      </SubSection>

      <SubSection title="Download Timeouts">
        <p className="text-gray-600 mb-4">
          <strong>Problem:</strong> Mirroring times out when downloading large providers
        </p>
        <p className="text-gray-600 mb-4">
          <strong>Solutions:</strong>
        </p>
        <ul className="list-disc list-inside text-gray-600 space-y-1">
          <li>Use a proxy with better connectivity</li>
          <li>Mirror specific OS/arch instead of "all"</li>
          <li>Check your network bandwidth and proxy settings</li>
        </ul>
      </SubSection>

      <SubSection title="Database Errors">
        <p className="text-gray-600 mb-4">
          <strong>Problem:</strong> SQLite database locked or corrupted
        </p>
        <p className="text-gray-600 mb-4">
          <strong>Solution:</strong> Restart the backend container:
        </p>
        <CodeBlock>docker compose restart backend</CodeBlock>
        <Note type="warning">
          If the database is corrupted, you may need to delete the database file and re-mirror your providers.
        </Note>
      </SubSection>

      <SubSection title="Logs">
        <p className="text-gray-600 mb-4">Check container logs for detailed error information:</p>
        <CodeBlock>{`# All services
docker compose logs -f

# Backend only
docker compose logs -f backend

# Nginx only
docker compose logs -f nginx`}</CodeBlock>
      </SubSection>
    </div>
  );
}
