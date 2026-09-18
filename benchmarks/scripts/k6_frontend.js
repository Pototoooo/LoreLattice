import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    ui_vu1: { executor: 'constant-vus', vus: 1, duration: '10s' },
    ui_vu10: { executor: 'constant-vus', vus: 10, duration: '10s', startTime: '15s' },
    ui_vu50: { executor: 'constant-vus', vus: 50, duration: '10s', startTime: '30s' },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200'],
  },
};

export default function () {
  const response = http.get(`${__ENV.LORELATTICE_HOST || 'http://localhost'}/`);
  check(response, {
    'status is 200': (r) => r.status === 200,
    'HTML shell returned': (r) => r.body.includes('<div id="app"></div>'),
  });
}

export function handleSummary(data) {
  return { [__ENV.K6_SUMMARY_PATH || 'k6-summary.json']: JSON.stringify(data, null, 2) };
}
