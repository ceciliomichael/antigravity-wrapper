import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/admin/:path*",
        destination: "http://localhost:3045/admin/:path*",
      },
      {
        source: "/v1/:path*",
        destination: "http://localhost:3045/v1/:path*",
      },
    ];
  },
};

export default nextConfig;
