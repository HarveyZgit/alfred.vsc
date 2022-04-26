import * as path from 'path';
import rollupTypescript from 'rollup-plugin-typescript2'

const buildTsConfigPath = path.join(__dirname, './tsconfig.build.json');

export default {
  input: 'src/index.ts',
  output: {
    file: 'build/index.js',
    format: 'cjs'
  },
  plugins: [
    rollupTypescript({
      tsconfig: buildTsConfigPath,
    }),
  ],
  external: ['alfy', 'fs', 'sql.js', 'lodash', 'os', 'path'],
};
