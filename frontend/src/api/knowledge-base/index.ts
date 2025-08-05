import { get, post, put, del, postUpload, getDown, getTestData } from "../../utils/request";
import { loadTestData } from "../test-data";

// 获取知识库ID（优先从设置中获取）
async function getKnowledgeBaseID() {
  // 从localStorage获取设置中的知识库ID
  const settingsStr = localStorage.getItem("WeKnora_settings");
  let knowledgeBaseId = "";
  
  if (settingsStr) {
    try {
      const settings = JSON.parse(settingsStr);
      if (settings.knowledgeBaseId) {
        return settings.knowledgeBaseId;
      }
    } catch (e) {
      console.error("解析设置失败:", e);
    }
  }
  
  // 如果设置中没有知识库ID，则使用测试数据
  await loadTestData();
  
  const testData = getTestData();
  if (!testData || testData.knowledge_bases.length === 0) {
    console.error("测试数据未初始化或不包含知识库");
    throw new Error("测试数据未初始化或不包含知识库");
  }
  return testData.knowledge_bases[0].id;
}

export async function uploadKnowledgeBase(data = {}) {
  const kbId = await getKnowledgeBaseID();
  return postUpload(`/api/v1/knowledge-bases/${kbId}/knowledge/file`, data);
}

export async function getKnowledgeBase({page, page_size}) {
  const kbId = await getKnowledgeBaseID();
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

export function getKnowledgeDetailsCon(id: any, page) {
  return get(`/api/v1/chunks/${id}?page=${page}&page_size=25`);
}