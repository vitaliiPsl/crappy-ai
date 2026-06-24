package utils

import "testing"

func TestParseFloat32Ptr(t *testing.T) {
	got, err := ParseFloat32Ptr(" 0.25 ")
	if err != nil {
		t.Fatalf("ParseFloat32Ptr: %v", err)
	}

	if got == nil || *got != 0.25 {
		t.Fatalf("value = %v, want 0.25", got)
	}
}

func TestParseFloat32PtrEmpty(t *testing.T) {
	got, err := ParseFloat32Ptr("")
	if err != nil {
		t.Fatalf("ParseFloat32Ptr: %v", err)
	}

	if got != nil {
		t.Fatalf("value = %v, want nil", got)
	}
}

func TestParseFloat32PtrRejectsNonfinite(t *testing.T) {
	if _, err := ParseFloat32Ptr("NaN"); err == nil {
		t.Fatal("error = nil, want error")
	}
}

func TestParseNonnegativeInt32Ptr(t *testing.T) {
	got, err := ParseNonnegativeInt32Ptr(" 1024 ")
	if err != nil {
		t.Fatalf("ParseNonnegativeInt32Ptr: %v", err)
	}

	if got == nil || *got != 1024 {
		t.Fatalf("value = %v, want 1024", got)
	}
}

func TestParseNonnegativeInt32PtrEmpty(t *testing.T) {
	got, err := ParseNonnegativeInt32Ptr("")
	if err != nil {
		t.Fatalf("ParseNonnegativeInt32Ptr: %v", err)
	}

	if got != nil {
		t.Fatalf("value = %v, want nil", got)
	}
}

func TestParseNonnegativeInt32PtrRejectsNegative(t *testing.T) {
	if _, err := ParseNonnegativeInt32Ptr("-1"); err == nil {
		t.Fatal("error = nil, want error")
	}
}
