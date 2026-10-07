export type RecordItem = {
  id: string;
  name: string;
  kind: string;
  uri: string;
  path: string;
  status: string;
  branch: string;
  hidden: boolean;
  pinned: boolean;
  editor: string;
};
export type RecordsData = {
  page: number;
  pages: number;
  total: number;
  all: number;
  hidden: number;
  pinned: number;
  items: RecordItem[];
  warnings?: string[];
  index_stale: boolean;
  data_dir: string;
};
export type Settings = {
  roots: string[];
  editor: string;
  editors: Record<string, string>;
  managed: boolean;
};
export type Report = {
  source: string;
  applied: boolean;
  already_applied: boolean;
  directory_order_imported: number;
  records: number;
  hidden_imported: number;
  hidden_records: number;
  notes?: string[];
  unmigrated?: { original: string; how_to_migrate: string }[];
};
export type History = { report: Report; historical: boolean };

export type PanelActions = {
  active: boolean;
  closed: boolean;
  pending: string | null;
  act: (key: string, fn: () => Promise<void>) => Promise<void>;
  message: (text: string, error?: boolean) => void;
};
