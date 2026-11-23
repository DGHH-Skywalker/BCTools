import { useEffect, useRef, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';

import { ImportSecretModal } from '~/components/ImportSecretModal';

import { kugouAddKey, kugouClearKeys, kugouImportKeys } from '../settingsSlice';
import { selectStagingKugouV5Keys } from '../settingsSelector';
import type { StagingKugouKey } from '../keyFormats';
import { AddKey } from '~/components/AddKey.tsx';
import { KugouEKeyItem } from '~/features/settings/panels/Kugou/KugouEKeyItem.tsx';
import { KugouAllInstructions } from '~/features/settings/panels/Kugou/KugouAllInstructions.tsx';
import { parseAndroidKugouMMKV } from '~/util/mmkv/kugou.ts';
import { DatabaseKeyExtractor } from '~/util/DatabaseKeyExtractor.ts';
import { KeyListContainer } from '~/components/KeyListContainer';
import { toastImportResult } from '~/util/toastImportResult';
import { hasLocalDatabaseApi, kugouDatabaseRequest, type KugouDatabaseStatus } from '../kugouDatabase';

export function PanelKGGKey() {
  const dispatch = useDispatch();
  const kugouKeys = useSelector(selectStagingKugouV5Keys);
  const [showImportModal, setShowImportModal] = useState(false);
  const [database, setDatabase] = useState<KugouDatabaseStatus>();
  const [databaseBusy, setDatabaseBusy] = useState(false);
  const [databaseError, setDatabaseError] = useState('');

  useEffect(() => {
    if (!hasLocalDatabaseApi()) return;
    const request = new AbortController();
    void kugouDatabaseRequest<KugouDatabaseStatus>('status', 'GET', undefined, request.signal)
      .then(setDatabase)
      .catch(() => {});
    return () => request.abort();
  }, []);

  async function updateDatabase(method: 'POST' | 'DELETE') {
    setDatabaseBusy(true);
    setDatabaseError('');
    try {
      const result = await kugouDatabaseRequest<KugouDatabaseStatus>('source', method);
      if (result) setDatabase(result);
    } catch (error) {
      setDatabaseError(error instanceof Error ? error.message : '无法更新酷狗数据库');
    } finally {
      setDatabaseBusy(false);
    }
  }

  const addKey = () => dispatch(kugouAddKey());
  const clearAll = () => dispatch(kugouClearKeys());
  const handleSecretImport = async (file: File) => {
    let keys: Omit<StagingKugouKey, 'id'>[] | null = null;
    if (/mggkey_multi_process/i.test(file.name)) {
      keys = parseAndroidKugouMMKV(new DataView(await file.arrayBuffer()));
    } else if (/^KGMusicV3\.db$/i.test(file.name)) {
      try {
        const extractor = await DatabaseKeyExtractor.getInstance();
        keys = extractor.extractKugouKeyFromEncryptedDb(await file.arrayBuffer());
      } catch {
        toastImportResult(file.name, null);
        return;
      }
    }

    if (keys && keys.length > 0) {
      dispatch(kugouImportKeys(keys));
      setShowImportModal(false);
    }
    toastImportResult(file.name, keys);
  };

  const refKeyContainer = useRef<HTMLDivElement>(null);

  return (
    <div className="container flex flex-col grow min-h-0 w-full">
      <h2 className="text-2xl font-bold">酷狗解密密钥 (KGG / KGM v5)</h2>

      <p>酷狗已经升级了加密方式，现在最新版本的客户端使用 KGG / KGM v5 加密。</p>

      {database?.canSelect && (
        <section className="mt-3 mb-2 space-y-2">
          <h3 className="text-xl font-bold">自动解密</h3>
          <p>
            {database.custom ? '使用选定的数据库。' : 'BCTools 启动时会查找本机酷狗数据库。'}
            {database.keyCount > 0 && ` 已加载 ${database.keyCount} 个密钥。`}
          </p>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className="btn btn-sm"
              disabled={databaseBusy}
              onClick={() => void updateDatabase('POST')}
            >
              选择本机数据库
            </button>
            {database.custom && (
              <button
                type="button"
                className="btn btn-sm btn-ghost"
                disabled={databaseBusy}
                onClick={() => void updateDatabase('DELETE')}
              >
                恢复自动查找
              </button>
            )}
          </div>
          {databaseError && (
            <p role="alert" className="text-error">
              {databaseError}
            </p>
          )}
        </section>
      )}

      <h3 className="mt-2 text-xl font-bold">密钥管理</h3>
      <AddKey
        addKey={addKey}
        refContainer={refKeyContainer}
        importKeyFromFile={() => setShowImportModal(true)}
        clearKeys={clearAll}
      />

      <KeyListContainer ref={refKeyContainer} keys={kugouKeys}>
        {kugouKeys.map(({ id, audioHash, ekey }, i) => (
          <KugouEKeyItem key={id} id={id} ekey={ekey} audioHash={audioHash} i={i} />
        ))}
      </KeyListContainer>

      <ImportSecretModal
        clientName="酷狗音乐"
        show={showImportModal}
        onClose={() => setShowImportModal(false)}
        onImport={handleSecretImport}
      >
        <KugouAllInstructions />
      </ImportSecretModal>
    </div>
  );
}
