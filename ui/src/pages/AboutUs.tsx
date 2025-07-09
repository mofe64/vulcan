import React from "react";

const AboutUs: React.FC = () => {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-700">
      {/* Header */}
      <div className="bg-white/10 backdrop-blur-sm border-b border-white/20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <h1 className="text-2xl font-bold text-white">Vulkan Platform</h1>
            <nav className="flex space-x-8">
              <a
                href="/login"
                className="text-white/80 hover:text-white transition-colors"
              >
                Login
              </a>
              <a href="/about" className="text-white font-semibold">
                About Us
              </a>
            </nav>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-16">
          <h2 className="text-5xl font-bold text-white mb-6">
            About Vulkan Platform
          </h2>
          <p className="text-xl text-white/80 max-w-3xl mx-auto">
            Empowering organizations with a comprehensive Kubernetes-native
            platform for seamless application deployment, management, and
            governance.
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8 mb-16">
          <div className="bg-white/10 backdrop-blur-sm rounded-xl p-8 border border-white/20">
            <div className="w-12 h-12 bg-blue-500 rounded-lg flex items-center justify-center mb-6">
              <svg
                className="w-6 h-6 text-white"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-4">
              Lightning Fast Deployment
            </h3>
            <p className="text-white/70">
              Deploy applications with unprecedented speed using our optimized
              CI/CD pipelines and Kubernetes-native architecture.
            </p>
          </div>

          <div className="bg-white/10 backdrop-blur-sm rounded-xl p-8 border border-white/20">
            <div className="w-12 h-12 bg-green-500 rounded-lg flex items-center justify-center mb-6">
              <svg
                className="w-6 h-6 text-white"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-4">
              Enterprise Security
            </h3>
            <p className="text-white/70">
              Built with security-first principles, featuring OPA policies,
              RBAC, and comprehensive audit trails for enterprise-grade
              protection.
            </p>
          </div>

          <div className="bg-white/10 backdrop-blur-sm rounded-xl p-8 border border-white/20">
            <div className="w-12 h-12 bg-purple-500 rounded-lg flex items-center justify-center mb-6">
              <svg
                className="w-6 h-6 text-white"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-4">
              Multi-Cluster Management
            </h3>
            <p className="text-white/70">
              Manage multiple Kubernetes clusters from a single interface with
              advanced orchestration and monitoring capabilities.
            </p>
          </div>
        </div>

        {/* Mission Section */}
        <div className="bg-white/10 backdrop-blur-sm rounded-xl p-12 border border-white/20 mb-16">
          <div className="max-w-4xl mx-auto text-center">
            <h3 className="text-3xl font-bold text-white mb-8">Our Mission</h3>
            <p className="text-lg text-white/80 leading-relaxed">
              We believe that modern application deployment should be seamless,
              secure, and scalable. Vulkan Platform bridges the gap between
              development and operations, providing teams with the tools they
              need to deliver value faster while maintaining the highest
              standards of security and reliability.
            </p>
          </div>
        </div>

        {/* Technology Stack */}
        <div className="text-center">
          <h3 className="text-3xl font-bold text-white mb-12">
            Built with Modern Technologies
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-8">
            <div className="flex flex-col items-center">
              <div className="w-16 h-16 bg-white/10 rounded-lg flex items-center justify-center mb-4">
                <span className="text-white font-bold text-sm">K8s</span>
              </div>
              <span className="text-white/70 text-sm">Kubernetes</span>
            </div>
            <div className="flex flex-col items-center">
              <div className="w-16 h-16 bg-white/10 rounded-lg flex items-center justify-center mb-4">
                <span className="text-white font-bold text-sm">Go</span>
              </div>
              <span className="text-white/70 text-sm">Golang</span>
            </div>
            <div className="flex flex-col items-center">
              <div className="w-16 h-16 bg-white/10 rounded-lg flex items-center justify-center mb-4">
                <span className="text-white font-bold text-sm">React</span>
              </div>
              <span className="text-white/70 text-sm">React</span>
            </div>
            <div className="flex flex-col items-center">
              <div className="w-16 h-16 bg-white/10 rounded-lg flex items-center justify-center mb-4">
                <span className="text-white font-bold text-sm">OPA</span>
              </div>
              <span className="text-white/70 text-sm">Open Policy Agent</span>
            </div>
            <div className="flex flex-col items-center">
              <div className="w-16 h-16 bg-white/10 rounded-lg flex items-center justify-center mb-4">
                <span className="text-white font-bold text-sm">Helm</span>
              </div>
              <span className="text-white/70 text-sm">Helm Charts</span>
            </div>
            <div className="flex flex-col items-center">
              <div className="w-16 h-16 bg-white/10 rounded-lg flex items-center justify-center mb-4">
                <span className="text-white font-bold text-sm">Tekton</span>
              </div>
              <span className="text-white/70 text-sm">Tekton Pipelines</span>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="bg-white/10 backdrop-blur-sm border-t border-white/20 mt-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="text-center">
            <p className="text-white/60">
              © 2024 Vulkan Platform. Built with ❤️ for the Kubernetes
              community.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AboutUs;
