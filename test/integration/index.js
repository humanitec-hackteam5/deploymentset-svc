const fetch = require('node-fetch');

const baseURL = "http://localhost:8080";

var showDiagnostics = false

function POST(url, body) {
  if (showDiagnostics) { console.log(`POST ${url}`) }
  return fetch(baseURL + url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
}

function PATCH(url, body) {
  if (showDiagnostics) { console.log(`PATCH ${url}`) }
  return fetch(baseURL + url, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
}

function GET(url) {
  if (showDiagnostics) { console.log(`GET ${url}`) }
  return fetch(baseURL + url);
}

function DELETE(url) {
  if (showDiagnostics) { console.log(`DELETE ${url}`) }
  return fetch(baseURL + url, { method: 'DELETE'});
}

function CheckHttpOK(res) {
  if (res.ok) {
    if (res.headers.get("Content-Type") && res.headers.get("Content-Type").startsWith("application/json")) {
      return res.json();
    }
    return res.text();
  }
  throw `Expected Success for ${res.url}. Got ${res.status}`;
}

function CheckHttpStatus(status) {
  return function (res) {
    if (res.status === status) {
      if (res.headers.get("Content-Type") && res.headers.get("Content-Type").startsWith("application/json")) {
        return res.json();
      }
      return res.text();
    }
    throw `Expected status of ${status} for ${res.url}. Got ${res.status}`;
  };
}

// Create a random organization for our integration test
const orgId = "test-org-" + Math.floor(Math.random()*65536).toString(16)

const appId = "my-app";

const startDelta = {
  "modules": {
    "add": {
      "module-one": {
        "profile": "humanitec/base-module",
        "image": "registry.humanitec.io/my-org/module-one:VERSION_ONE",
        "configmap": {
          "DBNAME": "${dbs.prostgress.name}",
          "REDIS_HOST": "${modules.redis-cache.service.name}"
        }
      },
      "redis-cache": {
        "profile": "humanitec/redis"
      }
    }
  }
}

const updateDeltas = [
  {
    "modules": {
      "update": {
        "module-one": [
          {op: "replace", path: "/configmap/DBNAME", value: "HARDCODED_NAME" }
        ]
      }
    }
  },
  {
    "modules": {
      "update": {
        "module-one": [
          {op: "add", path: "/configmap/NEW_VAR", value: "Hello!" }
        ]
      }
    }
  }
];

POST(`/orgs/${orgId}/apps/${appId}/sets/0`, startDelta)
  .then(CheckHttpOK)

  .then(id => GET(`/orgs/${orgId}/apps/${appId}/sets/${id}`))
  .then(CheckHttpOK)
  .then(set => {
    if (!set.modules || !set.modules["module-one"] || set.modules["module-one"].profile !== "humanitec/base-module") {
      throw "Generated Set was not as expected."
    }
  })
  .then(id => GET(`/orgs/${orgId}/apps/${appId}/sets`))
  .then(CheckHttpOK)
  .then(sets => {
    if (sets.length !== 1) {
      throw `Expected 1 set in the app, got ${sets.length}`;
    }
    if (!sets[0].modules || !sets[0].modules["module-one"] || sets[0].modules["module-one"].profile !== "humanitec/base-module") {
      throw "Generated Set was not as expected.";
    }
  })

  .then(() => console.log(`SUCCESS: Create Set`), err => console.log(`FAIL: Create Set: ${err}`));

POST(`/orgs/${orgId}/apps/${appId}/deltas`, startDelta)
  .then(CheckHttpOK)

  .then(id => GET(`/orgs/${orgId}/apps/${appId}/deltas/${id}`))
  .then(CheckHttpOK)

  .then(() => console.log(`SUCCESS: Create Delta`), err => console.log(`FAIL: Create Delta: ${err}`));

POST(`/orgs/${orgId}/apps/${appId}/deltas`, startDelta)
  .then(CheckHttpOK)

  .then(id => PATCH(`/orgs/${orgId}/apps/${appId}/deltas/${id}`, updateDeltas))
  .then(CheckHttpOK)

  .then(dw => GET(`/orgs/${orgId}/apps/${appId}/deltas/${dw.id}`))
  .then(CheckHttpOK)
  .then(dw => {
    if (!dw.modules || !dw.modules.add || !dw.modules.add["module-one"] || !dw.modules.add["module-one"].configmap || dw.modules.add["module-one"].configmap.NEW_VAR != "Hello!") {
      throw "Updated delta not retrieved!";
    }
  })

  .then(() => console.log(`SUCCESS: Update Delta`), err => console.log(`FAIL: Update Delta: ${err}`));
