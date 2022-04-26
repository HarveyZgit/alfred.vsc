import alfy from 'alfy';
import { GetFiles } from './files';

const databasePath = '/Users/bytedance/Library/Application Support/Code/User/globalStorage/state.vscdb';

async function main() {
  const res = await GetFiles(databasePath);

  alfy.output([
    {
      title: 'test',
      arg: JSON.stringify(res.slice(0, 10)),
    }
  ]);
}

main();

export default main;
