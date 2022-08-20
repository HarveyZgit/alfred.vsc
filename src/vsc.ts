import './setup';
import path from 'path';
import { setup, workflow } from 'halfred-tools';
import { fmtSearchList } from './common/utils';
import { filterRecordBySearchKey } from './records/utils';
import { vscliLogger, vscLogger } from './common/logger';
import { userConfig } from './storage';

setup({
  logsDir: path.join(process.cwd(), './logs'),
});

userConfig.setup();

async function main() {
  try {
    vscLogger.info(`workflow.input, ${workflow.input}`);

    const outputList = filterRecordBySearchKey(workflow.input);
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
