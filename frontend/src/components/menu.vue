<template>
    <div class="aside_box">
        <div class="logo_box" @click="router.push('/platform/knowledge-bases')" style="cursor: pointer;">
            <img class="logo" src="@/assets/img/weknora.png" alt="">
        </div>
        
        <!-- 上半部分：知识库和对话 -->
        <div class="menu_top">
            <div class="menu_box" :class="{ 'has-submenu': item.children }" v-for="(item, index) in topMenuItems" :key="index">
                <div @click="handleMenuClick(item.path)"
                    @mouseenter="mouseenteMenu(item.path)" @mouseleave="mouseleaveMenu(item.path)"
                     :class="['menu_item', item.childrenPath && item.childrenPath == currentpath ? 'menu_item_c_active' : isMenuItemActive(item.path) ? 'menu_item_active' : '']">
                    <div class="menu_item-box">
                        <div class="menu_icon">
                            <img class="icon" :src="getImgSrc(item.icon == 'zhishiku' ? knowledgeIcon :  item.icon == 'logout' ? logoutIcon : item.icon == 'tenant' ? tenantIcon : prefixIcon)" alt="">
                        </div>
                        <span class="menu_title" :title="item.path === 'knowledge-bases' && kbMenuItem ? kbMenuItem.title : item.title">{{ item.path === 'knowledge-bases' && kbMenuItem ? kbMenuItem.title : item.title }}</span>
                        <!-- 知识库切换下拉箭头 -->
                        <div v-if="item.path === 'knowledge-bases' && isInKnowledgeBase" 
                             class="kb-dropdown-icon" 
                             :class="{ 
                                 'rotate-180': showKbDropdown,
                                 'active': isMenuItemActive(item.path)
                             }"
                             @click.stop="toggleKbDropdown">
                            <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
                                <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z"/>
                            </svg>
                        </div>
                    </div>
                    <!-- 知识库切换下拉菜单 -->
                    <div v-if="item.path === 'knowledge-bases' && showKbDropdown && isInKnowledgeBase" 
                         class="kb-dropdown-menu">
                        <div v-for="kb in initializedKnowledgeBases" 
                             :key="kb.id" 
                             class="kb-dropdown-item"
                             :class="{ 'active': kb.name === currentKbName }"
                             @click.stop="switchKnowledgeBase(kb.id)">
                            {{ kb.name }}
                        </div>
                    </div>
                    <t-popup overlayInnerClassName="upload-popup" class="placement top center" content="上传知识"
                        placement="top" show-arrow destroy-on-close>
                        <div class="upload-file-wrap" @click.stop="uploadFile" variant="outline"
                             v-if="item.path === 'knowledge-bases' && $route.name === 'knowledgeBaseDetail'">
                            <img class="upload-file-icon" :class="[item.path == currentpath ? 'active-upload' : '']"
                                :src="getImgSrc(fileAddIcon)" alt="">
                        </div>
                    </t-popup>
                </div>
                <div ref="submenuscrollContainer" @scroll="handleScroll" class="submenu" v-if="item.children">
                    <template v-for="(group, groupIndex) in groupedSessions" :key="groupIndex">
                        <div class="timeline_header">{{ group.label }}</div>
                        <div class="submenu_item_p" v-for="(subitem, subindex) in group.items" :key="subitem.id">
                            <div :class="['submenu_item', currentSecondpath == subitem.path ? 'submenu_item_active' : '']"
                                @mouseenter="mouseenteBotDownr(subitem.id)" @mouseleave="mouseleaveBotDown"
                                @click="gotopage(subitem.path)">
                                <span class="submenu_title"
                                    :style="currentSecondpath == subitem.path ? 'margin-left:18px;max-width:160px;' : 'margin-left:18px;max-width:185px;'">
                                    {{ subitem.title }}
                                </span>
                                <t-dropdown 
                                    :options="[{ content: '删除记录', value: 'delete' }]"
                                    @click="(data) => data.value === 'delete' && delCard(subitem.originalIndex, subitem)"
                                    placement="bottom-right"
                                    trigger="click">
                                    <div @click.stop class="menu-more-wrap">
                                        <t-icon name="ellipsis" class="menu-more" />
                                    </div>
                                </t-dropdown>
                            </div>
                        </div>
                    </template>
                </div>
            </div>
        </div>
        
        
        <!-- 下半部分：用户菜单 -->
        <div class="menu_bottom">
            <UserMenu />
        </div>
        
        <input type="file" @change="upload" style="display: none" ref="uploadInput"
            accept=".pdf,.docx,.doc,.txt,.md,.jpg,.jpeg,.png" />
    </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { onMounted, watch, computed, ref, reactive, nextTick } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getSessionsList, delSession } from "@/api/chat/index";
