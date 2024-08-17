import fs from 'fs';
import { find, get } from 'lodash';
import { envNames, envs } from '../common/constant';
import { genRecordId, getIcon, getRecordType } from '../common/utils';
import { RecordItem } from '../typings/records';

interface BaseMenuItemUri {
  path: string;
  scheme: 'file' | 'vscode-remote';
  external?: string;
}

interface FileMenuItemUri extends BaseMenuItemUri {
  scheme: 'file';
}

interface RemoteMenuItemUri extends BaseMenuItemUri {
  scheme: 'vscode-remote';
  authority: string;
}

export function getVscodeMenuItemUriPath(
  uri: FileMenuItemUri | RemoteMenuItemUri
) {
  const { path, scheme, external } = uri;
  if (external) return external;
  if (scheme === 'file') return `${scheme}://${path}`;
  if (scheme === 'vscode-remote') return `${scheme}://${uri.authority}${path}`;
  return '';
}

function getFallbackName(path?: string) {
  if (!path) return 'Unknown Name';
  const lastIdx = path.lastIndexOf('/');
  return lastIdx > -1 ? path.slice(lastIdx + 1) : path;
}

interface MenuItem {
  id: string;
  name: string;
  submenu?: {
    items: MenuItem[];
  };
}

function isVscodeRecentOpenMenu(item: MenuItem) {
  const { id, submenu } = item;
  if (!id || !id.startsWith('submenuitem')) return false;
  if (!submenu || !submenu.items) return false;

  for (let i = 0; i < submenu.items.length; i++) {
    const item = submenu.items[i];
    if (item.id === 'openRecentFolder') return true;
  }
  return false;
}

export function getRecordsFromVscodeMenu(): RecordItem[] {
  try {
    const content = fs.readFileSync(
      `${envs.get(envNames.globalStoragePath)}/storage.json`,
      { encoding: 'utf-8' }
    );
    const config = JSON.parse(content);
    const fileMenusItems = get(
      config,
      'lastKnownMenubarData.menus.File.items',
      []
    );
    const recentFolderConfig = find(fileMenusItems, isVscodeRecentOpenMenu);
    const recentFolderList = get(
      recentFolderConfig,
      'submenu.items',
      []
    ) as any[];
    const openRecentFolderList = recentFolderList
      .filter((item) => item.id === 'openRecentFolder' && item.uri)
      .map((item) => {
        const path = getVscodeMenuItemUriPath(item.uri);
        return {
          __vsc_id__: genRecordId(path),
          name: getFallbackName(path),
          path,
          type: getRecordType(path),
          icon: getIcon(path),
          extra: {
            from: 'getRecordsFromVscodeMenu',
          },
        } as RecordItem;
      });
    return openRecentFolderList;
  } catch {
    return [];
  }
}
