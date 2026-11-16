package qemuimg

import "testing"

func TestOptionsOrderingAndString(t *testing.T) {
	options := NewOptions(
		Option{Key: "backing_file", Value: "/base"},
		Option{Key: "backing_fmt", Value: "qcow2"},
	)
	if got, want := options.String(), "backing_file=/base,backing_fmt=qcow2"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got := options.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
}

func TestOptionsSetUpdatesInPlace(t *testing.T) {
	options := NewOptions(
		Option{Key: "a", Value: "1"},
		Option{Key: "b", Value: "2"},
	)
	returned := options.Set("a", "9")
	if returned != options {
		t.Fatal("Set() did not return the receiver")
	}
	if got, want := options.String(), "a=9,b=2"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	options.Set("c", "3")
	if got, want := options.String(), "a=9,b=2,c=3"; got != want {
		t.Fatalf("String() after append = %q, want %q", got, want)
	}
}

func TestOptionsGetHasDelete(t *testing.T) {
	options := NewOptions(Option{Key: "a", Value: "1"}, Option{Key: "b", Value: "2"})

	value, ok := options.Get("a")
	if !ok || value != "1" {
		t.Fatalf("Get(a) = (%q, %v), want (1, true)", value, ok)
	}
	if !options.Has("b") {
		t.Fatal("Has(b) = false, want true")
	}

	options.Delete("a")
	if options.Has("a") {
		t.Fatal("Has(a) = true after Delete")
	}
	if got, want := options.String(), "b=2"; got != want {
		t.Fatalf("String() after Delete = %q, want %q", got, want)
	}

	options.Delete("missing")
	if got := options.Len(); got != 1 {
		t.Fatalf("Len() after deleting missing key = %d, want 1", got)
	}
}

func TestOptionsClone(t *testing.T) {
	var nilOptions *Options
	if got := nilOptions.Clone(); got != nil {
		t.Fatalf("nil Clone() = %#v, want nil", got)
	}

	original := NewOptions(Option{Key: "a", Value: "1"})
	clone := original.Clone()
	clone.Set("a", "2")
	clone.Set("b", "3")

	if got, want := original.String(), "a=1"; got != want {
		t.Fatalf("original String() = %q, want %q", got, want)
	}
	if got, want := clone.String(), "a=2,b=3"; got != want {
		t.Fatalf("clone String() = %q, want %q", got, want)
	}
}

func TestOptionsFromMapSorted(t *testing.T) {
	options := OptionsFromMap(map[string]string{
		"zeta":  "z",
		"alpha": "a",
		"mu":    "m",
	})
	if got, want := options.String(), "alpha=a,mu=m,zeta=z"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestOptionsMap(t *testing.T) {
	options := NewOptions(Option{Key: "a", Value: "1"}, Option{Key: "b", Value: "2"})
	m := options.Map()
	if len(m) != 2 || m["a"] != "1" || m["b"] != "2" {
		t.Fatalf("Map() = %#v, want a=1 b=2", m)
	}
	if got := (*Options)(nil).Map(); got == nil || len(got) != 0 {
		t.Fatalf("nil Map() = %#v, want empty non-nil map", got)
	}
}

func TestOptionsEmptyKeysIgnored(t *testing.T) {
	options := NewOptions(
		Option{Key: "", Value: "ignored"},
		Option{Key: "a", Value: "1"},
	)
	if got, want := options.String(), "a=1"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got := len(options.Map()); got != 1 {
		t.Fatalf("Map() length = %d, want 1", got)
	}

	var nilOptions *Options
	if got := nilOptions.String(); got != "" {
		t.Fatalf("nil String() = %q, want empty", got)
	}
	if got := NewOptions().String(); got != "" {
		t.Fatalf("empty String() = %q, want empty", got)
	}
}
