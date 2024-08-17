import '../setup';
import { Command } from 'commander';
import { set, uniq, uniqBy } from 'lodash';
import pkg from '../../package.json';
import { filterByIgnorePatterns, findRecordByPath } from '../records/utils';
import { records } from '../storage';
import { vscliLogger } from '../common/logger';
import { passResultToAlfred, vscliResult } from '../common/constant';
import { RecordItem } from '../typings/records';

const program = new Command();

program.name(`${pkg.name}-cli`).version(pkg.version);

program
  .command('rebuild')
  .description('Rebuild records index')
  .option('--drop-all', 'rebuild index without save trash records')
  .action(async ({ dropAll }) => {
    vscliLogger.info('Start Rebuilding Index.');
    vscliLogger.info(`Rebuild Mode: ${dropAll ? 'drop all' : 'save trash'}`);
    try {
      await records.setup(dropAll);
      vscliLogger.info('Update Records Success!');
      passResultToAlfred(vscliResult.rebuildIndex.success);
    } catch (error) {
      passResultToAlfred(vscliResult.rebuildIndex.fail);
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
      uniqBy([item, ...allRecords], (record: RecordItem) => record.path)
    );
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
  });

program
  .command('ignore <pattern>')
  .description('Ignore some files or folders by pattern')
  .option(
    '-t, --ignoreByType',
    "the pattern will used to filter by record's type"
  )
  .action((pattern, { ignoreByType } = {}) => {
    try {
      vscliLogger.info(
        '[cli::ignore] start to exec ignore, ' +
          JSON.stringify({ pattern, ignoreByType })
      );

      if (!pattern) return;

      const allIgnorePatterns = records.getContent('ignorePatterns');

      if (ignoreByType) {
        allIgnorePatterns.type = allIgnorePatterns.type
          ? uniq([...allIgnorePatterns.type, pattern])
          : [pattern];
      } else {
        allIgnorePatterns.path = allIgnorePatterns.path
          ? uniq([...allIgnorePatterns.path, pattern])
          : [pattern];
      }

      vscliLogger.info(
        '[cli::ignore] start to save ignorePatterns' +
          JSON.stringify(allIgnorePatterns)
      );

      // save pattern
      records.update('ignorePatterns', allIgnorePatterns, true);

      vscliLogger.info('[cli::ignore] start to save ignorePatterns, success');

      const allRecords = records.getContent('records');
      if (!allRecords || !allRecords.length) {
        vscliLogger.info(
          '[cli::ignore] records are empty, skip filter records'
        );
        return;
      }

      const { records: nextRecords, needDeleteRecords } =
        filterByIgnorePatterns(allRecords);

      vscliLogger.info('[cli::ignore] start to update trash');

      // update trash
      records.update(
        'trash',
        (prevContent) => ({ ...prevContent, ...needDeleteRecords }),
        true
      );
      vscliLogger.info('[cli::ignore] start to update trash, success');

      vscliLogger.info('[cli::ignore] start to update records, success');

      // update records
      records.update('records', nextRecords);
      vscliLogger.info('[cli::ignore] start to update records, success');

      passResultToAlfred(vscliResult.ignore.success);
    } catch (error) {
      passResultToAlfred(vscliResult.ignore.fail);
      vscliLogger.error(error);
    }
  });

program.parse();
