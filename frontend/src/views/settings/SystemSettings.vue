<template>
    <div class="system-settings-container">
        <!-- 页面标题区域 -->
        <div class="settings-header">
            <h2>系统设置</h2>
            <p class="settings-subtitle">管理和更新系统模型与服务配置</p>
        </div>
        
        <!-- 配置内容 -->
        <div class="settings-content">
            <!-- 直接引用初始化配置组件的内容，不使用外层卡片包装 -->
            <InitializationContent />
        </div>
    </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'

// 异步加载初始化配置组件
const InitializationContent = defineAsyncComponent(() => import('../initialization/InitializationContent.vue'))
</script>

<style lang="less" scoped>
.system-settings-container {
    padding: 20px;
    background-color: #fff;
    margin: 0 20px 0 20px;
    height: calc(100vh);
    overflow-y: auto;
    box-sizing: border-box;
    flex: 1;
}

.settings-header {
    margin-bottom: 20px;
    border-bottom: 1px solid #f0f0f0;
    padding-bottom: 16px;

    h2 {
        font-size: 20px;
        font-weight: 600;
        color: #000000;
        margin: 0 0 8px 0;
    }

    .settings-subtitle {
        font-size: 14px;
        color: #666666;
        margin: 0;
    }
}

.settings-content {
    margin-top: 0;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .system-settings-container {
        padding: 16px;
        margin: 10px;
        height: calc(100vh - 20px);
    }
    
    .settings-header h2 {
        font-size: 18px;
    }
}

/* 覆盖TDesign组件样式，与账户信息页面保持一致 */
:deep(.t-card) {
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    border: 1px solid #e5e7eb;
}

/* 调整InitializationContent内部样式，使每个配置区域显示为独立卡片 */
:deep(.config-section) {
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 20px;
    margin-bottom: 20px;
    
    &:last-child {
        margin-bottom: 0;
    }
    
    h3 {
        font-size: 16px;
        font-weight: 600;
        color: #000000;
        margin: 0 0 16px 0;
        display: flex;
        align-items: center;
        padding: 0;
        background: none;
        border-left: none;
        border-radius: 0;
        border-bottom: 1px solid #f0f0f0;
        padding-bottom: 12px;
        
        .section-icon {
            margin-right: 8px;
            color: #07c05f;
            font-size: 18px;
        }
    }
}

:deep(.ollama-summary-card) {
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 20px;
    margin-bottom: 20px;
}

:deep(.submit-section) {
    margin-top: 20px;
    text-align: center;
}
</style>
