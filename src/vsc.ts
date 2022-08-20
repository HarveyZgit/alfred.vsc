import './setup';
import path from 'path';
import { setup, workflow } from 'halfred-tools';
import { fmtInput4JsonStringify, fmtSearchList } from './common/utils';
import { filterRecordBySearchKey } from './records/utils';
import { vscliLogger, vscLogger } from './common/logger';
import { userConfig } from './storage';
import { inputParser } from './input';

setup({
  logsDir: path.join(process.cwd(), './logs'),
});

userConfig.setup();

async function main() {
  try {
    const inputInfo = inputParser(workflow.input);

    vscLogger.info(
      `workflow.input, [original]=${workflow.input} [parsed]=${JSON.stringify(
        inputInfo,
        fmtInput4JsonStringify
      )}`
    );

    const outputList = filterRecordBySearchKey(inputInfo);
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
