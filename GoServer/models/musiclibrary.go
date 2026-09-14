package models

import "strconv"

// WeekFileKind 枚举周文件夹里一个编号文件（NN.mp3）的内容类型。
const (
	// WeekFileSilent 静音占位（空时段，17.43s，标签为「空音频」）
	WeekFileSilent = "silent"
	// WeekFileSong 单首歌
	WeekFileSong = "song"
	// WeekFileMerged 同时段多首歌按顺序合并成一个 MP3
	WeekFileMerged = "merged"
)

// WeekFile 描述周文件夹里一个编号文件的当前内容，持久化在 data.json 的
// weeks 字段里：weeks["2026年第34周"]["05"] = {...}。
//
// 它就是歌曲文件的索引：编号 -> 文件内容（哪些歌 / 是否合并 / 如何还原）。
type WeekFile struct {
	Kind    string       `json:"kind"`
	SongIDs []int64      `json:"songIds,omitempty"`
	Parts   []MergedPart `json:"parts,omitempty"`
}

// MergedPart 记录合并文件中一个成员的还原信息：
//
//	还原文件 = ID3v2 标签 + 合并文件[Offset, Offset+Length) + ID3v1 标签
//
// 与 converter.MergeMP3s 的「只保留第一个文件的 ID3v2、剥掉所有 ID3v1」
// 拼接策略对称，因此任何成员都能无损拆回独立 MP3。
type MergedPart struct {
	SongID int64  `json:"songId"`
	ID3v2  string `json:"id3v2,omitempty"` // 原始 ID3v2 头（base64）
	ID3v1  string `json:"id3v1,omitempty"` // 原始 ID3v1 尾（base64）
	Offset int64  `json:"offset"`          // 裸音频帧在合并文件中的起始字节
	Length int64  `json:"length"`          // 裸音频帧长度
}

// Signature 返回用于增量同步的比较键：类型 + 成员歌曲列表。
// 只要内容没变（同一批歌、同样的合并顺序），就无需重建文件。
func (f WeekFile) Signature() string {
	sig := f.Kind
	for _, id := range f.SongIDs {
		sig += "|" + strconv.FormatInt(id, 10)
	}
	return sig
}
