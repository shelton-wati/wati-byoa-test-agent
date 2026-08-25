package webhook

import "testing"

func TestChannelTypeUnmarshal(t *testing.T) {
	var v ChannelTypeValue
	if err := v.UnmarshalJSON([]byte(`"INSTAGRAM"`)); err != nil {
		t.Fatal(err)
	}
	if v.String() != "INSTAGRAM" {
		t.Fatalf("got %q", v)
	}

	var v2 ChannelTypeValue
	if err := v2.UnmarshalJSON([]byte(`1`)); err != nil {
		t.Fatal(err)
	}
	if v2.String() != "INSTAGRAM" {
		t.Fatalf("got %q", v2)
	}
}
