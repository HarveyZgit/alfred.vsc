import alfy, { IconElement, ScriptFilterItem } from 'alfy';
import { values } from 'lodash';
import { EnvKeys, envNames, envs } from './constant';
import { getFiles } from './files';

interface SearchListItem {
  name: string;
  path: string;
  icon: IconElement;
};

function fmtSearchList(list: SearchListItem[]) {
  return list.map<ScriptFilterItem>(item => ({
    title: item.name,
    subtitle: item.path,
    arg: item.path,
    icon: item.icon,
  }));
}

async function getHistory(keyWord: string) {
  const recentCountLenth = +envs.getWithDefault(envNames.recentLength);
  const res = await getFiles(recentCountLenth);

  if (!keyWord) return fmtSearchList(res);;
  const search = res.filter(item => item.name.includes(keyWord));
  return fmtSearchList(search);
}

function checkEnvs() {
  const noConfEnvs = values(envNames).filter((name: EnvKeys) => !envs.has(name));
  return noConfEnvs.length ? Promise.reject(`Can not get envs: ${noConfEnvs.join(', ')}`) : Promise.resolve;
}

async function main() {
  try {
    await checkEnvs();

    const showList = await getHistory(alfy.input);
    alfy.output(showList);
  } catch(err) {
    alfy.log(err);
  }
}

main();
