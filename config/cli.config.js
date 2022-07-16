import { external, input, plugins } from './share';

export default {
  input: input('src/cli/command.ts'),
  output: {
    file: 'build/lib/cli.js',
    format: 'cjs'
  },
  plugins,
  external
};
