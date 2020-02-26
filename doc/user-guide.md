



# Deployment Sets: A User Guide

## Definitions
### Deployment Set ("Set")
A Deployment Set (or "Set" for short) represents all the _non-environment specific_ configuration for an application. This allows _the same_ Deployment Set to be deployed to different environments.
A Deployment Set includes amongst other things:
 - modules and their versions,
 - whether ingress is configured,
 - static and dynamic configuration (i.e. configuration that is the same between environments and those containing variable substitutions.)
 - static secrets (i.e. secrets that are the same between environments.)
 - whether a database is required for a module.

It does not include:
- The generated DNS name for a module that has ingress defined,
- the name or credentials for a database,
- any cluster information.

Deployment Sets are _Immutable_. This means that the ID generated for a Deployment Set always uniquely described that exact Deployment Set. Changes to a Deployment Set can be applied using Deployment Deltas.

### Deployment Delta ("Delta")
A Deployment Delta (or "Delta" for short) describes the changes that should be made to convert from one Set to another.
For example, given an initial Set that looks like:

    {
      "module-one": {
        "version": "1.0.0"
      },
      "module-two": {
        "version": "2.0.0"
      }
    }
  And a target Set that should look like:

    {
      "module-one": {
        "version": "1.0.1"
      },
      "module-three": {
        "version": "3.0.0"
      }
    }

The Delta describing how to convert the first Set to the second Set would be:

    {
      "modules": {
        add: {
          "module-three": {
            "version": "3.0.0"
          }
        },
        remove: [
          "module-two"
        ],
        "update": {
          "module-one": [
            { "op": "replace", "path": "/version", "value": "1.0.1" }
          ]
        }
      }
    }

The above Delta describes the following:
A new module (`module-three`) is added, `module-two` is removed and the version of `module-one` is changed from `1.0.0` to `1.0.1`.

## Worked Examples
For a full description of the API, please refer to:

For all of these examples, we will assume the organisation in question is: `my-org` and the app is `my-app`.

### Creating a new Deployment Set from nothing
You will notice that the API has no way to directly create a new Set. Instead, new Sets must be created by "applying" a Delta to an existing Set. There is a special case of a Deployment Set known as the _Empty Set_. This Set contains nothing at all - it is empty. It also provides a convenient way to bootstrap new Sets. We can simply apply a Delta to the _Empty Set_ in order to generate whatever initial Set we wish.
A useful property of Deployment Deltas is that a Delta containing only an `add` section that is applied to the _Empty Set_ will generate a Set which is equal to the content of the `add` section.
If  we wish to generate the following Set:

    {
      "module-one": {
        "module": {
          "name": "module-one",
          "version": "1.0.0"
        }
        "configmap": {
          "HELLO": "World!"
        }
      },
      "module-two": {
        "module": {
          "name": "module-one",
          "version": "2.0.0"
        }
        "ingress": {
              "enabled": true
            },
            "container": {
              "port": "8080"
            }
          }
        }

Our Delta would look as follows:

    {
      "modules": {
        "add": {
          "module-one": {
            "module": {
              "name": "module-one",
              "version": "1.0.0"
            }
            "configmap": {
              "HELLO": "World!"
            }
          },
          "module-two": {
            "module": {
              "name": "module-one",
              "version": "2.0.0"
            }
            "ingress": {
              "enabled": true
            },
            "container": {
              "port": "8080"
            }
          }
        }
      }
    }

It would be applied as follows:

    POST /orgs/my-org/apps/my-app/sets/0

The _Empty Set_ always has an ID value of `0`. (Note, this can contain 1 or more zeros)

### Complex Updates using Deltas

The `update` action in a Delta is based on [jsonpatch](https://tools.ietf.org/html/rfc6902) This allows for arbitrary updates to be applied to arbitrary JSON. The `update` actions in Deltas support a subset of jsonpatch operations. Specifically:
| Operation | Description |
|--|--|
| `add` | Adds a new property into an object or a new value into an array. (If the property exists, the value will be replaced. An index of `-` indicates insertion at the end of the array.) |
| `remove` | Removes an existing property from an object or an existing value from an array. (If the property or index does not exist, nothing happens.) |
| `replace` | Removes the value of an existing property in an object or an existing value at a specific index an array. If the property or index does not exist, this is an error. |

The remaining operations of `copy`, `move` and `test` are not supported.

#### Example 1: Adding updating properties in a sub object

Start Set:

    {
      "module-one": {
        "version": "1.0.0",
        "configmap": {
          "HELLO": "World!",
          "UNWANTED_KEY": "Unwanted Value!",
          "KEY": "Value"
        }
      }
    }

Target Set:

    {
      "module-one": {
        "version": "1.0.0",
        "configmap": {
          "HELLO": "Alice!",
          "NEW_KEY": "New Value!",
          "KEY": "Value"
        }
      }
    }

Delta:

    {
      "modules" {
        "module-one": {
          "update": [
            {"op": "add", "path": "/configmap/NEW_KEY", "value": "New Value!"},
            {"op": "remove", "path": "/configmap/UNWANTED_KEY"},
            {"op": "replace", "path": "/configmap/HELLO", "value": "Alice!"}
          ]
        }
      }
    }

This approach allows for only making specific updates to a sub object. Notice how the `KEY` property and value remain untouched by the above delta.

### Generating a Delta between 2 Sets

Deltas can also describe the differences between two Sets.

It would be applied as follows:

To find the difference between set `A` and `B`, use the following:

    POST /orgs/my-org/apps/my-app/sets/A?diff=B

This will return the Delta that if applied to `B` would give `A`.
