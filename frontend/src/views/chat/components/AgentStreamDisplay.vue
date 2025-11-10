<template>
  <div class="agent-stream-display">
    
    <!-- Collapsed intermediate steps -->
    <div v-if="shouldShowCollapsedSteps" class="intermediate-steps-collapsed">
      <div class="intermediate-steps-header" @click="toggleIntermediateSteps">
        <div class="intermediate-steps-title">
          <img :src="agentIcon" alt="" />
          <span>{{ intermediateStepsSummary }}</span>
        </div>
        <div class="intermediate-steps-show-icon">
          <t-icon :name="showIntermediateSteps ? 'chevron-up' : 'chevron-down'" />
        </div>
      </div>
    </div>
    
    <!-- Event Stream -->
    <template v-for="(event, index) in displayEvents" :key="getEventKey(event, index)">
      <div v-if="event && event.type" class="event-item" :data-event-index="index">
        
        <!-- Plan Task Change Event -->
        <div v-if="event.type === 'plan_task_change'" class="plan-task-change-event">
          <div class="plan-task-change-card">
            <div class="plan-task-change-content">
              <strong>任务:</strong> {{ event.task }}
            </div>
          </div>
        </div>
        
        <!-- Thinking Event -->
        <div v-if="event.type === 'thinking'" class="thinking-event">
          <div 
            class="thinking-phase" 
            :class="{ 
              'thinking-active': event.thinking,
              'thinking-last': isLastThinking(event.event_id)
            }"
          >
            <div v-if="event.content" 
                 class="thinking-content markdown-content" 
                 v-html="renderMarkdown(event.content)">
            </div>
          </div>
        </div>
        
        <!-- Answer Event -->
        <div v-else-if="event.type === 'answer' && event.content && event.content.trim()" class="answer-event">
          <div 
            class="answer-content-wrapper"
            :class="{ 
              'answer-active': !event.done,
              'answer-done': event.done
            }"
          >
            <div v-if="event.content" 
                 class="answer-content markdown-content" 
                 v-html="renderMarkdown(event.content)">
            </div>
          </div>
        </div>
        
        <!-- Tool Call Event -->
        <div v-else-if="event.type === 'tool_call'" class="tool-event">
        <div 
          class="action-card" 
          :class="{ 
            'action-pending': event.pending,
            'action-error': event.success === false 
          }"
        >
          <div class="action-header" @click="toggleEvent(event.tool_call_id)">
            <div class="action-title">
              <img v-if="event.tool_name && !isBookIcon(event.tool_name)" class="action-title-icon" :src="getToolIcon(event.tool_name)" alt="" />
              <t-icon v-if="event.tool_name && isBookIcon(event.tool_name)" class="action-title-icon" name="book" />
              <!-- Custom header for todo_write tool -->
              <t-tooltip v-if="event.tool_name === 'todo_write' && event.tool_data?.steps" :content="'更新计划'" placement="top">
                <span class="action-name">
                  更新计划
                </span>
              </t-tooltip>
              <!-- Use tool summary as title if available, otherwise use description -->
              <t-tooltip v-else :content="getToolTitle(event)" placement="top">
                <span class="action-name">{{ getToolTitle(event) }}</span>
              </t-tooltip>
            </div>
            <div v-if="!event.pending" class="action-show-icon">
              <t-icon :name="isEventExpanded(event.tool_call_id) ? 'chevron-up' : 'chevron-down'" />
            </div>
          </div>
          
          <!-- Plan Status Summary (Fixed, always visible, outside action-details) -->
          <div v-if="!event.pending && event.tool_name === 'todo_write' && event.tool_data?.steps" class="plan-status-summary-fixed">
            <div class="plan-status-text">
              <template v-for="(part, partIndex) in getPlanStatusItems(event)" :key="partIndex">
                <t-icon :name="part.icon" :class="['status-icon', part.class]" />
                <span>{{ part.label }} {{ part.count }}</span>
                <span v-if="partIndex < getPlanStatusItems(event).length - 1" class="separator">·</span>
              </template>
            </div>
          </div>
          
          <!-- Search Results Summary (Fixed, always visible, outside action-details) -->
          <div v-if="!event.pending && (event.tool_name === 'search_knowledge' || event.tool_name === 'knowledge_search') && event.tool_data" class="search-results-summary-fixed">
            <div class="results-summary-text">{{ getSearchResultsSummary(event.tool_data) }}</div>
          </div>
          
          <div v-if="isEventExpanded(event.tool_call_id) && !event.pending" class="action-details">
            <!-- Thinking tool: only render markdown thought content -->
            <template v-if="event.tool_name === 'thinking' && event.tool_data?.thought">
              <div class="thinking-thought-content">
                <div class="thinking-thought-markdown markdown-content" v-html="renderMarkdown(event.tool_data.thought)"></div>
              </div>
            </template>
            
            <!-- For other tools: show ToolResultRenderer or output -->
            <template v-else>
              <!-- Use ToolResultRenderer if display_type is available -->
              <div v-if="event.display_type && event.tool_data" class="tool-result-wrapper">
                <ToolResultRenderer 
                  :display-type="event.display_type"
                  :tool-data="event.tool_data"
                  :output="event.output"
                  :arguments="event.arguments"
                />
              </div>
              
              <!-- Fallback to original output display -->
              <div v-else-if="event.output" class="tool-output-wrapper">
                <div class="detail-output">{{ event.output }}</div>
              </div>
              
              <!-- Show Arguments only if no display_type and not for todo_write -->
              <div v-if="event.arguments && event.tool_name !== 'todo_write' && !event.display_type" class="tool-arguments-wrapper">
                <div class="arguments-header">
                  <span class="arguments-label">参数</span>
                </div>
                <pre class="detail-code">{{ formatJSON(event.arguments) }}</pre>
              </div>
            </template>
          </div>
        </div>
      </div>
      </div>
    </template>
    
    <!-- Loading Indicator -->
    <div v-if="!isConversationDone && eventStream.length > 0" class="loading-indicator">
      <img class="botanswer_loading_gif" src="@/assets/img/botanswer_loading.gif" alt="Processing...">
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import ToolResultRenderer from './ToolResultRenderer.vue';

