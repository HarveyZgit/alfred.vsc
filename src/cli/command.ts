import '../setup';
import { Command } from 'commander';
import { uniqBy } from 'lodash';
import pkg from '../../package.json';
import { SearchListItem } from '../common/utils';
import { findRecordByPath } from '../records/utils';
import { records } from '../storage';
import { vscliLogger } from '../common/logger';
import { vscliResult } from '../common/constant';

const program = new Command();

program.name(`${pkg.name}-cli`).version(pkg.version);

program
  .command('rebuild')
  .description('Rebuild records index')
  .action(async () => {
    vscliLogger.info('Start Rebuilding Index.');
    try {
      await records.setup();
      vscliLogger.info('Update Records Success!');
      console.log(vscliResult.rebuildIndex.success);
    } catch (error) {
      console.log(vscliResult.rebuildIndex.fail);
      vscliLogger.error(error);
    }
  });

program
  .command('selected')
  .description('Do something after selected one record')
  .option('--record <filePath>', 'record file path')
  .action((params) => {
    const { record } = params;
    const [item, allRecords] = findRecordByPath(record || '');
    if (!record || !item) return;
    records.update(
      'records',
      uniqBy([item, ...allRecords], (record: SearchListItem) => record.path)
    );
  });

program
  .command('delete')
  .description('Delete one item in records.')
  .option('--record <filePath>', 'record file path')
  .action((params) => {
    const { record } = params;
    const [item, allRecords] = findRecordByPath(record || '');
    if (!record || !item) return;
    records.update(
      'records',
      allRecords.filter((item) => item.path !== record)
    );
  });

program.parse();
