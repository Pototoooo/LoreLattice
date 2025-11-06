import { fetchEventSource } from '@microsoft/fetch-event-source'
import { ref, type Ref, onUnmounted, nextTick } from 'vue'
import { generateRandomString } from '@/utils/index';



interface StreamOptions {
  // 请求方法 (默认POST)
  method?: 'GET' | 'POST'
  // 请求头
  headers?: Record<string, string>
  // 请求体自动序列化
  body?: Record<string, any>
  // 流式渲染间隔 (ms)
  chunkInterval?: number
}

export function useStream() {
  // 响应式状态
  const output = ref('')              // 显示内容
  const isStreaming = ref(false)      // 流状态
  const isLoading = ref(false)        // 初始加载
  const error = ref<string | null>(null)// 错误信息
  let controller = new AbortController()

  // 流式渲染缓冲
  let buffer: string[] = []
  let renderTimer: number | null = null

  // 启动流式请求
  const startStream = async (params: { session_id: any; query: any; knowledge_base_ids?: string[]; agent_enabled?: boolean; method: string; url: string }) => {
    // 重置状态
    output.value = '';
    error.value = null;
    isStreaming.value = true;
    isLoading.value = true;

    // 获取API配置
    const apiUrl = import.meta.env.VITE_IS_DOCKER ? "" : "http://localhost:8080";
    
    // 获取JWT Token
    const token = localStorage.getItem('weknora_token');
    if (!token) {
      error.value = "未找到登录令牌，请重新登录";
      stopStream();
      return;
    }

    // Validate knowledge_base_ids for agent-chat requests
    const isAgentChat = params.url === '/api/v1/agent-chat';
    if (isAgentChat && params.method === 'POST') {
      if (!params.knowledge_base_ids || params.knowledge_base_ids.length === 0) {
        error.value = "请至少选择一个知识库";
        stopStream();
        return;
      }
    }

    try {
      let url =
        params.method == "POST"
          ? `${apiUrl}${params.url}/${params.session_id}`
          : `${apiUrl}${params.url}/${params.session_id}?message_id=${params.query}`;
      
      // Prepare POST body with required fields for agent-chat
      // knowledge_base_ids array and agent_enabled can update Session's SessionAgentConfig
      const postBody: any = { 
        query: params.query,
        agent_enabled: params.agent_enabled !== undefined ? params.agent_enabled : true
      };
      // Always include knowledge_base_ids for agent-chat (already validated above)
      if (params.knowledge_base_ids !== undefined && params.knowledge_base_ids.length > 0) {
        postBody.knowledge_base_ids = params.knowledge_base_ids;
      }
      
      await fetchEventSource(url, {
        method: params.method,
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
          "X-Request-ID": `${generateRandomString(12)}`,
        },
        body:
          params.method == "POST"
            ? JSON.stringify(postBody)
            : null,
        signal: controller.signal,
        openWhenHidden: true,

        onopen: async (res) => {
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          isLoading.value = false;
        },

        onmessage: (ev) => {
          buffer.push(JSON.parse(ev.data)); // 数据存入缓冲
          // 执行自定义处理
          if (chunkHandler) {
            chunkHandler(JSON.parse(ev.data));
          }
        },

        onerror: (err) => {
          throw new Error(`流式连接失败: ${err}`);
        },

        onclose: () => {
          stopStream();
        },
      });
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      stopStream()
    }
  }

  let chunkHandler: ((data: any) => void) | null = null
  // 注册块处理器
  const onChunk = (handler: () => void) => {
    chunkHandler = handler
  }


  // 停止流
  const stopStream = () => {
    controller.abort();
    controller = new AbortController(); // 重置控制器（如需重新发起）
    isStreaming.value = false;
    isLoading.value = false;
  }

  // 组件卸载时自动清理
  onUnmounted(stopStream)

  return {
    output,          // 显示内容
    isStreaming,     // 是否在流式传输中
    isLoading,       // 初始连接状态
    error,
    onChunk,
    startStream,     // 启动流
    stopStream       // 手动停止
  }
}