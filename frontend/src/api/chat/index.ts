import { get, post, put, del, postChat } from "../../utils/request";
import { loadTestData } from "../test-data";



export async function createSessions(data = {}) {
  await loadTestData();
  return post("/api/v1/sessions", data);
}

export async function getSessionsList(page: number, page_size: number) {
  await loadTestData();
  return get(`/api/v1/sessions?page=${page}&page_size=${page_size}`);
}

export async function generateSessionsTitle(session_id: string, data: any) {
  await loadTestData();
  return post(`/api/v1/sessions/${session_id}/generate_title`, data);
}

export async function knowledgeChat(data: { session_id: string; query: string; }) {
  await loadTestData();
  return postChat(`/api/v1/knowledge-chat/${data.session_id}`, { query: data.query });
}

export async function getMessageList(data: { session_id: string; limit: number, created_at: string }) {
  await loadTestData();
  if (data.created_at) {
    return get(`/api/v1/messages/${data.session_id}/load?before_time=${encodeURIComponent(data.created_at)}&limit=${data.limit}`);
  } else {
    return get(`/api/v1/messages/${data.session_id}/load?limit=${data.limit}`);
  }
}

export async function delSession(session_id: string) {
  await loadTestData();
  return del(`/api/v1/sessions/${session_id}`);
}