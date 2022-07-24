import { Workflow } from 'halfred-tools';
import { extname } from 'path';

export function getIcon(inputPath: string): Workflow.Icon {
  let iconFileName = 'file';

  if (inputPath.includes('remote')) {
    iconFileName = 'remote';
  } else if (!extname(inputPath)) {
    iconFileName = 'folder';
  }

  return {
    path: `./assets/${iconFileName}.png`,
  };
}

export interface SearchListItem {
  name: string;
  path: string;
  icon: Workflow.Icon;
}

export function fmtSearchList(list: SearchListItem[]) {
  return list.map<Workflow.Item>((item) => ({
    title: item.name,
    subtitle: item.path,
    arg: item.path,
    icon: item.icon,
  }));
}
