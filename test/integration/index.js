const fetch = require('node-fetch');

const baseURL = "http://localhost:8080"

function POST(url, body) {
  return fetch(baseURL + url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
}

function PATCH(url, body) {
  return fetch(baseURL + url, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
}

function GET(url) {
  return fetch(baseURL + url)
}

function CheckHttpOK(res) {
  if (res.ok) {
    return res.json();
  }
  console.log(JSON.stringify(res))
  throw `Expected Success for ${res.url}. Got ${res.status}`;
}

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

POST(`/orgs/my-org/apps/my-app/sets/0`, startDelta)
  .then(CheckHttpOK)

  .then(id => GET(`/orgs/my-org/apps/my-app/sets/${id}`))
  .then(CheckHttpOK)

  .catch(err => console.log(`FAIL: ${err}`))
  .then(() => console.log(`SUCCESS: Create Set past`), err => console.log(`FAIL: ${err}`));

POST(`/orgs/my-org/apps/my-app/deltas`, startDelta)
  .then(CheckHttpOK)

  .then(id => GET(`/orgs/my-org/apps/my-app/deltas/${id}`))
  .then(CheckHttpOK)

  .then(() => console.log(`SUCCESS: Create Delta past`), err => console.log(`FAIL: ${err}`));

POST(`/orgs/my-org/apps/my-app/deltas`, startDelta)
  .then(CheckHttpOK)

  .then(id => PATCH(`/orgs/my-org/apps/my-app/deltas/${id}`, updateDeltas))
  .then(CheckHttpOK)

  .then(dw => GET(`/orgs/my-org/apps/my-app/deltas/${dw.id}`))
  .then(CheckHttpOK)
  .then(dw => {
    if (dw.content.modules.add["module-one"].configmap.NEW_VAR != "Hello!") {
      throw "Updated delta not retrieved!";
    }
  })

  .then(() => console.log(`SUCCESS: Update Delta past`), err => console.log(`FAIL: ${err}`));
