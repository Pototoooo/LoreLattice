<template>
  <div class="agent-stream-display">
    
    <!-- Collapsed intermediate steps -->
    <div v-if="shouldShowCollapsedSteps" class="intermediate-steps-collapsed" @click="toggleIntermediateSteps">
      <span class="collapse-icon">{{ showIntermediateSteps ? '▼' : '▶' }}</span>
      <span class="collapse-text">
        {{ intermediateStepsCount }} intermediate step{{ intermediateStepsCount !== 1 ? 's' : '' }} (click to {{ showIntermediateSteps ? 'collapse' : 'expand' }})
      </span>
    </div>
    
    <!-- Event Stream -->
    <template v-for="(event, index) in displayEvents" :key="getEventKey(event, index)">
      <div v-if="event && event.type" class="event-item">
        
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
        <div v-else-if="event.type === 'answer'" class="answer-event">
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
              <span class="action-icon" :class="{ 
                'success': event.success === true, 
                'error': event.success === false,
                'pending': event.pending
              }">
                {{ event.pending ? '○' : (event.success ? '✓' : '✗') }}
              </span>
              <!-- Custom header for todo_write tool -->
              <span v-if="event.tool_name === 'todo_write' && event.tool_data?.steps" class="action-name">
                制定计划 {{ getPlanStatusSummary(event) }}
              </span>
              <!-- Default header for other tools -->
              <span v-else class="action-name">{{ getToolDescription(event) }}</span>
            </div>
            <span v-if="!event.pending" class="expand-icon">
              {{ isEventExpanded(event.tool_call_id) ? '▼' : '▶' }}
            </span>
          </div>
          
          <!-- External Tool Summary -->
          <div v-if="getToolSummary(event) && !event.pending" class="tool-summary">
            <!-- Thinking tool: render markdown -->
            <div v-if="event.tool_name === 'thinking' && event.tool_data?.thought" 
                 class="tool-summary-markdown" 
                 v-html="renderMarkdown(event.tool_data.thought)">
            </div>
            <!-- Other tools: plain text -->
            <span v-else>{{ getToolSummary(event) }}</span>
          </div>
          
          <div v-if="isEventExpanded(event.tool_call_id) && !event.pending" class="action-details">
            <div v-if="event.tool_name" class="detail-row">
              <span class="detail-label">Tool:</span>
              <code class="detail-value">{{ event.tool_name }}</code>
            </div>
            <!-- Hide Arguments for todo_write tool -->
            <div v-if="event.arguments && event.tool_name !== 'todo_write'" class="detail-row">
              <span class="detail-label">Arguments:</span>
              <pre class="detail-code">{{ formatJSON(event.arguments) }}</pre>
            </div>
            
            <!-- Use ToolResultRenderer if display_type is available -->
            <div v-if="event.display_type && event.tool_data" class="detail-row">
              <ToolResultRenderer 
                :display-type="event.display_type"
                :tool-data="event.tool_data"
                :output="event.output"
              />
            </div>
            
            <!-- Fallback to original output display -->
            <div v-else-if="event.output" class="detail-row">
              <span class="detail-label">{{ event.success ? 'Output:' : 'Error:' }}</span>
              <div class="detail-output">{{ event.output }}</div>
            </div>
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
const expandedEvents = ref<Set<string>>(new Set());

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

