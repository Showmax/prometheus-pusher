package main

import "testing"

func TestResources(t *testing.T) {
	grm := newRouteMap("test/routes", "test")
	c, _ := parseConfig(cfgTest)
	var r *resources
	t.Run("create", func(t *testing.T) {
		r = createResources(c, grm)
	})
	t.Run("run", func(t *testing.T) {
		<-r.run()
	})
	t.Run("process", func(t *testing.T) {
		r.process()
	})
	t.Run("shutdown", func(t *testing.T) {
		r.shutdown()
	})
	t.Run("stop", func(t *testing.T) {
		<-r.stop()
	})

}
