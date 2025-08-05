import { get, setTestData } from '../../utils/request';

export interface TestDataResponse {
  success: boolean;
  data: {
    tenant: {
      id: number;
      name: string;
      api_key: string;
    };
    knowledge_bases: Array<{
      id: string;
      name: string;
      description: string;
    }>;
  }
}

// 是否已加载测试数据
let isTestDataLoaded = false;

/**
 * 加载测试数据
 * 在API调用前调用此函数以确保测试数据已加载
 * @returns Promise<boolean> 是否成功加载
 */
export async function loadTestData(): Promise<boolean> {
  // 如果已经加载过，直接返回
  if (isTestDataLoaded) {
    return true;
  }

  try {
    console.log('开始加载测试数据...');
    const response = await get('/api/v1/test-data');
    console.log('测试数据', response);
    
    if (response && response.data) {
      // 设置测试数据
      setTestData({
        tenant: response.data.tenant,
        knowledge_bases: response.data.knowledge_bases
      });
      isTestDataLoaded = true;
      console.log('测试数据加载成功');
      return true;
    } else {
      console.warn('测试数据响应为空');
      return false;
    }
  } catch (error) {
    console.error('加载测试数据失败:', error);
    return false;
  }
} 
