// @ts-check
import { defineConfig } from 'astro/config';

import tailwind from '@astrojs/tailwind';
import node from '@astrojs/node';

// https://astro.build/config
export default defineConfig({
  integrations: [tailwind(), node({
    mode: 'standalone'
  })],
  output: 'server',
  devToolbar: {
    enabled: false
  }
});
