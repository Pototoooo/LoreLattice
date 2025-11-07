<script setup lang="ts">
import { ref, defineEmits, onMounted, defineProps, computed, watch, nextTick, h } from "vue";
import { useRoute, useRouter } from 'vue-router';
import { onBeforeRouteUpdate } from 'vue-router';
import { MessagePlugin } from "tdesign-vue-next";
import { useSettingsStore } from '@/stores/settings';
import { useUIStore } from '@/stores/ui';
import { listKnowledgeBases } from '@/api/knowledge-base';
import KnowledgeBaseSelector from './KnowledgeBaseSelector.vue';

const route = useRoute();
const router = useRouter();
const settingsStore = useSettingsStore();
const uiStore = useUIStore();
let query = ref("");
const showKbSelector = ref(false);
const atButtonRef = ref<HTMLElement>();

const props = defineProps({
  isReplying: {
    type: Boolean,
    required: false
  }
});

const isAgentEnabled = computed(() => settingsStore.isAgentEnabled);
const selectedKbIds = computed(() => settingsStore.settings.selectedKnowledgeBases || []);

// 获取已选择的知识库信息
const knowledgeBases = ref<Array<{ id: string; name: string }>>([]);
const selectedKbs = computed(() => {
  return knowledgeBases.value.filter(kb => selectedKbIds.value.includes(kb.id));
});

// 显示的知识库标签（最多显示2个）
const displayedKbs = computed(() => selectedKbs.value.slice(0, 2));
const remainingCount = computed(() => Math.max(0, selectedKbs.value.length - 2));

// 加载知识库列表
const loadKnowledgeBases = async () => {
  try {
    const response: any = await listKnowledgeBases();
    if (response.data && Array.isArray(response.data)) {
      knowledgeBases.value = response.data.filter((kb: any) => 
        kb.embedding_model_id && kb.embedding_model_id !== '' &&
        kb.summary_model_id && kb.summary_model_id !== ''
      );
    }
  } catch (error) {
    console.error('Failed to load knowledge bases:', error);
  }
};

onMounted(() => {
  loadKnowledgeBases();
  
  // 如果从知识库内部进入，自动选中该知识库
  const kbId = (route.params as any)?.kbId as string;
  if (kbId && !selectedKbIds.value.includes(kbId)) {
    settingsStore.addKnowledgeBase(kbId);
  }
});

// 监听路由变化
watch(() => route.params.kbId, (newKbId) => {
  if (newKbId && typeof newKbId === 'string' && !selectedKbIds.value.includes(newKbId)) {
    settingsStore.addKnowledgeBase(newKbId);
  }
});

const emit = defineEmits(['send-msg']);

const createSession = (val: string) => {
  if (!val.trim()) {
    MessagePlugin.info("请先输入内容!");
    return;
  }
  if (selectedKbIds.value.length === 0) {
    MessagePlugin.warning("请先选择知识库!");
    return;
  }
  if (props.isReplying) {
    return MessagePlugin.error("正在回复中，请稍后再试!");
  }
  emit('send-msg', val);
  clearvalue();
}

const clearvalue = () => {
  query.value = "";
}

const onKeydown = (val: string, event: { e: { preventDefault(): unknown; keyCode: number; shiftKey: any; ctrlKey: any; }; }) => {
  if ((event.e.keyCode == 13 && event.e.shiftKey) || (event.e.keyCode == 13 && event.e.ctrlKey)) {
    return;
  }
  if (event.e.keyCode == 13) {
    event.e.preventDefault();
    createSession(val)
  }
}

const handleGoToAgentSettings = () => {
  // 使用 uiStore 打开设置并跳转到 agent 部分
  uiStore.openSettings('agent');
  // 如果当前不在设置页面，导航到设置页面
  if (route.path !== '/platform/settings') {
    router.push('/platform/settings');
  }
}

