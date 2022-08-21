import path from 'path';
import { setup } from 'halfred-tools';
import { userConfig } from './storage';
import { vscLogger, vscliLogger } from './common/logger';

setup({
  logsDir: path.join(process.cwd(), './logs'),
});

vscLogger.gcStart();
vscliLogger.gcStart();

userConfig.setup();
