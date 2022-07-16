const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const configDir = path.join(__dirname, '../config');

const rollupConfigs = fs
  .readdirSync(
    configDir,
    { withFileTypes: true, encoding: 'utf-8' },
  )
  .filter(item => item.isFile() && item.name.endsWith('.config.js'));

const { VSC_SCOPE_CONF = '' } = process.env;
const buildScopes = VSC_SCOPE_CONF.split(',');

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

console.log('[start building]', rollupConfigs.filter(filterConfigWithScope).map(item => item.name));

rollupConfigs
  .filter(filterConfigWithScope)
  .forEach(item => {
    spawnSync('npx', ['rollup', '-c', `${configDir}/${item.name}`], { stdio: 'inherit' });
  });
