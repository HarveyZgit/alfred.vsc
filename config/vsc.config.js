import { defineConfig } from 'rollup';
import { external, input, plugins } from './share';

export default defineConfig({
  input: input('src/vsc.ts'),
  output: {
    file: 'build/lib/vsc.js',
    format: 'cjs',
  },
  plugins,
  external,
});
