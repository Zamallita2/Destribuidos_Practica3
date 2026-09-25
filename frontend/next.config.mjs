/** @type {import('next').NextConfig} */
const nextConfig = {
    eslint: { ignoreDuringBuilds: true },
    typescript: { ignoreBuildErrors: true },
    transpilePackages: ['three', 'three-stdlib'],
    async rewrites() {
        const apiTarget = process.env.API_PROXY_TARGET || 'http://localhost:8080';
        return [{ source: '/api/:path*', destination: `${apiTarget}/api/:path*` }];
    },
};

export default nextConfig;