// Import icons
import agentIcon from '@/assets/img/agent.svg';
import thinkingIcon from '@/assets/img/Frame3718.svg';
import knowledgeIcon from '@/assets/img/zhishiku-thin.svg';
import documentIcon from '@/assets/img/ziliao.svg';
import fileAddIcon from '@/assets/img/file-add-green.svg';

interface SessionData {
  isAgentMode?: boolean;
  agentEventStream?: any[];
  knowledge_references?: any[];
}

const props = defineProps<{
  session: SessionData;
}>();

// Configure marked for security
marked.use({
  mangle: false,
  headerIds: false
});

// Event stream
const eventStream = computed(() => props.session?.agentEventStream || []);

// Expanded events tracking (for tool calls)
// Initialize with thinking tools expanded by default
const expandedEvents = ref<Set<string>>(new Set());

// Watch event stream to auto-expand thinking tools
watch(eventStream, (stream) => {
  if (!stream || !Array.isArray(stream)) return;
  
  stream.forEach((event: any) => {
    if (event?.type === 'tool_call' && event?.tool_name === 'thinking' && event?.tool_call_id) {
      expandedEvents.value.add(event.tool_call_id);
    }
  });
}, { immediate: true, deep: true });

// State for intermediate steps collapse
const showIntermediateSteps = ref(false);

// Find the last thinking event in the current message's event stream
// Only the last thinking should have the green border-left
const lastThinkingEventId = computed(() => {
  const stream = eventStream.value;
  if (!stream || stream.length === 0) return null;
  
  // Find all thinking events
  const thinkingEvents = stream.filter((e: any) => e.type === 'thinking');
  if (thinkingEvents.length === 0) return null;
  
  // Return the event_id of the last thinking event
  const lastThinking = thinkingEvents[thinkingEvents.length - 1];
  return lastThinking.event_id;
});

// Check if a thinking event is the last one (should have green border)
const isLastThinking = (eventId: string): boolean => {
  return eventId === lastThinkingEventId.value;
};

// Check if conversation is done (based on answer event with done=true or stop event)
const isConversationDone = computed(() => {
  const stream = eventStream.value;
  if (!stream || stream.length === 0) {
    console.log('[Collapse] No stream or empty stream');
    return false;
  }
  
  // Check for stop event (user cancelled)
  const stopEvent = stream.find((e: any) => e.type === 'stop');
  if (stopEvent) {
    console.log('[Collapse] Found stop event, conversation done');
    return true;
  }
  
  // Check for answer event with done=true
  const answerEvents = stream.filter((e: any) => e.type === 'answer');
  const doneAnswer = answerEvents.find((e: any) => e.done === true);
  
  console.log('[Collapse] Answer events:', answerEvents.length, 'Done answer:', !!doneAnswer);
  
  return !!doneAnswer;
});

// Find the final content to display (last thinking or answer)
const finalContent = computed(() => {
  const stream = eventStream.value;
  if (!stream || stream.length === 0) {
    console.log('[Collapse] finalContent: no stream');
    return null;
  }
  
  if (!isConversationDone.value) {
    console.log('[Collapse] finalContent: not done yet');
    return null;
  }
  
  // Check if there's an answer with content
  const answerEvents = stream.filter((e: any) => e.type === 'answer');
  const hasAnswerContent = answerEvents.some((e: any) => e.content && e.content.trim());
  
  console.log('[Collapse] Answer events:', answerEvents.length, 'Has content:', hasAnswerContent);
  
  if (hasAnswerContent) {
    // Answer has content, it's the final content
    console.log('[Collapse] finalContent: showing answer');
    return { type: 'answer' };
  } else {
    // Answer is empty, find last thinking with content
    const thinkingEvents = stream.filter((e: any) => e.type === 'thinking' && e.content && e.content.trim());
    console.log('[Collapse] Thinking events with content:', thinkingEvents.length);
    
    if (thinkingEvents.length > 0) {
      const lastThinking = thinkingEvents[thinkingEvents.length - 1];
      console.log('[Collapse] finalContent: showing last thinking', lastThinking.event_id);
      return { type: 'thinking', event_id: lastThinking.event_id };
    }
  }
  
  console.log('[Collapse] finalContent: no final content found');
  return null;
});

