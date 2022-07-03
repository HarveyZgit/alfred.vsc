/**
 * 通过 sql.js 读取 vscode 的记录，这样查询的结果会比较慢，但是结果会比较准确，且排序是按照最近打开的顺序排序的
 */

import { values } from 'lodash';
import { EnvKeys, envNames, envs } from '../constant';
import { getFiles } from '../files';
import { fmtSearchList } from '../utils';

function checkEnvs() {
  const noConfEnvs = values(envNames).filter((name: EnvKeys) => !envs.has(name));
  return noConfEnvs.length ? Promise.reject(`Can not get envs: ${noConfEnvs.join(', ')}`) : Promise.resolve;
}

export async function useSqlJs2SearchHistory(keyWord: string) {
  await checkEnvs();
  const recentCountLength = +envs.getWithDefault(envNames.recentLength);
  const res = await getFiles(recentCountLength);

  if (!keyWord) return fmtSearchList(res);;
  const search = res.filter(item => new RegExp(keyWord, 'i').test(item.name));
  return fmtSearchList(search);
}
