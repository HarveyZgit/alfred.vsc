import alfy, { ScriptFilterItem } from 'alfy';
import { GetFiles } from './files';

const databasePath = '/Users/bytedance/Library/Application Support/Code/User/globalStorage/state.vscdb';

async function getHistory(keyWord: string) {
  if (!keyWord) return [];
  const res = await GetFiles(databasePath);
  const search = res.filter(item => item.name.includes(keyWord));

  const outputContent = search.map<ScriptFilterItem>(item => ({
    title: item.name,
    subtitle: item.path,
    arg: item.path,
    icon: item.icon,
  }));

  return outputContent;
}

async function main() {
  const showList = await getHistory(alfy.input);
  alfy.output(showList);
}

main();
