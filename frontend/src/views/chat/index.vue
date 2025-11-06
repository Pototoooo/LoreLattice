<template>
    <div class="chat">
        <div ref="scrollContainer" class="chat_scroll_box" @scroll="handleScroll">
            <div class="msg_list">
                <div v-for="(session, id) in messagesList" :key='id'>
                    <div v-if="session.role == 'user'">
                        <usermsg :content="session.content"></usermsg>
                    </div>
                    <div v-if="session.role == 'assistant'">
                        <botmsg :content="session.content" :session="session" @scroll-bottom="scrollToBottom"
                            :isFirstEnter="isFirstEnter"></botmsg>
                    </div>
                </div>
                <div v-if="loading"
                    style="height: 41px;display: flex;align-items: center;background: #fff;width: 58px;">
                    <img class="botanswer_laoding_gif" src="@/assets/img/botanswer_loading.gif" alt="正在等待答案……">
                </div>
            </div>
        </div>
        <div style="min-height: 115px; margin: 16px auto 4px;width: 100%;max-width: 800px;">
            <InputField @send-msg="sendMsg" :isReplying="isReplying"></InputField>
        </div>
    </div>
</template>
<script setup>
import { storeToRefs } from 'pinia';
import { ref, onMounted, onUnmounted, nextTick, watch, reactive, onBeforeUnmount } from 'vue';
import { useRoute, useRouter, onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router';
import InputField from '../../components/Input-field.vue';
import botmsg from './components/botmsg.vue';
import usermsg from './components/usermsg.vue';
import { getMessageList, generateSessionsTitle, getSession } from "@/api/chat/index";
import { useStream } from '../../api/chat/streame'
import { useMenuStore } from '@/stores/menu';
import { useSettingsStore } from '@/stores/settings';
import { MessagePlugin } from 'tdesign-vue-next';
const usemenuStore = useMenuStore();
const useSettingsStoreInstance = useSettingsStore();
const { menuArr, isFirstSession, firstQuery } = storeToRefs(usemenuStore);
const { output, onChunk, isStreaming, isLoading, error, startStream, stopStream } = useStream();
const route = useRoute();
const router = useRouter();
const session_id = ref(route.params.chatid);
const sessionData = ref(null);
const created_at = ref('');
const limit = ref(20);
const messagesList = reactive([]);
const isReplying = ref(false);
const scrollLock = ref(false);
const isNeedTitle = ref(false);
const isFirstEnter = ref(true);
const loading = ref(false);
let fullContent = ref('')
let userquery = ref('')
const scrollContainer = ref(null)
watch([() => route.params], (newvalue) => {
    isFirstEnter.value = true;
    if (newvalue[0].chatid) {
        if (!firstQuery.value) {
            scrollLock.value = false;
        }
        messagesList.splice(0);
        session_id.value = newvalue[0].chatid;
        checkmenuTitle(session_id.value)
        let data = {
            session_id: session_id.value,
            created_at: '',
            limit: limit.value
        }
        getmsgList(data);
    }
});
const scrollToBottom = () => {
    nextTick(() => {
        if (scrollContainer.value) {
            scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight;
        }
    })
}
const debounce = (fn, delay) => {
    let timer
    return (...args) => {
        clearTimeout(timer)
        timer = setTimeout(() => fn(...args), delay)
    }
}
const onChatScrollTop = () => {
    if (scrollLock.value) return;
    const { scrollTop, scrollHeight } = scrollContainer.value;
    isFirstEnter.value = false
    if (scrollTop == 0) {
        let data = {
            session_id: session_id.value,
            created_at: created_at.value,
            limit: limit.value
        }
        getmsgList(data, true, scrollHeight);
    }
}
const handleScroll = debounce(onChatScrollTop, 500);

const getmsgList = (data, isScrollType = false, scrollHeight) => {
    getMessageList(data).then(res => {
        if (res && res.data?.length) {
            created_at.value = res.data[0].created_at;
            handleMsgList(res.data, isScrollType, scrollHeight);
        }
    })
}

// Reconstruct agentEventStream from agent_steps stored in database
// This allows the frontend to restore the exact conversation state including all agent reasoning steps
const reconstructEventStreamFromSteps = (agentSteps) => {
    if (!agentSteps || !Array.isArray(agentSteps) || agentSteps.length === 0) {
        return [];
    }
    
    const events = [];
    
    agentSteps.forEach((step) => {
        // Add thinking event if thought content exists
        if (step.thought && step.thought.trim()) {
            events.push({
                type: 'thinking',
                event_id: `step-${step.iteration}-thought`,
                content: step.thought,
                done: true,
                thinking: false,
            });
        }
        
        // Add tool call and result events
        if (step.tool_calls && Array.isArray(step.tool_calls)) {
            step.tool_calls.forEach((toolCall) => {
                events.push({
                    type: 'tool_call',
                    tool_call_id: toolCall.id,
                    tool_name: toolCall.name,
                    arguments: toolCall.args,
                    pending: false,
                    success: toolCall.result?.success !== false,
                    output: toolCall.result?.output || '',
                    error: toolCall.result?.error || undefined,
                    duration: toolCall.duration,
                    display_type: toolCall.result?.data?.display_type,
                    tool_data: toolCall.result?.data,
                });
            });
        }
    });
    
    // 添加一个完成标记的 answer 事件，触发折叠逻辑
    if (events.length > 0) {
        events.push({
            type: 'answer',
            content: '',  // 空内容，因为最终答案会在 message.content 中
            done: true
        });
    }
    
    return events;
};
const handleMsgList = async (data, isScrollType = false, newScrollHeight) => {
    let chatlist = data.reverse()
    for (let i = 0, len = chatlist.length; i < len; i++) {
        let item = chatlist[i];
        item.isAgentMode = false; // Agent 模式标记
        item.agentEventStream = item.agentEventStream || [];
        item._eventMap = new Map();
        item._pendingToolCalls = new Map();
        
        // Check if this message has agent_steps from database (historical agent conversation)
        // If so, reconstruct the agentEventStream to restore the exact conversation state
        if (item.agent_steps && Array.isArray(item.agent_steps) && item.agent_steps.length > 0) {
            console.log('[Message Load] Reconstructing agent steps for message:', item.id, 'steps:', item.agent_steps.length);
            item.isAgentMode = true;
            item.agentEventStream = reconstructEventStreamFromSteps(item.agent_steps);
            // 隐藏最终答案内容，用户可以通过展开步骤查看完整过程
            item.hideContent = true;
            console.log('[Message Load] Reconstructed', item.agentEventStream.length, 'events from agent steps');
        }
        
        if (item.content) {
            if (!item.content.includes('<think>') && !item.content.includes('<\/think>')) {
                item.thinkContent = "";
                item.content = item.content;
                item.showThink = false;
            } else if (item.content.includes('<\/think>')) {
                const arr = item.content.trim().split('<\/think>');
                item.showThink = true;
                item.thinkContent = arr[0].trim().replace('<think>', '');
                let index = item.content.trim().lastIndexOf('<\/think>')
                item.content = item.content.substring(index + 8);
            }
        }
        if (item.is_completed && !item.content) {
            item.content = "抱歉，我无法回答这个问题。";
        }
        messagesList.unshift(item);
        if (isFirstEnter.value) {
            scrollToBottom();
        } else if (isScrollType) {
            nextTick(() => {
                const { scrollHeight } = scrollContainer.value;
                scrollContainer.value.scrollTop = scrollHeight - newScrollHeight
            })
        }
    }
    if (messagesList[messagesList.length - 1] && !messagesList[messagesList.length - 1].is_completed) {
        isReplying.value = true;
        await startStream({ session_id: session_id.value, query: messagesList[messagesList.length - 1].id, method: 'GET', url: '/api/v1/sessions/continue-stream' });
    }

}
const checkmenuTitle = (session_id) => {
    menuArr.value[1].children?.forEach(item => {
        if (item.id == session_id) {
            isNeedTitle.value = item.isNoTitle;
        }
    });
}
// 发送消息
const sendMsg = async (value) => {
    userquery.value = value;
    isReplying.value = true;
    loading.value = true;
    messagesList.push({ content: value, role: 'user' });
    scrollToBottom();
    
    // Always use agent mode with unified architecture
    // Get knowledge_base_ids from session's agent_config.knowledge_bases to update SessionAgentConfig
    let kbIds = [];
    if (sessionData.value?.agent_config?.knowledge_bases?.length > 0) {
        kbIds = sessionData.value.agent_config.knowledge_bases;
    }
    
    // Validate knowledge_base_ids before sending
    if (kbIds.length === 0) {
        MessagePlugin.warning('请至少选择一个知识库');
        isReplying.value = false;
        loading.value = false;
        // Remove the user message that was just added
        messagesList.pop();
        return;
    }
    
    await startStream({ 
        session_id: session_id.value, 
        knowledge_base_ids: kbIds,
        agent_enabled: true,
        query: value, 
        method: 'POST', 
        url: '/api/v1/agent-chat'
    });
}

// Watch for stream errors and show message
watch(error, (newError) => {
    if (newError) {
        MessagePlugin.error(newError);
        isReplying.value = false;
        loading.value = false;
    }
});

// 处理流式数据
onChunk((data) => {
    // 日志：打印接收到的事件
    console.log('[Agent Event Received]', {
        response_type: data.response_type,
        id: data.id,
        done: data.done,
        content_length: data.content?.length || 0,
        content_preview: data.content ? data.content.substring(0, 50) : '',
        data: data.data
    });
    
    // 处理 agent query 事件 - 保持 loading 状态
    if (data.response_type === 'agent_query') {
        console.log('[Agent Query Event]', {
            session_id: data.data?.session_id,
            query: data.data?.query,
            request_id: data.data?.request_id
        });
        // 保持 loading 状态，等待实际内容
        loading.value = true;
        return;
    }
    
    // 处理会话标题更新事件 - 不关闭 loading
    if (data.response_type === 'session_title') {
        const title = data.content || data.data?.title;
        if (title && data.data?.session_id) {
            console.log('[Session Title Update]', {
                session_id: data.data.session_id,
                title: title
            });
            usemenuStore.updatasessionTitle(data.data.session_id, title);
            usemenuStore.changeIsFirstSession(false);
            isNeedTitle.value = false;
        }
        // 不关闭 loading，等待实际内容
        return;
    }
    
    // 判断是否是 Agent 模式的响应
    const isAgentResponse = data.response_type === 'thinking' || 
                           data.response_type === 'tool_call' || 
                           data.response_type === 'references';
    
    // Agent 模式处理
    if (isAgentResponse || messagesList[messagesList.length - 1]?.isAgentMode) {
        // 在 handleAgentChunk 中处理 loading 状态
        handleAgentChunk(data);
        return;
    }
    
    // 原有的知识库 QA 处理逻辑
    fullContent.value += data.content;
    let obj = { ...data, content: '', role: 'assistant', showThink: false };

    if (fullContent.value.includes('<think>') && !fullContent.value.includes('<\/think>')) {
        obj.thinking = true;
        obj.showThink = true;
        obj.content = '';
        obj.thinkContent = fullContent.value.replace('<think>', '').trim();
    } else if (fullContent.value.includes('<think>') && fullContent.value.includes('<\/think>')) {
        obj.thinking = false;
        obj.showThink = true;
        const index = fullContent.value.indexOf('<\/think>');
        obj.thinkContent = fullContent.value.substring(0, index).replace('<think>', '').trim();
        obj.content = fullContent.value.substring(index + 8).trim();
    } else {
        obj.content = fullContent.value;
    }
    
    // 检查是否已有消息，如果没有则说明这是第一次，关闭 loading
    const existingMessage = messagesList.findLast((item) => {
        if (item.request_id === obj.id) {
            return true
        }
        return item.id === obj.id;
    });
    if (!existingMessage) {
        loading.value = false; // 消息即将创建，关闭 loading
    }
    
    if (data.done) {
        // 标题生成已改为异步事件推送，不再需要在这里手动调用
        // 如果标题还未生成，前端会通过 SSE 事件接收
        isReplying.value = false;
        fullContent.value = "";
    }
    updateAssistantSession(obj);
})
// 处理 Agent 流式数据 (Cursor-style UI)
const handleAgentChunk = (data) => {
    const message = messagesList.findLast((item) => item.request_id === data.id || item.id === data.id);
    
    if (!message) {
        // 创建新的 Assistant 消息 - 此时开始显示内容，关闭 loading
        const newMsg = {
            id: data.id,
            request_id: data.id,
            role: 'assistant',
            content: '',
            isAgentMode: true,
            // Event stream: ordered list of all agent events (thinking, tool calls, etc)
            agentEventStream: [],
            // Map to track event by event_id for quick lookup
            _eventMap: new Map(),
            knowledge_references: []
        };
        messagesList.push(newMsg);
        loading.value = false; // 消息已创建，关闭 loading
        scrollToBottom();
        return;
    }
    
    message.isAgentMode = true;
    
    switch(data.response_type) {
        case 'thinking':
            {
                const eventId = data.data?.event_id;
                console.log('[Thinking Event]', {
                    event_id: eventId,
                    done: data.done,
                    content_length: data.content?.length || 0
                });
                
                // Initialize structures
                if (!message.agentEventStream) message.agentEventStream = [];
                if (!message._eventMap) message._eventMap = new Map();
                
                if (!data.done) {
                    // Check if this thinking event already exists
                    let thinkingEvent = message._eventMap.get(eventId);
                    
                    if (!thinkingEvent) {
                        // Create new thinking event
                        console.log('[Thinking] Creating new thinking event, event_id:', eventId);
                        thinkingEvent = {
                            type: 'thinking',
                            event_id: eventId,
                            content: '',
                            done: false,
                            startTime: Date.now(),
                            thinking: true
                        };
                        
                        // Add to event stream
                        message.agentEventStream.push(thinkingEvent);
                        message._eventMap.set(eventId, thinkingEvent);
                    }
                    
                    // Accumulate content
                    if (data.content) {
                        thinkingEvent.content += data.content;
                        console.log('[Thinking] Event', eventId, 'accumulated:', thinkingEvent.content.length, 'chars');
                    }
                    
                } else {
                    // Thinking completed
                    const thinkingEvent = message._eventMap.get(eventId);
                    if (thinkingEvent) {
                        console.log('[Thinking] Completing event, event_id:', eventId, 'content length:', thinkingEvent.content.length);
                        
                        // Mark as done
                        thinkingEvent.done = true;
                        thinkingEvent.thinking = false;
                        thinkingEvent.duration_ms = data.data?.duration_ms || (Date.now() - thinkingEvent.startTime);
                        thinkingEvent.completed_at = data.data?.completed_at || Date.now();
                        
                        console.log('[Thinking] Event completed, duration:', thinkingEvent.duration_ms, 'ms');
                    } else {
                        console.warn('[Thinking] Received done for unknown event_id:', eventId);
                    }
                }
            }
            break;
            
        case 'tool_call':
            // Store pending tool call to pair with result later
            if (data.data && data.data.tool_name) {
                console.log('[Tool Call]', data.data.tool_name);
                
                if (!message.agentEventStream) message.agentEventStream = [];
                if (!message._pendingToolCalls) message._pendingToolCalls = new Map();
                
                const toolCallId = data.data.tool_call_id || (data.data.tool_name + '_' + Date.now());
                
                // Create tool call event
                const toolCallEvent = {
                    type: 'tool_call',
                    tool_call_id: toolCallId,
                    tool_name: data.data.tool_name,
                    arguments: data.data.arguments,
                    timestamp: Date.now(),
                    pending: true
                };
                
                // Add to event stream
                message.agentEventStream.push(toolCallEvent);
                message._pendingToolCalls.set(toolCallId, toolCallEvent);
            }
            break;
            
        case 'tool_result':
        case 'error':
            // Tool result - update the corresponding tool call event
            if (data.data) {
                const toolCallId = data.data.tool_call_id;
                const toolName = data.data.tool_name;
                const success = data.response_type !== 'error' && data.data.success !== false;
                
                console.log('[Tool Result]', {
                    tool_call_id: toolCallId,
                    tool_name: toolName,
                    success: success
                });
                
                // Find and update the pending tool call event
                let toolCallEvent = null;
                if (message._pendingToolCalls) {
                    if (toolCallId && message._pendingToolCalls.has(toolCallId)) {
                        toolCallEvent = message._pendingToolCalls.get(toolCallId);
                        message._pendingToolCalls.delete(toolCallId);
                    } else {
                        // Try to find by tool_name if no tool_call_id match
                        for (const [key, value] of message._pendingToolCalls.entries()) {
                            if (value.tool_name === toolName) {
                                toolCallEvent = value;
                                message._pendingToolCalls.delete(key);
                                break;
                            }
                        }
                    }
                }
                
                if (toolCallEvent) {
                    // Update the existing event with result
                    toolCallEvent.pending = false;
                    toolCallEvent.success = success;
                    toolCallEvent.output = success ? (data.data.output || data.content) : (data.data.error || data.content);
                    toolCallEvent.error = !success ? (data.data.error || data.content) : undefined;
                    toolCallEvent.duration = data.data.duration_ms;
                    toolCallEvent.display_type = data.data.display_type;
                    toolCallEvent.tool_data = data.data;
                    
                    console.log('[Tool Result] Updated event in stream');
                } else {
                    console.warn('[Tool Result] No pending tool call found for', toolCallId || toolName);
                }
                
                // If this is an error response without tool data, handle it
                if (data.response_type === 'error' && !toolName) {
                    message.content = data.content || '处理出错';
                    isReplying.value = false;
                }
            } else if (data.response_type === 'error') {
                // Generic error without tool context
                message.content = data.content || '处理出错';
                isReplying.value = false;
            }
            break;
            

        case 'references':
            // 知识引用
            if (data.knowledge_references) {
                message.knowledge_references = data.knowledge_references;
            }
            break;
            
        case 'answer':
            // 最终答案
            message.thinking = false;
            message.content = (message.content || '') + (data.content || '');
            fullContent.value += data.content || '';
            
            // Add or update answer event in agentEventStream
            if (!message.agentEventStream) message.agentEventStream = [];
            
            let answerEvent = message.agentEventStream.find((e) => e.type === 'answer');
            if (!answerEvent) {
                answerEvent = {
                    type: 'answer',
                    content: '',
                    done: false
                };
                message.agentEventStream.push(answerEvent);
            }
            
            answerEvent.content = message.content;
            answerEvent.done = data.done;
            
            if (data.done) {
                console.log('[Agent] Answer done, content length:', message.content?.length || 0);
                
                // 完成
                isReplying.value = false;
                fullContent.value = '';
                
                // 标题生成已改为异步事件推送，不再需要在这里手动调用
                // 如果标题还未生成，前端会通过 SSE 事件接收
            }
            break;
    }
    
    scrollToBottom();
};

const updateAssistantSession = (payload) => {
    const message = messagesList.findLast((item) => {
        if (item.request_id === payload.id) {
            return true
        }
        return item.id === payload.id;
    });
    if (message) {
        message.content = payload.content;
        message.thinking = payload.thinking;
        message.thinkContent = payload.thinkContent;
        message.showThink = payload.showThink;
        message.knowledge_references = message.knowledge_references ? message.knowledge_references : payload.knowledge_references;
    } else {
        messagesList.push(payload);
    }
    scrollToBottom();
}
onMounted(async () => {
    messagesList.splice(0);
    
    // Load session data to get agent_config
    try {
        const sessionRes = await getSession(session_id.value);
        if (sessionRes?.data) {
            sessionData.value = sessionRes.data;
        }
    } catch (error) {
        console.error('Failed to load session data:', error);
    }
    
    checkmenuTitle(session_id.value)
    if (firstQuery.value) {
        scrollLock.value = true;
        sendMsg(firstQuery.value);
        usemenuStore.changeFirstQuery('');
    } else {
        scrollLock.value = false;
        let data = {
            session_id: session_id.value,
            created_at: '',
            limit: limit.value
        }
        getmsgList(data)
    }
})
const clearData = () => {
    stopStream();
    isReplying.value = false;
    fullContent.value = '';
    userquery.value = '';

}
onBeforeRouteLeave((to, from, next) => {
    clearData()
    next()
})
onBeforeRouteUpdate((to, from, next) => {
    clearData()
    next()
})
</script>
<style lang="less" scoped>
.chat {
    font-size: 20px;
    padding: 20px;
    box-sizing: border-box;
    flex: 1;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    max-width: calc(100vw - 260px);
    min-width: 400px;

    :deep(.answers-input) {
        position: static;
        transform: translateX(0);

        .t-textarea__inner {
            width: 100% !important;
        }
    }
}

.chat_scroll_box {
    flex: 1;
    width: 100%;
    overflow-y: auto;

    &::-webkit-scrollbar {
        width: 0;
        height: 0;
        color: transparent;
    }
}


.agent-mode-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: linear-gradient(135deg, #e6f7ff 0%, #bae7ff 100%);
    border: 1px solid #91d5ff;
    border-radius: 6px;
    margin-bottom: 12px;
    max-width: 800px;
    width: 100%;

    .agent-icon {
        font-size: 20px;
    }

    .agent-text {
        font-size: 14px;
        font-weight: 500;
        color: #0050b3;
        flex: 1;
    }
}

.msg_list {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 800px;
    flex: 1;
    margin: 0 auto;
    width: 100%;

    .botanswer_laoding_gif {
        width: 24px;
        height: 18px;
        margin-left: 16px;
    }
}
</style>