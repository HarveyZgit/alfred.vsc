import * as os from 'os';
import { get, has, merge, pick, values } from 'lodash';

/** vscode db 实例缓存 */
export const __VSC_DB_CACHE__ = '__VSC_DB_CACHE__';

export const HOME_PATH = os.homedir();

/** 环境变量 */
export const envNames = {
  dbPath: 'VSC_DB_PATH',
  codeBin: 'VSC_CODE_BIN',
};

export const defaultEnvs = {
  [envNames.dbPath]: `${HOME_PATH}/Library/Application Support/Code/User/globalStorage/state.vscdb`,
  [envNames.codeBin]: '/usr/local/bin/code',
}

export type EnvKeys = string;

export const envs = {
  get all() {
    const env = pick(process.env, values(envNames));
    return merge({}, defaultEnvs, env);
  },

  get(key: EnvKeys) {
    return get(envs.all, key);
  },

  has(key: EnvKeys) {
    return has(envs.all, key);
  },

  hasUserConf(key: EnvKeys) {
    const { env } = process;
    return has(env, key);
  },
}
