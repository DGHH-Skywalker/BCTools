package converter

import "time"

// DefaultSilentPlaceholder 是空时段导出到 SD 卡时生成的静音占位时长。
//
// 为什么是 17.43s：换卡导出按「时段位置」编号，编号必须与贴出来的歌单一一对应；
// 整天没点歌时该天所有时段都生成静音占位（见 Web/src/utils/exportEntries.ts），
// 占位时长直接决定 SD 卡上一周文件的总时长，时长太短会让两首歌之间的间隔听起来
// 像被切了一截，太长又会让整周的曲序在播放器里显得拖沓。17.43s 是与实际广播录
// 音里两首歌之间的留白一致的固定值——**不允许改成 setting / 用户可配**，否则
// 老用户改完一次之后再升级会被覆盖，反而带来数据迁移问题。
const DefaultSilentPlaceholder = 17430 * time.Millisecond // 17.43s