const toggleAgentMode = () => {
  // 如果要启用 Agent，先检查是否就绪
  // 注意：isAgentReady 是从 store 中计算的，需要确保 store 中的配置是最新的
  if (!isAgentEnabled.value) {
    // 尝试启用 Agent，先检查是否就绪
    const agentReady = settingsStore.isAgentReady
    if (!agentReady) {
      // 创建带跳转链接的自定义消息
      const messageContent = h('div', { style: 'display: flex; align-items: center; gap: 8px; flex-wrap: wrap;' }, [
        h('span', { style: 'flex: 1; min-width: 0;' }, 'Agent 未就绪，请先在设置中完成 Agent 配置（思考模型、Rerank 模型和允许的工具）'),
        h('a', {
          href: '#',
          onClick: (e: Event) => {
            e.preventDefault();
            handleGoToAgentSettings();
          },
          style: 'color: #07C05F; text-decoration: none; font-weight: 500; cursor: pointer; white-space: nowrap; flex-shrink: 0;',
          onMouseenter: (e: Event) => {
            (e.target as HTMLElement).style.textDecoration = 'underline';
          },
          onMouseleave: (e: Event) => {
            (e.target as HTMLElement).style.textDecoration = 'none';
          }
        }, '去设置 →')
      ]);
      
      MessagePlugin.warning({
        content: messageContent,
        duration: 5000
      });
      return
    }
  }
  
  // 正常切换 Agent 状态
  settingsStore.toggleAgent(!isAgentEnabled.value);
  const message = isAgentEnabled.value ? 'Agent 模式已启用' : 'Agent 模式已禁用';
  MessagePlugin.success(message);
}

const toggleKbSelector = () => {
  showKbSelector.value = !showKbSelector.value;
}

const removeKb = (kbId: string) => {
  settingsStore.removeKnowledgeBase(kbId);
}

onBeforeRouteUpdate((to, from, next) => {
  clearvalue()
  next()
})

</script>
<template>
  <div class="answers-input">
    <t-textarea v-model="query" placeholder="基于知识库提问" name="description" :autosize="true" @keydown="onKeydown" />
    
    <!-- 控制栏 -->
    <div class="control-bar">
      <!-- 左侧控制按钮 -->
      <div class="control-left">
        <!-- Agent 模式按钮 -->
        <t-tooltip 
          placement="top"
          :show-arrow="true"
          :duration="0"
        >
          <template #content>
            <div style="display: flex; flex-direction: column; gap: 4px; align-items: flex-start;">
              <span>
                {{ isAgentEnabled ? 'Agent 模式（已启用）' : (settingsStore.isAgentReady ? 'Agent 模式（已禁用，点击启用）' : 'Agent 未就绪，请先完成配置') }}
              </span>
              <a 
                href="#"
                @click.prevent="handleGoToAgentSettings"
                style="color: #07C05F; text-decoration: none; font-size: 12px; cursor: pointer;"
                @mouseenter="(e) => e.target.style.textDecoration = 'underline'"
                @mouseleave="(e) => e.target.style.textDecoration = 'none'"
              >
                去设置 Agent →
              </a>
            </div>
          </template>
          <div 
            class="control-btn agent-btn"
            :class="{ 
              'active': isAgentEnabled,
              'disabled': !isAgentEnabled && !settingsStore.isAgentReady
            }"
            @click="toggleAgentMode"
          >
            <img 
              :src="isAgentEnabled ? getImgSrc('agent-active.svg') : getImgSrc('agent.svg')" 
              alt="Agent" 
              class="control-icon"
            />
          </div>
        </t-tooltip>

        <!-- @ 知识库选择按钮 -->
        <div 
          ref="atButtonRef"
          class="control-btn kb-btn"
          :class="{ 'active': selectedKbIds.length > 0 }"
          @click="toggleKbSelector"
        >
          <img :src="getImgSrc('at-icon.svg')" alt="@" class="control-icon" />
          <span class="kb-btn-text">
            {{ selectedKbIds.length > 0 ? `知识库(${selectedKbIds.length})` : '知识库' }}
          </span>
          <svg 
            width="12" 
            height="12" 
            viewBox="0 0 12 12" 
            fill="currentColor"
            class="dropdown-arrow"
            :class="{ 'rotate': showKbSelector }"
          >
            <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z"/>
          </svg>
        </div>

        <!-- 已选择的知识库标签 -->
        <div v-if="displayedKbs.length > 0" class="kb-tags">
          <div 
            v-for="kb in displayedKbs" 
            :key="kb.id" 
            class="kb-tag"
          >
            <span class="kb-tag-text">{{ kb.name }}</span>
            <span class="kb-tag-remove" @click.stop="removeKb(kb.id)">×</span>
          </div>
          <div v-if="remainingCount > 0" class="kb-tag more-tag">
            +{{ remainingCount }}
          </div>
        </div>
      </div>

      <!-- 右侧发送按钮 -->
      <div 
        @click="createSession(query)" 
        class="control-btn send-btn"
        :class="{ 'disabled': !query.length || selectedKbIds.length === 0 }"
      >
        <img src="../assets/img/sending-aircraft.svg" alt="发送" />
      </div>
    </div>

    <!-- 知识库选择下拉（使用 Teleport 传送到 body，避免父容器定位影响） -->
    <Teleport to="body">
    <KnowledgeBaseSelector
      v-model:visible="showKbSelector"
        :anchorEl="atButtonRef"
      @close="showKbSelector = false"
    />
    </Teleport>
  </div>
