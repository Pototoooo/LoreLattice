<script setup lang="ts">
import { onMounted } from 'vue'
import { useUIStore } from '@/stores/ui'

const uiStore = useUIStore()

// 在应用启动时初始化主题
onMounted(() => {
  uiStore.initTheme()
})
</script>
<template>
  <div id="app">
    <RouterView />
  </div>
</template>
<style>
body,
html,
#app {
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 0;
    font-size: 14px;
    font-family: Helvetica Neue, Helvetica, PingFang SC, Hiragino Sans GB,
        Microsoft YaHei, SimSun, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    background: var(--td-bg-color-page);
}

/* 暗色模式下的TDesign组件样式增强 */
:root[theme-mode="dark"] {
    /* 表格表头 */
    .t-table thead th {
        color: var(--td-text-color-primary) !important;
        background-color: var(--td-bg-color-container) !important;
        font-weight: 600 !important;
    }
    
    /* 表格内容 */
    .t-table tbody td {
        color: var(--td-text-color-primary) !important;
        background-color: var(--td-bg-color-container) !important;
    }
    
    /* 表格行hover效果 */
    .t-table tbody tr:hover td {
        background-color: var(--td-bg-color-container-hover) !important;
    }
    
    /* 表格边框 */
    .t-table,
    .t-table th,
    .t-table td {
        border-color: var(--td-component-border) !important;
    }
    
    /* 滑块标记文字 */
    .t-slider__mark-text {
        color: var(--td-text-color-primary) !important;
        font-weight: 500 !important;
    }
    
    /* 滑块刻度 */
    .t-slider__stops .t-slider__stop {
        background-color: var(--td-text-color-placeholder) !important;
    }
    
    /* 对话框 */
    .t-dialog {
        background: var(--td-bg-color-container) !important;
        color: var(--td-text-color-primary) !important;
    }
    
    .t-dialog__header {
        color: var(--td-text-color-primary) !important;
        border-bottom-color: var(--td-component-border) !important;
    }
    
    .t-dialog__body {
        color: var(--td-text-color-primary) !important;
    }
    
    /* 表单标签 */
    .t-form__label {
        color: var(--td-text-color-primary) !important;
        font-weight: 500 !important;
    }
    
    /* 输入框 */
    .t-input,
    .t-textarea {
        background-color: var(--td-bg-color-component) !important;
        color: var(--td-text-color-primary) !important;
        border-color: var(--td-component-border) !important;
    }
    
    .t-input::placeholder,
    .t-textarea::placeholder {
        color: var(--td-text-color-placeholder) !important;
    }
    
    /* 下拉选择器 */
    .t-select-input {
        background-color: var(--td-bg-color-component) !important;
        color: var(--td-text-color-primary) !important;
        border-color: var(--td-component-border) !important;
    }
    
    .t-select__placeholder {
        color: var(--td-text-color-placeholder) !important;
    }
    
    /* 开关组件 */
    .t-switch__label {
        color: var(--td-text-color-primary) !important;
    }
    
    /* 标签页 */
    .t-tabs__nav-item {
        color: var(--td-text-color-secondary) !important;
    }
    
    .t-tabs__nav-item.t-is-active {
        color: var(--td-brand-color) !important;
    }
}
</style>
