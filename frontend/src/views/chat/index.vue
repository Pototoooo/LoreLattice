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
import { getMessageList, generateSessionsTitle } from "@/api/chat/index";
import { useStream } from '../../api/chat/streame'
import { useMenuStore } from '@/stores/menu';
import { useSettingsStore } from '@/stores/settings';
const usemenuStore = useMenuStore();
const useSettingsStoreInstance = useSettingsStore();
const { menuArr, isFirstSession, firstQuery } = storeToRefs(usemenuStore);
const { output, onChunk, isStreaming, isLoading, error, startStream, stopStream } = useStream();
const route = useRoute();
const router = useRouter();
const session_id = ref(route.params.chatid);
const knowledge_base_id = ref(route.params.kbId);
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
        knowledge_base_id.value = newvalue[0].kbId;
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
const handleMsgList = async (data, isScrollType = false, newScrollHeight) => {
    let chatlist = data.reverse()
    for (let i = 0, len = chatlist.length; i < len; i++) {
        let item = chatlist[i];
        item.isAgentMode = false; // Agent 模式标记
        item.agentEventStream = item.agentEventStream || [];
        item._eventMap = new Map();
        item._pendingToolCalls = new Map();
        
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
    
    // 判断是否使用 Agent 模式
    const isAgentMode = useSettingsStoreInstance.isAgentEnabled;
    const chatUrl = isAgentMode ? '/api/v1/agent-chat' : '/api/v1/knowledge-chat';
    
    await startStream({ 
        session_id: session_id.value, 
        knowledge_base_id: knowledge_base_id.value,
        query: value, 
        method: 'POST', 
        url: chatUrl
    });
}

// 处理流式数据
onChunk((data) => {
    loading.value = false;
    
    // 日志：打印接收到的事件
    console.log('[Agent Event Received]', {
        response_type: data.response_type,
        id: data.id,
        done: data.done,
        content_length: data.content?.length || 0,
        content_preview: data.content ? data.content.substring(0, 50) : '',
        data: data.data
    });
    
    // 判断是否是 Agent 模式的响应
    const isAgentResponse = data.response_type === 'thinking' || 
                           data.response_type === 'tool_call' || 
                           data.response_type === 'references';
    
    // Agent 模式处理
    if (isAgentResponse || messagesList[messagesList.length - 1]?.isAgentMode) {
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
    if (data.done) {
        if (isFirstSession.value || isNeedTitle.value) {
            generateSessionsTitle(session_id.value, {
                messages: [{ role: "user", content: userquery.value }]
            }).then(res => {
                if (res.data) {
                    usemenuStore.changeIsFirstSession(false);
                    usemenuStore.updatasessionTitle(session_id.value, res.data);
                    isNeedTitle.value = false;
                }
            })
        }
        isReplying.value = false;
        fullContent.value = "";
    }
    updateAssistantSession(obj);
})
// 处理 Agent 流式数据 (Cursor-style UI)
const handleAgentChunk = (data) => {
    const message = messagesList.findLast((item) => item.request_id === data.id || item.id === data.id);
    
    if (!message) {
        // 创建新的 Assistant 消息
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
                
                // 生成标题
                if (isFirstSession.value || isNeedTitle.value) {
                    generateSessionsTitle(session_id.value, {
                        messages: [{ role: "user", content: userquery.value }]
                    }).then(res => {
                        if (res.data) {
                            usemenuStore.changeIsFirstSession(false);
                            usemenuStore.updatasessionTitle(session_id.value, res.data);
                            isNeedTitle.value = false;
                        }
                    });
                }
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
onMounted(() => {
    messagesList.splice(0);
    // scrollContainer.value.addEventListener("scroll", () => {
    //     if (scrollContainer.value.scrollTop == 0) {
    //         onChatScrollTop();
    //     }
    // });
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
    max-width: 800px;

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

    .botanswer_laoding_gif {
        width: 24px;
        height: 18px;
        margin-left: 16px;
    }
}
</style>