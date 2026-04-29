//go:build mongo

package checker

func registerMongo(r *Registry) {
	r.Register(&MongoChecker{})
}
