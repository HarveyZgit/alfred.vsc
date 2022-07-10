import alfy from 'alfy';
import { fmtSearchList } from './common/utils';
import { filterRecordBySearchKey } from './records/utils';

async function main() {
  try {
    const outputList = filterRecordBySearchKey(alfy.input);
    const result = fmtSearchList(outputList);

    alfy.output(result);
  } catch(err) {
    alfy.log(err);
  }
}

main();