import { getKnowledgeBaseById, listKnowledgeBases, uploadKnowledgeFile } from '@/api/knowledge-base';
import { logout as logoutApi } from '@/api/auth';
import { kbFileTypeVerification } from '@/utils/index';
import { useMenuStore } from '@/stores/menu';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';
import { MessagePlugin } from "tdesign-vue-next";
import UserMenu from '@/components/UserMenu.vue';
let uploadInput = ref();
const usemenuStore = useMenuStore();
const authStore = useAuthStore();
const uiStore = useUIStore();
const route = useRoute();
const router = useRouter();
const currentpath = ref('');
const currentPage = ref(1);
const page_size = ref(30);
const total = ref(0);
const currentSecondpath = ref('');
const submenuscrollContainer = ref(null);
// 计算总页数
const totalPages = computed(() => Math.ceil(total.value / page_size.value));
const hasMore = computed(() => currentPage.value < totalPages.value);
type MenuItem = { title: string; icon: string; path: string; childrenPath?: string; children?: any[] };
const { menuArr } = storeToRefs(usemenuStore);
let activeSubmenu = ref<string>('');

// 是否处于知识库详情页（不包括全局聊天）
const isInKnowledgeBase = computed<boolean>(() => {
    return route.name === 'knowledgeBaseDetail' || 
           route.name === 'kbCreatChat' || 
           route.name === 'knowledgeBaseSettings';
});

// 统一的菜单项激活状态判断
const isMenuItemActive = (itemPath: string): boolean => {
    const currentRoute = route.name;
    
    switch (itemPath) {
        case 'knowledge-bases':
            return currentRoute === 'knowledgeBaseList' || 
                   currentRoute === 'knowledgeBaseDetail' || 
                   currentRoute === 'knowledgeBaseSettings';
        case 'creatChat':
            return currentRoute === 'kbCreatChat' || currentRoute === 'globalCreatChat';
        case 'settings':
            return currentRoute === 'settings';
        default:
            return itemPath === currentpath.value;
    }
};

// 统一的图标激活状态判断
const getIconActiveState = (itemPath: string) => {
    const currentRoute = route.name;
    
    return {
        isKbActive: itemPath === 'knowledge-bases' && (
            currentRoute === 'knowledgeBaseList' || 
            currentRoute === 'knowledgeBaseDetail' || 
            currentRoute === 'knowledgeBaseSettings'
        ),
        isCreatChatActive: itemPath === 'creatChat' && (currentRoute === 'kbCreatChat' || currentRoute === 'globalCreatChat'),
        isSettingsActive: itemPath === 'settings' && currentRoute === 'settings',
        isChatActive: itemPath === 'chat' && currentRoute === 'chat'
    };
};

// 分离上下两部分菜单
const topMenuItems = computed<MenuItem[]>(() => {
    return (menuArr.value as unknown as MenuItem[]).filter((item: MenuItem) => 
        item.path === 'knowledge-bases' || item.path === 'creatChat'
    );
});

const bottomMenuItems = computed<MenuItem[]>(() => {
    return (menuArr.value as unknown as MenuItem[]).filter((item: MenuItem) => {
        if (item.path === 'knowledge-bases' || item.path === 'creatChat') {
            return false;
        }
        return true;
    });
});

