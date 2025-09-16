import { get, post, put, del, postUpload, getDown, getTestData } from "../../utils/request";
import { loadTestData } from "../test-data";
export async function getDefaultKnowledgeBaseId(): Promise<string> {
  // 如果设置中没有知识库ID，则使用测试数据
  await loadTestData();
  const testData = getTestData();
  if (!testData || testData.knowledge_bases.length === 0) {
    throw new Error('没有可用的知识库');
  }
  
  return testData.knowledge_bases[0].id;
}

export async function uploadKnowledgeBase(data = {}) {
  const kbId = await getDefaultKnowledgeBaseId();
  return postUpload(`/api/v1/knowledge-bases/${kbId}/knowledge/file`, data);
}

export async function getKnowledgeBase({page, page_size}: {page: number, page_size: number}) {
  const kbId = await getDefaultKnowledgeBaseId();
  return get(
    `/api/v1/knowledge-bases/${kbId}/knowledge?page=${page}&page_size=${page_size}`
  );
}

export function getKnowledgeDetails(id: any) {
  return get(`/api/v1/knowledge/${id}`);
}

export function delKnowledgeDetails(id: any) {
  return del(`/api/v1/knowledge/${id}`);
}

export function downKnowledgeDetails(id: any) {
  return getDown(`/api/v1/knowledge/${id}/download`);
}

export function batchQueryKnowledge(ids: any) {
  return get(`/api/v1/knowledge/batch?${ids}`);
}

export function getKnowledgeDetailsCon(id: any, page: number) {
  return get(`/api/v1/chunks/${id}?page=${page}&page_size=25`);
}