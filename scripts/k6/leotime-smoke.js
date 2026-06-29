import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: Number(__ENV.K6_VUS || 10),
  duration: __ENV.K6_DURATION || '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const baseUrl = __ENV.BASE_URL || 'http://127.0.0.1:8080';

export default function () {
  const health = http.get(`${baseUrl}/api/health`);
  check(health, {
    'health is 200': (response) => response.status === 200,
  });

  const session = http.get(`${baseUrl}/api/v1/session`);
  check(session, {
    'session is 200': (response) => response.status === 200,
  });

  const metrics = http.get(`${baseUrl}/metrics`);
  check(metrics, {
    'metrics is 200': (response) => response.status === 200,
  });

  sleep(1);
}

