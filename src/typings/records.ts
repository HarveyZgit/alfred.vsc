import type { Workflow } from 'halfred-tools';
import { GitInfo } from './gitInfo';

export type RecordType = 'file' | 'remote' | 'folder';

export interface RecordItem {
  __vsc_id__: string;
  name: string;
  path: string;
  pathWithoutProtocol: string;
  gitInfo: GitInfo | null;
  type: RecordType;
  icon: Workflow.Icon;
  extra?: {
    from: string;
    [x: string]: any;
  };
}