// Count intermediate steps (tools + thinking that will be collapsed)
const intermediateStepsCount = computed(() => {
  if (!finalContent.value) {
    console.log('[Collapse] intermediateStepsCount: no final content');
    return 0;
  }
  
  const stream = eventStream.value;
  let count = 0;
  
  for (const event of stream) {
    if (event.type === 'tool_call') {
      count++;
    } else if (event.type === 'thinking' && event.content) {
      // Count if it's not the final thinking
      if (finalContent.value.type !== 'thinking' || event.event_id !== finalContent.value.event_id) {
        count++;
      }
    }
  }
  
  console.log('[Collapse] intermediateStepsCount:', count);
  return count;
});

// Get intermediate steps summary with special info
const intermediateStepsSummary = computed(() => {
  if (!finalContent.value || !eventStream.value) {
    return '';
  }
  
  const stream = eventStream.value;
  const toolCalls: string[] = [];
  let searchCount = 0;
  let thinkingCount = 0;
  let totalDuration = 0;
  
  for (const event of stream) {
    if (event.type === 'tool_call' && event.tool_name) {
      const toolName = event.tool_name;
      if (toolName === 'search_knowledge' || toolName === 'knowledge_search') {
        searchCount++;
      } else if (toolName === 'thinking') {
        // Count if it's not the final thinking
        if (finalContent.value.type !== 'thinking' || event.event_id !== finalContent.value.event_id) {
          thinkingCount++;
        }
      } else if (toolName !== 'todo_write') {
        // Only add unique tool names
        if (!toolCalls.includes(toolName)) {
          toolCalls.push(toolName);
        }
      }
      // Accumulate duration for tool calls
      if (event.duration) {
        totalDuration += event.duration;
      } else if (event.duration_ms) {
        totalDuration += event.duration_ms;
      }
    } else if (event.type === 'thinking' && event.content) {
      // Count if it's not the final thinking
      if (finalContent.value.type !== 'thinking' || event.event_id !== finalContent.value.event_id) {
        thinkingCount++;
        // Accumulate duration for thinking events
        if (event.duration_ms) {
          totalDuration += event.duration_ms;
        }
      }
    }
  }
  
  const parts: string[] = [];
  if (searchCount > 0) {
    parts.push(`检索知识库 ${searchCount} 次`);
  }
  if (thinkingCount > 0) {
    parts.push(`思考 ${thinkingCount} 次`);
  }
  if (toolCalls.length > 0) {
    const toolNames = toolCalls.map(name => {
      if (name === 'get_document_info') return '获取文档';
      if (name === 'get_related_chunks') return '获取相关片段';
      return name;
    });
    if (toolNames.length === 1) {
      parts.push(`调用 ${toolNames[0]}`);
    } else {
      parts.push(`调用工具 ${toolNames.join('、')}`);
    }
  }
  
  // Add duration info
  if (totalDuration > 0) {
    parts.push(`耗时 ${formatDuration(totalDuration)}`);
  }
  
  if (parts.length === 0) {
    return `${intermediateStepsCount.value} 个中间步骤`;
  }
  
  // 优化连接词，使语句更流畅
  if (parts.length === 1) {
    return parts[0];
  } else if (parts.length === 2) {
    return `${parts[0]}，${parts[1]}`;
  } else {
    // 3个或以上：前几个用顿号，最后一个用逗号
    const last = parts.pop();
    return `${parts.join('、')}，${last}`;
  }
});

// Should show the collapsed steps indicator
const shouldShowCollapsedSteps = computed(() => {
  const result = isConversationDone.value && intermediateStepsCount.value > 0;
  console.log('[Collapse] shouldShowCollapsedSteps:', result, 'done:', isConversationDone.value, 'count:', intermediateStepsCount.value);
  return result;
});

