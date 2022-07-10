import { external, input, plugins } from './share';

export default {
  input: input('src/vsc.ts'),
  output: {
    file: 'build/vsc.js',
    format: 'cjs'
  },
  plugins,
  external
};
