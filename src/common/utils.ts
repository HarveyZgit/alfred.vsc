import { Workflow } from 'halfred-tools';
import fs from 'fs';
import { vscLogger } from './logger';

function isDirectory(input: string) {
  if (fs.existsSync(input)) {
    return fs.statSync(input).isDirectory();
  }
  return false;
}

export function getIcon(inputPath: string): Workflow.Icon {
  const pathWithoutScheme = inputPath.replace(/^(.*?):\/\//, '');
  let iconFileName = 'file';

  if (inputPath.startsWith('remote')) {
    iconFileName = 'remote';
  } else if (isDirectory(pathWithoutScheme)) {
    iconFileName = 'folder';
  }

  vscLogger.info(`${iconFileName} -- ${inputPath}`);

  return {
    path: `./assets/${iconFileName}.png`,
  };
}

export interface SearchListItem {
  name: string;
  path: string;
  icon: Workflow.Icon;
  extra?: {
    from: string;
    [x: string]: any;
  };
}

export function fmtSearchList(list: SearchListItem[]) {
  return list.map<Workflow.Item>((item) => ({
    title: item.name,
    subtitle: item.path,
    arg: item.path,
    icon: item.icon,
  }));
}
