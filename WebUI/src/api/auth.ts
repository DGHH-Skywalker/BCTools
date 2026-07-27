import apiClient from "./client"

export async function verifyPassword(password: string): Promise<{ success: boolean; hint?: string }> {
  const res = await apiClient.post("/auth/verify", { password })
  return res.data
}
