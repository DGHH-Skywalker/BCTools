import { computed, shallowRef } from "vue"

import type { components } from "./generated/contracts"

type Schemas = components["schemas"]
export type PluginSnapshot = Schemas["PluginSnapshot"]
export type Resource = Schemas["Resource"]
export type Resources = Schemas["ResourceMap"]
export type UpdateStatus = Schemas["UpdateStatus"]

export async function apiV1<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch("/api/v1" + path, { cache: "no-store", ...init })
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error?.message || "请求失败 (" + response.status + ")")
  }
  return response.status === 204 ? undefined as T : response.json()
}

export function localCall<T>(path: string, method = "GET", body?: BodyInit) {
  const headers: Record<string, string> = { "X-BCTools-Local": "1" }
  if (typeof body === "string") headers["Content-Type"] = "application/json"
  return apiV1<T>(path, { method, headers, body })
}

export const activePlugins = shallowRef<PluginSnapshot>({ plugins: [], layoutProvider: "" })
export const resources = shallowRef<Resources>({})
export const resourceTemplates = computed(() => Object.assign({}, ...Object.keys(resources.value).sort().map(id => resources.value[id].templates || {})))

export async function refreshExtensions() {
  const [plugins, assets] = await Promise.allSettled([
    apiV1<PluginSnapshot>("/plugins"),
    apiV1<Resources>("/updates/resources"),
  ])
  if (plugins.status === "fulfilled") activePlugins.value = plugins.value
  if (assets.status === "fulfilled") resources.value = assets.value
}

export async function checkWebRevision() {
  const status = await apiV1<UpdateStatus>("/updates")
  const current = location.pathname.match(/^\/_ui\/([0-9a-f]{64})\//)?.[1] || ""
  if (current !== status.web) {
    location.replace("/" + location.hash)
  } else if (current) {
    await localCall("/updates/web/" + current + "/confirm", "POST").catch(() => {})
  }
}