</template>
<script lang="ts">
const getImgSrc = (url: string) => {
  return new URL(`/src/assets/img/${url}`, import.meta.url).href;
}
</script>
<style scoped lang="less">
.answers-input {
  position: absolute;
  z-index: 99;
  bottom: 60px;
  left: 50%;
  transform: translateX(-400px);
}

:deep(.t-textarea__inner) {
  width: 100%;
  width: 800px;
  max-height: 250px !important;
  min-height: 112px !important;
  resize: none;
  color: #000000e6;
  font-size: 16px;
  font-weight: 400;
  line-height: 24px;
  font-family: "PingFang SC";
  padding: 16px 12px 52px 16px;  /* 增加底部padding为控制栏腾出空间 */
  border-radius: 12px;
  border: 1px solid #E7E7E7;
  box-sizing: border-box;
  background: #FFF;
  box-shadow: 0 6px 6px 0 #0000000a, 0 12px 12px -1px #00000014;

  &:focus {
    border: 1px solid #07C05F;
  }

  &::placeholder {
    color: #00000066;
    font-family: "PingFang SC";
    font-size: 16px;
    font-weight: 400;
    line-height: 24px;
  }
}

/* 控制栏 */
.control-bar {
  position: absolute;
  bottom: 12px;
  left: 16px;
  right: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.control-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  overflow: hidden;
}

.control-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 10px;
  border-radius: 6px;
  background: #f5f5f5;
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
  flex-shrink: 0;

  &:hover {
    background: #e6e6e6;
    transform: scale(1.02);
  }

  &:active {
    transform: scale(0.98);
  }

  &.disabled {
    opacity: 0.5;
    cursor: not-allowed;
    
    &:hover {
      background: #f5f5f5;
      transform: none;
    }
  }
}

.agent-btn {
  width: 28px;
  height: 28px;
  padding: 0;

  &.active {
    background: rgba(7, 192, 95, 0.1);
    
    &:hover {
      background: rgba(7, 192, 95, 0.15);
    }
  }

  &.disabled {
    opacity: 0.5;
    cursor: not-allowed;
    
    &:hover {
      background: #f5f5f5;
      transform: none;
    }
  }
}

.control-icon {
  width: 18px;
  height: 18px;
  transition: all 0.2s ease;
}

.kb-btn {
  height: 28px;
  padding: 0 10px;
  min-width: auto;
  
  &.active {
    background: rgba(7, 192, 95, 0.1);
    color: #07C05F;
    
    &:hover {
      background: rgba(7, 192, 95, 0.15);
    }
  }
}

.kb-btn-text {
  font-size: 13px;
  color: #666;
  font-weight: 500;
  white-space: nowrap;
}

.kb-btn.active .kb-btn-text {
  color: #07C05F;
}

.dropdown-arrow {
  transition: transform 0.2s ease;
  width: 10px;
  height: 10px;
  margin-left: 2px;
  
  &.rotate {
    transform: rotate(180deg);
  }
}

.kb-tags {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  overflow-x: auto;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
}

.kb-tag {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: rgba(7, 192, 95, 0.1);
  border: 1px solid rgba(7, 192, 95, 0.3);
  border-radius: 4px;
  font-size: 12px;
  color: #07C05F;
  white-space: nowrap;
  transition: all 0.15s ease;
  
  &:hover {
    background: rgba(7, 192, 95, 0.15);
  }
}

.kb-tag-text {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kb-tag-remove {
  cursor: pointer;
  font-weight: bold;
  font-size: 16px;
  line-height: 1;
  opacity: 0.7;
  
  &:hover {
    opacity: 1;
  }
}

.more-tag {
  background: rgba(0, 0, 0, 0.05);
  border-color: rgba(0, 0, 0, 0.1);
  color: #666;
  
  &:hover {
    background: rgba(0, 0, 0, 0.08);
  }
}

.send-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  background-color: #07C05F;
  
  &:hover:not(.disabled) {
    background-color: #00a651;
    transform: scale(1.05);
  }
  
  &.disabled {
    background-color: #b5eccf;
  }
  
  img {
    width: 16px;
    height: 16px;
  }
}
</style>
