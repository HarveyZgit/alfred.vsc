import alfy from 'alfy';
import { fmtSearchList } from './common/utils';
import { getRecordsFromVscodeMenu } from './records/getRecordsFromVscodeMenu';
import { filterRecordBySearchKey } from './records/utils';

async function main() {
  try {
    const outputList = alfy.input
      ? filterRecordBySearchKey(alfy.input)
      : getRecordsFromVscodeMenu();
    const result = fmtSearchList(outputList);

    alfy.output(result);
  } catch(err) {
    alfy.log(err);
  }
}

main();
