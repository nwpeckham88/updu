//go:build !mongo

package checker

// registerMongo is a no-op when the binary is built without the `mongo` build
// tag. The MongoChecker stub type still exists so shared tests and config
// validation compile, but the type is not exposed to users via the registry.
func registerMongo(r *Registry) {}