// Events to display (based on collapse state)
const displayEvents = computed(() => {
  const stream = eventStream.value;
  if (!stream || !Array.isArray(stream)) {
    console.log('[Collapse] displayEvents: no stream or not array');
    return [];
  }
  
  // Filter out invalid events
  const validStream = stream.filter((e: any) => e && typeof e === 'object' && e.type);
  
  console.log('[Collapse] displayEvents: total stream length:', validStream.length);
  
  // Track task changes for todo_write events
  // This works for both real-time streaming and historical messages
  let lastTask: string | null = null;
  const result: any[] = [];
  
  for (let i = 0; i < validStream.length; i++) {
    const event = validStream[i];
    
    // Check if this is a todo_write event with task change
    if (event.type === 'tool_call' && event.tool_name === 'todo_write' && event.tool_data?.task) {
      const currentTask = event.tool_data.task;
      
      // If task changed (or is first task), insert a task change event before the todo_write event
      // For historical messages, we need to show the first task as well
      if (lastTask === null || currentTask !== lastTask) {
        result.push({
          type: 'plan_task_change',
          task: currentTask,
          event_id: `plan-task-change-${event.tool_call_id || i}`,
          timestamp: event.timestamp || Date.now()
        });
      }
      
      lastTask = currentTask;
    }
    
    result.push(event);
  }
  
  // If not done, show everything (with task change events)
  if (!isConversationDone.value) {
    console.log('[Collapse] displayEvents: not done, showing all', result.length);
    return result;
  }
  
  // If done but user wants to see intermediate steps, show all
  if (showIntermediateSteps.value) {
    console.log('[Collapse] displayEvents: user expanded, showing all', result.length);
    return result;
  }
  
  // Otherwise, show only final content
  const final = finalContent.value;
  if (!final) {
    console.log('[Collapse] displayEvents: no final content, showing all', result.length);
    return result;
  }
  
  if (final.type === 'answer') {
    // Filter to show only answer events
    const filtered = result.filter((e: any) => e.type === 'answer');
    console.log('[Collapse] displayEvents: showing answer only', filtered.length);
    return filtered;
  } else if (final.type === 'thinking') {
    // Filter to show only the last thinking
    const filtered = result.filter((e: any) => 
      e.type === 'thinking' && e.event_id === final.event_id
    );
    console.log('[Collapse] displayEvents: showing last thinking only', filtered.length);
    return filtered;
  }
  
  console.log('[Collapse] displayEvents: fallback, showing all', result.length);
  return result;
});

// Get unique key for event
const getEventKey = (event: any, index: number): string => {
  if (!event) return `event-${index}`;
  if (event.event_id) return `event-${event.event_id}`;
  if (event.tool_call_id) return `tool-${event.tool_call_id}`;
  return `event-${index}-${event.type || 'unknown'}`;
};

const toggleIntermediateSteps = () => {
  showIntermediateSteps.value = !showIntermediateSteps.value;
};

const toggleEvent = (eventId: string) => {
  if (expandedEvents.value.has(eventId)) {
    expandedEvents.value.delete(eventId);
  } else {
    expandedEvents.value.add(eventId);
  }
};

const isEventExpanded = (eventId: string): boolean => {
  return expandedEvents.value.has(eventId);
};

// Markdown rendering function
const renderMarkdown = (content: any): string => {
  if (!content) return '';
  
  // Ensure content is a string
  const contentStr = typeof content === 'string' ? content : String(content || '');
  if (!contentStr.trim()) return '';
  
  try {
    const html = marked.parse(contentStr) as string;
    if (!html) return '';
    
    return DOMPurify.sanitize(html, {
      ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'u', 'code', 'pre', 'ul', 'ol', 'li', 'blockquote', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'a', 'table', 'thead', 'tbody', 'tr', 'th', 'td'],
      ALLOWED_ATTR: ['href', 'title', 'target', 'rel']
    });
  } catch (e) {
    console.error('Markdown rendering error:', e, 'Content:', contentStr.substring(0, 100));
    // Return escaped HTML instead of raw content for safety
    return contentStr.replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }
};

// Tool summary - extract key info to display externally
const getToolSummary = (event: any): string => {
  if (!event || event.pending || !event.success) return '';
  
  const toolName = event.tool_name;
  const toolData = event.tool_data;
  
  // For search tools, don't return summary here - it will be displayed in SearchResults component
  if (toolName === 'search_knowledge' || toolName === 'knowledge_search') {
    return '';
  } else if (toolName === 'get_document_info') {
    if (toolData?.title) {
      return `获取文档：${toolData.title}`;
    }
  } else if (toolName === 'get_related_chunks') {
    if (toolData?.count !== undefined) {
      return `找到 ${toolData.count} 个相关片段`;
    }
  } else if (toolName === 'todo_write') {
    // Extract steps from tool data
    const steps = toolData?.steps;
    if (Array.isArray(steps)) {
      const inProgress = steps.filter((s: any) => s.status === 'in_progress').length;
      const pending = steps.filter((s: any) => s.status === 'pending').length;
      const completed = steps.filter((s: any) => s.status === 'completed').length;
      
      const parts = [];
      if (inProgress > 0) parts.push(`🚀 进行中 ${inProgress}`);
      if (pending > 0) parts.push(`📋 待处理 ${pending}`);
      if (completed > 0) parts.push(`✅ 已完成 ${completed}`);
      
      return parts.join(' · ');
    }
  } else if (toolName === 'thinking') {
    // Return truthy value to trigger rendering, actual content rendered in template
    return toolData?.thought ? '深度思考' : '';
  }
  
  return '';
};

