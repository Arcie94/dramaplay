import { CapacitorConfig } from "@capacitor/cli";

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
};

export default config;
