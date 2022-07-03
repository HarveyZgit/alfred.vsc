import fs from 'fs';
import { find, get } from 'lodash';
import { envNames, envs } from '../common/constant';
import { getIcon, SearchListItem } from '../common/utils';

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

export function getVscodeMenuItemUriPath(uri: FileMenuItemUri | RemoteMenuItemUri) {
  const { path, scheme, external } = uri;
  if (external) return external;
  if (scheme === 'file') return `${scheme}://${path}`;
  if (scheme === 'vscode-remote') return `${scheme}://${encodeURIComponent(uri.authority)}${path}`;
  return '';
}

function getFallbackName(path?: string) {
  if (!path) return 'Unknown Name';
  const lastIdx = path.lastIndexOf('/');
  return lastIdx > -1 ? path.slice(lastIdx + 1) : path;
}

export function getRecordsFromVscodeMenu() {
  try {
    const content = fs.readFileSync(
      `${envs.get(envNames.globalStoragePath)}/storage.json`,
      { encoding: 'utf-8' }
    );
    const config = JSON.parse(content);
    const fileMenusItems = get(config, 'lastKnownMenubarData.menus.File.items', []);
    const recentFolderConfig = find(fileMenusItems, item => item.id === 'submenuitem.36');
    const recentFolderList = get(recentFolderConfig, 'submenu.items', []) as any[];
    const openRecentFolderList = recentFolderList
      .filter(item => item.id === 'openRecentFolder' && item.uri)
      .map(item => {
        const path = getVscodeMenuItemUriPath(item.uri);
        return {
          name: getFallbackName(path),
          path,
          icon: getIcon(path),
        } as SearchListItem;
      });
    return openRecentFolderList;
  } catch {
    return [];
  }
}
