import { get, post, put, del, postChat } from "../../utils/request";
import { loadTestData } from "../test-data";

// 从localStorage获取设置
function getSettings() {
  const settingsStr = localStorage.getItem("WeKnora_settings");
  if (settingsStr) {
    try {
      const settings = JSON.parse(settingsStr);
      if (settings.apiKey && settings.endpoint) {
        return settings;
      }
    } catch (e) {
      console.error("解析设置失败:", e);
    }
  }
  return null;
}

// 根据是否有设置决定是否需要加载测试数据
async function ensureConfigured() {
  const settings = getSettings();
  // 如果没有设置APIKey和Endpoint，则加载测试数据
  if (!settings) {
    await loadTestData();
  }
}

export async function createSessions(data = {}) {
  await ensureConfigured();
  return post("/api/v1/sessions", data);
}

export async function getSessionsList(page: number, page_size: number) {
  await ensureConfigured();
  return get(`/api/v1/sessions?page=${page}&page_size=${page_size}`);
}

export async function generateSessionsTitle(session_id: string, data: any) {
  await ensureConfigured();
  return post(`/api/v1/sessions/${session_id}/generate_title`, data);
}

export async function knowledgeChat(data: { session_id: string; query: string; }) {
  await ensureConfigured();
  return postChat(`/api/v1/knowledge-chat/${data.session_id}`, { query: data.query });
}

export async function getMessageList(data: { session_id: string; limit: number, created_at: string }) {
  await ensureConfigured();

  if (data.created_at) {
    return get(`/api/v1/messages/${data.session_id}/load?before_time=${encodeURIComponent(data.created_at)}&limit=${data.limit}`);
  } else {
    return get(`/api/v1/messages/${data.session_id}/load?limit=${data.limit}`);
  }
}

export async function delSession(session_id: string) {
  await ensureConfigured();
  return del(`/api/v1/sessions/${session_id}`);
}