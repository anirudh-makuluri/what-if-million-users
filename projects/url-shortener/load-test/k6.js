import http from "k6/http";
import { check, sleep } from "k6";

const baseUrl = __ENV.BASE_URL || "http://localhost:8083";
const vus = Number(__ENV.VUS || 500);
const duration = __ENV.DURATION || "30s";

export const options = {
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<300"],
  },
  stages: [
	{ duration: "10s", target: vus },
	{ duration: "15s", target: vus * 1.5},
	{ duration: "15s", target: vus * 2 },
	{ duration: "15s", target: vus * 2.5 },
  ],
};

export function setup() {
  return {
    runId: Date.now(),
  };
}

export default function (data) {
  const shortCode = `k6-${data.runId}-vu${__VU}-iter${__ITER}`;
  const longUrl = "https://google.com";

  const payload = JSON.stringify({
    short_code: shortCode,
    long_url: longUrl,
  });

  const shortenResponse = http.post(`${baseUrl}/shorten`, payload, {
    headers: { "Content-Type": "application/json" },
    tags: { name: "POST /shorten" },
  });

  check(shortenResponse, {
    "shorten returned 200": (r) => r.status === 200,
    "no duplicate 409s": (r) => r.status !== 409,
  });

  if (shortenResponse.status !== 200) {
    return;
  }

  const response = http.get(`${baseUrl}/${shortCode}`, {
    redirects: 0,
    tags: { name: "GET /:shortCode" },
  });

  check(response, {
    "redirect returned 301": (r) => r.status === 301,
    "location header matches": (r) => r.headers.Location === longUrl,
  });

}
