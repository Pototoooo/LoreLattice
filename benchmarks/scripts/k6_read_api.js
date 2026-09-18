import http from 'k6/http';
import { check } from 'k6';

const host = __ENV.LORELATTICE_HOST || 'http://localhost';
const token = __ENV.LORELATTICE_TOKEN;

export const options = {
  scenarios: {
    api_vu1: { executor: 'constant-vus', vus: 1, duration: '15s', tags: { load: 'vu1' } },
    api_vu10: { executor: 'constant-vus', vus: 10, duration: '15s', startTime: '20s', tags: { load: 'vu10' } },
    api_vu50: { executor: 'constant-vus', vus: 50, duration: '15s', startTime: '40s', tags: { load: 'vu50' } },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  const response = http.get(`${host}/api/v1/knowledge-bases`, {
    headers: { Authorization: `Bearer ${token}` },
    tags: { endpoint: 'knowledge_bases_list' },
  });
  check(response, {
    'status is 200': (r) => r.status === 200,
    'JSON success is true': (r) => r.json('success') === true,
  });
}

export function handleSummary(data) {
  return { [__ENV.K6_SUMMARY_PATH || 'k6-summary.json']: JSON.stringify(data, null, 2) };
}