// Get plan status parts for todo_write tool header
const getPlanStatusParts = (event: any) => {
  if (!event || !event.tool_data?.steps) {
    return { inProgress: 0, pending: 0, completed: 0 };
  }
  
  const steps = event.tool_data.steps;
  if (!Array.isArray(steps)) {
    return { inProgress: 0, pending: 0, completed: 0 };
  }
  
  return {
    inProgress: steps.filter((s: any) => s.status === 'in_progress').length,
    pending: steps.filter((s: any) => s.status === 'pending').length,
    completed: steps.filter((s: any) => s.status === 'completed').length
  };
};

// Get plan status items for display with icons
const getPlanStatusItems = (event: any) => {
  const parts = getPlanStatusParts(event);
  const items: Array<{ icon: string; class: string; label: string; count: number }> = [];
  
  if (parts.inProgress > 0) {
    items.push({
      icon: 'play-circle-filled',
      class: 'in-progress',
      label: '进行中',
      count: parts.inProgress
    });
  }
  
  if (parts.pending > 0) {
    items.push({
      icon: 'time',
      class: 'pending',
      label: '待处理',
      count: parts.pending
    });
  }
  
  if (parts.completed > 0) {
    items.push({
      icon: 'check-circle-filled',
      class: 'completed',
      label: '已完成',
      count: parts.completed
    });
  }
  
  return items;
};

// Get plan status summary for todo_write tool header (deprecated, use getPlanStatusParts instead)
const getPlanStatusSummary = (event: any): string => {
  const parts = getPlanStatusParts(event);
  const textParts = [];
  if (parts.inProgress > 0) textParts.push(`🚀 进行中 ${parts.inProgress}`);
  if (parts.pending > 0) textParts.push(`📋 待处理 ${parts.pending}`);
  if (parts.completed > 0) textParts.push(`✅ 已完成 ${parts.completed}`);
  return textParts.length > 0 ? textParts.join(' · ') : '';
};

// Check if tool should use book icon
const isBookIcon = (toolName: string): boolean => {
  return false; // 不再使用 t-icon 的 book，改用 SVG 图标
};

// Get icon for tool type
const getToolIcon = (toolName: string): string => {
  if (toolName === 'thinking') {
    return thinkingIcon;
  } else if (toolName === 'search_knowledge' || toolName === 'knowledge_search') {
    return knowledgeIcon;
  } else if (toolName === 'get_document_info' || toolName === 'get_related_chunks') {
    return documentIcon;
  } else if (toolName === 'todo_write') {
    return fileAddIcon;
  } else {
    return documentIcon; // default icon
  }
};

// Get search results summary text
const getSearchResultsSummary = (toolData: any): string => {
  if (!toolData) return '';
  
  const count = toolData.results?.length || toolData.count || 0;
  if (count === 0) return '';
  
  const kbCount = toolData.kb_counts ? Object.keys(toolData.kb_counts).length : 0;
  if (kbCount > 0) {
    return `找到 ${count} 个结果，来自 ${kbCount} 个知识库`;
  }
  return `找到 ${count} 个结果`;
};

// Extract and format query parameters from args
const getQueryText = (args: any): string => {
  if (!args) return '';
  
  // Parse if it's a string
  let parsedArgs = args;
  if (typeof parsedArgs === 'string') {
    try {
      parsedArgs = JSON.parse(parsedArgs);
    } catch (e) {
      return '';
    }
  }
  
  if (!parsedArgs || typeof parsedArgs !== 'object') return '';
  
  const queries: string[] = [];
  
  // Add query if exists
  if (parsedArgs.query && typeof parsedArgs.query === 'string') {
    queries.push(parsedArgs.query);
  }
  
  // Add vector_queries if exists
  if (Array.isArray(parsedArgs.vector_queries) && parsedArgs.vector_queries.length > 0) {
    const vectorQueries = parsedArgs.vector_queries
      .filter((q: any) => q && typeof q === 'string')
      .join(' ');
    if (vectorQueries) {
      queries.push(vectorQueries);
    }
  }
  
  // Add keyword_queries if exists
  if (Array.isArray(parsedArgs.keyword_queries) && parsedArgs.keyword_queries.length > 0) {
    const keywordQueries = parsedArgs.keyword_queries
      .filter((q: any) => q && typeof q === 'string')
      .join(' ');
    if (keywordQueries) {
      queries.push(keywordQueries);
    }
  }
  
  // Join all queries with space and remove duplicates
  const uniqueQueries = Array.from(new Set(queries));
  return uniqueQueries.join(' ');
};

// Get tool title - prefer summary over description, add query for search tools
const getToolTitle = (event: any): string => {
  if (event.pending) {
    return `正在调用 ${event.tool_name}...`;
  }
  
  const toolName = event.tool_name;
  const isSearchTool = toolName === 'search_knowledge' || toolName === 'knowledge_search';
  
  // For search tools, use description with query text
  if (isSearchTool) {
    const baseTitle = getToolDescription(event);
    if (event.arguments) {
      const queryText = getQueryText(event.arguments);
      if (queryText) {
        return `${baseTitle}：「${queryText}」`;
      }
    }
    return baseTitle;
  }
  
  // Use tool summary if available
  const summary = getToolSummary(event);
  return summary || getToolDescription(event);
};

