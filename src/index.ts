import alfy from 'alfy';
import { fmtSearchList } from './common/utils';
import { records } from './storage';

async function main() {
  try {
    const originalRecords = records.getContent();

    const outputList = alfy.input
      ? originalRecords.filter(item => new RegExp(alfy.input, 'i').test(item.name))
      : originalRecords;

    const result = fmtSearchList(outputList);

    alfy.output(result);
  } catch(err) {
    alfy.log(err);
  }
}

main();