// Check if conversation is done (based on answer event with done=true)
const isConversationDone = computed(() => {
  const stream = eventStream.value;
  if (!stream || stream.length === 0) {
    console.log('[Collapse] No stream or empty stream');
    return false;
  }
  
  // Only check for answer event with done=true
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
  
  // If not done, show everything
  if (!isConversationDone.value) {
    console.log('[Collapse] displayEvents: not done, showing all', validStream.length);
    return validStream;
  }
  
  // If done but user wants to see intermediate steps, show all
  if (showIntermediateSteps.value) {
    console.log('[Collapse] displayEvents: user expanded, showing all', validStream.length);
    return validStream;
  }
  
  // Otherwise, show only final content
  const final = finalContent.value;
  if (!final) {
    console.log('[Collapse] displayEvents: no final content, showing all', validStream.length);
    return validStream;
  }
  
  if (final.type === 'answer') {
    // Filter to show only answer events
    const filtered = validStream.filter((e: any) => e.type === 'answer');
    console.log('[Collapse] displayEvents: showing answer only', filtered.length);
    return filtered;
  } else if (final.type === 'thinking') {
    // Filter to show only the last thinking
    const filtered = validStream.filter((e: any) => 
      e.type === 'thinking' && e.event_id === final.event_id
    );
    console.log('[Collapse] displayEvents: showing last thinking only', filtered.length);
    return filtered;
  }
  
  console.log('[Collapse] displayEvents: fallback, showing all', validStream.length);
  return validStream;
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
  
  if (toolName === 'search_knowledge' || toolName === 'knowledge_search') {
    if (toolData?.count !== undefined) {
      const kbCount = toolData.kb_counts ? Object.keys(toolData.kb_counts).length : 0;
      if (kbCount > 0) {
        return `Found ${toolData.count} result${toolData.count !== 1 ? 's' : ''} from ${kbCount} knowledge base${kbCount !== 1 ? 's' : ''}`;
      }
      return `Found ${toolData.count} result${toolData.count !== 1 ? 's' : ''}`;
    }
  } else if (toolName === 'get_document_info') {
    if (toolData?.title) {
      return `Retrieved document: ${toolData.title}`;
    }
  } else if (toolName === 'get_related_chunks') {
    if (toolData?.count !== undefined) {
      return `Found ${toolData.count} related chunk${toolData.count !== 1 ? 's' : ''}`;
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
    return toolData?.thought ? 'thinking_content' : '';
  }
  
  return '';
};

// Get plan status summary for todo_write tool header
const getPlanStatusSummary = (event: any): string => {
  if (!event || !event.tool_data?.steps) return '';
  
  const steps = event.tool_data.steps;
  if (!Array.isArray(steps)) return '';
  
  const inProgress = steps.filter((s: any) => s.status === 'in_progress').length;
  const pending = steps.filter((s: any) => s.status === 'pending').length;
  
  const parts = [];
  if (inProgress > 0) parts.push(`🚀 进行中 ${inProgress}`);
  if (pending > 0) parts.push(`📋 待处理 ${pending}`);
  
  return parts.length > 0 ? parts.join(' · ') : '';
};

// Tool description
const getToolDescription = (event: any): string => {
  if (event.pending) {
    return `Calling ${event.tool_name}...`;
  }
  
  const success = event.success === true;
  const toolName = event.tool_name;
  
  if (toolName === 'search_knowledge' || toolName === 'knowledge_search') {
    return success ? 'Searched knowledge base' : 'Failed to search knowledge base';
  } else if (toolName === 'get_document_info') {
    return success ? 'Retrieved document information' : 'Failed to get document info';
  } else if (toolName === 'thinking') {
    return success ? 'Completed thinking' : 'Thinking failed';
  } else if (toolName === 'todo_write') {
    return success ? 'Updated task list' : 'Failed to update tasks';
  } else {
    return success ? `Called ${toolName}` : `Error calling ${toolName}`;
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
  cursor: pointer;
  padding: 10px 14px;
  background: #ffffff;
  border-left: 3px solid #07c05f;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  user-select: none;
  margin-bottom: 8px;
  box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
  animation: slideInDown 0.3s ease-out;
  
  &:hover {
    background: rgba(7, 192, 95, 0.04);
    box-shadow: 0 2px 6px rgba(7, 192, 95, 0.12);
  }
  
  &:active {
    background: rgba(7, 192, 95, 0.08);
  }
  
  .collapse-icon {
    font-size: 12px;
    color: #07c05f;
    transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    font-weight: 600;
  }
  
  .collapse-text {
    font-size: 14px;
    color: #333333;
    font-weight: 500;
    letter-spacing: -0.01em;
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
  align-items: center;
  justify-content: space-between;
  padding: 6px 14px;
  cursor: pointer;
  user-select: none;
  transition: background-color 0.25s cubic-bezier(0.4, 0, 0.2, 1);

  &:hover {
    background: rgba(7, 192, 95, 0.04);
  }
  
  &:active {
    background: rgba(7, 192, 95, 0.08);
  }
}

.action-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.action-icon {
  font-size: 14px;
  font-weight: bold;
  width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;

  &.success {
    color: #00a870;
    background: rgba(0, 168, 112, 0.1);
  }

  &.error {
    color: #e34d59;
    background: rgba(227, 77, 89, 0.1);
  }
  
  &.pending {
    color: #666666;
    background: rgba(102, 102, 102, 0.1);
    animation: pulse 1.5s ease-in-out infinite;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
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
}

.expand-icon {
  font-size: 12px;
  color: #07c05f;
  transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.2s ease;
  font-weight: 600;
  opacity: 0.8;
  display: inline-block;
  
  &:hover {
    opacity: 1;
  }
}

.action-header:hover .expand-icon {
  transform: scale(1.1);
}

.action-details {
  padding: 0 12px 12px 12px;
  border-top: 1px solid #e5e7eb;
  background: #fff;
}

.detail-row {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  font-weight: 600;
  color: #666;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-value {
  font-size: 13px;
  color: #07c05f;
  background: rgba(7, 192, 95, 0.05);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Courier New', monospace;
  display: inline-block;
}

.detail-code {
  font-size: 12px;
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  font-family: 'Monaco', 'Courier New', monospace;
  color: #333;
  margin: 0;
  overflow-x: auto;
  border: 1px solid #e5e7eb;
}

.detail-output {
  font-size: 13px;
  color: #333;
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-word;
  border: 1px solid #e5e7eb;
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