// Tool description
const getToolDescription = (event: any): string => {
  if (event.pending) {
    return `正在调用 ${event.tool_name}...`;
  }
  
  const success = event.success === true;
  const toolName = event.tool_name;
  
  if (toolName === 'search_knowledge' || toolName === 'knowledge_search') {
    return success ? '检索知识库' : '检索知识库失败';
  } else if (toolName === 'get_document_info') {
    return success ? '获取文档信息' : '获取文档信息失败';
  } else if (toolName === 'thinking') {
    return success ? '完成思考' : '思考失败';
  } else if (toolName === 'todo_write') {
    return success ? '更新任务列表' : '更新任务列表失败';
  } else {
    return success ? `调用 ${toolName}` : `调用 ${toolName} 失败`;
  }
};

// Helper functions
const formatDuration = (ms?: number): string => {
  if (!ms) return '0s';
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
};

const formatJSON = (obj: any): string => {
  try {
    if (typeof obj === 'string') {
      // Try to parse if it's a JSON string
      try {
        const parsed = JSON.parse(obj);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return obj;
      }
    }
    return JSON.stringify(obj, null, 2);
  } catch {
    return String(obj);
  }
};
</script>

<style lang="less" scoped>
@import '../../../components/css/markdown.less';

.agent-stream-display {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.intermediate-steps-collapsed {
  display: flex;
  flex-direction: column;
  font-size: 14px;
  width: 100%;
  border-radius: 8px;
  background-color: #ffffff;
  border-left: 3px solid #07c05f;
  box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
  overflow: hidden;
  box-sizing: border-box;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  margin-bottom: 8px;
  
  .intermediate-steps-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    color: #333333;
    font-weight: 500;
    cursor: pointer;
  }
  
  .intermediate-steps-title {
    display: flex;
    align-items: center;
    
    img {
      width: 16px;
      height: 16px;
      margin-right: 8px;
    }
    
    span {
      white-space: nowrap;
      font-size: 14px;
    }
  }
  
  .intermediate-steps-show-icon {
    font-size: 14px;
    padding: 0 2px 1px 2px;
    color: #07c05f;
  }
  
  .intermediate-steps-header:hover {
    background-color: rgba(7, 192, 95, 0.04);
  }
}

.event-item {
  margin-bottom: 0;
}

// Thinking Event
.thinking-event {
  animation: fadeInUp 0.3s ease-out;
  
  .thinking-phase {
    background: #ffffff;
    border-radius: 8px;
    padding: 1px 14px;
    // 默认不显示 border-left
    border-left: 3px solid transparent;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    
    // 只有最后一个 thinking 显示绿色 border-left
    &.thinking-last {
      border-left-color: #07c05f;
      box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
    }
    
    // 进行中的 thinking（实时对话）显示绿色 border-left 和动画
    &.thinking-active {
      border-left-color: #07c05f;
      box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
      animation: pulseBorder 2s ease-in-out infinite;
    }
  }
  
  .thinking-content {
    font-size: 14px;
    color: #333333;
    line-height: 1.6;
    
    &.markdown-content {
      :deep(p) {
        margin: 8px 0;
        line-height: 1.6;
      }
      
      :deep(code) {
        background: #f0f0f0;
        padding: 2px 6px;
        border-radius: 3px;
        font-family: 'Monaco', 'Courier New', monospace;
        font-size: 12px;
      }
      
      :deep(pre) {
        background: #f5f5f5;
        padding: 12px;
        border-radius: 4px;
        overflow-x: auto;
        margin: 8px 0;
        
        code {
          background: none;
          padding: 0;
        }
      }
      
      :deep(ul), :deep(ol) {
        margin: 8px 0;
        padding-left: 24px;
      }
      
      :deep(li) {
        margin: 4px 0;
      }
      
      :deep(blockquote) {
        border-left: 3px solid #07c05f;
        padding-left: 12px;
        margin: 8px 0;
        color: #666;
      }
      
      :deep(h1), :deep(h2), :deep(h3), :deep(h4), :deep(h5), :deep(h6) {
        margin: 12px 0 8px 0;
        font-weight: 600;
        color: #333;
      }
      
      :deep(a) {
        color: #07c05f;
        text-decoration: none;
        
        &:hover {
          text-decoration: underline;
        }
      }
      
      :deep(table) {
        border-collapse: collapse;
        margin: 8px 0;
        font-size: 12px;
        
        th, td {
          border: 1px solid #e5e7eb;
          padding: 6px 10px;
        }
        
        th {
          background: #f5f5f5;
          font-weight: 600;
        }
      }
    }
  }
}

