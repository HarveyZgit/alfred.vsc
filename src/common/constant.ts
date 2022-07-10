import * as os from 'os';
import { get, has, merge, pick, values } from 'lodash';

/** vscode db 实例缓存 */
export const __VSC_DB_CACHE__ = '__VSC_DB_CACHE__';

export const HOME_PATH = os.homedir();

/** 环境变量 */
export const envNames = {
  IDEPath: 'VSC_IDE_PATH',
  globalStoragePath: 'VSC_IDE_GS_PATH',
  dbPath: 'VSC_IDE_DB_PATH',
  codeBin: 'VSC_CODE_BIN',
  watchDirectories: 'VSC_DIRECTORIES',
};

export const defaultEnvs: Record<string, any> = {
  [envNames.IDEPath]: `${HOME_PATH}/Library/Application Support/Code`,
  [envNames.codeBin]: '/usr/local/bin/code',
  [envNames.watchDirectories]: '',
}

defaultEnvs[envNames.globalStoragePath] = `${defaultEnvs[envNames.IDEPath]}/User/globalStorage`;
defaultEnvs[envNames.dbPath] = `${defaultEnvs[envNames.globalStoragePath]}/state.vscdb`;

export type EnvKeys = string;

export const envs = {
  get all() {
    const env = pick(process.env, values(envNames));
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
}
