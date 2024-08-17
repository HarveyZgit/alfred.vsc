import type { Workflow } from 'halfred-tools';

export type RecordType = 'file' | 'remote' | 'folder';

export interface RecordItem {
  __vsc_id__: string;
  name: string;
  path: string;
  type: RecordType;
  icon: Workflow.Icon;
  extra?: {
    from: string;
    [x: string]: any;
  };
}
