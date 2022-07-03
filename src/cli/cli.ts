import alfy from 'alfy';
import { getRecordsFromSpecifiedDirectory } from '../records/getRecordsFromSpecifiedDirectory';

async function main() {
  try {
    if (alfy.input === '@update') {
      getRecordsFromSpecifiedDirectory(true);
      alfy.output([{
        title: 'update record success',
      }]);
      return;
    }
  } catch(err) {
    alfy.log(err);
  }
}

main();
