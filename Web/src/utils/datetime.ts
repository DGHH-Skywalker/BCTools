// 时间工具单一入口。
//
// 此前有 12 处文件各自 `import isoWeek` 再 `dayjs.extend(isoWeek)`。dayjs.extend
// 本身幂等，但把插件注册散落各处意味着：谁忘了 extend，`isoWeek()` 就在运行时
// 静默返回 undefined。集中在这里注册一次，其余模块从本文件取 dayjs。
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"

dayjs.extend(isoWeek)

export { dayjs }
export type Dayjs = dayjs.Dayjs

/** 把日期字符串转为 ISO 周内序号：1=周一 … 7=周日。 */
export function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

/** 返回给定日期所在 ISO 周的 7 天，格式 YYYY-MM-DD。 */
export function weekDates(anchor: Dayjs): string[] {
  const start = anchor.startOf("isoWeek")
  return Array.from({ length: 7 }, (_, i) => start.add(i, "day").format("YYYY-MM-DD"))
}

/** 判断一组日期是否跨越了多个 ISO 周。 */
export function isMultiWeek(dates: string[]): boolean {
  return new Set(dates.map((d) => dayjs(d).isoWeek())).size > 1
}

/** 把 "HH:MM" 转为可比较的毫秒数，用于按时分先后排序。 */
export function parseTime(time: string): number {
  const [h, m] = (time || "").split(":").map(Number)
  return new Date(1970, 0, 1, h || 0, m || 0).getTime()
}
