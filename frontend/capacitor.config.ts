import type { CapacitorConfig } from "@capacitor/cli";

const config: CapacitorConfig = {
  appId: "com.dramaplay.app",
  appName: "DramaPlay",
  webDir: "dist",
  server: {
    url: "http://10.0.2.2:4321",
    cleartext: true,
  },
  android: {
    allowMixedContent: true,
  },
  plugins: {
    GoogleAuth: {
      scopes: ["profile", "email"],
      serverClientId: "948421850128-kh10okq8tvc2rnl6vd4d460s1r3r7vir.apps.googleusercontent.com",
      forceCodeForRefreshToken: true,
    },
  },
};

export default config;
