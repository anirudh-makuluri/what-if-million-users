import http from 'k6/http';
import { Counter } from 'k6/metrics';
import { check, group } from 'k6';

const rateLimited200 = new Counter('api_request_200');
const rateLimited429 = new Counter('api_request_429');
const rateLimitedOther = new Counter('api_request_other');

export let options = {
  stages: [
    { duration: '20s', target: 100 },  // Ramp-up to 100 users
    { duration: '20s', target: 100 },  // Hold at 100 users
    { duration: '20s', target: 200 },  // Ramp-up to 200 users
    { duration: '30s', target: 200 },  // Hold at 200 users
    { duration: '10s', target: 0 },    // Ramp-down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<500'],
    'http_req_duration{staticAsset:yes}': ['p(95)<100'],
  },
};

const API_URL = __ENV.API_URL || 'http://localhost:8080';

export default function () {
  let clientId = `client-${__VU}`;

  group('Rate Limited Request', function () {
    let payload = JSON.stringify({
      client_id: clientId,
      action: 'api_request',
    });

    let params = {
      headers: {
        'Content-Type': 'application/json',
      },
    };

    let response = http.post(`${API_URL}/api/request`, payload, params);

    if (response.status === 200) {
      rateLimited200.add(1);
    } else if (response.status === 429) {
      rateLimited429.add(1);
    } else {
      rateLimitedOther.add(1);
    }

    check(response, {
      'is status 200 or 429': (r) => r.status === 200 || r.status === 429,
      'response has body': (r) => r.body && r.body.length > 0,
      'response time < 100ms': (r) => r.timings.duration < 100,
    });
  });

  group('Health Check', function () {
    let response = http.get(`${API_URL}/health`);
    check(response, {
      'health is 200': (r) => r.status === 200,
    });
  });
}
