const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const configDir = path.join(__dirname, '../config');

const rollupConfigs = fs.readdirSync(
  configDir,
  { withFileTypes: true, encoding: 'utf-8' },
);

rollupConfigs
  .filter(item => item.isFile() && item.name.endsWith('.config.js'))
  .forEach(item => {
    spawnSync('npx', ['rollup', '-c', `${configDir}/${item.name}`], { stdio: 'inherit' });
  });
