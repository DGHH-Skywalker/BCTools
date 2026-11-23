import { DecipherInstance, DecipherOK, DecipherResult, Status } from '~/decrypt-worker/Deciphers';
import { KuGou, KuGouHeader } from '@unlock-music/crypto';
import type { DecryptCommandOptions } from '~/decrypt-worker/types.ts';
import { chunkBuffer } from '~/decrypt-worker/util/buffer.ts';

export class KugouMusicDecipher implements DecipherInstance {
  cipherName = 'Kugou';

  async decrypt(buffer: Uint8Array, options: DecryptCommandOptions): Promise<DecipherResult | DecipherOK> {
    let kgm: KuGou | undefined;
    let kgmHdr: KuGouHeader | undefined;

    try {
      kgmHdr = new KuGouHeader(buffer.subarray(0, 0x400));
      const offset = kgmHdr.offsetToData;
      if (offset <= 0 || offset >= buffer.byteLength) {
        throw new Error('Invalid Kugou audio offset');
      }
      kgm = KuGou.fromHeaderV5(kgmHdr, options.kugouKey);

      const audioBuffer = new Uint8Array(buffer.subarray(offset));
      for (const [block, offset] of chunkBuffer(audioBuffer)) {
        kgm.decrypt(block, offset);
      }

      return Promise.resolve({
        status: Status.OK,
        cipherName: this.cipherName,
        data: audioBuffer,
      });
    } finally {
      kgmHdr?.free();
      kgm?.free();
    }
  }

  public static make(this: void) {
    return new KugouMusicDecipher();
  }
}
