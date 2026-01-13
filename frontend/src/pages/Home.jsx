import React from 'react';
import { Link } from 'react-router-dom';
import SearchBar from '../components/SearchBar';

// SVG Icons
const icons = {
  lock: (
    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
    </svg>
  ),
  package: (
    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  ),
  rocket: (
    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15.59 14.37a6 6 0 01-5.84 7.38v-4.8m5.84-2.58a14.98 14.98 0 006.16-12.12A14.98 14.98 0 009.631 8.41m5.96 5.96a14.926 14.926 0 01-5.841 2.58m-.119-8.54a6 6 0 00-7.381 5.84h4.8m2.581-5.84a14.927 14.927 0 00-2.58 5.84m2.699 2.7c-.103.021-.207.041-.311.06a15.09 15.09 0 01-2.448-2.448 14.9 14.9 0 01.06-.312m-2.24 2.39a4.493 4.493 0 00-1.757 4.306 4.493 4.493 0 004.306-1.758M16.5 9a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0z" />
    </svg>
  ),
  sync: (
    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
    </svg>
  ),
  search: (
    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
  ),
  book: (
    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
    </svg>
  ),
};

function Home() {
  return (
    <div>
      <section className="bg-gradient-to-b from-white to-gray-50 py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h1 className="text-5xl font-bold text-gray-900 mb-6">
              Private Terraform Registry
            </h1>
            <p className="text-xl text-gray-600 mb-8 max-w-3xl mx-auto">
              Enterprise-grade provider and module management for your infrastructure as code
            </p>
            <SearchBar />
          </div>
        </div>
      </section>

      <section className="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          <FeatureCard
            icon={icons.lock}
            title="Private & Secure"
            description="Host your providers and modules in a secure, private environment"
          />
          <FeatureCard
            icon={icons.package}
            title="Offline Deployment"
            description="Deploy in fully offline environments without external dependencies"
          />
          <FeatureCard
            icon={icons.rocket}
            title="Easy Setup"
            description="Get started quickly with Docker Compose one-click deployment"
          />
          <FeatureCard
            icon={icons.sync}
            title="Version Control"
            description="Complete version management and history tracking"
          />
          <FeatureCard
            icon={icons.search}
            title="Fast Search"
            description="Quickly discover and search through providers and modules"
          />
          <FeatureCard
            icon={icons.book}
            title="Full Documentation"
            description="Built-in documentation system with Markdown support"
          />
        </div>
      </section>

      <section className="bg-gray-50 py-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <h2 className="text-3xl font-bold text-gray-900 text-center mb-12">
            Getting Started
          </h2>
          <div className="bg-white rounded-lg shadow-sm p-8 max-w-3xl mx-auto">
            <div className="space-y-6">
              <Step
                number="1"
                title="Clone the repository"
                code="git clone https://github.com/Veritas-Calculus/vc-terraform-registry.git"
              />
              <Step
                number="2"
                title="Start the service"
                code="docker-compose up -d"
              />
              <Step
                number="3"
                title="Access the registry"
                description="Open your browser and navigate to http://localhost:8080"
              />
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}

function FeatureCard({ icon, title, description }) {
  return (
    <div className="bg-white rounded-xl p-6 shadow-sm hover:shadow-md transition-shadow">
      <div className="text-blue-600 mb-4">{icon}</div>
      <h3 className="text-xl font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-600">{description}</p>
    </div>
  );
}

function Step({ number, title, code, description }) {
  return (
    <div className="flex gap-4">
      <div className="flex-shrink-0">
        <div className="w-8 h-8 bg-primary-600 text-white rounded-full flex items-center justify-center font-semibold">
          {number}
        </div>
      </div>
      <div className="flex-1">
        <h4 className="text-lg font-semibold text-gray-900 mb-2">{title}</h4>
        {code && (
          <pre className="bg-gray-50 rounded-lg p-4 overflow-x-auto">
            <code className="text-sm text-gray-800">{code}</code>
          </pre>
        )}
        {description && <p className="text-gray-600">{description}</p>}
      </div>
    </div>
  );
}

export default Home;
