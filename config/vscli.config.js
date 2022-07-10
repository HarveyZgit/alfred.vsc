import { external, input, plugins } from './share';

export default {
  input: input('src/vscli.ts'),
  output: {
    file: 'build/vscli.js',
    format: 'cjs'
  },
  plugins,
  external
};
