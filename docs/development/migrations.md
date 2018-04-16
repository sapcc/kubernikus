Migrations
==========
Kubernikus incorporates a simple migration system to update the internal representation of existing Klusters.
This can be used to update the CRD of a Kluster or update any other accompanying kubernetes resource in the control plane (e.g. Kluster Secret, Ingress, etc).

To create a migration a [Migration](https://github.com/sapcc/kubernikus/blob/master/pkg/migration/migration.go) function needs to be written and [registered](https://github.com/sapcc/kubernikus/blob/master/pkg/migration/register.go).

```
type Migration func(klusterRaw []byte, kluster *v1.Kluster, client kubernetes.Interface) (err error)
```
* `klusterRaw` contains the current raw JSON data of the Kluster, it can be used to access old data not exposed by the `v1.Kluster` type anymore.
* `kluster` is the Kluster which should be migrated. Modifications to this object are persisted if the function does not return an error
* `client` is a kubernetes client which can be used to modify other kubernetes resources belonging to the Kluster.

The migration controller takes care of invoking the defined function for every Kluster until it succeeds. When a migration is successfully executed the `Status.SpecVersion` is updated to reflect the migration has been applied to the respective Kluster.

### Pending migrations
As long as a kluster is not up to date with respect to `Status.SpecVersion` it will have the condition `Status.MigrationsPending` set to true.
Klusters with pending migrations are ignored by all controllers that modify the internal Kluster representation (e.g. update the CRD).
`Status.MigrationsPending` is updated during operator startup before any controller is running and continuously kept in sync by the migration controller.

**Controllers should call the `Disabled()` method on the `v1.Kluster` and exit early in reconciliation functions in case a Kluster is disabled.**
