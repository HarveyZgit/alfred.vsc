import { get, has, set } from "lodash";

type Key = string | symbol;

class Store {
  _store: Record<Key, any>;

  get store() {
    return this._store;
  }

  constructor() {
    this._store = {};
  }

  get<T = any>(key: Key, defaultValue?: T): T {
    return get(this._store, key, defaultValue);
  }

  set<T = any>(key: Key, value: T): T {
    set(this._store, key, value);
    return value;
  }

  has(key: Key) {
    return has(this._store, key);
  }
}

export const store = new Store();
