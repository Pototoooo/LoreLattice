import { ref, computed, reactive } from 'vue'

import { defineStore } from 'pinia';

export const useMenuStore = defineStore('menuStore', {
    state: () => ({
        menuArr: reactive([
            { title: '知识库', icon: 'zhishiku', path: 'knowledge-bases' },
            {
                title: '对话',
                icon: 'prefixIcon',
                path: 'creatChat',
                childrenPath: 'chat',
                children: reactive<object[]>([]),
            },
            { title: '设置', icon: 'setting', path: 'settings' },
            { title: '退出登录', icon: 'logout', path: 'logout' }
        ]),
        isFirstSession: false,
        firstQuery: ''
    }
    ),
    actions: {
        clearMenuArr() {
            this.menuArr[1].children = reactive<object[]>([]);
        },
        updatemenuArr(obj: any) {
            // 检查是否已存在相同 ID 的 session，避免重复添加
            const exists = this.menuArr[1].children?.some((item: any) => item.id === obj.id);
            if (!exists) {
                this.menuArr[1].children?.push(obj);
            }
        },
        updataMenuChildren(item: object) {
            this.menuArr[1].children?.unshift(item)
        },
        updatasessionTitle(session_id: string, title: string) {
            this.menuArr[1].children?.forEach((item: any) => {
                if (item.id == session_id) {
                    item.title = title;
                    item.isNoTitle = false;
                }
            });
        },
        changeIsFirstSession(payload: boolean) {
            this.isFirstSession = payload;
        },
        changeFirstQuery(payload: string) {
            this.firstQuery = payload;
        }
    }
});


