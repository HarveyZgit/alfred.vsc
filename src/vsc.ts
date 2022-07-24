import path from 'path';
import { setup, workflow } from 'halfred-tools';
import { fmtSearchList } from './common/utils';
import { getRecordsFromVscodeMenu } from './records/getRecordsFromVscodeMenu';
import { filterRecordBySearchKey } from './records/utils';
import { vscliLogger, vscLogger } from './common/logger';

setup({
  logs: path.join(__dirname, './logs'),
});

async function main() {
  try {
    const outputList = workflow.input
      ? filterRecordBySearchKey(workflow.input)
      : getRecordsFromVscodeMenu();
    const result = fmtSearchList(outputList);

    workflow.output(result);
  } catch (err) {
    vscliLogger.error(err);
    workflow.output([
      {
        title: 'Some Error',
        subtitle: `See log ${vscLogger.logFile}`,
        arg: '',
      },
    ]);
  }
}

main();
