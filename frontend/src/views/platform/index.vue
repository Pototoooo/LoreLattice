<template>
    <div class="main" ref="dropzone" @dragover="dragover" @drop="drop" @dragstart="dragstart">
        <Menu></Menu>
        <RouterView />
        <div class="upload-mask" v-show="ismask">
            <input type="file" style="display: none" ref="uploadInput" accept=".pdf,.docx,.doc,.txt,.md" />
            <UploadMask></UploadMask>
        </div>
    </div>
</template>
<script setup lang="ts">
import Menu from '@/components/menu.vue'
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router'
import { storeToRefs } from "pinia";
import { knowledgeStore } from "@/stores/knowledge";
const usemenuStore = knowledgeStore();
import useKnowledgeBase from '@/hooks/useKnowledgeBase'
import UploadMask from '@/components/upload-mask.vue'
import { getKnowledgeBaseById } from '@/api/knowledge-base/index'
import { MessagePlugin } from 'tdesign-vue-next'
let { requestMethod } = useKnowledgeBase()
const router = useRouter();
const route = useRoute();
let ismask = ref(false)
let dropzone = ref();
let uploadInput = ref();

// 获取当前知识库ID
const getCurrentKbId = async (): Promise<string | null> => {
    let kbId = (route.params as any)?.kbId as string
    if (!kbId && route.name === 'chat' && (route.params as any)?.kbId) {
        kbId = (route.params as any).kbId
    }
    return kbId || null
}

// 检查知识库初始化状态
const checkKnowledgeBaseInitialization = async (): Promise<boolean> => {
    const currentKbId = await getCurrentKbId();
    
    if (!currentKbId) {
        MessagePlugin.error("缺少知识库ID");
        return false;
    }
    
    try {
        const kbResponse = await getKnowledgeBaseById(currentKbId);
        const kb = kbResponse.data;
        
        // 检查知识库是否已初始化（有 EmbeddingModelID 和 SummaryModelID）
        if (!kb.embedding_model_id || kb.embedding_model_id === '' || 
            !kb.summary_model_id || kb.summary_model_id === '') {
            MessagePlugin.warning("该知识库尚未完成初始化配置，请先前往设置页面配置模型信息后再上传文件");
            return false;
        }
        return true;
    } catch (error) {
        console.error('获取知识库信息失败:', error);
        MessagePlugin.error("获取知识库信息失败，无法上传文件");
        return false;
    }
}

const dragover = (event: DragEvent) => {
    event.preventDefault();
    ismask.value = true;
    if (((window.innerWidth - event.clientX) < 50) || ((window.innerHeight - event.clientY) < 50) || event.clientX < 50 || event.clientY < 50) {
        ismask.value = false
    }
}
const drop = async (event: DragEvent) => {
    event.preventDefault();
    ismask.value = false
    
    // 检查知识库初始化状态
    const isInitialized = await checkKnowledgeBaseInitialization();
    if (!isInitialized) {
        return;
    }
    
    const DataTransferItemList = event.dataTransfer?.items;
    if (DataTransferItemList) {
        for (const dataTransferItem of DataTransferItemList) {
            const fileEntry = dataTransferItem.webkitGetAsEntry() as FileSystemFileEntry | null;
            if (fileEntry) {
                fileEntry.file((file: File) => {
                    requestMethod(file, uploadInput)
                    // 修复页面跳转问题：不跳转，让上传成功后的逻辑处理
                })
            }
        }
    }
}
const dragstart = (event: DragEvent) => {
    event.preventDefault();
}
</script>
<style lang="less">
.main {
    display: flex;
    width: 100%;
    height: 100%;
    min-width: 600px;
}

.upload-mask {
    background-color: rgba(255, 255, 255, 0.8);
    position: fixed;
    width: 100%;
    height: 100%;
    z-index: 999;
    display: flex;
    justify-content: center;
    align-items: center;
}

img {
    -webkit-user-drag: none;
    -khtml-user-drag: none;
    -moz-user-drag: none;
    -o-user-drag: none;
    user-drag: none;
}
</style>