// 当前知识库名称和列表
const currentKbName = ref<string>('')
const allKnowledgeBases = ref<Array<{ id: string; name: string; embedding_model_id?: string; summary_model_id?: string }>>([])
const showKbDropdown = ref<boolean>(false)

// 过滤已初始化的知识库
const initializedKnowledgeBases = computed(() => {
    return allKnowledgeBases.value.filter(kb => 
        kb.embedding_model_id && kb.embedding_model_id !== '' && 
        kb.summary_model_id && kb.summary_model_id !== ''
    )
})

// 时间分组函数
const getTimeCategory = (dateStr: string): string => {
    if (!dateStr) return '更早';
    
    const date = new Date(dateStr);
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000);
    const sevenDaysAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);
    const thirtyDaysAgo = new Date(today.getTime() - 30 * 24 * 60 * 60 * 1000);
    const oneYearAgo = new Date(today.getTime() - 365 * 24 * 60 * 60 * 1000);
    
    const sessionDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    
    if (sessionDate.getTime() >= today.getTime()) {
        return '今天';
    } else if (sessionDate.getTime() >= yesterday.getTime()) {
        return '昨天';
    } else if (date.getTime() >= sevenDaysAgo.getTime()) {
        return '近7天';
    } else if (date.getTime() >= thirtyDaysAgo.getTime()) {
        return '近30天';
    } else if (date.getTime() >= oneYearAgo.getTime()) {
        return '近1年';
    } else {
        return '更早';
    }
};

// 按时间分组Session列表
const groupedSessions = computed(() => {
    const chatMenu = (menuArr.value as unknown as MenuItem[]).find((item: MenuItem) => item.path === 'creatChat');
    if (!chatMenu || !chatMenu.children || chatMenu.children.length === 0) {
        return [];
    }
    
    const groups: { [key: string]: any[] } = {
        '今天': [],
        '昨天': [],
        '近7天': [],
        '近30天': [],
        '近1年': [],
        '更早': []
    };
    
    // 将sessions按时间分组
    (chatMenu.children as any[]).forEach((session: any, index: number) => {
        const category = getTimeCategory(session.updated_at || session.created_at);
        groups[category].push({
            ...session,
            originalIndex: index
        });
    });
    
    // 按顺序返回非空分组
    const orderedLabels = ['今天', '昨天', '近7天', '近30天', '近1年', '更早'];
    return orderedLabels
        .filter(label => groups[label].length > 0)
        .map(label => ({
            label,
            items: groups[label]
        }));
});

// 动态更新知识库菜单项标题
const kbMenuItem = computed(() => {
    const kbItem = topMenuItems.value.find(item => item.path === 'knowledge-bases')
    if (kbItem && isInKnowledgeBase.value && currentKbName.value) {
        return { ...kbItem, title: currentKbName.value }
    }
    return kbItem
})

