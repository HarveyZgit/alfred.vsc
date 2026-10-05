import { rspack } from '@rspack/core';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.dirname(fileURLToPath(import.meta.url));
export default {
  experiments: { css: true },
  mode: 'production',
  context: root,
  entry: './src/main.tsx',
  devtool: false,
  output: {
    path: path.resolve(root, '../internal/workflow/panel/dist'),
    filename: 'assets/app.[contenthash].js',
    publicPath: '/',
    clean: true,
  },
  resolve: { extensions: ['.tsx', '.ts', '.js'] },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        exclude: /node_modules/,
        loader: 'builtin:swc-loader',
        options: {
          jsc: {
            parser: { syntax: 'typescript', tsx: true },
            transform: { react: { runtime: 'automatic' } },
          },
        },
      },
      {
        test: /\.css$/,
        type: 'css',
        use: [
          {
            loader: 'postcss-loader',
            options: {
              postcssOptions: {
                plugins: {
                  tailwindcss: { config: path.resolve(root, 'tailwind.config.cjs') },
                  autoprefixer: {},
                },
              },
            },
          },
        ],
      },
    ],
  },
  plugins: [new rspack.HtmlRspackPlugin({ template: './index.html' })],
};
