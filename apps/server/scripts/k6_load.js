import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '20s', target: 100 }, // 在此阶段可尝试向 server 发 SIGTERM 验证零报错优雅停机
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/samples?page=1&page_size=10`);
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
  sleep(0.1);
}