const loading = ref(false)
const uploadFile = async () => {
    // 获取当前知识库ID
    const currentKbId = await getCurrentKbId();
    
    // 检查当前知识库的初始化状态
    if (currentKbId) {
        try {
            const kbResponse = await getKnowledgeBaseById(currentKbId);
            const kb = kbResponse.data;
            
            // 检查知识库是否已初始化（有 EmbeddingModelID 和 SummaryModelID）
            if (!kb.embedding_model_id || kb.embedding_model_id === '' || 
                !kb.summary_model_id || kb.summary_model_id === '') {
                MessagePlugin.warning("该知识库尚未完成初始化配置，请先前往设置页面配置模型信息后再上传文件");
                return;
            }
        } catch (error) {
            console.error('获取知识库信息失败:', error);
            MessagePlugin.error("获取知识库信息失败，无法上传文件");
            return;
        }
    }
    
    uploadInput.value.click()
}
const upload = async (e: any) => {
    const file = e.target.files[0];
    if (!file) return;
    
    // 文件类型验证
    if (kbFileTypeVerification(file)) {
        return;
    }
    
    // 获取当前知识库ID
    const currentKbId = (route.params as any)?.kbId as string;
    if (!currentKbId) {
        MessagePlugin.error("缺少知识库ID");
        return;
    }
    
    try {
        const result = await uploadKnowledgeFile(currentKbId, { file });
        const responseData = result as any;
        console.log('上传API返回结果:', responseData);
        
        // 如果没有抛出异常，就认为上传成功，先触发刷新事件
        console.log('文件上传完成，发送事件通知页面刷新，知识库ID:', currentKbId);
        window.dispatchEvent(new CustomEvent('knowledgeFileUploaded', { 
            detail: { kbId: currentKbId } 
        }));
        
        // 然后处理UI消息
        // 判断上传是否成功 - 检查多种可能的成功标识
        const isSuccess = responseData.success || responseData.code === 200 || responseData.status === 'success' || (!responseData.error && responseData);
        
        if (isSuccess) {
            MessagePlugin.info("上传成功！");
        } else {
            // 改进错误信息提取逻辑
            let errorMessage = "上传失败！";
            if (responseData.error && responseData.error.message) {
                errorMessage = responseData.error.message;
            } else if (responseData.message) {
                errorMessage = responseData.message;
            }
            if (responseData.code === 'duplicate_file' || (responseData.error && responseData.error.code === 'duplicate_file')) {
                errorMessage = "文件已存在";
            }
            MessagePlugin.error(errorMessage);
        }
    } catch (err: any) {
        let errorMessage = "上传失败！";
        if (err.code === 'duplicate_file') {
            errorMessage = "文件已存在";
        } else if (err.error && err.error.message) {
            errorMessage = err.error.message;
        } else if (err.message) {
            errorMessage = err.message;
        }
        MessagePlugin.error(errorMessage);
    } finally {
        uploadInput.value.value = "";
    }
}
const mouseenteBotDownr = (val: string) => {
    activeSubmenu.value = val;
}
const mouseleaveBotDown = () => {
    activeSubmenu.value = '';
}

const delCard = (index: number, item: any) => {
    delSession(item.id).then((res: any) => {
        if (res && (res as any).success) {
            // 使用 originalIndex 找到正确的位置进行删除
            const actualIndex = index !== undefined ? index : 
                (menuArr.value as any[])[1]?.children?.findIndex((s: any) => s.id === item.id);
            
            if (actualIndex !== -1) {
                (menuArr.value as any[])[1]?.children?.splice(actualIndex, 1);
            }
            
            if (item.id == route.params.chatid) {
                // 删除当前会话后，跳转到全局创建聊天页面
                router.push('/platform/creatChat');
            }
        } else {
            MessagePlugin.error("删除失败，请稍后再试!");
        }
    })
}
const debounce = (fn: (...args: any[]) => void, delay: number) => {
    let timer: ReturnType<typeof setTimeout>
    return (...args: any[]) => {
        clearTimeout(timer)
        timer = setTimeout(() => fn(...args), delay)
    }
}
// 滚动处理
const checkScrollBottom = () => {
    const container = submenuscrollContainer.value
    if (!container || !container[0]) return

    const { scrollTop, scrollHeight, clientHeight } = container[0]
    const isBottom = scrollHeight - (scrollTop + clientHeight) < 100 // 触底阈值
    
    if (isBottom && hasMore.value && !loading.value) {
        currentPage.value++;
        getMessageList(true);
    }
}
const handleScroll = debounce(checkScrollBottom, 200)
const getMessageList = async (isLoadMore = false) => {
    // 全局显示对话列表（所有有"对话"入口的地方都显示）
    let kbId = (route.params as any)?.kbId as string
    
    // 如果在知识库内部，获取知识库名称和所有知识库列表
    if (kbId && isInKnowledgeBase.value) {
        try {
            const [kbRes, allKbRes]: any[] = await Promise.all([
                getKnowledgeBaseById(kbId),
                listKnowledgeBases()
            ])
            if (kbRes?.data?.name) {
                currentKbName.value = kbRes.data.name
            }
            if (allKbRes?.data) {
                allKnowledgeBases.value = allKbRes.data
            }
        } catch {}
    } else {
        // 不在知识库内部时，清空知识库名称
        currentKbName.value = '';
        // 不在知识库列表页时才调用API（避免与页面组件KnowledgeBaseList.vue重复调用）
        // 知识库列表页会自己调用listKnowledgeBases来显示列表
        if (route.name !== 'knowledgeBaseList') {
            try {
                const allKbRes: any = await listKnowledgeBases()
                if (allKbRes?.data) {
                    allKnowledgeBases.value = allKbRes.data
                }
            } catch {}
        }
    }
    
    if (loading.value) return Promise.resolve();
    loading.value = true;
    
    // 只有在首次加载或路由变化时才清空数组，滚动加载时不清空
    if (!isLoadMore) {
        currentPage.value = 1; // 重置页码
        usemenuStore.clearMenuArr();
    }
    
    return getSessionsList(currentPage.value, page_size.value).then((res: any) => {
        if (res.data && res.data.length) {
            // Display all sessions globally without filtering
            res.data.forEach((item: any) => {
                let obj = { 
                    title: item.title ? item.title : "新会话", 
                    path: `chat/${item.id}`, 
                    id: item.id, 
                    isMore: false, 
                    isNoTitle: item.title ? false : true,
                    created_at: item.created_at,
                    updated_at: item.updated_at
                }
                usemenuStore.updatemenuArr(obj)
            });
        }
        if ((res as any).total) {
            total.value = (res as any).total;
        }
        loading.value = false;
    }).catch(() => {
        loading.value = false;
    })
}