// Answer Event - 类似 thinking 但有独特样式
.answer-event {
  animation: fadeInUp 0.3s ease-out;
  
  .answer-content-wrapper {
    background: #ffffff;
    border-radius: 8px;
    padding: 1px 14px;
    border-left: 3px solid #07c05f;
    box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    
    // 进行中的 answer 显示动画
    &.answer-active {
      animation: pulseBorder 2s ease-in-out infinite;
    }
    
    // 完成的 answer 保持绿色边框
    &.answer-done {
      border-left-color: #07c05f;
    }
  }
  
  .answer-content {
    font-size: 14px;
    color: #333333;
    line-height: 1.6;
    
    &.markdown-content {
      :deep(p) {
        margin: 8px 0;
        line-height: 1.6;
      }
      
      :deep(code) {
        background: #f0f0f0;
        padding: 2px 6px;
        border-radius: 3px;
        font-family: 'Monaco', 'Courier New', monospace;
        font-size: 12px;
      }
      
      :deep(pre) {
        background: #f5f5f5;
        padding: 12px;
        border-radius: 4px;
        overflow-x: auto;
        margin: 8px 0;
        
        code {
          background: none;
          padding: 0;
        }
      }
      
      :deep(ul), :deep(ol) {
        margin: 8px 0;
        padding-left: 24px;
      }
      
      :deep(li) {
        margin: 4px 0;
      }
      
      :deep(blockquote) {
        border-left: 3px solid #07c05f;
        padding-left: 12px;
        margin: 8px 0;
        color: #666;
      }
      
      :deep(h1), :deep(h2), :deep(h3), :deep(h4), :deep(h5), :deep(h6) {
        margin: 12px 0 8px 0;
        font-weight: 600;
        color: #333;
      }
      
      :deep(a) {
        color: #07c05f;
        text-decoration: none;
        
        &:hover {
          text-decoration: underline;
        }
      }
      
      :deep(table) {
        border-collapse: collapse;
        margin: 8px 0;
        font-size: 12px;
        
        th, td {
          border: 1px solid #e5e7eb;
          padding: 6px 10px;
        }
        
        th {
          background: #f5f5f5;
          font-weight: 600;
        }
      }
    }
  }
}

// Tool Event
.tool-event {
  animation: fadeInUp 0.3s ease-out;
  
  .action-card {
    background: #ffffff;
    border-radius: 8px;
    border: 1px solid #e5e7eb;
    overflow: hidden;
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);

    &:hover {
      border-color: #07c05f;
      box-shadow: 0 2px 8px rgba(7, 192, 95, 0.12);
      transform: translateY(-1px);
    }

    &.action-error {
      border-left: 3px solid #e34d59;
      animation: shakeError 0.4s ease-out;
    }
    
    &.action-pending {
      opacity: 0.7;
      box-shadow: none;
    }
  }
  
  .tool-summary {
    padding: 8px 14px;
    font-size: 13px;
    color: #333333;
    background: #ffffff;
    border-top: 1px solid #f0f0f0;
    line-height: 1.6;
    font-weight: 500;
    animation: slideIn 0.25s ease-out;
    
    .tool-summary-markdown {
      font-weight: 400;
      line-height: 1.6;
      color: #333333;
      
      :deep(p) {
        margin: 4px 0;
        color: #333333;
      }
      
      :deep(ul), :deep(ol) {
        margin: 4px 0;
        padding-left: 20px;
      }
      
      :deep(code) {
        background: #f5f5f5;
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 12px;
        color: #07c05f;
        font-weight: 500;
      }
      
      :deep(strong) {
        font-weight: 600;
        color: #333333;
      }
    }
  }
}

.action-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  color: #333333;
  font-weight: 500;
  cursor: pointer;
  user-select: none;
  transition: background-color 0.25s cubic-bezier(0.4, 0, 0.2, 1);

  &:hover {
    background-color: rgba(7, 192, 95, 0.04);
  }
}

.action-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0; // Allow flex item to shrink below content size
  
  .action-title-icon {
    width: 16px;
    height: 16px;
    color: #07c05f;
    fill: currentColor;
    flex-shrink: 0;
    
    // For t-icon component
    :deep(svg) {
      width: 16px;
      height: 16px;
      color: #07c05f;
      fill: currentColor;
    }
  }
  
  // Tooltip wrapper should also allow shrinking
  :deep(.t-tooltip) {
    flex: 1;
    min-width: 0;
  }
  
  .action-name {
    white-space: nowrap;
    font-size: 14px;
  }
}


