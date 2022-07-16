const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { exit } = require('process');

const configDir = path.join(__dirname, '../config');

const allRollupConfigs = fs
  .readdirSync(
    configDir,
    { withFileTypes: true, encoding: 'utf-8' },
  )
  .filter(item => item.isFile() && item.name.endsWith('.config.js'));

const { VSC_SCOPE_CONF } = process.env;
const buildScopes = VSC_SCOPE_CONF ? VSC_SCOPE_CONF.split(',') : [];

/**
 * @param {fs.Dirent} item
 */
function filterConfigWithScope(item) {
  if (buildScopes.length) {
    try {
      const [scope] = item.name.split('.');
      return buildScopes.includes(scope);
    } catch (error) {
      return false;
    }
  }
  return true;
}


const rollupConfigs = allRollupConfigs.filter(filterConfigWithScope);
console.log('[start building]', buildScopes.length ? rollupConfigs.map(item => item.name) : 'building all configs');

if (rollupConfigs.length) {
  rollupConfigs.forEach(item => {
    spawnSync('npx', ['rollup', '-c', `${configDir}/${item.name}`], { stdio: 'inherit' });
  });
} else {
  console.log('[Warning] No config need to build');
  exit(1);
}

