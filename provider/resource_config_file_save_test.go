package provider

import "testing"

func TestMakeConfigSaveID(t *testing.T) {
	t.Parallel()

	t.Run("deterministic", func(t *testing.T) {
		triggers := map[string]string{"a": "1", "b": "2"}
		ts := "2026-02-14T00:00:00Z"
		id1 := makeConfigSaveID(triggers, ts)
		id2 := makeConfigSaveID(triggers, ts)
		if id1 != id2 {
			t.Errorf("same inputs produced different IDs: %s vs %s", id1, id2)
		}
	})

	t.Run("different triggers", func(t *testing.T) {
		ts := "2026-02-14T00:00:00Z"
		id1 := makeConfigSaveID(map[string]string{"a": "1"}, ts)
		id2 := makeConfigSaveID(map[string]string{"a": "2"}, ts)
		if id1 == id2 {
			t.Error("different triggers should produce different IDs")
		}
	})

	t.Run("different timestamps", func(t *testing.T) {
		triggers := map[string]string{"a": "1"}
		id1 := makeConfigSaveID(triggers, "2026-02-14T00:00:00Z")
		id2 := makeConfigSaveID(triggers, "2026-02-14T00:00:01Z")
		if id1 == id2 {
			t.Error("different timestamps should produce different IDs")
		}
	})

	t.Run("nil triggers", func(t *testing.T) {
		id := makeConfigSaveID(nil, "2026-02-14T00:00:00Z")
		if id == "" {
			t.Error("nil triggers should still produce an ID")
		}
	})

	t.Run("key order independent", func(t *testing.T) {
		ts := "2026-02-14T00:00:00Z"
		id1 := makeConfigSaveID(map[string]string{"a": "1", "b": "2", "c": "3"}, ts)
		id2 := makeConfigSaveID(map[string]string{"c": "3", "a": "1", "b": "2"}, ts)
		if id1 != id2 {
			t.Errorf("key order should not matter: %s vs %s", id1, id2)
		}
	})
}
