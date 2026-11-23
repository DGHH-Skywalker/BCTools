package kugoudb

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"testing/fstest"

	_ "modernc.org/sqlite"
	"modernc.org/sqlite/vfs"
	"unlock-music.dev/cli/algo/kgm/pc_kugou_db"
)

// The Windows database page format follows unlock-music.dev/cli/algo/kgm/pc_kugou_db.
// Copyright (c) 2020-2021 Unlock Music. Use under the MIT license in LICENSE.unlock-music.
func decryptPages(data []byte) error {
	if bytes.HasPrefix(data, pc_kugou_db.SQLITE_HEADER) {
		return nil
	}
	const pageSize = pc_kugou_db.PAGE_SIZE
	if len(data) < pageSize || len(data)%pageSize != 0 {
		return errors.New("不支持的酷狗数据库页大小")
	}
	header := [8]byte(data[16:24])
	if !bytes.Equal(header[4:], []byte{0x00, 0x40, 0x20, 0x20}) {
		return errors.New("未识别到酷狗数据库")
	}
	copy(data[16:24], data[8:16])
	decryptPage(data[16:pageSize], 1)
	if !bytes.Equal(data[16:24], header[:]) {
		return errors.New("酷狗数据库首页校验失败")
	}
	copy(data, pc_kugou_db.SQLITE_HEADER)
	for start := pageSize; start < len(data); start += pageSize {
		decryptPage(data[start:start+pageSize], uint32(start/pageSize+1))
	}
	return nil
}

func decryptPage(page []byte, index uint32) {
	block, iv := pageCipher(index)
	cipher.NewCBCDecrypter(block, iv[:]).CryptBlocks(page, page)
}

func pageCipher(index uint32) (cipher.Block, [16]byte) {
	seed := [24]byte(pc_kugou_db.DEFAULT_MASTER_KEY)
	binary.LittleEndian.PutUint32(seed[16:20], index)
	key := md5.Sum(seed[:])
	var ivSeed [16]byte
	value := index + 1
	for offset := 0; offset < len(ivSeed); offset += 4 {
		value = value*0x9ef4 - (value/0xce26)*0x7fffff07
		if value&0x80000000 != 0 {
			value += 0x7fffff07
		}
		binary.LittleEndian.PutUint32(ivSeed[offset:], value)
	}
	iv := md5.Sum(ivSeed[:])
	block, _ := aes.NewCipher(key[:])
	return block, iv
}

func extractKeys(ctx context.Context, data []byte) (map[string]string, error) {
	if err := decryptPages(data); err != nil {
		return nil, err
	}
	// Keep the decrypted bytes in memory. A read-only VFS also retains the
	// source page size without sqlite3_deserialize changing pager state.
	name, filesystem, err := vfs.New(fstest.MapFS{"kugou.db": {Data: data}})
	if err != nil {
		return nil, err
	}
	defer filesystem.Close()
	db, err := sql.Open("sqlite", "file:kugou.db?mode=ro&immutable=1&vfs="+name)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	keys := make(map[string]string)
	var failures error
	valid := false
	for _, query := range []string{
		`SELECT EncryptionKeyId, EncryptionKey FROM ShareFileItems
		 WHERE length(EncryptionKeyId) BETWEEN 1 AND 256 AND length(EncryptionKey) BETWEEN 1 AND 4096`,
		`SELECT EnHash, EnKey FROM DownloadItem
		 WHERE length(EnHash) BETWEEN 1 AND 256 AND length(EnKey) BETWEEN 1 AND 4096`,
	} {
		if err := readKeys(ctx, db, query, keys); err != nil {
			failures = errors.Join(failures, err)
		} else {
			valid = true
		}
	}
	if !valid {
		return nil, fmt.Errorf("未找到可读取的酷狗密钥表: %w", failures)
	}
	return keys, nil
}

func readKeys(ctx context.Context, db *sql.DB, query string, keys map[string]string) error {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var hash, key string
		if err := rows.Scan(&hash, &key); err != nil {
			return err
		}
		if _, found := keys[hash]; !found {
			keys[hash] = key
		}
	}
	return rows.Err()
}
