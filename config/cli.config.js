import { external, input, plugins } from './share';

export default {
  input: input('src/cli/command.ts'),
  output: {
    file: 'build/cli.js',
    format: 'cjs'
  },
  plugins,
  external
};
