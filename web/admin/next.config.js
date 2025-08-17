/** @type {import('next').NextConfig} */
const nextConfig = {
    reactStrictMode: true,
    swcMinify: true,
    env: {
        CUSTOM_KEY: process.env.CUSTOM_KEY || "default-value",
    },
    images: {
        domains: ['avatars.githubusercontent.com', 'images.unsplash.com'],
    },
    async rewrites() {
        return [
            {
                source: '/api/control-plane/:path*',
                destination: `${process.env.CONTROL_PLANE_API_URL || 'http://localhost:8080'}/api/v1/:path*`,
            },
            {
                source: '/api/analytics/:path*',
                destination: `${process.env.ANALYTICS_API_URL || 'http://localhost:8084'}/api/v1/:path*`,
            },
        ];
    },
    async headers() {
        return [
            {
                source: '/api/:path*',
                headers: [
                    { key: 'Access-Control-Allow-Credentials', value: 'true' },
                    { key: 'Access-Control-Allow-Origin', value: '*' },
                    { key: 'Access-Control-Allow-Methods', value: 'GET,OPTIONS,PATCH,DELETE,POST,PUT' },
                    { key: 'Access-Control-Allow-Headers', value: 'X-CSRF-Token, X-Requested-With, Accept, Accept-Version, Content-Length, Content-MD5, Content-Type, Date, X-Api-Version, Authorization' },
                ],
            },
        ];
    },
};

module.exports = nextConfig;
