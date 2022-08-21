import '../setup';
import { Command } from 'commander';
import { set, uniqBy } from 'lodash';
import pkg from '../../package.json';
import { SearchListItem } from '../common/utils';
import { findRecordByPath } from '../records/utils';
import { records } from '../storage';
import { vscLogger } from '../common/logger';
import { passResultToAlfred, vscliResult } from '../common/constant';

const program = new Command();

program.name(`${pkg.name}-cli`).version(pkg.version);

program
  .command('rebuild')
  .description('Rebuild records index')
  .option('--drop-all', 'rebuild index without save trash records')
  .action(async ({ dropAll }) => {
    vscLogger.info('Start Rebuilding Index.');
    vscLogger.info(`Rebuild Mode: ${dropAll ? 'drop all' : 'save trash'}`);
    try {
      await records.setup(dropAll);
      vscLogger.info('Update Records Success!');
      passResultToAlfred(vscliResult.rebuildIndex.success);
    } catch (error) {
      passResultToAlfred(vscliResult.rebuildIndex.fail);
      vscLogger.error(error);
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

    vscLogger.info(`select item ${item.__vsc_id__}`, {
      target: item,
    });
  });

program
  .command('delete')
  .description('Delete one item in records.')
  .option('--record <filePath>', 'record file path')
  .action((params) => {
    const { record: recordPath } = params;
    const [deleteTarget, allRecords] = findRecordByPath(recordPath || '');
    if (!recordPath || !deleteTarget) return;

    // update trash
    records.update(
      'trash',
      (prevContent) => set(prevContent, deleteTarget.__vsc_id__, deleteTarget),
      true
    );

    // update records
    records.update(
      'records',
      allRecords.filter((item) => item.__vsc_id__ !== deleteTarget.__vsc_id__)
    );

    vscLogger.info(`delete item ${deleteTarget.__vsc_id__}`, {
      target: deleteTarget,
    });
  });

program.parse();
