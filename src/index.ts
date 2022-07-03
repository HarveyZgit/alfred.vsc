import alfy from 'alfy';
import { envNames, envs } from './constant';
import { fmtSearchList } from './utils';
import getDirectoryRecords, { updateDirectoryRecords } from './directory';
import { useSqlJs2SearchHistory } from './legacy/useSqlJs2SearchHistory';

function getHistory(keyWord: string) {
  const res = getDirectoryRecords();

  if (!keyWord) return fmtSearchList(res);;
  const search = res.filter(item => new RegExp(keyWord, 'i').test(item.name));
  return fmtSearchList(search);
}

async function main() {
  try {
    if (alfy.input === '@update') {
      updateDirectoryRecords();
      alfy.output([{
        title: 'update record success',
      }]);
      return;
    }

    let showList = [];
    if (envs.get(envNames.useVscodeRecent)) {
      showList = await useSqlJs2SearchHistory(alfy.input);
    } else {
      showList = getHistory(alfy.input);
    }

    alfy.output(showList);
  } catch(err) {
    alfy.log(err);
  }
}

main();
