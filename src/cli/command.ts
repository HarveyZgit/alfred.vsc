import alfy from 'alfy';
import { Command } from 'commander';
import { uniqBy } from 'lodash';
import pkg from '../../package.json';
import { SearchListItem } from '../common/utils';
import { findRecordByPath } from '../records/utils';
import { records } from '../storage';

const program = new Command();

program
  .name(`${pkg.name}-cli`)
  .version(pkg.version);

program
  .command('rebuild')
  .description('Rebuild records index')
  .action(async () => {
    alfy.output([{
      title: 'Start Rebuilding Index.',
    }]);
    console.log('Start Rebuilding Index.');
    await records.setup();
    console.log('Update Records Success!');
    alfy.output([{
      title: 'Update Records Success!',
    }]);
  });

program
  .command('selected')
  .description('Do something after selected one record')
  .option('--record <filePath>')
  .action((params) => {
    const { record } = params;
    const [item, allRecords] = findRecordByPath(record || '');
    if (!record || !item) return;
    records.update(uniqBy([item, ...allRecords], (record: SearchListItem) => record.path));
  });

program.parse();
