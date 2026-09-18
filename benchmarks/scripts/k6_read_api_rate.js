import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const status200 = new Counter('status_200');
const status429 = new Counter('status_429');
const statusOther = new Counter('status_other');

const scenario = (rate, startTime) => ({
  executor: 'constant-arrival-rate',
  rate,
  timeUnit: '1s',
  duration: '15s',
  preAllocatedVUs: Math.max(2, Math.ceil(rate / 20)),
  maxVUs: Math.max(10, Math.ceil(rate / 5)),
  startTime,
  tags: { target_rate: String(rate) },
});

export const options = {
  scenarios: {
    api_10rps: scenario(10, '0s'),
    api_100rps: scenario(100, '20s'),
    api_300rps: scenario(300, '40s'),
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  const response = http.get(`${__ENV.LORELATTICE_HOST || 'http://localhost'}/api/v1/knowledge-bases`, {
    headers: { Authorization: `Bearer ${__ENV.LORELATTICE_TOKEN}` },
  });
  if (response.status === 200) status200.add(1);
  else if (response.status === 429) status429.add(1);
  else statusOther.add(1);
  check(response, { 'status is 200': (r) => r.status === 200 });
}

export function handleSummary(data) {
  return { [__ENV.K6_SUMMARY_PATH || 'k6-rate-summary.json']: JSON.stringify(data, null, 2) };
}
