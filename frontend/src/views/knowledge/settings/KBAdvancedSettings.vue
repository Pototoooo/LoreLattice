<template>
  <div class="kb-advanced-settings">
    <div class="section-header">
      <h2>高级设置</h2>
      <p class="section-description">配置多模态、知识图谱等高级功能</p>
    </div>

    <div class="settings-group">
      <!-- 多模态功能 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>多模态功能</label>
          <p class="desc">启用图片、视频等多模态内容的理解能力</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localMultimodal.enabled"
            @change="handleMultimodalToggle"
            size="large"
          />
        </div>
      </div>

      <!-- 多模态存储配置 -->
      <div v-if="localMultimodal.enabled" class="subsection">
        <!-- VLLM 视觉模型 -->
        <div class="setting-row">
          <div class="setting-info">
            <label>VLLM 视觉模型 <span class="required">*</span></label>
            <p class="desc">用于多模态理解的视觉语言模型（必选）</p>
          </div>
          <div class="setting-control">
            <ModelSelector
              ref="vllmSelectorRef"
              model-type="VLLM"
              :selected-model-id="localMultimodal.vllmModelId"
              :all-models="allModels"
              @update:selected-model-id="handleVLLMChange"
              @add-model="handleAddModel('vllm')"
              placeholder="请选择VLLM模型（必选）"
            />
          </div>
        </div>

        <div class="subsection-header">
          <h4>存储配置 <span class="required">*</span></h4>
        </div>
        
        <div class="setting-row">
          <div class="setting-info">
            <label>存储类型 <span class="required">*</span></label>
            <p class="desc">选择多模态文件的存储方式（MinIO或腾讯云COS二选一）</p>
          </div>
          <div class="setting-control">
            <t-radio-group v-model="localMultimodal.storageType" @change="handleStorageTypeChange">
              <t-radio value="minio">MinIO</t-radio>
              <t-radio value="cos">腾讯云COS</t-radio>
            </t-radio-group>
          </div>
        </div>

        <!-- MinIO配置 -->
        <div v-if="localMultimodal.storageType === 'minio'" class="storage-config">
          <div class="setting-row">
            <div class="setting-info">
              <label>Bucket名称 <span class="required">*</span></label>
              <p class="desc">MinIO存储桶名称（必填）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.minio.bucketName"
                placeholder="请输入Bucket名称（必填）"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>使用SSL</label>
              <p class="desc">是否使用SSL连接</p>
            </div>
            <div class="setting-control">
              <t-switch
                v-model="localMultimodal.minio.useSSL"
                @change="handleConfigChange"
                size="large"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>路径前缀</label>
              <p class="desc">文件存储路径前缀（可选）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.minio.pathPrefix"
                placeholder="请输入路径前缀"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>
        </div>

        <!-- COS配置 -->
        <div v-if="localMultimodal.storageType === 'cos'" class="storage-config">
          <div class="setting-row">
            <div class="setting-info">
              <label>SecretId <span class="required">*</span></label>
              <p class="desc">腾讯云API密钥ID（必填）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.cos.secretId"
                placeholder="请输入SecretId（必填）"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>SecretKey <span class="required">*</span></label>
              <p class="desc">腾讯云API密钥Key（必填）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.cos.secretKey"
                type="password"
                placeholder="请输入SecretKey（必填）"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>地域 <span class="required">*</span></label>
              <p class="desc">COS存储桶所在地域（必填）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.cos.region"
                placeholder="如：ap-guangzhou（必填）"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>Bucket名称 <span class="required">*</span></label>
              <p class="desc">COS存储桶名称（必填）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.cos.bucketName"
                placeholder="请输入Bucket名称（必填）"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>AppId <span class="required">*</span></label>
              <p class="desc">腾讯云应用ID（必填）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.cos.appId"
                placeholder="请输入AppId（必填）"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>路径前缀</label>
              <p class="desc">文件存储路径前缀（可选）</p>
            </div>
            <div class="setting-control">
              <t-input
                v-model="localMultimodal.cos.pathPrefix"
                placeholder="请输入路径前缀"
                @change="handleConfigChange"
                style="width: 280px;"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- 知识图谱提取 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>知识图谱提取</label>
          <p class="desc">从文档中自动提取实体和关系构建知识图谱</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localNodeExtract.enabled"
            @change="handleNodeExtractToggle"
            size="large"
          />
        </div>
      </div>

      <!-- 知识图谱配置 -->
      <div v-if="localNodeExtract.enabled" class="subsection">
        <div class="subsection-header">
          <h4>图谱配置</h4>
        </div>
        
        <div class="setting-row">
          <div class="setting-info">
            <label>提示文本</label>
            <p class="desc">用于引导模型提取实体和关系的提示文本</p>
          </div>
          <div class="setting-control">
            <t-textarea
              v-model="localNodeExtract.text"
              placeholder="请输入提示文本"
              :autosize="{ minRows: 3, maxRows: 6 }"
              @change="handleConfigChange"
              style="width: 280px;"
            />
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label>标签</label>
            <p class="desc">预定义的实体标签（多个标签用逗号分隔）</p>
          </div>
          <div class="setting-control">
            <t-tag-input
              v-model="localNodeExtract.tags"
              placeholder="输入标签后按回车"
              @change="handleConfigChange"
              style="width: 280px;"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ModelSelector from '@/components/ModelSelector.vue'
