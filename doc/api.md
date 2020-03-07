# Deployment Sets API
## Overview

### Deployment Set Schema
A Deployment Set is made up of a series of modules or resources. Each module or resource has a unique name within
the Deployment Set. For example:

    {
      "modules": {
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

Deployment Sets have a property that their ID is a cryptographic hash of their content. This means that a Deployment Set
can be globally identified based on its ID. It also means that referencing a Deployment Set by ID will always return the
same deployment set.

### Deployment Delta Schema
A Deployment Delta defines 3 different actions can be applied to entities in a deployment set. These three actions are:

- `add`
- `remove`
- `update`

### Wrapped Entities
Deployment Sets and Deployment Deltas are returned from the API along with metadata (e.g. when they were created and
what their ID is. The structure of the wrapper is the same in both cases:

    {
      "id": "<ENITITY_ID>",
      "metadata": { <ENTITY_METADATA> },
      "content": { <ENTITY_CONTENT> }
    }

When data is sent to the API, e.g. via a POST, PUT or PATCH, then the raw entity should be sent.

## API

### Conventions

All payloads are expected to be JSON and so require the `Content-Type` header to be set to `application/json`

### GET /org/{orgId}/apps/{appId}/sets/{setId}

#### Description

Fetches the Deployment Set defined by the specific ID.

#### Returns

A Wrapped Deployment Set.

    {
      "id": "uf6OiM_uMN_xhOO9iYVCGULbLlQjPqc2y6wHyfy6eBQ",
      "metadata": {},
      "content": {
        "modules": {
          "module-one": {
            "profile": "humanitec/base-module",
            "image": "registry.humanitec.io/my-org/module-one:VERSION_ONE",
            "configmap": {
              "DBCONNECTION": "jdbc:postgresql://${dbs.prostgress.host}/${dbs.prostgress.name}?user=${dbs.prostgress.username}&password=${dbs.prostgress.password}",
              "REDIS_HOST": "${modules.redis-cache.service.name}"
            }
          },
          "redis-cache": {
            "profile": "humanitec/redis"
          }
        }
      }
    }

#### Status Codes

| Code | Description |
|--|--|
| 200 | Success |
| 404 | ID does not match a known Deployment Set |

### POST /org/{orgId}/apps/{appId}/sets/{setId}

#### Description

Applies a Deployment Delta to the specified Deployment Set.

#### Payload
A raw Deployment Delta

    {
      "modules": {
        "update": {
          "module-one": [
            { "op": "add", "path": "/configmap/NEW_KEY", "value": "new value!" }
          ]
        }
      }
    }

#### Returns

A Wrapped Deployment Set.

    {
      "id": "uf6OiM_uMN_xhOO9iYVCGULbLlQjPqc2y6wHyfy6eBQ",
      "metadata": {},
      "content": {
        "modules": {
          "module-one": {
            "profile": "humanitec/base-module",
            "image": "registry.humanitec.io/my-org/module-one:VERSION_ONE",
            "configmap": {
              "DBCONNECTION": "jdbc:postgresql://${dbs.prostgress.host}/${dbs.prostgress.name}?user=${dbs.prostgress.username}&password=${dbs.prostgress.password}",
              "REDIS_HOST": "${modules.redis-cache.service.name}",
              "NEW_KEY": "new value!"
            }
          },
          "redis-cache": {
            "profile": "humanitec/redis"
          }
        }
      }
    }

#### Status Codes

| Code | Description |
|--|--|
| 200 | Success |
| 400 | The Delta is not compatible with the Set |
| 404 | ID does not match a known Deployment Set |
| 422 | The Delta is malformed |

### GET /org/{orgId}/apps/{appId}/sets/{leftSetId}?diff={rightSetId}

#### Description

Returns the Deployment Delta that if applied to the Set with ID `{rightSetId}` would return the Set with ID `{leftSetId}`

#### Returns

A raw Deployment Delta

    {
      "modules": {
        "update": {
          "module-one": [
            { "op": "add", "path":"/configmap/NEW_KEY", "value": "new value!" }
          ]
        }
      }
    }

#### Status Codes

| Code | Description |
|--|--|
| 200 | Success |
| 404 | On or other of the IDs does not match a known Deployment Set |



### GET /org/{orgId}/apps/{appId}/deltas/{deltaId}

#### Description

Fetches the Deployment Delta defined by the specific ID.

#### Returns

A Wrapped Deployment Delta.

    {
      "id": "21942db2e54233ea736cbac07c9fcba78",
      "metadata": {
        "created_by": "user@example.com",
        "created_at": "2020-03-05T12:23:56Z",
        "collaborators": []
      },
      "content": {
        "modules": {
          "add": {
            "module-one": {
              "profile": "humanitec/base-module",
              "image": "registry.humanitec.io/my-org/module-one:VERSION_ONE",
              "configmap": {
                "DBCONNECTION": "jdbc:postgresql://${dbs.prostgress.host}/${dbs.prostgress.name}?user=${dbs.prostgress.username}&password=${dbs.prostgress.password}",
                "REDIS_HOST": "${modules.redis-cache.service.name}"
              }
            },
            "redis-cache": {
              "profile": "humanitec/redis"
            }
          }
        }
      }
    }

#### Status Codes

| Code | Description |
|--|--|
| 200 | Success |
| 404 | ID does not match a known Deployment Delta in the scope of this app and organization |

### POST /org/{orgId}/apps/{appId}/deltas

#### Description

Stores a new Deployment Delta. A unique ID, scoped to the app and org is generated.

#### Payload
A raw Deployment Delta

    {
      "modules": {
        "update": {
          "module-one": [
            { "op": "add", "path": "/configmap/NEW_KEY", "value": "new value!" }
          ]
        }
      }
    }
#### Returns

An ID.

    "6YTBKCDFBNWLSUEM7KONWLVQD4T7F2PAKA"

#### Status Codes

| Code | Description |
|--|--|
| 201 | Success |
| 422 | The Delta is malformed |

### PUT /org/{orgId}/apps/{appId}/deltas/{deltaId}

#### Description

Replaces an existing Deployment Delta with a new one. There is no requirement for the Deltas to be compatible.

The replacement gets treated as an edit so the `last_modified_at` is updated and `collaboriators` is updated if
necessary.

#### Payload
A raw Deployment Delta

    {
      "modules": {
        "update": {
          "module-other": [
            { "op": "add", "path": "/configmap/OTHER_KEY", "value": "different value!" }
          ]
        }
      }
    }
#### Returns

Empty Response.

#### Status Codes

| Code | Description |
|--|--|
| 200 | Success |
| 404 | ID does not match a known Deployment Delta in the scope of this app and organization |
| 422 | The Delta is malformed |

### PATCH /org/{orgId}/apps/{appId}/deltas/{deltaId}

#### Description

Merges one or more Deployment Deltas into the specified Delta

This allows for piecewise updating of a Deployment Delta. The end result is that a single Deployment Delta can be
applied having the same effect as applying the merged Delta.

#### Payload
An array of raw Deployment Delta

    [
      {
        "modules": {
          "update": {
            "module-other": [
              { "op": "add", "path": "/configmap/OTHER_KEY", "value": "different value!" }
            ]
          }
        }
      },
      {
        "modules": {
          "add": {
            "module-new": {
              "profile": "humanitec/redis"
            }
          }
        }
      }
    ]

#### Returns

A wrapped Deployment Delta

    {
      "id": "6YTBKCDFBNWLSUEM7KONWLVQD4T7F2PAKA",
      "metadata": {
        "created_by": "user@example.com",
        "created_at": "2020-03-05T12:23:56Z",
        "collaborators": []
      },
      "content": {
        "modules": {
          "add": {
            "module-new": {
              "profile": "humanitec/redis"
            }
          },
          "update": {
            "module-one": [
              { "op": "add", "path": "/configmap/NEW_KEY", "value": "new value!" }
            ],
            "module-other": [
              { "op": "add", "path": "/configmap/OTHER_KEY", "value": "different value!" }
            ]
          }
        }
      }
    }

#### Status Codes

| Code | Description |
|--|--|
| 200 | Success |
| 400 | Deltas could not be merged as they are incompatible |
| 404 | ID does not match a known Deployment Delta in the scope of this app and organization |
| 422 | The Delta is malformed |
