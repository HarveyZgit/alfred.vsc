import fs from 'fs';
import { noop } from 'lodash';

export function readFile<T>(path: string, parse: true, defaultValue?: T): T;
export function readFile<T>(
  path: string,
  parse: false,
  defaultValue?: T
): string;
export function readFile<T>(path: string, parse: boolean, defaultValue?: T) {
  const content = fs.readFileSync(path, { encoding: 'utf-8' });
  if (parse) {
    try {
      return JSON.parse(content) as T;
    } catch (error) {
      return defaultValue;
    }
  }

  return content;
}

export function writeFile<T>(path: string, content: T, sync: boolean) {
  const write = sync ? fs.writeFileSync : fs.writeFile;
  write.call(
    fs,
    path,
    JSON.stringify(content, null, 2),
    sync ? undefined : noop
  );
}