@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes slideInDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(-8px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes pulseBorder {
  0%, 100% {
    border-left-color: #07c05f;
    box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
  }
  50% {
    border-left-color: #0ae06f;
    box-shadow: 0 2px 6px rgba(7, 192, 95, 0.15);
  }
}

@keyframes shakeError {
  0%, 100% {
    transform: translateX(0);
  }
  10%, 30%, 50%, 70%, 90% {
    transform: translateX(-3px);
  }
  20%, 40%, 60%, 80% {
    transform: translateX(3px);
  }
}

.action-name {
  font-size: 14px;
  font-weight: 500;
  color: #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  max-width: 100%;
  vertical-align: middle;
}

.action-show-icon {
  font-size: 14px;
  padding: 0 2px 1px 2px;
  color: #07c05f;
}

.action-details {
  padding: 0;
  border-top: 1px solid #f0f0f0;
  background: #ffffff;
  display: flex;
  flex-direction: column;
}

.tool-result-wrapper {
  margin: 0;
}

.search-results-summary-fixed {
  padding: 8px 12px;
  background: #f8f9fa;
  border-top: 1px solid #e7e7e7;
  
  .results-summary-text {
    font-size: 13px;
    font-weight: 500;
    color: #333;
    line-height: 1.5;
  }
}

.plan-status-summary-fixed {
  padding: 8px 12px;
  background: #f8f9fa;
  border-top: 1px solid #e7e7e7;
  
  .plan-status-text {
    font-size: 13px;
    font-weight: 500;
    color: #333;
    line-height: 1.5;
    display: flex;
    align-items: center;
    gap: 4px;
    flex-wrap: wrap;
    
    .status-icon {
      font-size: 14px;
      flex-shrink: 0;
      
      &.in-progress {
        color: #07C05F;
      }
      
      &.pending {
        color: #fa8c16;
      }
      
      &.completed {
        color: #07C05F;
      }
    }
    
    .separator {
      color: #999;
      margin: 0 4px;
    }
    
    span:not(.separator) {
      display: inline-flex;
      align-items: center;
      gap: 4px;
    }
  }
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.plan-task-change-event {
  margin-bottom: 8px;
  
  .plan-task-change-card {
    padding: 8px 12px;
    background: #f8f9fa;
    border-radius: 4px;
    border-left: 3px solid #07C05F;
    font-size: 13px;
    color: #333;
    
    .plan-task-change-content {
      strong {
        color: #333;
        font-weight: 600;
        margin-right: 4px;
      }
    }
  }
}

.tool-output-wrapper {
  .detail-output {
    font-size: 13px;
    color: #333;
    background: #ffffff;
    padding: 12px;
    border-radius: 6px;
    white-space: pre-wrap;
    word-break: break-word;
    border: 1px solid #e5e7eb;
    line-height: 1.6;
  }
}

.thinking-thought-content {
  padding: 8px 12px;
  
  .thinking-thought-markdown {
    font-size: 14px;
    color: #333333;
    line-height: 1.6;
    
    :deep(p) {
      margin: 6px 0;
      line-height: 1.6;
      font-size: 14px;
      color: #333333;
      
      &:first-child {
        margin-top: 0;
      }
      
      &:last-child {
        margin-bottom: 0;
      }
    }
    
    :deep(code) {
      background: #f0f0f0;
      padding: 2px 6px;
      border-radius: 3px;
      font-family: 'Monaco', 'Courier New', monospace;
      font-size: 12px;
      color: #333;
    }
    
    :deep(pre) {
      background: #f5f5f5;
      padding: 8px;
      border-radius: 4px;
      overflow-x: auto;
      margin: 6px 0;
      
      code {
        background: none;
        padding: 0;
      }
    }
    
    :deep(ul), :deep(ol) {
      margin: 6px 0;
      padding-left: 24px;
    }
    
    :deep(li) {
      margin: 2px 0;
      line-height: 1.6;
    }
    
    :deep(blockquote) {
      border-left: 3px solid #07c05f;
      margin: 6px 0;
      color: #666;
      background: rgba(7, 192, 95, 0.05);
      padding: 6px 12px;
      border-radius: 4px;
    }
    
    :deep(h1), :deep(h2), :deep(h3), :deep(h4), :deep(h5), :deep(h6) {
      margin: 8px 0 4px 0;
      font-weight: 600;
      color: #333;
      
      &:first-child {
        margin-top: 0;
      }
    }
    
    :deep(a) {
      color: #07c05f;
      text-decoration: none;
      
      &:hover {
        text-decoration: underline;
      }
    }
    
    :deep(table) {
      border-collapse: collapse;
      margin: 6px 0;
      font-size: 12px;
      
      th, td {
        border: 1px solid #e5e7eb;
        padding: 6px 10px;
      }
      
      th {
        background: #f5f5f5;
        font-weight: 600;
      }
    }
  }
}

.tool-arguments-wrapper {
  margin-top: 8px;
  
  .arguments-header {
    margin-bottom: 6px;
    
    .arguments-label {
      font-size: 12px;
      font-weight: 600;
      color: #666;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
  }
  
  .detail-code {
    font-size: 12px;
    background: #ffffff;
    padding: 10px;
    border-radius: 6px;
    font-family: 'Monaco', 'Courier New', monospace;
    color: #333;
    margin: 0;
    overflow-x: auto;
    border: 1px solid #e5e7eb;
    line-height: 1.5;
  }
}

.loading-indicator {
  display: flex;
  align-items: center;
  padding: 8px 0;
  margin-top: 8px;
  animation: fadeInUp 0.3s ease-out;
  
  .botanswer_loading_gif {
    width: 24px;
    height: 18px;
    margin-left: 0;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

</style>
