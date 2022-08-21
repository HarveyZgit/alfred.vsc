import './setup';
import { workflow } from 'halfred-tools';
import { fmtInput4JsonStringify, fmtSearchList } from './common/utils';
import { filterRecordBySearchKey } from './records/utils';
import { vscliLogger, vscLogger } from './common/logger';
import { inputParser } from './input';

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