onMounted(() => {
    const routeName = typeof route.name === 'string' ? route.name : (route.name ? String(route.name) : '')
    currentpath.value = routeName;
    if (route.params.chatid) {
        currentSecondpath.value = `chat/${route.params.chatid}`;
    }
    getMessageList();
});

watch([() => route.name, () => route.params], (newvalue) => {
    const nameStr = typeof newvalue[0] === 'string' ? (newvalue[0] as string) : (newvalue[0] ? String(newvalue[0]) : '')
    currentpath.value = nameStr;
    if (newvalue[1].chatid) {
        currentSecondpath.value = `chat/${newvalue[1].chatid}`;
    } else {
        currentSecondpath.value = "";
    }
    // 路由变化时刷新对话列表
    getMessageList();
    // 路由变化时更新图标状态
    getIcon(nameStr);
});
let fileAddIcon = ref('file-add-green.svg');
let knowledgeIcon = ref('zhishiku-green.svg');
let prefixIcon = ref('prefixIcon.svg');
let logoutIcon = ref('logout.svg');
let settingIcon = ref('setting.svg'); // 设置图标
let pathPrefix = ref(route.name)
  const getIcon = (path: string) => {
      // 根据当前路由状态更新所有图标
      const kbActiveState = getIconActiveState('knowledge-bases');
      const creatChatActiveState = getIconActiveState('creatChat');
      const settingsActiveState = getIconActiveState('settings');
      
      // 上传图标：只在知识库相关页面显示绿色
      fileAddIcon.value = kbActiveState.isKbActive ? 'file-add-green.svg' : 'file-add.svg';
      
      // 知识库图标：只在知识库页面显示绿色
      knowledgeIcon.value = kbActiveState.isKbActive ? 'zhishiku-green.svg' : 'zhishiku.svg';
      
      // 对话图标：只在对话创建页面显示绿色，在知识库页面显示灰色，其他情况显示默认
      prefixIcon.value = creatChatActiveState.isCreatChatActive ? 'prefixIcon-green.svg' : 
                        kbActiveState.isKbActive ? 'prefixIcon-grey.svg' : 
                        'prefixIcon.svg';
      
      // 设置图标：只在设置页面显示绿色
      settingIcon.value = settingsActiveState.isSettingsActive ? 'setting-green.svg' : 'setting.svg';
      
      // 退出图标：始终显示默认
      logoutIcon.value = 'logout.svg';
}
getIcon(typeof route.name === 'string' ? route.name as string : (route.name ? String(route.name) : ''))
const handleMenuClick = async (path: string) => {
    if (path === 'knowledge-bases') {
        // 知识库菜单项：如果在知识库内部，跳转到当前知识库文件页；否则跳转到知识库列表
        const kbId = await getCurrentKbId()
        if (kbId) {
            router.push(`/platform/knowledge-bases/${kbId}`)
        } else {
            router.push('/platform/knowledge-bases')
        }
    } else if (path === 'settings') {
        // 设置菜单项：打开设置弹窗并跳转路由
        uiStore.openSettings()
        router.push('/platform/settings')
    } else {
        gotopage(path)
    }
}

