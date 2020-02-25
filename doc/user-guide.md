
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
            { "op": "replace", "path": "version", "value": "1.0.1" }
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
        "version": "1.0.0",
        "configMap": {
          "HELLO": "World!"
        }
      },
      "module-two": {
        "version": "2.0.0",
        "ingress": true,
        "container_port": "8080"
      }
    }

Our Delta would look as follows:

    {
      "modules": {
        "add": {
          "module-one": {
            "version": "1.0.0",
            "configMap": {
              "HELLO": "World!"
            }
          },
          "module-two": {
            "version": "2.0.0",
            "ingress": true,
            "container_port": "8080"
          }
        }
      }
    }

It would be applied as follows:

    POST /orgs/my-org/apps/my-app/sets/0

The _Empty Set_ always has an ID value of `0`.

### Generating a delta between 2 Sets
