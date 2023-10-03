/** @type {import('next').NextConfig} */
const nextConfig = {
    output: process.env.EXPORT ? 'export' : undefined,
    distDir: process.env.EXPORT ? 'dist' : undefined,

    rewrites: (!process.env.EXPORT) ? async function() {
        return [
            {
                source: "/api/:path*",
                destination: "http://localhost:3001/api/:path*",
            }
        ]
    } : undefined,
}

module.exports = nextConfig
