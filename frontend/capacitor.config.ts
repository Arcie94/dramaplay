import type { CapacitorConfig } from "@capacitor/cli";

const config: CapacitorConfig = {
  appId: "com.dramaplay.app",
  appName: "DramaPlay",
  webDir: "dist",
  server: {
    url: "https://dramaplay.online",
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
    SplashScreen: {
      launchShowDuration: 2000,
      launchAutoHide: true,
      backgroundColor: "#09090b", // zinc-950
      androidSplashResourceName: "splash",
      androidScaleType: "CENTER_CROP",
      showSpinner: true,
      spinnerColor: "#ef4444", // red-500
    },
  },
};

export default config;
