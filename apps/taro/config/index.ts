import path from 'node:path';
import { defineConfig } from '@tarojs/cli';

export default defineConfig({
  projectName: 'taro-starter',
  designWidth: 750,
  deviceRatio: {
    640: 2.34 / 2,
    750: 1,
    375: 2,
    828: 1.81 / 2,
  },
  sourceRoot: 'src',
  outputRoot: 'dist',
  framework: 'react',
  compiler: 'vite',
  alias: {
    '@': path.resolve(__dirname, '..', 'src'),
  },
  mini: {},
  h5: {},
});
