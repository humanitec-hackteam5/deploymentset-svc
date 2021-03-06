
# deploymentset-svc

`deploymentset-svc` provides core manipulations required to use Deployment Sets.


## Configuration
It takes the following environment variables:

| Variable | Description |
|---|---|
| `DATABASE_NAME` | The name of the Postgress DB to connect to. |
| `DATABASE_USER` | The userame that the service should access the database under. |
| `DATABASE_PASSWORD` | The password associated with the useranme. |
| `DATABASE_HOST` | The DNS name or IP address that the databse server resides on. |
| `DATABASE_PORT` | The port on the server that the database is listening on. It defaults to `5432`.|
| `PORT` | The port number the server should be exposed on. It defaults to `8080`. |

## Supported endpoints

| Method | Path Template | Description |
| --- | --- | ---|
| `GET` | `/orgs/{orgId}/apps/{appId}/sets` | List of all Deployment Sets for the specified app. (Sets are wrapped.)|
| `GET` | `/orgs/{orgId}/apps/{appId}/sets/{setId}` | A specific deployment set for an app. (Set is wrapped.) |
| `POST` | `/orgs/{orgId}/apps/{appId}/sets/{setId}` | Create a new deployment set by applying a Deployment delta. (`setId` can be `0` to indicate the null set.) - Delta should be provided as body and should not be wrapped. |
| `GET` | `/orgs/{orgId}/apps/{appId}/sets/{leftSetId}?diff={rightSetId}` | Generate a Delta that defines how to get from the right set to the left set. (i.e. `POST` `/orgs/{orgId}/apps/{appId}/sets/{rightSetId}` with the returned Delta returns `leftSetId`.) |
| `GET` | `/orgs/{orgId}/apps/{appId}/deltas` | Lists all Deltas for an app, |
| `POST` | `/orgs/{orgId}/apps/{appId}/deltas` | Creates a new delta, returns a unique ID. |
| `GET` | `/orgs/{orgId}/apps/{appId}/deltas/{deltaId}` | Fetches a particular delta. |
| `PUT` | `/orgs/{orgId}/apps/{appId}/deltas/{deltaId}` | Replaces the content of a delta with a new delta. |
| `PATCH` | `/orgs/{orgId}/apps/{appId}/deltas/{deltaId}` | Applies an array of deltas to a current delta. See [Updating a Delta](doc/user-guide.md#updating-a-delta) |

## Running locally

The service can be built with:

    $ go build humanitec.io/deploymentset-svc/cmd/depset

Tests can be run with:

    $ go test humanitec.io/deploymentset-svc/cmd/depset \
	    humanitec.io/deploymentset-svc/pkg/depset

Mock for the `humanitec.io/deploymentset-svc/cmd/depset` tests can be regenerated with:

    $ mockgen -source=main.go -destination=modeler_mock.go -package=main modeler

## Testing with a database

The Go unit tests do not cover any of the database code. Tests on this can be run as follows:

Build the image and run it with docker-compose:

    $ docker-compose build && docker-compose up

Build the package and run the integration tests:

    $ cd tests/integration
    $ npm install
    $ node index.js

## Implementation Notes
The code is divided into a reusable package `humanitec.io/deploymentset-svc/pkg/depset` a command that provides the service endpoints itself.
### humanitec.io/deploymentset-svc/pkg/depset
This provides the three core deployment set manipulations:
| Operation | Description |
|---|---|
| Apply | Apply a Delta to a Deployment Set, generating a new Deployment Set |
| Diff | Generate a Delta describing how to get from one Deployment Set to another. |
| Hash | Generate an invariant ID from a deployment set. |

It provides one operation for merging Deltas:
| Operation | Description |
|---|---|
| MergeDeltas | Combines 2 or more Deltas into a single Delta |


### humanitec.io/deploymentset-svc/cmd/depset
Provides the command that actually runs the server serving the REST endpoints.
