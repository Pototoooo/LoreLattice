import { ref, reactive, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { formatStringDate, kbFileTypeVerification } from "../utils/index";
import { MessagePlugin } from "tdesign-vue-next";
import {
  uploadKnowledgeFile,
  listKnowledgeFiles,
  getKnowledgeDetails,
  delKnowledgeDetails,
  getKnowledgeDetailsCon,
} from "@/api/knowledge-base/index";
import { knowledgeStore } from "@/stores/knowledge";
import { useRoute } from 'vue-router';
const usemenuStore = knowledgeStore();
export default function (knowledgeBaseId?: string) {
  const route = useRoute();
  const { cardList, total } = storeToRefs(usemenuStore);
  let moreIndex = ref(-1);
  const details = reactive({
    title: "",
    time: "",
    md: [] as any[],
    id: "",
    total: 0
  });
  const getKnowled = (query = { page: 1, page_size: 35 }) => {
    if (!knowledgeBaseId) return;
    listKnowledgeFiles(knowledgeBaseId, query)
      .then((result: any) => {
        let { data, total: totalResult } = result;
        let cardList_ = data.map((item: any) => {
            item["file_name"] = item.file_name.substring(
              0,
              item.file_name.lastIndexOf(".")
            );
            return {
              ...item,
              updated_at: formatStringDate(new Date(item.updated_at)),
              isMore: false,
              file_type: item.file_type.toLocaleUpperCase(),
            };
        });
        if (query.page == 1) {
          cardList.value = cardList_ as any[];
        } else {
          (cardList.value as any[]).push(...cardList_);
        }
        total.value = totalResult;
      })
      .catch((_err) => {});
  };
  const delKnowledge = (index: number, item: any) => {
    (cardList.value as any[])[index].isMore = false;
    moreIndex.value = -1;
    delKnowledgeDetails(item.id)
      .then((result: any) => {
        if (result.success) {
          MessagePlugin.info("知识删除成功！");
          getKnowled();
        } else {
          MessagePlugin.error("知识删除失败！");
        }
      })
      .catch((_err) => {
        MessagePlugin.error("知识删除失败！");
      });
  };
  const openMore = (index: number) => {
    moreIndex.value = index;
  };
  const onVisibleChange = (visible: boolean) => {
    if (!visible) {
      moreIndex.value = -1;
    }
  };
  const requestMethod = (file: any, uploadInput: any) => {
    if (file instanceof File && uploadInput) {
      if (kbFileTypeVerification(file)) {
        return;
      }
      // 每次上传时动态解析当前 kbId（优先：路由 -> URL -> 初始参数）
      let currentKbId: string | undefined;
      try {
        currentKbId = (route.params as any)?.kbId as string;
      } catch {}
      if (!currentKbId && typeof window !== 'undefined') {
        try {
          const match = window.location.pathname.match(/knowledge-bases\/([^/]+)/);
          if (match && match[1]) currentKbId = match[1];
        } catch {}
      }
      if (!currentKbId) {
        currentKbId = knowledgeBaseId;
      }
      if (!currentKbId) {
        MessagePlugin.error("缺少知识库ID");
        return;
      }
      uploadKnowledgeFile(currentKbId, { file })
        .then((result: any) => {
          if (result.success) {
            MessagePlugin.info("上传成功！");
            getKnowled();
          } else {
            // 改进错误信息提取逻辑
            let errorMessage = "上传失败！";
            if (result.error && result.error.message) {
              errorMessage = result.error.message;
            } else if (result.message) {
              errorMessage = result.message;
            }
            if (result.code === 'duplicate_file' || (result.error && result.error.code === 'duplicate_file')) {
              errorMessage = "文件已存在";
            }
            MessagePlugin.error(errorMessage);
          }
          uploadInput.value.value = "";
        })
        .catch((err: any) => {
          let errorMessage = "上传失败！";
          if (err.code === 'duplicate_file') {
            errorMessage = "文件已存在";
          } else if (err.error && err.error.message) {
            errorMessage = err.error.message;
          } else if (err.message) {
            errorMessage = err.message;
          }
          MessagePlugin.error(errorMessage);
          uploadInput.value.value = "";
        });
    } else {
      MessagePlugin.error("file文件类型错误！");
    }
  };
  const getCardDetails = (item: any) => {
    Object.assign(details, {
      title: "",
      time: "",
      md: [],
      id: "",
    });
    getKnowledgeDetails(item.id)
      .then((result: any) => {
        if (result.success && result.data) {
          let { data } = result;
          Object.assign(details, {
            title: data.file_name,
            time: formatStringDate(new Date(data.updated_at)),
            id: data.id,
          });
        }
      })
      .catch((_err) => {});
      getfDetails(item.id, 1);
  };
  const getfDetails = (id: string, page: number) => {
    getKnowledgeDetailsCon(id, page)
      .then((result: any) => {
        if (result.success && result.data) {
          let { data, total: totalResult } = result;
          if (page == 1) {
            (details.md as any[]) = data;
          } else {
            (details.md as any[]).push(...data);
          }
          details.total = totalResult;
        }
      })
      .catch((_err) => {});
  };
  return {
    cardList,
    moreIndex,
    getKnowled,
    details,
    delKnowledge,
    openMore,
    onVisibleChange,
    requestMethod,
    getCardDetails,
    total,
    getfDetails,
  };
}
