// src/utils/request.js
import axios from "axios";
import { generateRandomString } from "./index";

// 从localStorage获取设置
function getSettings() {
  const settingsStr = localStorage.getItem("WeKnora_settings");
  if (settingsStr) {
    try {
      return JSON.parse(settingsStr);
    } catch (e) {
      console.error("解析设置失败:", e);
    }
  }
  return {
    endpoint: import.meta.env.VITE_IS_DOCKER ? "" : "http://localhost:8080",
    apiKey: "",
    knowledgeBaseId: "",
  };
}

// API基础URL，优先使用设置中的endpoint
const settings = getSettings();
const BASE_URL = settings.endpoint;

// 测试数据
let testData: {
  tenant: {
    id: number;
    name: string;
    api_key: string;
  };
  knowledge_bases: Array<{
    id: string;
    name: string;
    description: string;
  }>;
} | null = null;

// 创建Axios实例
const instance = axios.create({
  baseURL: BASE_URL, // 使用配置的API基础URL
  timeout: 30000, // 请求超时时间
  headers: {
    "Content-Type": "application/json",
    "X-Request-ID": `${generateRandomString(12)}`,
  },
});

// 设置测试数据
export function setTestData(data: typeof testData) {
  testData = data;
  if (data) {
    // 优先使用设置中的ApiKey，如果没有则使用测试数据中的
    const apiKey = settings.apiKey || (data?.tenant?.api_key || "");
    if (apiKey) {
      instance.defaults.headers["X-API-Key"] = apiKey;
    }
  }
}

// 获取测试数据
export function getTestData() {
  return testData;
}

instance.interceptors.request.use(
  (config) => {
    // 每次请求前检查是否有更新的设置
    const currentSettings = getSettings();
    
    // 更新BaseURL (如果有变化)
    if (currentSettings.endpoint && config.baseURL !== currentSettings.endpoint) {
      config.baseURL = currentSettings.endpoint;
    }
    
    // 更新API Key (如果有)
    if (currentSettings.apiKey) {
      config.headers["X-API-Key"] = currentSettings.apiKey;
    }
    
    config.headers["X-Request-ID"] = `${generateRandomString(12)}`;
    return config;
  },
  (error) => {}
);

instance.interceptors.response.use(
  (response) => {
    // 根据业务状态码处理逻辑
    const { status, data } = response;
    if (status === 200 || status === 201) {
      return data;
    } else {
      return Promise.reject(data);
    }
  },
  (error: any) => {
    if (!error.response) {
      return Promise.reject({ message: "网络错误，请检查您的网络连接" });
    }
    const { data } = error.response;
    return Promise.reject(data);
  }
);

export function get(url: string) {
  return instance.get(url);
}

export async function getDown(url: string) {
  let res = await instance.get(url, {
    responseType: "blob",
  });
  return res
}

export function postUpload(url: string, data = {}) {
  return instance.post(url, data, {
    headers: {
      "Content-Type": "multipart/form-data",
      "X-Request-ID": `${generateRandomString(12)}`,
    },
  });
}

export function postChat(url: string, data = {}) {
  return instance.post(url, data, {
    headers: {
      "Content-Type": "text/event-stream;charset=utf-8",
      "X-Request-ID": `${generateRandomString(12)}`,
    },
  });
}

export function post(url: string, data = {}) {
  return instance.post(url, data);
}

export function put(url: string, data = {}) {
  return instance.put(url, data);
}

export function del(url: string) {
  return instance.delete(url);
}
