import * as os from 'os';
import { get, has, merge, omit, pick, values } from 'lodash';

/** vscode db 实例缓存 */
export const __VSC_DB_CACHE__ = '__VSC_DB_CACHE__';

export const HOME_PATH = os.homedir();

/** 环境变量 */
export const envNames = {
  IDEPath: 'VSC_IDE_PATH',
  watchDirectories: 'VSC_DIRECTORIES',

  // 以下环境变量不允许手动配置
  globalStoragePath: 'VSC_IDE_GS_PATH',
  databasePath: 'VSC_IDE_DB_PATH',
};

export const defaultEnvs: Record<string, any> = {
  [envNames.IDEPath]: `${HOME_PATH}/Library/Application Support/Code`,
  [envNames.watchDirectories]: '',
  get [envNames.globalStoragePath]() {
    return `${defaultEnvs[envNames.IDEPath]}/User/globalStorage`;
  },
  get [envNames.databasePath]() {
    return `${defaultEnvs[envNames.globalStoragePath]}/state.vscdb`;
  },
};

export type EnvKeys = string;

export const envs = {
  get all() {
    const allowConfEnvs = omit(process.env, [
      envNames.globalStoragePath,
      envNames.databasePath,
    ]);
    const env = pick(allowConfEnvs, values(envNames));
    return merge({}, defaultEnvs, env);
  },

  get(key: EnvKeys) {
    return get(envs.all, key);
  },

  getWithDefault(key: EnvKeys, dft?: any) {
    return get(envs.all, key, dft ?? get(envNames, key));
  },

  has(key: EnvKeys) {
    return has(envs.all, key);
  },

  hasUserConf(key: EnvKeys) {
    const { env } = process;
    return has(env, key);
  },
};

/** 用于 console 输出，并将结果传递给 alfred */
export const vscliResult = {
  rebuildIndex: {
    success: 'rebuild_index_success',
    fail: 'rebuild_index_fail',
  },
};

export const passResultToAlfred = console.log;