// 处理退出登录确认
const handleLogout = () => {
    gotopage('logout')
}

const getCurrentKbId = async (): Promise<string | null> => {
    let kbId = (route.params as any)?.kbId as string
    // 新的路由格式：/platform/chat/:kbId/:chatid，直接从路由参数获取
    if (!kbId && route.name === 'chat' && (route.params as any)?.kbId) {
        kbId = (route.params as any).kbId
    }
    return kbId || null
}

const gotopage = async (path: string) => {
    pathPrefix.value = path;
    // 处理退出登录
    if (path === 'logout') {
        try {
            // 调用后端API注销
            await logoutApi();
        } catch (error) {
            // 即使API调用失败，也继续执行本地清理
            console.error('注销API调用失败:', error);
        }
        // 清理所有状态和本地存储
        authStore.logout();
        MessagePlugin.success('已退出登录');
        router.push('/login');
        return;
    } else {
        if (path === 'creatChat') {
            // 尝试获取当前知识库ID
            const kbId = await getCurrentKbId()
            if (kbId) {
                // 如果在知识库内部，进入该知识库的对话页
                router.push(`/platform/knowledge-bases/${kbId}/creatChat`)
            } else {
                // 如果不在知识库内，也进入对话创建页，让用户通过 @ 按钮选择知识库
                router.push(`/platform/creatChat`)
            }
        } else {
            router.push(`/platform/${path}`);
        }
    }
    getIcon(path)
}

const getImgSrc = (url: string) => {
    return new URL(`/src/assets/img/${url}`, import.meta.url).href;
}

const mouseenteMenu = (path: string) => {
    if (pathPrefix.value != 'knowledge-bases' && pathPrefix.value != 'creatChat' && path != 'knowledge-bases') {
        prefixIcon.value = 'prefixIcon-grey.svg';
    }
}
const mouseleaveMenu = (path: string) => {
    if (pathPrefix.value != 'knowledge-bases' && pathPrefix.value != 'creatChat' && path != 'knowledge-bases') {
        const nameStr = typeof route.name === 'string' ? route.name as string : (route.name ? String(route.name) : '')
        getIcon(nameStr)
    }
}

// 知识库下拉相关方法
const toggleKbDropdown = (event?: Event) => {
    if (event) {
        event.stopPropagation()
    }
    showKbDropdown.value = !showKbDropdown.value
}

const switchKnowledgeBase = (kbId: string, event?: Event) => {
    if (event) {
        event.stopPropagation()
    }
    showKbDropdown.value = false
    const currentRoute = route.name
    
    // 路由跳转
    if (currentRoute === 'knowledgeBaseDetail') {
        router.push(`/platform/knowledge-bases/${kbId}`)
    } else if (currentRoute === 'kbCreatChat') {
        router.push(`/platform/knowledge-bases/${kbId}/creatChat`)
    } else if (currentRoute === 'knowledgeBaseSettings') {
        router.push(`/platform/knowledge-bases/${kbId}/settings`)
    } else {
        router.push(`/platform/knowledge-bases/${kbId}`)
    }
    
    // 刷新右侧内容 - 通过触发页面重新加载或发送事件
    nextTick(() => {
        // 发送全局事件通知页面刷新知识库内容
        window.dispatchEvent(new CustomEvent('knowledgeBaseChanged', { 
            detail: { kbId } 
        }))
    })
}

