import { IconElement, ScriptFilterItem } from 'alfy';
import { extname } from 'path';


export function getIcon(inputPath: string): IconElement {
  let iconFileName = 'file';

  if (inputPath.includes('remote')) {
    iconFileName = 'remote'
  } else if (!extname(inputPath)) {
    iconFileName = 'folder'
  }

  return {
    path: `./assets/${iconFileName}.png`,
  }
}

export interface SearchListItem {
  name: string;
  path: string;
  icon: IconElement;
};

export function fmtSearchList(list: SearchListItem[]) {
  return list.map<ScriptFilterItem>(item => ({
    title: item.name,
    subtitle: item.path,
    arg: item.path,
    icon: item.icon,
  }));
}