import { useUIStore } from '@/stores/ui'

const uiStore = useUIStore()

interface MultimodalConfig {
  enabled: boolean
  storageType: 'minio' | 'cos'
  vllmModelId?: string
  minio: {
    bucketName: string
    useSSL: boolean
    pathPrefix: string
  }
  cos: {
    secretId: string
    secretKey: string
    region: string
    bucketName: string
    appId: string
    pathPrefix: string
  }
}

interface NodeExtractConfig {
  enabled: boolean
  text: string
  tags: string[]
}

interface Props {
  multimodal: MultimodalConfig
  nodeExtract: NodeExtractConfig
  allModels?: any[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:multimodal': [value: MultimodalConfig]
  'update:nodeExtract': [value: NodeExtractConfig]
}>()

const localMultimodal = ref<MultimodalConfig>({ ...props.multimodal })
const localNodeExtract = ref<NodeExtractConfig>({ ...props.nodeExtract })

const vllmSelectorRef = ref()

// 监听props变化
watch(() => props.multimodal, (newVal) => {
  localMultimodal.value = { ...newVal }
}, { deep: true })

watch(() => props.nodeExtract, (newVal) => {
  localNodeExtract.value = { ...newVal }
}, { deep: true })

// 处理多模态开关
const handleMultimodalToggle = () => {
  // 如果关闭多模态，清空相关配置
  if (!localMultimodal.value.enabled) {
    localMultimodal.value.vllmModelId = ''
    localMultimodal.value.minio = {
      bucketName: '',
      useSSL: false,
      pathPrefix: ''
    }
    localMultimodal.value.cos = {
      secretId: '',
      secretKey: '',
      region: '',
      bucketName: '',
      appId: '',
      pathPrefix: ''
    }
  }
  emit('update:multimodal', localMultimodal.value)
}

// 处理存储类型变化
const handleStorageTypeChange = () => {
  emit('update:multimodal', localMultimodal.value)
}

// 处理VLLM模型变化
const handleVLLMChange = (modelId: string) => {
  localMultimodal.value.vllmModelId = modelId
  emit('update:multimodal', localMultimodal.value)
}

// 处理添加模型
const handleAddModel = (subSection: string) => {
  uiStore.openSettings('models', subSection)
}

// 处理知识图谱开关
const handleNodeExtractToggle = () => {
  emit('update:nodeExtract', localNodeExtract.value)
}

// 处理配置变化
const handleConfigChange = () => {
  emit('update:multimodal', localMultimodal.value)
  emit('update:nodeExtract', localNodeExtract.value)
}

// 由于使用了 allModels prop，不再需要单独刷新
</script>

<style lang="less" scoped>
.kb-advanced-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: #333333;
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: #666666;
    margin: 0;
    line-height: 1.5;
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid #e5e7eb;

  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  flex: 1;
  max-width: 65%;
  padding-right: 24px;

  label {
    font-size: 15px;
    font-weight: 500;
    color: #333333;
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: 13px;
    color: #666666;
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.subsection {
  padding: 16px 20px;
  margin: 12px 0 0 0;
  background: #f8fafb;
  border-radius: 8px;
  border-left: 3px solid #07C05F;
  position: relative;
}

.subsection-header {
  margin: 16px 0 8px 0;
  
  &:first-child {
    margin-top: 0;
  }
  
  h4 {
    font-size: 15px;
    font-weight: 600;
    color: #333333;
    margin: 0;
    padding-left: 8px;
    border-left: 2px solid #07C05F;
    
    .required {
      color: #e34d59;
      margin-left: 4px;
    }
  }
}

.required {
  color: #e34d59;
  margin-left: 2px;
  font-weight: 500;
}

.storage-config {
  margin-top: 8px;
}
</style>

