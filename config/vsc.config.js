import copy from 'rollup-plugin-copy';
import { external, input, plugins } from './share';

export default {
  input: input('src/vsc.ts'),
  output: {
    file: 'build/lib/vsc.js',
    format: 'cjs'
  },
  plugins: [
    ...plugins,
    copy({
      targets: [
        { src: 'public/*', dest: 'build' },
        { src: 'assets', dest: 'build' },
        { src: ['package.json', 'yarn.lock'], dest: 'build' },
      ]
    }),
  ],
  external
};
