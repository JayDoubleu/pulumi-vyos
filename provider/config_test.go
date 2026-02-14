package provider

import "testing"

func TestShouldSaveConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		c := Config{}
		if c.ShouldSaveConfig() {
			t.Error("nil SaveConfig should return false")
		}
	})

	t.Run("false", func(t *testing.T) {
		f := false
		c := Config{SaveConfig: &f}
		if c.ShouldSaveConfig() {
			t.Error("false SaveConfig should return false")
		}
	})

	t.Run("true", func(t *testing.T) {
		tr := true
		c := Config{SaveConfig: &tr}
		if !c.ShouldSaveConfig() {
			t.Error("true SaveConfig should return true")
		}
	})
}