// 点击外部关闭下拉菜单
const handleClickOutside = () => {
    showKbDropdown.value = false
}

onMounted(() => {
    document.addEventListener('click', handleClickOutside)
})

watch(() => route.params.kbId, () => {
    showKbDropdown.value = false
})

</script>
<style lang="less" scoped>
.aside_box {
    min-width: 260px;
    padding: 8px;
    background: #fff;
    box-sizing: border-box;
    height: 100vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;

    .logo_box {
        height: 80px;
        display: flex;
        align-items: center;
        .logo{
            width: 134px;
            height: auto;
            margin-left: 24px;
        }
    }

    .logo_img {
        margin-left: 24px;
        width: 30px;
        height: 30px;
        margin-right: 7.25px;
    }

    .logo_txt {
        transform: rotate(0.049deg);
        color: #000000;
        font-family: "TencentSans";
        font-size: 24.12px;
        font-style: normal;
        font-weight: W7;
        line-height: 21.7px;
    }

    .menu_top {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        min-height: 0;
    }

    .menu_bottom {
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
    }

    .menu_box {
        display: flex;
        flex-direction: column;
        
        &.has-submenu {
            flex: 1;
            min-height: 0;
        }
    }


    .upload-file-wrap {
        padding: 6px;
        border-radius: 3px;
        height: 32px;
        width: 32px;
        box-sizing: border-box;
    }

    .upload-file-wrap:hover {
        background-color: #dbede4;
        color: #07C05F;

    }

    .upload-file-icon {
        width: 20px;
        height: 20px;
        color: rgba(0, 0, 0, 0.6);
    }

    .active-upload {
        color: #07C05F;
    }

    .menu_item_active {
        border-radius: 4px;
        background: #07c05f1a !important;

        .menu_icon,
        .menu_title {
            color: #07c05f !important;
        }
    }

    .menu_item_c_active {

        .menu_icon,
        .menu_title {
            color: #000000e6;
        }
    }

    .menu_p {
        height: 56px;
        padding: 6px 0;
        box-sizing: border-box;
    }

    .menu_item {
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: space-between;
        height: 48px;
        padding: 13px 8px 13px 16px;
        box-sizing: border-box;
        margin-bottom: 4px;

        .menu_item-box {
            display: flex;
            align-items: center;
        }

        &:hover {
            border-radius: 4px;
            background: #30323605;
            color: #00000099;

            .menu_icon,
            .menu_title {
                color: #00000099;
            }
        }
    }

    .menu_icon {
        display: flex;
        margin-right: 10px;
        color: #00000099;

        .icon {
            width: 20px;
            height: 20px;
            fill: currentColor;
            overflow: hidden;
        }
    }

    .menu_title {
        color: #00000099;
        text-overflow: ellipsis;
        font-family: "PingFang SC";
        font-size: 14px;
        font-style: normal;
        font-weight: 600;
        line-height: 22px;
        overflow: hidden;
        white-space: nowrap;
        max-width: 120px;
        flex: 1;
    }

    .submenu {
        font-family: "PingFang SC";
        font-size: 14px;
        font-style: normal;
        overflow-y: auto;
        scrollbar-width: none;
        flex: 1;
        min-height: 0;
        margin-left: 4px;
    }
    
    .timeline_header {
        font-family: "PingFang SC";
        font-size: 12px;
        font-weight: 600;
        color: #00000066;
        padding: 12px 18px 6px 18px;
        margin-top: 8px;
        line-height: 20px;
        user-select: none;
        
        &:first-child {
            margin-top: 4px;
        }
    }

    .submenu_item_p {
        height: 44px;
        padding: 4px 0px 4px 0px;
        box-sizing: border-box;
    }


    .submenu_item {
        cursor: pointer;
        display: flex;
        align-items: center;
        color: #00000099;
        font-weight: 400;
        line-height: 22px;
        height: 36px;
        padding-left: 0px;
        padding-right: 14px;
        position: relative;

        .submenu_title {
            overflow: hidden;
            white-space: nowrap;
            text-overflow: ellipsis;
        }

        .menu-more-wrap {
            margin-left: auto;
            opacity: 0;
            transition: opacity 0.2s ease;
        }

        .menu-more {
            display: inline-block;
            font-weight: bold;
            color: #07C05F;
        }

        .sub_title {
            margin-left: 14px;
        }

        &:hover {
            background: #30323605;
            color: #00000099;
            border-radius: 3px;

            .menu-more {
                color: #00000099;
            }

            .menu-more-wrap {
                opacity: 1;
            }

            .submenu_title {
                max-width: 160px !important;

            }
        }
    }

    .submenu_item_active {
        background: #07c05f1a !important;
        color: #07c05f !important;
        border-radius: 3px;

        .menu-more {
            color: #07c05f !important;
        }

        .menu-more-wrap {
            opacity: 1;
        }

        .submenu_title {
            max-width: 160px !important;
        }
    }
}

