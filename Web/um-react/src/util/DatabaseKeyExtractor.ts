import { getFileName } from './pathHelper';
import { SQLDatabase, SQLStatic, loadSQL } from './sqlite';
import { KuGou } from '@unlock-music/crypto';

export interface QMAndroidKeyEntry {
  name: string;
  ekey: string;
}

export type KugouKeyEntry = {
  audioHash: string;
  ekey: string;
};

export class DatabaseKeyExtractor {
  private static _instance: DatabaseKeyExtractor;

  static async getInstance() {
    if (!DatabaseKeyExtractor._instance) {
      DatabaseKeyExtractor._instance = new DatabaseKeyExtractor(await loadSQL());
    }
    return DatabaseKeyExtractor._instance;
  }

  constructor(private SQL: SQLStatic) {}

  private hasTable(db: SQLDatabase, name: string): boolean {
    const tables = db.exec('SELECT name FROM sqlite_master WHERE type="table"')[0]?.values.map((x) => x[0]);
    return tables?.includes(name) ?? false;
  }

  extractQmcV2KeysFromSqliteDb(buffer: ArrayBuffer): null | QMAndroidKeyEntry[] {
    let db: SQLDatabase | null = null;

    try {
      db = new this.SQL.Database(new Uint8Array(buffer));

      let sql: undefined | string;
      if (this.hasTable(db, 'audio_file_ekey_table')) {
        sql = 'select file_path, ekey from audio_file_ekey_table';
      } else if (this.hasTable(db, 'EKeyFileInfo')) {
        sql = 'select filePath, eKey from EKeyFileInfo';
      }
      if (!sql) return null;

      const result = db.exec(sql);
      if (result.length === 0) {
        return [];
      }

      const keys = result[0].values;
      return keys.map(([path, ekey]) => ({
        // strip dir name
        name: getFileName(String(path)),
        ekey: String(ekey),
      }));
    } finally {
      db?.close();
    }
  }

  extractKugouKeyFromEncryptedDb(buffer: ArrayBuffer): null | KugouKeyEntry[] {
    const dbBuffer = new Uint8Array(buffer);
    let db: SQLDatabase | null = null;

    try {
      const header = new TextDecoder().decode(dbBuffer.subarray(0, 16));
      if (header !== 'SQLite format 3\0') KuGou.decryptDatabase(dbBuffer);
      db = new this.SQL.Database(dbBuffer);

      const queries: string[] = [];
      if (this.hasTable(db, 'ShareFileItems')) {
        queries.push('SELECT EncryptionKeyId AS H, EncryptionKey AS K FROM ShareFileItems');
      }
      if (this.hasTable(db, 'DownloadItem')) {
        queries.push('SELECT EnHash AS H, EnKey AS K FROM DownloadItem');
      }
      if (!queries.length) return null;

      const result = db.exec(`SELECT H, K FROM (${queries.join(' UNION ALL ')})
        WHERE LENGTH(H) BETWEEN 1 AND 256 AND LENGTH(K) BETWEEN 1 AND 4096`);
      if (result.length === 0) {
        return [];
      }

      const keys = new Map<string, string>();
      for (const [hash, key] of result[0].values) {
        const audioHash = String(hash).normalize();
        if (!keys.has(audioHash)) keys.set(audioHash, String(key).normalize());
      }
      return Array.from(keys, ([audioHash, ekey]) => ({ audioHash, ekey }));
    } finally {
      db?.close();
    }
  }
}