/* 知识库下拉菜单样式 */
.kb-dropdown-icon {
    margin-left: auto;
    color: #666;
    transition: transform 0.3s ease, color 0.2s ease;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    
    &.rotate-180 {
        transform: rotate(180deg);
    }
    
    &:hover {
        color: #07c05f;
    }
    
    &.active {
        color: #07c05f;
    }
    
    &.active:hover {
        color: #05a04f;
    }
    
    svg {
        width: 12px;
        height: 12px;
        transition: inherit;
    }
}

.kb-dropdown-menu {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    z-index: 1000;
    max-height: 200px;
    overflow-y: auto;
}

.kb-dropdown-item {
    padding: 8px 16px;
    cursor: pointer;
    transition: background-color 0.2s ease;
    font-size: 14px;
    color: #333;
    
    &:hover {
        background-color: #f5f5f5;
    }
    
    &.active {
        background-color: #07c05f1a;
        color: #07c05f;
        font-weight: 500;
    }
    
    &:first-child {
        border-radius: 6px 6px 0 0;
    }
    
    &:last-child {
        border-radius: 0 0 6px 6px;
    }
}

.menu_item-box {
    display: flex;
    align-items: center;
    width: 100%;
    position: relative;
}

.menu_box {
    position: relative;
}
</style>
<style lang="less">
.upload-popup {
    background-color: rgba(0, 0, 0, 0.9);
    color: #FFFFFF;
    border-color: rgba(0, 0, 0, 0.9) !important;
    box-shadow: none;
    margin-bottom: 10px !important;

    .t-popup__arrow::before {
        border-color: rgba(0, 0, 0, 0.9) !important;
        background-color: rgba(0, 0, 0, 0.9) !important;
        box-shadow: none !important;
    }
}


// 退出登录确认框样式
:deep(.t-popconfirm) {
    .t-popconfirm__content {
        background: #fff;
        border: 1px solid #e7e7e7;
        border-radius: 6px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        padding: 12px 16px;
        font-size: 14px;
        color: #333;
        max-width: 200px;
    }
    
    .t-popconfirm__arrow {
        border-bottom-color: #e7e7e7;
    }
    
    .t-popconfirm__arrow::after {
        border-bottom-color: #fff;
    }
    
    .t-popconfirm__buttons {
        margin-top: 8px;
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }
    
    .t-button--variant-outline {
        border-color: #d9d9d9;
        color: #666;
    }
    
    .t-button--theme-danger {
        background-color: #ff4d4f;
        border-color: #ff4d4f;
    }
    
    .t-button--theme-danger:hover {
        background-color: #ff7875;
        border-color: #ff7875;
    }
}
</style>